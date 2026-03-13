# Plan: TSDB Service Implementation

## Background

Time series data (CPU, memory, temperature, etc.) is modeled as `[]*L8TimeSeriesPoint` slice fields on protobuf types. The ORM already skips these fields during relational conversion (see `skip-l8timeseries-in-orm.md`). This plan implements the TSDB storage layer using TimescaleDB on a **separate PostgreSQL database** from the relational ORM.

### Data Model

- `L8TSDBNotification`: `{ PropertyId string, Point *L8TimeSeriesPoint }`
- `L8TimeSeriesPoint`: `{ Stamp int64, Value float64 }`
- `ITSDBService` interface: `AddTSDB(notifications)` + `GetTSDB(propertyId, start, end)`
- `PropertyId` is a fully qualified L8 property path, e.g., `NetworkDevice[dev-001].Cpu`

### Scale Reference

- 30K devices x 3 metrics = 90K time series
- 5-minute intervals = 288 writes/day per series
- ~26M rows/day, ~9.5B rows/year
- ~1.7 GB/day uncompressed, ~100-170 MB/day after TimescaleDB compression

## Design Decisions

### Separate Database

The TSDB uses its own `*sql.DB` connection, separate from the relational ORM's database. Reasons:
- Write patterns clash (low-frequency CRUD vs high-frequency append-only)
- TimescaleDB maintenance (compression, chunk management, retention) shouldn't compete with relational vacuuming
- PostgreSQL tuning is opposite (joins/indexes vs sequential append/chunk scanning)
- Independent scaling and failure isolation

### Single Hypertable, Narrow Schema

One table for all time series data. TimescaleDB handles internal partitioning (chunks). Narrow schema (prop_id + value) rather than wide (per-metric columns) because:
- New metrics don't require schema changes
- Matches the generic `L8TimeSeriesPoint` type and `GetTSDB(propertyId)` interface
- TimescaleDB compression with `segmentby = 'prop_id'` deduplicates the string efficiently

### Batch Inserts

`AddTSDB` receives a slice of notifications. These are written in a single transaction using a prepared statement for efficiency at 90K inserts per 5-minute cycle.

## Phase 0: L8TSDBQuery Protobuf Type

### File: `../l8types/proto/api.proto`

Add the `L8TSDBQuery` message after the existing `L8TimeSeriesPoint` (line 247):

```protobuf
message L8TSDBQuery {
  string property_id = 1;
  int64 tsdbstart = 2;
  int64 tsdbend = 3;
}
```

After adding, the owner will build, push, and update the vendor directory in l8orm.

The generated Go struct will be in `l8types/go/types/l8api/api.pb.go` with JSON fields `propertyId`, `tsdbstart`, `tsdbend`. These field names are what the L8Query where clause will match against.

## Phase 1: ITSDB Interface and Common Types

### File: `go/orm/common/IORM.go`

Add the `ITSDB` interface alongside the existing `IORM`:

```go
type ITSDB interface {
    // AddTSDB writes time series data points to the TSDB.
    AddTSDB(notifications []*l8notify.L8TSDBNotification) error

    // GetTSDB retrieves time series data points for a property within a time range.
    // start and end are Unix timestamps (seconds).
    GetTSDB(propertyId string, start, end int64) ([]*l8api.L8TimeSeriesPoint, error)

    // Close releases database connections and cleans up resources.
    Close() error
}
```

This import requires `l8notify` and `l8api` packages.

## Phase 2: TimescaleDB Plugin

### File: `go/orm/plugins/postgres/Tsdb.go`

New file implementing `ITSDB` using TimescaleDB.

```go
type Tsdb struct {
    db        *sql.DB
    mtx       *sync.Mutex
    verified  bool
}
```

**Constructor**: `NewTsdb(db *sql.DB) *Tsdb`

