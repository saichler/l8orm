# Layer 8 Distributed ORM as a Service

A distributed, horizontally and vertically scalable ORM (Object-Relational Mapping) service built on the Layer8 networking framework. L8ORM provides a comprehensive ORM solution designed for modern distributed applications with built-in support for PostgreSQL, automatic object conversion, and advanced querying capabilities.

## Features

- **Distributed Architecture**: Horizontally and vertically scalable ORM service with service mesh integration
- **Layer8 Integration**: Built on the Layer8 networking framework for reliable distributed communication and service discovery
- **PostgreSQL Support**: Native PostgreSQL database integration with connection pooling and optimization
- **Object Conversion**: Automatic conversion between Go objects and relational data structures with type safety
- **Protocol Buffers**: Uses protobuf for efficient data serialization and cross-platform compatibility
- **Transaction Support**: Built-in transaction management with replication capabilities and ACID compliance
- **Advanced Querying**: Sophisticated query builder with filtering, timeout support, and query optimization
- **Web Service Integration**: REST API support through integrated web services for external access
- **Pluggable Design**: Modular architecture allowing custom database implementations and service extensions
- **Service Bus Integration**: Built-in support for l8bus messaging and event-driven architecture
- **Reflection Utilities**: Advanced reflection capabilities for dynamic object handling and conversion

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

### Latest Updates (2025)

Recent improvements include:
- **Enhanced Interface Support**: Improved interface handling and type conversion
- **Timeout Functionality**: Added timeout support for long-running operations
- **Filter Mode**: Advanced filtering capabilities for query optimization
- **Crash Prevention**: Enhanced error handling and stability improvements
- **Repository Updates**: Updated repository structure and dependencies

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

- **PostgreSQL Driver**: `github.com/lib/pq` v1.10.9
- **Protocol Buffers**: `google.golang.org/protobuf` v1.36.9
- **Layer8 Bus**: `github.com/saichler/l8bus` - Service mesh communication
- **Layer8 Query Language**: `github.com/saichler/l8ql` - Advanced query capabilities
- **Layer8 Reflection**: `github.com/saichler/l8reflect` - Dynamic object handling
- **Layer8 Serialization**: `github.com/saichler/l8srlz` - Data serialization utilities
- **Layer8 Types**: `github.com/saichler/l8types` - Common type definitions
- **Layer8 Utils**: `github.com/saichler/l8utils` - General utilities
- **Layer8 Services**: `github.com/saichler/l8services` - Service framework components
- **UUID**: `github.com/google/uuid` v1.6.0 - Unique identifier generation

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