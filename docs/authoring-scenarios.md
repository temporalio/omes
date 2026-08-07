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
up front) and `Resumable` (`Snapshot`/`LoadState`). To declare default iterations/concurrency, prefer the
scenario's `DefaultConfiguration` field over the older `HasDefaultConfiguration` executor interface.

## Exposing options

Scenario configuration arrives through **two separate channels**, and knowing which is which matters:

1. **Built-in run flags** — iterations, duration, concurrency, rate, attempts, timeout. These are
   framework-level and apply to every scenario, so you neither declare nor read them; see
   [running.md](./running.md#configuring-the-load) for the list.
2. **Your own options** — `--option key=value` pairs that you **declare** on the scenario.

The dividing line is that run flags shape the load while options decide what the load does. Two
consequences when you add a knob:

- **Don't declare an option for something a run flag already controls.** If you want fewer iterations or
  less concurrency, that is the user's `--iterations` or `--max-concurrent`, not an option of yours.
- **Don't borrow a run flag's vocabulary for a different meaning.** `throughput_stress` has
  `internal-iterations`, which is not `--iterations`; a user who confuses them gets a valid run at the
  wrong load rather than an error. Name the thing your option actually controls.

### Declare them

Declare options with the same typed registrars used for omes's own CLI flags — an `OptionSet` embeds a
`pflag.FlagSet`, so `Int`, `Duration`, `Bool`, `Float64`, `String` and the rest are all available, and
pflag does the parsing and type checking:

```go
loadgen.MustRegisterScenario(loadgen.Scenario{
	Description: "Each iteration executes a single workflow with child workflows and/or activities.",
	Options: func(o *loadgen.OptionSet) {
		o.Int("children-per-workflow", 30, "Number of child workflows started per iteration.")
		o.Duration("sleep-time", time.Second, "How long each sleep activity sleeps.")

		o.Int("task-queue-count", 0, "Number of task queues to spread iterations across.")
		o.MarkRequired("task-queue-count")

		// Gated on a namespace capability; see "Feature options" below.
		o.Feature("include-standalone-activity", "Include standalone activities.",
			func(c loadgen.Capabilities) bool { return c.Namespace.GetStandaloneActivities() })
	},
	ExecutorFn: /* ... */,
})
```

Declaring buys you three things:

- **Unknown names are rejected.** A user who types `--option activites-per-workflow=50` gets an error
  listing what the scenario actually accepts, instead of a silently-ignored option and a run at the default.
- **Malformed values are rejected before the run starts**, with a message naming the option and the
  expected type — and every problem is reported at once, not one per attempt.
- **`list-scenarios` shows your options**, their types, defaults, and descriptions, so users can discover
  them without reading your source.

If your scenario uses a shared executor that reads options of its own, call that executor's declaration
helper from yours — for example `loadgen.DeclareFuzzExecutorOptions(o)`.

Options shared by several scenarios are declared the same way, so their names, defaults, and meaning
are defined once. If your scenario can generate Nexus load, declare it with
`loadgen.DeclareNexusOptions(o)` and settle it at run time with
`info.ResolveNexusConfig(ctx)`, which decides whether the server supports Nexus and creates an
endpoint if the run needs one. For an option that adds to Nexus load rather than producing it — a
standalone-operations switch, say — gate it with `loadgen.IncludeNexusSubFeature`, so Nexus being off
takes its sub-features with it instead of failing the run.

### Read them

```go
children := info.OptionInt("children-per-workflow")
sleep := info.OptionDuration("sleep-time")
```

Accessors exist for int, float64, bool, duration, and string. They take **no default** — the declaration
is the only place a default lives, so an option the user didn't supply reads back as whatever you
declared. Values are already validated, so these can't fail on user input; reading an option you didn't
declare is a bug in the scenario and reads as the zero value.

If an option's real default can't be written as a constant — say it derives from the run's `--duration` —
declare a sentinel, say so in the usage string, and resolve it in code:

```go
o.Duration(IterTimeoutFlag, 0, "Timeout for internal iterations. 0 means auto: the run duration plus a minute.")
...
if config.InternalIterTimeout == 0 {
    config.InternalIterTimeout = cmp.Or(info.Configuration.Duration+time.Minute, time.Minute)
}
```

### Declaring a default run configuration

If your scenario only makes sense at a particular scale, declare it and `list-scenarios` will show that
too:

```go
DefaultConfiguration: &loadgen.RunConfiguration{Iterations: 100, MaxConcurrent: 5},
```

### Feature options

An option can also be gated on namespace capability, rather than fixed at declaration time,
with `o.Feature`:

```go
o.Feature("include-standalone-nexus", "Include standalone Nexus operations.",
	func(c loadgen.Capabilities) bool { return c.Namespace.GetStandaloneNexusOperation() })
```

| Config | Capability reported | Result |
| --- | --- | --- |
| unset | yes | enabled |
| unset | no | disabled |
| `=false` | either | disabled |
| `=true` | yes | enabled |
| `=true` | no | run fails with a usage error |
| unset | probe failed after retries | run fails |

Resolution needs a dialed client, so it happens in `loadgen.ResolveFeatureOptions` after the run
connects, not at declaration or `ResolveOptions` time. A test that calls `Configure` directly
without also calling `ResolveFeatureOptions` sees the feature at its pflag default (`false`), not
its capability-resolved value.

**If you drive a scenario as a library rather than through the omes CLI, you must call
`loadgen.ResolveFeatureOptions` yourself** once the client is dialed. The CLI does it for you; code
that builds a `ScenarioInfo` and calls `executor.Run` directly does not get it for free. Skipping it
does not fail the run — every feature option reads `false`, so the load quietly omits the feature —
so reading an unresolved feature logs an error naming the option and this function. Treat that line
as a bug in the calling code.

A feature option cannot also be `MarkRequired`: required means the user must supply a value, gated
means the namespace supplies it. Declaring both is rejected when options resolve.

The predicate is handed a `loadgen.Capabilities`, carrying both what the namespace reports
(`DescribeNamespace`) and what the server as a whole reports (`GetSystemInfo`) — gate on whichever
answers your question. Reach for `o.Feature` only when one of them has a matching field; an ordinary
knob that every server supports stays a plain `o.Bool`.

### Conventions for options

- **Declare every option you read.** A scenario accepts exactly what it declares, and the accessors read
  from the declarations — so an undeclared option can't be passed and can't be read.
- **Give each option a usage string.** It's what a user sees in `list-scenarios`.
- **Use `MarkRequired` instead of hand-rolling a check** in `PrepareTestInput` or `Configure`; the
  framework fails the run before any load starts.
- **Don't restate options in the `Description`.** `list-scenarios` renders the declarations, so listing
  names and defaults in prose only creates something to drift.

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
3. Declare every option you read, with a type, a default or `MarkRequired`, and a usage string.
4. Declare a `DefaultConfiguration` if the scenario assumes a particular scale.
5. Put helpers other scenario authors would want in the `loadgen` package, not in your scenario file.
