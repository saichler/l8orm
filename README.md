# L8ORM - Distributed Object-Relational Mapping

A distributed, horizontally and vertically scalable ORM (Object-Relational Mapping) service built on the Layer8 networking framework. L8ORM provides a pluggable ORM solution designed for modern distributed applications with built-in support for PostgreSQL and automatic object conversion.

## Features

- **Distributed Architecture**: Horizontally and vertically scalable ORM service
- **Layer8 Integration**: Built on the Layer8 networking framework for reliable distributed communication
- **PostgreSQL Support**: Native PostgreSQL database integration
- **Object Conversion**: Automatic conversion between Go objects and relational data structures
- **Protocol Buffers**: Uses protobuf for efficient data serialization
- **Transaction Support**: Built-in transaction management with replication capabilities
- **Web Service Integration**: REST API support through integrated web services
- **Pluggable Design**: Modular architecture allowing custom database implementations

## Architecture

L8ORM consists of several key components:

- **ORM Interface** (`IORM`): Core interface for read/write operations
- **Convert Service**: Handles object-to-relational data conversion
- **Persist Service**: Manages database persistence operations
- **Statement Builder**: SQL statement generation and query building
- **Web Service Layer**: REST API endpoints for external access

## Installation

```bash
go get github.com/saichler/l8orm/go
```

## Usage

### Basic Setup

```go
import (
    "github.com/saichler/l8orm/go/orm/common"
    "github.com/saichler/l8orm/go/orm/persist"
)

// Initialize ORM service
ormService := &persist.OrmService{}
```

### Database Operations

The ORM provides both low-level relational data operations and high-level object operations:

```go
// Read relational data
data, err := orm.Read(query)

// Write relational data
err := orm.Write(relationalData)

// Read objects
elements := orm.ReadObjects(query, resources)

// Write objects
err := orm.WriteObjects(elements, resources)
```

## Configuration

### PostgreSQL Setup

Use the provided scripts to manage PostgreSQL:

```bash
# Start PostgreSQL
./scripts/start-postgres.sh

# Stop PostgreSQL
./scripts/stop-postgres.sh
```

### Protocol Buffers

Generate Go bindings from proto files:

```bash
cd proto
./make-bindings.sh
```

## Testing

Run the test suite:

```bash
cd go
./test.sh
```

The project includes comprehensive tests for:
- Convert service functionality
- PostgreSQL operations
- Service point integrations
- Missing table handling

## Dependencies

- **PostgreSQL Driver**: `github.com/lib/pq`
- **Protocol Buffers**: `google.golang.org/protobuf`
- **Layer8 Framework**: `github.com/saichler/layer8`
- **Additional Utilities**: Various saichler packages for serialization, types, and utilities

## API Reference

### Core Interfaces

#### IORM Interface
```go
type IORM interface {
    Read(ifs.IQuery) (*types.RelationalData, error)
    Write(*types.RelationalData) error
    ReadObjects(ifs.IQuery, ifs.IResources) ifs.IElements
    WriteObjects(ifs.IElements, ifs.IResources) error
    Close() error
}
```

### Data Structures

The ORM uses Protocol Buffer definitions for data structures:

- **RelationalData**: Root container for table data
- **Table**: Represents database tables with columns and rows
- **Row**: Individual record with column values
- **InstanceRows**: Grouped rows by instance
- **AttributeRows**: Grouped rows by attribute

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite
6. Submit a pull request

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Support

For issues, questions, or contributions, please visit the project repository or contact the maintainer.