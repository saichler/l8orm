# Plan: Postgres Schema Migration — Add Missing Columns on Startup

## Background

Today, the postgres plugin's `verifyTable()` (`go/orm/plugins/postgres/Postgres.go:182`) does the following during initialization:

1. Runs `select * from <tableName> where false` to check if the table exists.
2. If the error contains `"does not exist"`, calls `createTable()` to create it.
3. Otherwise assumes the existing table is up-to-date and marks it verified.

The gap: when a proto definition gains new attributes between releases, the existing table is silently left with the old schema. New columns never appear, and `Write`/`Read` paths either error on missing columns or drop data depending on the SQL statement builder. We need to detect this drift on startup and **ALTER TABLE ... ADD COLUMN** for every attribute present in the proto but missing from the live table.

Scope for this plan:
- **In scope**: adding missing columns for attributes that already exist in the proto but not in the table.
- **Out of scope** (explicitly deferred — listed under "Deferred" below): dropping removed columns, renaming columns, changing column types, backfilling default values, non-unique index drift, primary key / constraint drift.

## Current Code Reference

- `Postgres.verifyTables()` (Postgres.go:164) — iterates collected tables and calls `verifyTable` once per table, caching the result in `this.verifyed`.
- `Postgres.verifyTable()` (Postgres.go:182) — existence check + create-if-missing.
- `Postgres.createTable()` (Postgres.go:194) — builds CREATE TABLE DDL by iterating `node.Attributes`, skipping `attr.IsStruct`, mapping types via `postgresTypeOf()`. Also creates non-unique indexes from the decorator.
- `postgresTypeOf()` (Postgres.go:236) — Go type → Postgres type mapping. This is the single source of truth for new-column types and must be reused by the migration path.
- `collectTables()` (Postgres.go:145) — walks the root node's nested struct attributes to build the set of tables needing verification. Already skips time-series types.

The `verifyed` map caches the tables that have been checked in this process. Because migration runs in the same flow as creation, we only need to extend `verifyTable` — nothing else needs to change for when the check runs.

## Design

### 1. New function: `migrateTable(tableName string) error`

Called from `verifyTable` after confirming the table already exists (i.e., the existence probe did NOT return `"does not exist"`). Responsibilities:

1. Look up the introspector node for `tableName`. If missing, return an error with the same wording as `createTable` ("Cannot find node for table ...").
2. Query the live column set from Postgres using `information_schema.columns`:
   ```sql
   SELECT column_name
   FROM information_schema.columns
   WHERE table_name = $1
   ```
   Note: `information_schema` lowercases identifiers. The table-creation DDL currently uses mixed case (e.g., `ParentKey`, `RecKey`) without quoting, so Postgres stores the identifiers lowercased too. We must compare **case-insensitively** against the attribute names from the proto. Build a `map[string]bool` of lowercased column names.
3. Iterate `node.Attributes` in the same way `createTable` does:
   - Skip `attr.IsStruct` (nested structs become their own tables via `collectTables`, not columns).
   - Skip time-series attributes using the same guard `common.IsTimeSeriesType(attr.TypeName)` that `collectTables` uses, to stay consistent.
   - For each remaining attribute, check whether `strings.ToLower(attrName)` exists in the live column map.
   - If missing, execute:
     ```sql
     ALTER TABLE <tableName> ADD COLUMN <attrName> <postgresTypeOf(attr)>;
     ```
     Reuse `postgresTypeOf(attr)` verbatim so new columns match the types `createTable` would have generated. Log the action via `this.res.Logger().Info(...)` so operators can see the migration run.
4. After adding all missing columns, also reconcile non-unique indexes for any newly added columns:
   - Look up the non-unique field list via `this.res.Introspector().Decorators().Fields(node, l8reflect.L8DecoratorType_NonUnique)`.
   - For every field in that list that was just added (i.e., was in the "missing columns" set), create the index using the exact same DDL pattern `createTable` uses: `CREATE INDEX <tableName>_<fieldName>_idx ON <tableName> (<fieldName>);`. Wrap in `IF NOT EXISTS` (`CREATE INDEX IF NOT EXISTS ...`) so a partial prior migration does not fail.
5. Return `nil` on success, or the first Postgres error encountered (fail fast — the caller already bubbles errors up and leaves `this.verifyed[tableName]` unset, so a retry happens on next startup if the operator fixes the underlying issue).

### 2. Call site change in `verifyTable`

Current:
```go
func (this *Postgres) verifyTable(tableName string) error {
    q := strings.New("select * from ", tableName, " where false;")
    _, err := this.db.Exec(q.String())
    if err != nil && strings2.Contains(err.Error(), "does not exist") {
        return this.createTable(tableName)
    }
    return err
}
```

New:
```go
func (this *Postgres) verifyTable(tableName string) error {
    q := strings.New("select * from ", tableName, " where false;")
    _, err := this.db.Exec(q.String())
    if err != nil {
        if strings2.Contains(err.Error(), "does not exist") {
            return this.createTable(tableName)
        }
        return err
    }
    // Table exists — reconcile its columns with the current proto definition.
    return this.migrateTable(tableName)
}
```

The early `return err` preserves today's behavior for any non-"does not exist" error (e.g., permission denied, connection dropped). Only a cleanly-existing table proceeds to migration.

### 3. Logging

Use the existing resources logger (`this.res.Logger()`). Two log lines per table when drift is detected:
- `Info`: "Migrating table <name>: adding columns [col1, col2, ...]"
- `Info` (per index added): "Creating non-unique index <tableName>_<field>_idx"

No logging when a table is already in sync — migration must be quiet on the happy path so startup output stays clean.

### 4. File placement

