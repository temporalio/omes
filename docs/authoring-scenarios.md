# Authoring scenarios

There are two ways to add load shapes to omes:

|                     | You write                                                                                                       | Reach for it when                                                 |
| ------------------- | --------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| **A project (app)** | Plain Temporal code — workflows, activities, Nexus operations — like a sample, in any supported worker language | You want to run _your own_ workload concurrently. **Start here.** |
| **A scenario**      | A Go file in `scenarios/` using the omes loadgen framework                                                      | You need a load _shape_ omes's steady-rate executor can't express |

A project needs no knowledge of omes internals: you write the workload the way you normally would, and
omes runs it repeatedly at a configured concurrency. A scenario gives you control over how load is
generated, at the cost of learning the framework.

For running either, see [running.md](./running.md).

## Writing a project

A project is an **app** under `workers/<lang>/apps/<name>`, registered in `workers/<lang>/apps/registry.*`.
Each language ships a `helloworld` app to copy — Go, Python, TypeScript, .NET, and Ruby.

You provide a worker that registers your workflows and activities as usual, plus an `Execute` handler that
performs **one iteration**. omes calls `Execute` repeatedly, concurrently, for as many iterations or as
long as you ask. From [`workers/go/apps/helloworld/helloworld.go`](../workers/go/apps/helloworld/helloworld.go):

```go
var App = harness.App{
	Worker:        buildWorker,
	ClientFactory: harness.DefaultClientFactory,
	Project:       &harness.ProjectHandlers{Execute: executeProjectIteration},
}

func buildWorker(client sdkclient.Client, context harness.WorkerContext) sdkworker.Worker {
	w := sdkworker.New(client, context.TaskQueue, context.WorkerOptions)
	w.RegisterWorkflowWithOptions(helloWorldWorkflow, workflow.RegisterOptions{Name: workflowName})
	return w
}

func executeProjectIteration(client sdkclient.Client, executeContext harness.ProjectExecuteContext) error {
	run, err := client.ExecuteWorkflow(context.Background(), sdkclient.StartWorkflowOptions{
		ID:        fmt.Sprintf("%s-%d", executeContext.Run.ExecutionID, executeContext.Iteration),
		TaskQueue: executeContext.TaskQueue,
	}, workflowName, "World")
	if err != nil {
		return err
	}
	return run.Get(context.Background(), nil)
}
```

That is the whole surface: nothing omes-specific beyond `harness.App`. `ProjectHandlers` also takes an
optional `Init` handler for one-time setup before iterations begin. Give each iteration a unique workflow
ID — `ExecutionID` plus `Iteration`, as above — so concurrent iterations don't collide.

Your project runs through the built-in `project` scenario, selected with `--option language=<lang>` and
`--option project-name=<name>`:

```sh
go run ./cmd/omes run-scenario-with-worker --scenario project \
  --language go --app helloworld --run-id local-project-test --embedded-server \
  --iterations 5 --option language=go --option project-name=helloworld
```

See the [README's Project section](../README.md#project) for the remaining options, running the worker and
scenario separately, and running under Docker.

**The limitation to know:** the `project` scenario drives load at a steady rate — run _x_ iterations, or
run for _y_ duration. If you need a load shape it can't express (oscillating backlog, bursts, a custom
control loop), that's when you write a scenario instead.

## Writing a scenario

A scenario is a Go file in `scenarios/` that tells omes what work to generate each iteration. It registers
itself, chooses an executor, and reads any options it accepts. The rest of this guide covers that path.

## Registration, and the filename rule

Scenarios live in [`scenarios/`](../scenarios/) and register themselves from `init()`. There is no central
list to edit — importing the package is what registers them.

```go
func init() {
	loadgen.MustRegisterScenario(loadgen.Scenario{
		Description: "Each iteration executes a single workflow with a noop activity.",
		ExecutorFn: func() loadgen.Executor {
			return loadgen.KitchenSinkExecutor{
				TestInput: &kitchensink.TestInput{
					WorkflowInput: &kitchensink.WorkflowInput{
						InitialActions: []*kitchensink.ActionSet{
							kitchensink.NoOpSingleActivityActionSet(),
						},
					},
				},
			}
		},
	})
}
```

