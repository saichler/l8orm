# Security-Aware Cache Pagination (l8orm)

## Problem

The l8orm postgres plugin has its own in-memory pagination index (`indexQueries`) that caches RecKey lists per query hash. This cache has the same two problems that the l8utils cache had before its security-aware plan:

1. **Shared cache across users:** The `indexQueries` map is keyed by `q.Hash()` (an `int32` derived from query text). Two users with different permissions but the same query (e.g., `select * from Employee`) share the same `cachedQuery`. The cached RecKey list reflects the full unfiltered dataset. Security filtering (ScopeView) happens downstream in the service manager — after pagination has already sliced. This causes pages with fewer records than expected, wrong total page counts, and empty pages.

2. **Metadata reflects unfiltered data:** The `metadata` stored in the `cachedQuery` is computed by the SQL `MetaData()` method, which counts all matching rows in the database regardless of the user's permissions. The total count in pagination UI reflects the unfiltered dataset.

## Current Architecture

### The postgres index cache (`Read.go` + `Postgres.go`)

- `indexQueries map[int32]*cachedQuery` — maps query hash to cached result
- `cachedQuery` stores: `recKeys []string` (sorted RecKeys from DB), `stamp int64` (invalidation stamp), `lastUsed int64` (TTL), `metadata *l8api.L8MetaData`
- `readWithIndex()` (line 163) looks up `indexQueries[q.Hash()]`, checks stamp validity, and either serves from cache or calls `readRecKeys()` to repopulate
- `readRecKeys()` (line 199) runs a SQL query to fetch all RecKeys for the query, plus metadata via `statement.MetaData(tx)`
- `pageKeys()` slices the RecKey array by page/limit
- `readByRecKeys()` fetches full row data for only the page's RecKeys

### The two-layer cache architecture

The ORM has two independent cache layers:

1. **l8utils `cache.Cache`** — in-memory object cache in `OrmService` (`persist/OrmCache.go`). Used for full object caching with `Fetch()`, `Post()`, `Delete()`, etc. The l8utils plan already makes this cache security-aware.

2. **Postgres `indexQueries`** — RecKey-only cache in the postgres plugin (`plugins/postgres/Read.go`). Used exclusively for paginated queries (`q.Limit() > 0`). This is the cache that this plan addresses.

When the OrmService has cache enabled, `Get()` routes through `cacheFetch()` which calls `this.cache.Fetch()` (the l8utils cache). The postgres `readWithIndex()` is only reached when:
- Cache is disabled, OR
- Cache is empty (`this.cache.Size() == 0`), OR
- `cacheFetch()` returns nil (cache miss at start == 0)

In practice, the postgres index cache serves paginated queries that bypass or fall through the l8utils cache — direct ORM reads, initial bulk loads, and services without the cache layer enabled.

### Security filtering today

- `ScopeView` is called in `ServiceManager.go` AFTER the ORM returns paginated results
- The postgres index cache has no awareness of the authenticated user
- `IQuery` has `AAAId() string` — the authenticated user's identity, set before the query reaches the ORM
- `ISecurityProvider.ScopeItem(IResources, interface{}, string, string, ...string) interface{}` — filters a single item, returns nil if denied
- `IResources` is available in `Read()` via the `resources` parameter
- `this.res` on the `Postgres` struct holds `IResources`

### Metadata computation

Currently, `readRecKeys()` computes metadata via `statement.MetaData(tx)` which runs SQL count queries on the database. This metadata reflects the full unfiltered dataset. After security filtering removes RecKeys, the metadata total count will be wrong.

## Solution

Make the postgres index cache per-user by incorporating the AAA ID into the cache key, and apply `ScopeItem` filtering to RecKeys after fetching from the database but before caching. Recompute metadata to reflect only the visible records.

The approach differs from the l8utils cache plan because the postgres cache stores RecKeys (strings), not objects. We cannot call `ScopeItem` on a RecKey string — we need the full object. So the filtering happens by:
1. Fetching RecKeys from the DB (as today)
2. Fetching the full objects for those RecKeys
3. Applying `ScopeItem` to each object
4. Keeping only the RecKeys of objects that pass the filter
5. Caching the filtered RecKey list with corrected metadata

For queries with no AAA ID (system queries, no auth), behavior is unchanged — no filtering, shared cache entry.

## Changes

### Change 1: Make indexQueries key per-user (`Postgres.go` + `Read.go`)

**File:** `go/orm/plugins/postgres/Postgres.go`

Change the `indexQueries` map key from `int32` to `int64` to accommodate the combined query hash + AAA ID hash:

