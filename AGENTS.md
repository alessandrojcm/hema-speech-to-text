# AGENTS.md - Development Guide for AI Coding Agents

## Build/Test Commands
- `make test` - Run all tests with race detection
- `go test ./internal/config -v` - Run single package tests
- `go test -run TestFunctionName ./path/to/package` - Run specific test
- `make test-integration` - Run OBS integration tests (requires OBS Studio)
- `make coverage` - Generate test coverage report
- `make build` - Build the application
- `make fmt && make vet` - Format and vet code
- `make dev` - Quick dev cycle (fmt, vet, test, build)

## Code Style Guidelines
- **Imports**: Standard library first, then third-party, then local packages with blank lines between groups
- **Naming**: Use camelCase for unexported, PascalCase for exported. Prefer short names (e.g., `cfg` not `configuration`)
- **Types**: Define structs with struct tags for config mapping (`mapstructure:"field_name"`)
- **Error Handling**: Always wrap errors with context using `fmt.Errorf("description: %w", err)`
- **Logging**: Use structured logging with zerolog, create component-specific loggers with `WithComponent()`
- **Testing**: Use testify/assert and testify/require, follow table-driven test pattern
- **Concurrency**: Use sync.RWMutex for shared state, channels for communication
- **Context**: Always pass context.Context as first parameter for cancellation support
- **Documentation**: Add godoc comments for all exported types and functions

## Project Structure
- **Phase 1 Complete**: Configuration, OBS integration, replay management, text overlays, scene management
- **Integration Tests**: Use `t.Skip("Integration test - requires running OBS Studio")` for OBS-dependent tests
- **Manager Pattern**: Each subsystem has a Manager struct with Start/Stop lifecycle methods