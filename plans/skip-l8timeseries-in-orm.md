# Plan: Skip L8TimeSeriesPoint in ORM

## Background

`L8TimeSeriesPoint` is a protobuf type (`l8api.L8TimeSeriesPoint`) with two scalar fields: `Stamp` (int64) and `Value` (float64). Any model that tracks time series metrics has `[]*L8TimeSeriesPoint` slice fields (e.g., CPU usage, memory, latency).

Currently, the ORM treats `L8TimeSeriesPoint` like any other nested struct:
- Creates a PostgreSQL table named `L8TimeSeriesPoint`
- Writes individual rows per data point (linked to parent via ParentKey/RecKey)
- Reads rows back and reconstructs the slice

This is wrong for time series data — it should be handled by the TSDB service, not stored as relational rows. The ORM must **completely skip** any attribute whose `TypeName` is `"L8TimeSeriesPoint"`.

Note: `l8reflect` already has special handling for this type in its `Setter.go` and `StructComparator.go`, using the same `TypeName == "L8TimeSeriesPoint"` check pattern.

## Affected Locations

The ORM currently processes nested struct attributes (`IsStruct == true`) at these 4 code points. Each needs a skip guard for `L8TimeSeriesPoint`:

| # | File | Function | Line(s) | What Happens Today |
|---|------|----------|---------|--------------------|
| 1 | `go/orm/convert/ConvertTo.go` | `convertTo` | 137-166 | Iterates `subTableAttributes`, recursively converts each `L8TimeSeriesPoint` element into relational rows |
| 2 | `go/orm/convert/ConvertFrom.go` | `convertFrom` | 74-79 | Collects `L8TimeSeriesPoint` into `subTableAttributes` for reconstruction |
| 3 | `go/orm/convert/Utils.go` | `addTable` | 73-80 | Recursively creates table schema for `L8TimeSeriesPoint` |
| 4 | `go/orm/plugins/postgres/Postgres.go` | `collectTables` | 144-156 | Includes `L8TimeSeriesPoint` in table list for verification/creation |

## Implementation

### Helper Function

Add a `IsTimeSeriesType` function in `go/orm/convert/Utils.go` (or a new small file) that all 4 locations call:

```go
// IsTimeSeriesType returns true for types that are handled by the TSDB service
// and should be skipped by the relational ORM.
func IsTimeSeriesType(typeName string) bool {
    return typeName == "L8TimeSeriesPoint"
}
```

### Change 1: ConvertTo.go — Skip during write conversion

In `convertTo()`, when building `subTableAttributes` (line 118-121), add the guard:

```go
for attrName, attrNode := range node.Attributes {
    if attrNode.IsStruct {
        if IsTimeSeriesType(attrNode.TypeName) {
            continue  // TSDB data — skip, handled by TSDB service
        }
        subTableAttributes[attrName] = attrNode
        continue
    }
    // ... scalar processing unchanged
}
```

### Change 2: ConvertFrom.go — Skip during read conversion

In `convertFrom()`, when building `subTableAttributes` (line 74-79), add the same guard:

```go
for attrName, attrNode := range node.Attributes {
    if attrNode.IsStruct {
        if IsTimeSeriesType(attrNode.TypeName) {
            continue  // TSDB data — skip, handled by TSDB service
        }
        if !subAttributesFull {
            subTableAttributes[attrName] = attrNode
        }
        continue
    }
    // ... column processing unchanged
}
```

### Change 3: Utils.go — Skip during table schema creation

In `addTable()`, when recursing into struct attributes (line 73-80):

```go
for _, attr := range node.Attributes {
    if attr.IsStruct {
        if IsTimeSeriesType(attr.TypeName) {
            continue  // TSDB data — no relational table
        }
        err := addTable(attr, rlData)
        if err != nil {
            return err
        }
    }
}
```

### Change 4: Postgres.go — Skip during table collection/creation

In `collectTables()`, when recursing into struct attributes (line 147-154):

```go
for _, attr := range node.Attributes {
    if attr.IsStruct {
        if convert.IsTimeSeriesType(attr.TypeName) {
            continue  // TSDB data — no PostgreSQL table
        }
        _, ok := tables[attr.TypeName]
        if !ok {
            collectTables(attr, tables)
        }
    }
}
```

This requires importing the `convert` package in `postgres`, or alternatively placing `IsTimeSeriesType` in a location both packages can access (e.g., the `common` package). If there's a circular import concern, move the function to `go/orm/common/`.

## Traceability Matrix

| # | Gap | Phase |
|---|-----|-------|
| 1 | ConvertTo writes L8TimeSeriesPoint rows to relational data | Change 1 |
| 2 | ConvertFrom reconstructs L8TimeSeriesPoint from relational data | Change 2 |
| 3 | addTable creates schema entry for L8TimeSeriesPoint | Change 3 |
| 4 | collectTables includes L8TimeSeriesPoint in PostgreSQL table list | Change 4 |
| 5 | No central helper to identify TSDB-only types | Helper Function |

## Verification

After implementation:

1. **Build**: `cd go && go build ./...` — must compile cleanly
2. **Existing tests**: Run `go test ./tests/...` — must pass (current test types likely don't have time series fields, but verify no regressions)
3. **Manual check**: If a test type with `[]*L8TimeSeriesPoint` fields exists or is added:
   - No `L8TimeSeriesPoint` table should be created in PostgreSQL
   - Parent object write/read should succeed with the time series field as nil/empty
   - No errors during conversion

## Notes

- This is a **skip-only** change. The time series data will simply be absent from PostgreSQL. The TSDB service will handle persistence of these fields separately.
- Parent objects with time series slice fields will have those fields as `nil` after ORM read — this is the intended behavior until TSDB integration fills them.
- The same `TypeName == "L8TimeSeriesPoint"` pattern is already used in `l8reflect` (Setter.go:100, StructComparator.go:83), so this is consistent with the ecosystem.