**Table creation** (`verifyTable`):
```sql
CREATE TABLE IF NOT EXISTS l8tsdb (
    stamp    TIMESTAMPTZ NOT NULL,
    prop_id  TEXT        NOT NULL,
    value    FLOAT8      NOT NULL
);

SELECT create_hypertable('l8tsdb', 'stamp',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_l8tsdb_prop_stamp
    ON l8tsdb (prop_id, stamp DESC);
```

**Compression setup** (run once after table creation):
```sql
ALTER TABLE l8tsdb SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'prop_id',
    timescaledb.compress_orderby = 'stamp DESC'
);

SELECT add_compression_policy('l8tsdb', INTERVAL '1 day', if_not_exists => true);
```

**AddTSDB** implementation:
```go
func (this *Tsdb) AddTSDB(notifications []*l8notify.L8TSDBNotification) error {
    this.mtx.Lock()
    defer this.mtx.Unlock()

    if !this.verified {
        if err := this.verifyTable(); err != nil {
            return err
        }
        this.verified = true
    }

    tx, err := this.db.Begin()
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            tx.Rollback()
        } else {
            err = tx.Commit()
        }
    }()

    stmt, err := tx.Prepare(
        "INSERT INTO l8tsdb (stamp, prop_id, value) VALUES (to_timestamp($1), $2, $3)")
    if err != nil {
        return err
    }
    defer stmt.Close()

    for _, n := range notifications {
        if n.Point == nil {
            continue
        }
        _, err = stmt.Exec(n.Point.Stamp, n.PropertyId, n.Point.Value)
        if err != nil {
            return err
        }
    }
    return nil
}
```

**GetTSDB** implementation:
```go
func (this *Tsdb) GetTSDB(propertyId string, start, end int64) ([]*l8api.L8TimeSeriesPoint, error) {
    rows, err := this.db.Query(
        "SELECT extract(epoch from stamp)::bigint, value FROM l8tsdb "+
            "WHERE prop_id = $1 AND stamp BETWEEN to_timestamp($2) AND to_timestamp($3) "+
            "ORDER BY stamp",
        propertyId, start, end)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var points []*l8api.L8TimeSeriesPoint
    for rows.Next() {
        p := &l8api.L8TimeSeriesPoint{}
        if err := rows.Scan(&p.Stamp, &p.Value); err != nil {
            return nil, err
        }
        points = append(points, p)
    }
    return points, rows.Err()
}
```

**Close**: `this.db.Close()`

## Phase 3: OrmService TSDB Integration

### File: `go/orm/persist/OrmService.go`

Add TSDB field and integrate with `ITSDBService`:

```go
type OrmService struct {
    orm   common.IORM
    tsdb  common.ITSDB                // New: TSDB plugin (nil if not enabled)
    sla   *ifs.ServiceLevelAgreement
    cache *cache.Cache
}
```

The existing `Activate` function signature is unchanged. The `*sql.DB` connections are passed as SLA args. The number of `*sql.DB` args determines the TSDB configuration:

- **One `*sql.DB`**: Same database for both relational and TSDB. Simple setups and testing.
- **Two `*sql.DB`**: First for relational, second for TSDB. Production/scale setups.

Caller with shared DB:
```go
sla.SetArgs(db, enableCache)
```

Caller with separate DBs:
```go
sla.SetArgs(db, enableCache, tsdbDb)
```

In `OrmService.Activate()`, create both plugins from the SLA args:

```go
db := this.sla.Args()[0].(*sql.DB)
this.orm = postgres.NewPostgres(db, vnic.Resources())

// TSDB uses separate DB if provided, otherwise shares the relational DB
tsdbDb := db
if len(this.sla.Args()) > 2 {
    if secondDb, ok := this.sla.Args()[2].(*sql.DB); ok {
        tsdbDb = secondDb
    }
}
this.tsdb = postgres.NewTsdb(tsdbDb)
```

This means TSDB is **always available** — there's no nil-TSDB case. The only question is whether it shares the relational database or has its own.

### File: `go/orm/persist/OrmTsdb.go`

New file implementing the `ITSDBService` interface methods on `OrmService`:

```go
package persist

import (
    "github.com/saichler/l8types/go/types/l8api"
    "github.com/saichler/l8types/go/types/l8notify"
)

// AddTSDB writes time series notifications to the TSDB.
func (this *OrmService) AddTSDB(notifications []*l8notify.L8TSDBNotification) {
    err := this.tsdb.AddTSDB(notifications)
    if err != nil {
        // Log but don't fail — TSDB writes should not block the service
        // The caller can retry on the next notification cycle
    }
}

// GetTSDB retrieves time series data for a property within a time range.
func (this *OrmService) GetTSDB(propertyId string, start, end int64) []*l8api.L8TimeSeriesPoint {
    points, err := this.tsdb.GetTSDB(propertyId, start, end)
    if err != nil {
        return nil
    }
    return points
}
```

Update `DeActivate` to close TSDB. Note: when sharing the same `*sql.DB`, `Tsdb.Close()` must be a no-op (the relational `Postgres.Close()` owns the connection). The `Tsdb` constructor should accept an `ownsDb` flag to control this:

```go
func (this *OrmService) DeActivate() error {
    if this.cache != nil {
        this.cache.Close()
        this.cache = nil
    }
    this.tsdb.Close()
    this.tsdb = nil
    err := this.orm.Close()
    this.orm = nil
    return err
}
```

In the `Tsdb` struct:
```go
type Tsdb struct {
    db       *sql.DB
    mtx      *sync.Mutex
    verified bool
    ownsDb   bool   // true if TSDB has its own DB connection to close
}

func (this *Tsdb) Close() error {
    if this.ownsDb {
        return this.db.Close()
    }
    return nil
}
```

The `OrmService.Activate()` sets `ownsDb` based on whether a separate DB was provided:
```go
tsdbOwnsDb := len(this.sla.Args()) > 2
this.tsdb = postgres.NewTsdb(tsdbDb, tsdbOwnsDb)
```

## Phase 4: L8Query TSDB Routing

The UI queries TSDB data using standard L8Query syntax:
```
select * from L8TSDBQuery where propertyId=NetworkDevice[dev-001].Cpu and tsdbstart>=1710000000 and tsdbend<=1710086400
```

`L8TSDBQuery` is a new protobuf type (to be added in `l8types`) with fields:
- `propertyId string` — the L8 property path
- `tsdbstart int64` — start timestamp (Unix seconds)
- `tsdbend int64` — end timestamp (Unix seconds)

### Interception in `OrmService.Get`

The `Get` method parses the L8Query into an `IQuery`. Before routing to the relational ORM, check if the query's root type is `L8TSDBQuery`. If so, extract the where-clause values and delegate to `GetTSDB`.

### File: `go/orm/persist/OrmTsdb.go`

Add the TSDB query handler:

```go
const tsdbQueryType = "L8TSDBQuery"

// isTsdbQuery checks if a parsed query targets the TSDB.
func isTsdbQuery(query ifs.IQuery) bool {
    return query.RootType().TypeName == tsdbQueryType
}

// handleTsdbQuery extracts propertyId, start, end from the query's
// where clause and delegates to GetTSDB.
func (this *OrmService) handleTsdbQuery(query ifs.IQuery) ifs.IElements {
    propertyId, start, end, err := extractTsdbParams(query)
    if err != nil {
        return object.NewError(err.Error())
    }
    points := this.GetTSDB(propertyId, start, end)
    return object.New(nil, points)
}
```

`extractTsdbParams` walks the query's where-clause conditions and extracts the three parameters by field name.

### File: `go/orm/persist/OrmService.go`

Update `Get` to intercept TSDB queries before the relational path:

```go
func (this *OrmService) Get(pb ifs.IElements, vnic ifs.IVNic) ifs.IElements {
    if pb.IsFilterMode() {
        // ... existing filter mode logic unchanged ...
    }

    query, err := pb.Query(vnic.Resources())
    if err != nil {
        return object.NewError(err.Error())
    }

    // Route TSDB queries to the time series store
    if isTsdbQuery(query) {
        return this.handleTsdbQuery(query)
    }

    // ... existing relational path (cache check, fetchFromDbAndCache) ...
}
```