That is a complete scenario — see
[`scenarios/workflow_with_single_noop_activity.go`](../scenarios/workflow_with_single_noop_activity.go).

> **The scenario's name is its filename.** `MustRegisterScenario` infers the name from the file that
> called it, so `scenarios/my_new_thing.go` becomes the scenario `my_new_thing`. There is no name field.
> **Renaming the file renames the scenario**, which changes the task queue and breaks anything that
> referenced the old name — including in-flight runs and cleanup. Use snake_case.

## Choosing an executor

A scenario must supply an `Executor`. There are two practical routes, and the gap between them is the
main decision you'll make.

### `KitchenSinkExecutor` — start here

You declare an action tree; the executor runs the loop. The Kitchen Sink workflow accepts
[actions](../workers/go/workerlib/kitchensink/kitchen_sink.go) and is implemented in **every** worker
language, so a KitchenSink scenario works across all SDKs for free.

Use it unless you need control it cannot express.

### A custom `Executor` — when you outgrow it

`Executor` is a single method:

```go
type Executor interface {
	Run(context.Context, ScenarioInfo) error
}
```

Writing your own gives full control, and hands you responsibilities the KitchenSink path covered:
parsing and validating your own options, and (unless you delegate to `GenericExecutor`) driving your own
iteration loop.

If you go custom, prefer `GenericExecutor` for the loop (it handles concurrency, rate limiting, duration,
and attempts) and use the `*loadgen.Run` helpers — `DefaultStartWorkflowOptions`, `TaskQueue`,
`ExecuteKitchenSinkWorkflow` — rather than re-deriving IDs and task queues yourself.

Optional interfaces an executor may implement: `Configurable` (a `Configure` hook for validating options
up front), `Resumable` (`Snapshot`/`LoadState`), and `HasDefaultConfiguration` (declare the scenario's
default iterations/concurrency).

## Exposing options

Scenario configuration arrives through **two separate channels**, and knowing which is which matters:

1. **Built-in run flags** — iterations, duration, concurrency, rate, attempts, timeout. These are
   framework-level and apply to every scenario, so you neither declare nor read them; see
   [running.md](./running.md#configuring-the-load) for the list.
2. **Your own options** — arbitrary `--option key=value` pairs, read from `ScenarioInfo`:

```go
children := info.ScenarioOptionInt("children-per-workflow", 30)
sleep := info.ScenarioOptionDuration("sleep-duration", 5*time.Second)
```

Accessors exist for int, float, bool, duration, and string, each taking the default inline.

### Conventions for options

- **Document every option in the scenario's `Description`, with its default.** That string is the only
  thing `list-scenarios` shows a user, so an undocumented option is an invisible one.
- **Validate required options explicitly** and return a clear error naming the option. If your executor
  implements `Configurable`, validate in `Configure` so the run fails immediately rather than mid-load.
- **Prefer the typed accessors** over reading the raw map, so values parse consistently.

> **Current sharp edges to be aware of** (they affect how carefully you document):
> the typed accessors **panic** on a malformed value rather than returning a friendly error, and an
> **unknown or misspelled option key is silently ignored** — the default is used and nothing warns. A
> user who typos your option name gets a quiet, wrong-shaped run, so exact option names in the
> description matter.

## Testing your scenario

Run it locally with a worker and an embedded server — see
[running.md](./running.md#run-a-scenario-with-a-worker-local-all-in-one) — and keep `--iterations` small
while iterating.

Then confirm it did what you intended. Pass `--log-level debug` to see per-iteration progress, since the
default output is near-silent, and check the workflows it actually produced on the server rather than
reading success into the absence of errors.

If your scenario relies on custom search attributes or a Nexus endpoint, those must exist before the run;
against a real cluster they generally have to be pre-registered.

## Conventions summary

1. snake_case filenames — the filename _is_ the scenario name.
2. Prefer `KitchenSinkExecutor`; go custom only when you must, and reuse `GenericExecutor` and
   `*loadgen.Run` when you do.
3. Document every option and its default in the `Description`.
4. Validate required options up front, with errors that name the option.
5. Put helpers other scenario authors would want in the `loadgen` package, not in your scenario file.