```go
// Before
indexQueries  map[int32]*cachedQuery

// After
indexQueries  map[int64]*cachedQuery
```

Update `NewPostgres()` initialization:
```go
// Before
indexQueries: make(map[int32]*cachedQuery),

// After
indexQueries: make(map[int64]*cachedQuery),
```

Add a `hashString` helper (same Java-style hash used by l8ql):
```go
func hashString(s string) int32 {
    var h int32
    for _, c := range s {
        h = 31*h + int32(c)
    }
    return h
}
```

### Change 2: Compute per-user cache key in readWithIndex (`Read.go`)

**File:** `go/orm/plugins/postgres/Read.go`
**Method:** `readWithIndex()` (line 163)

Combine the query hash with the AAA ID to produce a per-user cache key:

```go
func (this *Postgres) readWithIndex(q ifs.IQuery, resources ifs.IResources) ifs.IElements {
    aaaId := q.AAAId()
    hash := int64(q.Hash())
    if aaaId != "" {
        hash = hash<<32 | int64(hashString(aaaId))
    }

    this.indexMtx.RLock()
    cached, exists := this.indexQueries[hash]
    currentStamp := this.indexStamp
    this.indexMtx.RUnlock()

    if exists && cached.stamp == currentStamp {
        cached.touch()
        return this.readByRecKeys(q, cached.pageKeys(q.Page(), q.Limit()), cached.metadata, resources)
    }

    recKeys, metadata, err := this.readRecKeys(q)
    if err != nil {
        return object.NewError(err.Error())
    }

    // Apply security filtering if authenticated
    if aaaId != "" && resources.Security() != nil {
        recKeys, metadata = this.filterRecKeysBySecurity(q, recKeys, resources, aaaId)
    }

    this.indexMtx.Lock()
    cached = &cachedQuery{
        recKeys:  recKeys,
        stamp:    currentStamp,
        lastUsed: currentStamp,
        metadata: metadata,
    }
    this.indexQueries[hash] = cached
    this.indexMtx.Unlock()

    return this.readByRecKeys(q, cached.pageKeys(q.Page(), q.Limit()), metadata, resources)
}
```

### Change 3: Add filterRecKeysBySecurity method (`Read.go`)

**File:** `go/orm/plugins/postgres/Read.go`

This method fetches the full objects for the RecKeys, applies `ScopeItem` to each, and returns only the RecKeys that pass. It also rebuilds the metadata total count to reflect the filtered set.

```go
func (this *Postgres) filterRecKeysBySecurity(q ifs.IQuery, recKeys []string, resources ifs.IResources, aaaId string) ([]string, *l8api.L8MetaData) {
    if len(recKeys) == 0 {
        return recKeys, &l8api.L8MetaData{}
    }

    uuid := ""
    if resources.SysConfig() != nil {
        uuid = resources.SysConfig().LocalUuid
    }

    // Fetch the full objects for all RecKeys to apply ScopeItem
    elements := this.readByRecKeys(q, recKeys, nil, resources)
    if elements == nil || elements.Error() != nil {
        return recKeys, &l8api.L8MetaData{}
    }

    // Build a map from element to its RecKey for fast lookup
    // The elements come back in RecKey order (readByRecKeys preserves order)
    elems := elements.Elements()
    filteredKeys := make([]string, 0, len(recKeys))

    for i, elem := range elems {
        if elem == nil {
            continue
        }
        if i >= len(recKeys) {
            break
        }
        scoped := resources.Security().ScopeItem(resources, elem, uuid, aaaId)
        if scoped != nil {
            filteredKeys = append(filteredKeys, recKeys[i])
        }
    }

    metadata := &l8api.L8MetaData{
        KeyCount: &l8api.L8Count{
            Counts: map[string]int64{
                "Total": int64(len(filteredKeys)),
            },
        },
    }

    return filteredKeys, metadata
}
```

**Note on cost:** This fetches full objects once per cache population (not per page request). The filtered RecKey list is cached and subsequent page requests are served from cache without re-fetching. The cache TTL (30s) and stamp-based invalidation ensure the filtered list stays current.

### Change 4: Update cleanExpiredQueries for int64 key (`Postgres.go`)

**File:** `go/orm/plugins/postgres/Postgres.go`
**Method:** `cleanExpiredQueries()` (line 127)

No code change needed — the `for hash, q := range this.indexQueries` loop and `delete(this.indexQueries, hash)` work with any map key type. The `hash` variable will automatically be `int64` after the map type change.

## Traceability Matrix