### Registration

`L8TSDBQuery` must be registered in the introspector so the L8Query parser can resolve it. This happens in `OrmService.Activate()`:

```go
vnic.Resources().Registry().Register(&l8api.L8TSDBQuery{})
```

## Phase 5: Retention Policy Configuration (unchanged)

The `ITSDB` interface should support configurable retention. Add to `Tsdb`:

```go
// SetRetention configures automatic data expiry.
func (this *Tsdb) SetRetention(interval string) error {
    _, err := this.db.Exec(
        "SELECT add_retention_policy('l8tsdb', INTERVAL '" + interval + "', if_not_exists => true)")
    return err
}
```

Default retention is not set — data is kept indefinitely unless explicitly configured. Callers can set retention after activation (e.g., `tsdb.SetRetention("90 days")`).

## File Summary

| File | Action | Description |
|------|--------|-------------|
| `go/orm/common/IORM.go` | Modify | Add `ITSDB` interface |
| `go/orm/plugins/postgres/Tsdb.go` | **New** | TimescaleDB plugin implementing `ITSDB` |
| `go/orm/persist/OrmService.go` | Modify | Add `tsdb` field, extract TSDB from SLA args |
| `go/orm/persist/OrmTsdb.go` | **New** | `AddTSDB`/`GetTSDB` methods on `OrmService` |

## Traceability Matrix

| # | Requirement | Phase |
|---|-------------|-------|
| 0 | `L8TSDBQuery` protobuf type in `api.proto` | Phase 0 |
| 1 | ITSDB interface definition | Phase 1 |
| 2 | TimescaleDB hypertable creation with chunking | Phase 2 |
| 3 | Compression policy (segmentby prop_id) | Phase 2 |
| 4 | Batch insert via prepared statement in transaction | Phase 2 |
| 5 | Time-range query by propertyId | Phase 2 |
| 6 | OrmService holds separate TSDB connection | Phase 3 |
| 7 | Backward-compatible activation (no signature change, DB count determines mode) | Phase 3 |
| 8 | `ITSDBService` interface implementation on OrmService | Phase 3 |
| 9 | TSDB cleanup on DeActivate (ownsDb flag for shared vs separate DB) | Phase 3 |
| 10 | L8Query interception for `L8TSDBQuery` root type | Phase 4 |
| 11 | Extract propertyId/start/end from where clause | Phase 4 |
| 12 | `L8TSDBQuery` type registration in introspector | Phase 4 |
| 13 | Configurable retention policy | Phase 5 |
| 14 | Shared DB mode (1 arg) and separate DB mode (2 args) | Phase 3 |

## Phase 6: Verification

1. **Build**: `cd go && go build ./...` — must compile cleanly
2. **Unit test**: Create test that:
   - Opens a separate DB connection for TSDB
   - Calls `AddTSDB` with sample notifications
   - Calls `GetTSDB` with a time range and verifies returned points
   - Verifies the `l8tsdb` hypertable was created
   - Verifies data survives compression (insert, wait, compress, query)
3. **Shared DB test**: Activate with single `*sql.DB`, verify:
   - Relational CRUD works
   - `AddTSDB`/`GetTSDB` work on the same database
   - `DeActivate` closes the connection once (no double-close)
4. **Separate DB test**: Activate with two `*sql.DB` args, verify:
   - Relational and TSDB use different connections
   - `DeActivate` closes both connections
   - TSDB data is in the second database
5. **L8Query TSDB test**: Send `select * from L8TSDBQuery where propertyId=X and tsdbstart>=T1 and tsdbend<=T2` via the service Get, verify:
   - Query is routed to TSDB, not relational ORM
   - Returned points match what was written via `AddTSDB`
   - Invalid/missing params return error
