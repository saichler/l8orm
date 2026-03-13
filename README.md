# Layer 8 Distributed ORM as a Service

A distributed, horizontally and vertically scalable ORM (Object-Relational Mapping) service built on the Layer8 networking framework. L8ORM provides automatic conversion between Go objects and PostgreSQL, with built-in support for time series data, query caching, write-through caching, and service mesh integration.

## Features

- **Distributed Architecture**: Horizontally and vertically scalable ORM service with service mesh integration via l8bus
- **Two-Layer Conversion**: Objects ↔ L8OrmRData (relational intermediate format) ↔ Database
- **PostgreSQL Plugin**: Native PostgreSQL integration with connection pooling, upserts, and automatic table creation
- **Time Series Database (TSDB)**: TimescaleDB-backed time series storage with hypertable chunking, separate from the relational ORM
- **Query Cache**: 30-second TTL cache for pagination optimization with background TTL cleaner
- **Write-Through Cache**: Optional in-memory cache layer with automatic invalidation on writes/deletes
- **Protocol Buffers**: Protobuf-based relational intermediate format for efficient serialization
- **Transaction Support**: ACID-compliant transaction management with batch processing (default 500 elements)
- **Before/After Callbacks**: Hook into CRUD operations for validation and business logic
- **Composite Keys**: ParentKey + RecKey scheme supporting nested structures, slices, and maps
- **Pluggable Design**: `IORM` and `ITSDB` interfaces allow custom database implementations

## Architecture

```
                    ┌─────────────────────┐
                    │    ORM Service       │
                    │  (Service Mesh)      │
                    │  Before/After Hooks  │
                    │  Cache Layer         │
                    └────────┬────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
     ┌────────┴────────┐          ┌─────────┴────────┐
     │  Relational ORM │          │      TSDB         │
     │  (IORM)         │          │     (ITSDB)       │
     └────────┬────────┘          └─────────┬────────┘
              │                             │
    ┌─────────┴──────────┐                  │
    │                    │                  │
┌───┴────┐    ┌──────────┴──┐    ┌──────────┴──┐
│Convert │    │  Statement  │    │ TimescaleDB  │
│To/From │    │  Builder    │    │ Hypertable   │
└───┬────┘    └──────────┬──┘    └─────────────┘
    │                    │
    └────────┬───────────┘
             │
      ┌──────┴──────┐
      │  PostgreSQL  │
      └─────────────┘
```

### Key Components

- **ORM Service** (`orm/persist`): Service mesh wrapper exposing CRUD as distributed endpoints with cache, TSDB routing, and before/after callbacks
- **Convert Layer** (`orm/convert`): Bidirectional conversion between Go objects and the L8OrmRData relational format
- **Statement Builder** (`orm/stmt`): SQL generation for SELECT, INSERT, UPDATE, DELETE, and metadata queries with prepared statement caching
- **PostgreSQL Plugin** (`orm/plugins/postgres`): IORM implementation with query caching, automatic table/index creation, and batch processing
- **TSDB Plugin** (`orm/plugins/postgres`): ITSDB implementation using TimescaleDB hypertables for time series data

## Project Structure

```
go/
├── orm/
│   ├── common/             # Core interfaces (IORM, ITSDB, IORMRelational)
│   ├── convert/            # Object ↔ Relational conversion
│   │   ├── ConvertTo.go    # Go objects → L8OrmRData
│   │   ├── ConvertFrom.go  # L8OrmRData → Go objects
│   │   ├── ConvertService.go # Service mesh integration
│   │   └── Utils.go        # Shared conversion utilities
│   ├── persist/            # ORM service layer
│   │   ├── OrmService.go   # Main service (cache, TSDB, callbacks)
│   │   ├── OrmCallback.go  # Before/After hook execution
│   │   ├── OrmDoAction.go  # Core write pipeline
│   │   ├── OrmCache.go     # Write-through cache operations
│   │   ├── OrmTSDB.go      # TSDB query routing
│   │   └── utils.go        # Element/query utilities
│   ├── plugins/postgres/   # PostgreSQL implementation
│   │   ├── Postgres.go     # Connection, table creation, query cache
│   │   ├── Read.go         # SELECT with pagination and caching
│   │   ├── Write.go        # INSERT/UPDATE with transactions
│   │   ├── Delete.go       # Cascade delete with composite keys
│   │   └── Tsdb.go         # TimescaleDB TSDB implementation
│   └── stmt/               # SQL statement builders
│       ├── Statement.go    # Core statement management
│       ├── Select.go       # SELECT generation
│       ├── Insert.go       # INSERT ON CONFLICT (upsert)
│       ├── Update.go       # UPDATE with COALESCE (PATCH)
│       ├── Delete.go       # DELETE generation
│       ├── QueryToSql.go   # L8Query → SQL WHERE clause
│       └── MetaData.go     # COUNT for pagination
├── types/l8orms/           # Generated protobuf types
├── tests/                  # All tests
└── vendor/                 # Vendored dependencies
proto/
└── orms.proto              # Protobuf definitions
scripts/
├── start-postgres.sh       # Start PostgreSQL container
└── stop-postgres.sh        # Stop PostgreSQL container
```

