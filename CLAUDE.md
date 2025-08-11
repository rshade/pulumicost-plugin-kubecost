# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Building and Testing

```bash
# Build the plugin binary
make build

# Run tests
make test

# Run linting (requires golangci-lint)
make lint

# Install to local plugin directory
make install
```

## Project Architecture

This is a gRPC plugin that implements the CostSource service from `pulumicost-spec`. Key components:

- **gRPC Server**: Listens on port 50051, implements the CostSource service methods
- **Kubecost Client**: HTTP client that queries the Kubecost API for allocation data
- **Configuration**: Supports both environment variables and YAML config files

## Key Implementation Details

### Resource ID Mapping
The plugin maps resource IDs to Kubecost filters:
- `namespace/<name>` → filter by namespace
- `pod/<namespace>/<name>` → filter by namespace and pod
- `controller/<namespace>/<name>` → filter by namespace and controller
- `node/<name>` → filter by node

### Cost Projections
`GetProjectedCost` calculates a 30-day average from historical data and extrapolates monthly costs. This is a simple MVP approach that can be enhanced with more sophisticated forecasting.

### Error Handling
- HTTP client includes timeout support via context
- TLS certificate verification can be disabled for development
- All errors are propagated with context

## Dependencies

The plugin depends on:
- `github.com/yourorg/pulumicost-spec/sdk/go/proto` - Protocol buffer definitions (needs to be replaced with actual import path)
- Standard gRPC and protobuf libraries
- `gopkg.in/yaml.v3` for configuration parsing

## Testing Approach

- Unit tests for individual components (client, config, server methods)
- Integration tests can use the testdata/ JSON files
- For live testing, set `KUBECOST_BASE_URL` environment variable

## Common Development Tasks

### Adding New Resource Types
1. Update the `Supports()` method in `kubecost_server.go`
2. Add mapping logic in `GetActualCost()` for the new resource ID format
3. Update `plugin.manifest.json` with the new resource type

### Debugging the gRPC Server
The server includes reflection support, so you can use tools like `grpcurl`:
```bash
grpcurl -plaintext localhost:50051 list
grpcurl -plaintext localhost:50051 describe CostSource
```

### Modifying Kubecost API Calls
The HTTP client in `client.go` handles the Kubecost API interaction. To add new endpoints:
1. Add new methods to the Client struct
2. Define request/response types
3. Handle the API call with proper error handling and timeout