Everything can live in `Postgres.go` since the new function is ~40 lines and sits directly next to `verifyTable` and `createTable`. If the file is already near the 500-line maintainability threshold, split `verifyTable` + `createTable` + `migrateTable` + `postgresTypeOf` into a new file `Schema.go` in the same package. (Current `Postgres.go` is 264 lines, so a split is not required now — keep them together.)

## Edge Cases

| Case | Behavior |
|---|---|
| Table has extra columns the proto no longer declares | **Ignored.** We do not drop columns — that's explicitly out of scope and is destructive. |
| Column exists but with a different type | **Ignored.** Type migration is out of scope; we only add. A future plan can handle type changes via a widening-only policy. |
| Column name differs only in case (`parentkey` vs `ParentKey`) | Considered equal because we compare lowercased names. Postgres folds unquoted identifiers to lowercase, so this is the correct behavior. |
| Proto adds a nested struct attribute (`IsStruct == true`) | Skipped as a column. The new struct's table is picked up by `collectTables` and created by the normal `verifyTable`→`createTable` path. |
| Proto adds a `L8TimeSeriesPoint` attribute | Skipped by the same time-series guard `collectTables` uses. |
| Two startups race on the same fresh deployment | `this.mtx` is held by `WriteRelational` around `verifyTables`. The existing lock already serializes verification; migration runs under the same lock. No additional locking needed. |
| `information_schema` query returns zero rows | Means the table was dropped between the existence probe and the column query. Return the error from the query execution or an explicit error — do not silently fall through to create. |
| Postgres version differences | `information_schema.columns` is standard and has existed since PG 8.x. No version-specific handling needed. |

## Testing

Location: `go/tests/` per the test-location rule. Use the system API, not internal calls.

### New test file: `TestPostgresSchemaMigration_test.go`

Scenarios (all use the real postgres container spun up by the existing test harness):

1. **Add-single-column**: Create a table from proto version A (fewer attributes), then simulate proto version B by invoking `verifyTables` against a type node that has one extra scalar attribute. Assert:
   - The table now contains the new column (query `information_schema.columns`).
   - Existing rows are preserved and the new column is NULL for them.
   - A subsequent `Write` with the new field populated round-trips correctly via `Read`.
2. **Add-multiple-columns**: Two new scalar attributes across two different Postgres types (`text` and `bigint`). Same assertions.
3. **Add-column-with-non-unique-index**: New attribute is decorated as non-unique. Assert the index `<tbl>_<field>_idx` exists in `pg_indexes` after migration.
4. **Idempotent rerun**: Call `verifyTables` twice in a row with the same node. Second call must be a no-op (no ALTER, no errors). Use a query count or `information_schema` snapshot diff.
5. **No drift is a no-op**: Existing table that already matches proto. Assert no ALTER statements were issued (log capture or query counter).
6. **Case-insensitive match**: Manually create a table with lowercase column names (as Postgres stores them), then migrate. Assert no duplicate columns are added for attributes whose proto name is mixed-case.
7. **Nested struct attribute added**: Proto gains a new struct field. Assert the child table is created (via the normal `collectTables` path) and the parent table gets **no** new scalar column for it.

Because the test harness in `go/tests/` already spins up postgres for `TestPostgres_test.go` etc., reuse the same fixture setup.

## Deferred (Explicitly Out of Scope)

These are deliberately left for future plans so this change stays small and non-destructive:

- Dropping columns that exist in the table but no longer in the proto.
- Renaming columns (requires migration metadata the ORM does not currently track).
- Changing column types (widening, e.g., `integer` → `bigint`).
- Changing the primary key / constraints.
- Backfilling default values for new NOT NULL columns (we always add as nullable — Postgres default).
- Dropping non-unique indexes whose field is no longer decorated.

If any of these become necessary, each should get its own plan because the failure modes and rollback stories are very different from additive column changes.

## Traceability Matrix

| # | Concern | Resolution |
|---|---|---|
| 1 | Detect that a table already exists | Reuse existing `select * from <t> where false` probe in `verifyTable` |
| 2 | Enumerate live columns | New query against `information_schema.columns` in `migrateTable` |
| 3 | Compare proto attributes to live columns | Iterate `node.Attributes` in `migrateTable`, same skip rules as `createTable` |
| 4 | Add missing columns | `ALTER TABLE ... ADD COLUMN` using `postgresTypeOf(attr)` |
| 5 | Preserve non-unique index behavior | Create `<tbl>_<field>_idx` for newly added non-unique fields |
| 6 | Case folding of identifiers | Lowercase both sides before comparing |
| 7 | Skip nested structs | Same `attr.IsStruct` guard as `createTable` |
| 8 | Skip time-series types | Same `common.IsTimeSeriesType` guard as `collectTables` |
| 9 | Logging | `this.res.Logger().Info(...)` on drift detected |
| 10 | Locking | Existing `this.mtx` in `WriteRelational` serializes verification |
| 11 | Tests | New `TestPostgresSchemaMigration_test.go` under `go/tests/` using system API |

## Phases

1. **Phase 1 — Implement `migrateTable`** and wire it into `verifyTable`. Keep changes confined to `Postgres.go`. Build with `go build ./...`.
2. **Phase 2 — Add tests** in `go/tests/TestPostgresSchemaMigration_test.go` covering all 7 scenarios above. Run against the existing postgres test fixture.
3. **Phase 3 — End-to-end verification**:
   - Spin up a fresh postgres, run the full test suite (`go test ./tests/...`) to confirm no regressions on existing tests.
   - Manually: drop a column from a test table, rerun `verifyTables`, confirm the column is re-added and existing data survives.
   - Manually: run the ORM service twice in a row against the same DB and confirm startup is quiet (no migration log lines) on the second run.
