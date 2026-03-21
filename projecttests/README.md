# Project Tests

Self-contained Go programs for testing Temporal workflows under load. Each project is an independent Go module that implements its own workflows, activities, and execution logic, coordinated via gRPC.

## How It Works

The **harness** (`go/harness/`) provides a gRPC server framework. Each project registers three callbacks:

- **`RegisterWorker(WorkerFunc)`** — Starts a Temporal worker with the project's workflows and activities.
- **`OnInit(InitFunc)`** — Called once per run to parse config and set up state (optional).
- **`OnExecute(ExecuteFunc)`** — Called once per iteration to start workflows and verify results.

A built project binary exposes two subcommands:
- `<program> worker --task-queue <tq> --server-address <addr> --namespace <ns>` — Runs the Temporal worker.
- `<program> project-server --port <port>` — Runs the gRPC server that accepts Init/Execute RPCs.

The test runner (or CLI) spawns both processes and drives execution via gRPC.

## Creating a New Project

1. Copy `go/tests/helloworld/` to `go/tests/<yourproject>/`.
2. Update `go.mod` with your module name.
3. Implement your workflow(s) and activities.
4. Wire up `Main()`:

```go
func Main() {
    h := harness.New()
    h.RegisterWorker(workerMain)
    h.OnExecute(clientMain)
    h.Run()
}
```

5. Add a test case in `project_test.go`.

See `go/tests/helloworld/` for a minimal example and `go/tests/throughputstress/` for a complex example with JSON config, retry scenarios, and continue-as-new.

## Running

**Locally via integration tests:**

```bash
go test -v -race -timeout 10m ./projecttests/...
```

**Via Docker:** See [dockerfiles/README.md](dockerfiles/README.md) for Docker Compose setup.

## Proto

The gRPC service is defined in `proto/api.proto`. To regenerate after changes:

```bash
cd projecttests/proto && buf generate
```

Lint with `buf lint`. See `proto/buf.yaml` for lint configuration.