## Core Interfaces

### IORM - Object-Relational Mapping

```go
type IORM interface {
    Read(query ifs.IQuery, resources ifs.IResources) ifs.IElements
    Write(action ifs.Action, elements ifs.IElements, resources ifs.IResources) error
    Delete(query ifs.IQuery, resources ifs.IResources) error
    Close() error
}
```

### ITSDB - Time Series Database

```go
type ITSDB interface {
    AddTSDB(notifications []*l8notify.L8TSDBNotification) error
    GetTSDB(propertyId string, start, end int64) ([]*l8api.L8TimeSeriesPoint, error)
    Close() error
}
```

### IORMRelational - Raw Relational Access

```go
type IORMRelational interface {
    IORM
    ReadRelational(query ifs.IQuery) (*l8orms.L8OrmRData, error)
    WriteRelational(action ifs.Action, data *l8orms.L8OrmRData) error
}
```

## Installation

```bash
go get github.com/saichler/l8orm/go
```

## Usage

### PostgreSQL Setup

```bash
# Start PostgreSQL (Docker)
./go/start-db.sh

# Or using the scripts directory
./scripts/start-postgres.sh
```

### Basic ORM Operations

```go
import (
    "github.com/saichler/l8orm/go/orm/plugins/postgres"
)

// Create a PostgreSQL-backed ORM
orm := postgres.NewPostgres(host, port, user, password, dbName, resources)
defer orm.Close()

// Read objects via query
elements := orm.Read(query, resources)

// Write objects (POST = insert, PUT = replace, PATCH = partial update)
err := orm.Write(ifs.POST, elements, resources)

// Delete by query
err := orm.Delete(query, resources)
```

### Time Series Data

```go
import "github.com/saichler/l8orm/go/orm/plugins/postgres"

// Create TSDB instance (can share or own its DB connection)
tsdb := postgres.NewTsdb(db, true)
defer tsdb.Close()

// Write time series data points
err := tsdb.AddTSDB(notifications)

// Read time series data for a property within a time range
points, err := tsdb.GetTSDB(propertyId, startTimestamp, endTimestamp)
```

## Data Model

The ORM uses a protobuf-based relational intermediate format (`L8OrmRData`):

- **L8OrmRData**: Root container mapping table names to table schemas
- **L8OrmTable**: Table with column definitions and instance rows
- **L8OrmInstanceRows**: Rows grouped by parent key (hierarchy support)
- **L8OrmAttributeRows**: Rows grouped by attribute name
- **L8OrmRow**: Single record with composite key (parent_key + rec_key) and column values

Composite keys enable representation of nested Go structures:
- **ParentKey**: Encodes the object hierarchy path
- **RecKey**: Identifies the specific field, slice index, or map key

## Testing

```bash
cd go
./test.sh
```

Test coverage includes:
- PostgreSQL CRUD operations
- PATCH/partial update semantics
- Service mesh integration
- Object ↔ relational conversion
- Write-through cache operations
- Missing/empty table handling
- TSDB write, read, time range queries, and edge cases

## Dependencies

| Dependency | Purpose |
|---|---|
| `github.com/lib/pq` | PostgreSQL driver |
| `google.golang.org/protobuf` | Protocol Buffers runtime |
| `github.com/saichler/l8bus` | Service mesh communication |
| `github.com/saichler/l8ql` | Query language (L8Query) |
| `github.com/saichler/l8reflect` | Introspection and type registry |
| `github.com/saichler/l8srlz` | Serialization utilities |
| `github.com/saichler/l8types` | Shared interfaces and types |
| `github.com/saichler/l8utils` | Utilities (cache, strings) |
| `github.com/saichler/l8services` | Service framework |
| `github.com/saichler/l8test` | Testing framework |
| `github.com/saichler/l8pollaris` | Polling framework |
| `github.com/saichler/probler` | Error handling |

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.