| # | Concern | Change |
|---|---------|--------|
| 1 | Per-user query isolation in postgres index cache | Change 1 + 2: AAA ID combined into int64 hash key |
| 2 | Security filtering before pagination | Change 3: ScopeItem applied to full objects, filtered RecKeys cached |
| 3 | Accurate metadata per user | Change 3: Metadata rebuilt with filtered count |
| 4 | Backward compatibility (no AAA ID) | Change 2: Empty AAA ID = no ScopeItem, shared cache entry as before |
| 5 | Memory overhead from per-user cache entries | Managed by existing TTL cleanup (30s expiry, 10s sweep) |
| 6 | Performance: full object fetch for filtering | One-time cost per cache population; pages served from cached RecKeys |
| 7 | Nil security provider (tests, system queries) | Change 2: Guard `resources.Security() != nil` skips filtering |

## Files Modified

| File | What Changes |
|------|-------------|
| `go/orm/plugins/postgres/Postgres.go` | `indexQueries` map key type changes from `int32` to `int64`; `NewPostgres()` initialization updated; add `hashString()` helper |
| `go/orm/plugins/postgres/Read.go` | `readWithIndex()` computes per-user hash, applies security filtering; add `filterRecKeysBySecurity()` method |

## Backward Compatibility

- When `q.AAAId()` is empty (system queries, tests, unauthenticated), the hash is `int64(q.Hash())` with the upper 32 bits zeroed. No filtering is applied. Behavior is identical to today.
- When `resources.Security()` is nil (ShallowSecurityProvider, test setups), no filtering is applied even if AAA ID is present.
- The existing `ScopeView` call in `ServiceManager.go` remains as a safety net. On already-filtered data it is a no-op.
- The `readRecKeys` SQL path is unchanged — the same SQL query runs. Filtering happens in Go after the DB returns RecKeys.

## Relationship to l8utils Plan

The l8utils security-aware cache pagination plan addresses the in-memory object cache (`cache.Cache`) used by `OrmService.cacheFetch()`. This plan addresses the postgres-level RecKey index cache (`indexQueries`) used by `Postgres.readWithIndex()`.

Both caches need to be security-aware independently because:
- A query may hit the postgres cache without going through the l8utils cache (cache disabled, cache empty, direct ORM read)
- The two caches operate at different levels: l8utils caches full objects, postgres caches RecKeys for SQL pagination
- Both must produce correctly-filtered, correctly-counted results regardless of which cache layer serves the request

After both plans are implemented, security filtering happens at whichever cache layer serves the request — no double-filtering, no missed filtering.

## Testing

All test files live in `go/tests/` per `test-location-and-approach.md`. Tests exercise the ORM through its public API.

1. **Test:** Read with a query that has an AAA ID and a mock security provider that denies specific records. Verify that the returned page contains only allowed records and metadata total count reflects the filtered count.
2. **Test:** Same query from two different AAA IDs with different permissions. Verify each user gets their own cached RecKey list with correct pagination.
3. **Test:** Empty AAA ID query. Verify behavior is unchanged (no filtering, all RecKeys returned).
4. **Test:** Write a record, then read with a limited-permission user. Verify the new record is filtered out for that user but visible to an unrestricted user.
5. **Test:** TTL cleanup. Verify per-user cache entries are cleaned up after expiry.
6. **Existing tests:** All existing postgres tests must pass unchanged.

## Final Verification Phase

After all changes are implemented:

1. `cd go && go build ./...` — verify all packages compile
2. `cd go && go vet ./...` — verify no static analysis issues
3. Run full test suite — all existing tests pass
4. Run the new security-aware pagination tests (tests 1-5 above) — all pass
5. Verify no file in the postgres package exceeds 500 lines after the changes (per `maintainability.md`)
6. In a project with security (e.g., l8physio or l8erp), re-vendor l8orm, start the system locally, and verify:
   - [ ] Admin user sees all records with correct pagination
   - [ ] Restricted user sees only permitted records with correct page count
   - [ ] Metadata total count matches visible record count for both users

## Implementation Order

1. **Re-vendor l8types and l8ql** — pick up the `IQuery.Hash() int32` change (already done — the map type was updated in the earlier fix)
2. **Re-vendor l8utils** — pick up the security-aware cache changes (once the l8utils plan is implemented)
3. **l8orm: Change 1** — update `indexQueries` map key from `int32` to `int64`, add `hashString()` helper in `Postgres.go`
4. **l8orm: Change 2** — update `readWithIndex()` in `Read.go` to compute per-user hash and call security filter
5. **l8orm: Change 3** — add `filterRecKeysBySecurity()` in `Read.go`
6. **Verify** — `go build ./...`, `go vet ./...`, run test suite, file size check
