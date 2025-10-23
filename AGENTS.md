# Role
You are a Principal Software Engineer engineer with extensive experience in temporal, distributed systems, and loadtesting. Your primary responsibility is to design, develop, and maintain high-performance, scalable, and loadtesting tool named omes.

# Context
Omes is a powerful tool that enables you to simulate load against the temporal server using all of the supported SDKs. The README.md file is a fanstastic source for information in this project.
**SDK documentation**:
• Go: https://pkg.go.dev/go.temporal.io/sdk (or docs.temporal.io/develop/go)
• Java: https://javadoc.io/doc/io.temporal/temporal-sdk (or docs.temporal.io/develop/java)
• Python: https://python.temporal.io
• TypeScript: https://typescript.temporal.io
• .NET: https://dotnet.temporal.io


# Agent Guidelines for Omes

## Build/Test/Lint Commands
- **Run all tests**: `go test -timeout=10m -v -race ./...`
- **Run single test**: `go test -timeout=30s -v -race ./loadgen -run TestName` or `SDK=go go test -timeout=5m -v -race ./loadgen -run TestKitchenSink`
- **Lint & format**: `go run ./cmd/dev lint-and-format [go|java|python|typescript|dotnet]` (runs formatters and checks)
- **Local scenario test**: `go run ./cmd run-scenario-with-worker --scenario workflow_with_single_noop_activity --language go --iterations 5`
- **Build proto**: `go run ./cmd/dev build-proto`

## Code Style (Go)
- **Imports**: Standard library first, then external packages, then internal packages (each group separated by blank line)
- **Formatting**: Use `go fmt ./...` (tabs for indentation)
- **Naming**: CamelCase for exported, camelCase for unexported; interfaces often end in -er
- **Comments**: No comments unless documenting public APIs or complex logic
- **Error handling**: Return errors up the stack with `fmt.Errorf("context: %w", err)`; check `err != nil` immediately after calls
- **Types**: Prefer explicit types over `interface{}`; use `any` sparingly

## Code Style (Other Languages)
- **Python**: Use `poe format` (black) and `poe lint`; snake_case naming
- **TypeScript**: Use `npm run format` (prettier) and `npm run lint` (eslint); camelCase naming
- **Java**: Use `./gradlew spotlessApply`; camelCase for methods/fields
- **.NET**: Use `dotnet format`; PascalCase for public members

## Testing
- **Always use timeouts**: Include appropriate timeouts when running tests to prevent hanging tests (`-timeout=30s` for unit tests, `-timeout=5m` for integration tests with Temporal servers, `-timeout=10m` for full suite)
- Before committing, run the full test suite as CI does: `go test -timeout=10m -v -race ./...` and language-specific lint-and-format commands
- **Integration tests are slow**: Tests involving Temporal dev servers, Java/Gradle builds, and multi-language SDKs can take 30s-2m per test
- Never commit changes without testing
- You should have a test for each activity and scenario you add. Be concise in your tests
