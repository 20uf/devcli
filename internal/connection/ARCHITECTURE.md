# Connection Context - DDD Architecture

## Overview

The **Connection Context** is the first bounded context refactored according to Domain-Driven Design principles. It handles the complete flow of connecting to an ECS container.

## Structure

```
internal/connection/
├── domain/                 # Domain layer (business logic)
│   ├── cluster.go         # Value Object: Cluster name
│   ├── service.go         # Value Object: Service name
│   ├── container.go       # Value Object: Container name
│   ├── task.go            # Entity: ECS Task with containers
│   ├── connection.go      # Aggregate Root: Complete connection info
│   ├── repository.go      # Contracts (interfaces)
│   └── errors.go          # Domain-specific errors
│
├── application/           # Application layer (use cases)
│   ├── orchestrator.go    # UseCase: Connect to container
│   └── orchestrator_test.go # Acceptance tests
│
└── infra/                 # Infrastructure layer (adapters)
    ├── ecs_mapper.go      # Anti-corruption layer (AWS → Domain)
    ├── ecs_repository.go  # Repository implementations
    ├── file_repository.go # Persistence (connections history)
    └── cli_adapter.go     # Composition root (wiring)
```

## Key Concepts

### 1. Value Objects
- **Cluster**, **Service**, **Container** - immutable, identified by name
- Type-safe alternatives to `string`
- Encapsulate validation (e.g., non-empty names)

### 2. Entities
- **Task** - has identity (ID), mutable state (containers, status)
- Methods for smart selection (find by name, select preferred)

### 3. Aggregate Root
- **Connection** - represents an intended connection to a container
- Encapsulates all information needed for execution
- Guards invariants (container must exist in task)

### 4. Repositories (Contracts)
- `ClusterRepository` - list clusters
- `ServiceRepository` - list services in cluster
- `TaskRepository` - get running task
- `ConnectionRepository` - save/retrieve connections for replay

### 5. Anti-Corruption Layer
- **ECSMapper** - translates AWS ECS API objects to domain entities
- Shields domain from AWS SDK API changes
- Extracts IDs/names from ARNs

### 6. Application Service
- **ConnectOrchestrator** - orchestrates the complete connection flow
- Framework-agnostic (can be called from Cobra, HTTP API, tests, etc.)
- Pure business logic: cluster → service → task → container
- Automatically selects preferred containers (php, app, web, api)

## Usage Example

```go
// Create adapter (wires all dependencies)
adapter, err := infra.NewCLIAdapter(ctx, profile, region)
if err != nil {
    return err
}

// Execute connection flow
conn, err := adapter.Connect(ctx, clusterName, serviceName, containerName, shellCommand)
if err != nil {
    return err
}

// Execute in infrastructure (AWS CLI)
return executeConnection(conn)  // In cmd layer
```

## Testing

All domain logic is tested via **acceptance tests** in `orchestrator_test.go`:

- Container auto-selection (prefers "php")
- Fallback to single container
- Explicit container selection
- Error handling (no clusters, services, tasks)
- Full connection flow

Run tests:
```bash
go test ./internal/connection/application -v
```

## Integration with Cobra

The `cmd/connect.go` can be refactored to use the orchestrator:

```go
func runConnect(cmd *cobra.Command, args []string) error {
    adapter, err := infra.NewCLIAdapter(cmd.Context(), flagProfile, flagRegion)
    if err != nil {
        return err
    }

    // Domain handles cluster/service/task/container selection
    conn, err := adapter.Connect(
        cmd.Context(),
        flagCluster, flagService, flagContainer,
        resolveShell(),
    )
    if err != nil {
        return err
    }

    // Infrastructure executes the connection
    return client.ExecInteractive(cmd.Context(), conn)
}
```

## Benefits

✅ **Testable** - All domain logic tested via mocks
✅ **Maintainable** - Clear separation of concerns
✅ **Portable** - Can switch from AWS → GCP with one mapper
✅ **Type-safe** - Value objects vs strings
✅ **Self-documenting** - Ubiquitous language explicit
✅ **Flexible** - New UIs (CLI, HTTP, TUI) without changing domain

## Next Steps

1. Refactor `cmd/connect.go` to use `CLIAdapter`
2. Apply same pattern to Deployment context
3. Extract shared infrastructure (profiles, history)
4. Add integration tests with real AWS (or mocks)
