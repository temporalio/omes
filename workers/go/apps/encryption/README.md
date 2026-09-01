# `encryption` project

An omes project that generates load through an **encrypting data converter**. Each iteration runs a
workflow that touches every payload-bearing path the Go SDK exposes, so the load includes an
encrypt/decrypt of every kind of payload Temporal carries rather than just workflow arguments.

> **The encryption key is fake.** `testOnlyKey` in [`codec.go`](./codec.go) is a 32-byte string
> committed in plaintext, so anything this project writes can be decrypted by anyone with the repo.
> It exists so the worker process and the project-server process share a key with no configuration.
> Never point this app at a namespace holding real data.

## Running it

The app targets **cloud namespaces**, so it makes no operator-service calls: it creates nothing in
the namespace and expects two things to already exist there.

**1. Two custom search attributes**:

| Name | Type |
| --- | --- |
| `EncryptionKeyword` | Keyword |
| `EncryptionInt` | Int |

Without them the first `UpsertTypedSearchAttributes` fails the workflow task.

**2. Optionally, a Nexus endpoint**, named in a project config file. Without one the run generates
no Nexus load and everything else still runs; `Init` logs a warning saying how to supply it. The
endpoint is a loopback — the caller is the coverage workflow and the handler is the same worker — so
its target is **this namespace** and the run's task queue, `omes-<run-id>`. That target is fixed when
the endpoint is created, so **pin `--run-id`** to a stable value and the same endpoint keeps working
across runs.

`encryption.json`:

```json
{ "nexusEndpoint": "my-encryption-endpoint" }
```

Everything else in that file is optional and covered under [Fan-out](#fan-out).

```sh
go run ./cmd/omes run-scenario-with-worker --scenario project \
  --language go --app encryption --run-id enc-load --iterations 1 \
  --server-address <namespace>.<account>.tmprl.cloud:7233 \
  --namespace <namespace> --tls \
  --option language=go --option project-name=encryption \
  --option project-config-file=encryption.json \
  --do-not-register-search-attributes
```

`--do-not-register-search-attributes` stops omes itself from trying to register its `KS_*`
attributes, which the app does not use.

The task queue is not the app's to choose: omes derives it from the run ID in both
`cmd/omes/run_worker.go` and `scenarios/project/project.go`, which is how the worker and the driver
agree on one without shared config. Pinning `--run-id` is what makes it static.

Each iteration starts three workflows: a message target (started by update-with-start, then sent
updates and later signals), the coverage workflow (which continues-as-new once, and which starts
children of its own), and a two-attempt retry chain.

Inside the coverage workflow the steps run *concurrently*. It issues every command it can before
waiting on any of them, which is what makes one `RespondWorkflowTaskCompleted` carry the whole set
rather than one command per workflow task. See [Fan-out](#fan-out) for why that matters and how to
turn it up.

## What is configured

[`app.go`](./app.go) sets three things on the client, and the Go harness routes both the worker
process and the project-server process through it, so all three apply to both sides:

- `DataConverter` — `converter.NewCodecDataConverter` over the SDK default converter, with an
  AES-GCM codec that encrypts the serialized payload proto.
- `FailureConverter` — a `DefaultFailureConverter` built with **both** `DataConverter: <ours>` and
  `EncodeCommonAttributes: true`. The `DataConverter` is the part that is easy to miss:
  `NewDefaultFailureConverter` defaults it to the *default* converter, not the client's, so failure
  and cancellation details would not go through the codec at all if you only set `DataConverter` on
  the client — and the failure paths below would generate no codec load.
- `ContextPropagators` — a propagator that writes its header field with the default converter. See
  the header note below.

## Fan-out

Every count below defaults to 1, and an unconfigured run is the one-of-each history the rest of this
README describes. Raising a count does not add requests or change which paths run — it changes how
much a single request carries. That is the difference between load that is *varied* and load that is
*dense*: the same paths either way, but many payloads per request instead of one.

```json
{
  "nexusEndpoint": "my-encryption-endpoint",
  "memoEntries": 20,
  "activityCount": 20,
  "markerCount": 20,
  "childCount": 1,
  "signalCount": 5,
  "nexusCount": 5,
  "failureDepth": 4,
  "payloadBytes": 1024,
  "concurrentUpdates": 10
}
```

| Option | Fans out | Notes |
| --- | --- | --- |
| `memoEntries` | Fields per upserted memo, and per start and child memo | `map<string, Payload>`, so each field is a payload of its own |
| `activityCount` | The echo activity and the failing activity | One `Payloads` holding the lot |
| `markerCount` | Side effects and failing local activities | Marker details; cheap, nothing gets scheduled |
| `childCount` | Echo child workflows | Each is a real execution — the expensive knob |
| `signalCount` | Signals to the target, which waits for exactly this many | |
| `nexusCount` | Nexus operations, when an endpoint is configured | |
| `failureDepth` | Levels in a failure's `cause` chain | Each level is its own `Failure` proto with its own details and `encoded_attributes` |
| `payloadBytes` | Padding on every payload body | Moves payload size independently of payload count |
| `concurrentUpdates` | Updates sent at once, beyond the one in update-with-start | The `Any` lever — see below |


### Rate

`--iterations` sets how many iterations run, not how fast. The rate lever is `--max-concurrent`
(default 10), because omes runs iterations as a bounded pool:

```
requests/sec ≈ (max-concurrent / iteration duration) × requests per iteration
```

Fan-out multiplies the payloads inside those requests, so payload throughput is reachable by turning
up either one. Preferring fan-out is what keeps the paths that take wall clock to reach — the
heartbeat timeout, the cancellation, the retry chain — in the same run as the dense load.

The sleeping activities are what bounds concurrency: a workflow waiting on a timer holds nothing,
but a sleeping activity holds a worker slot for its whole duration. Raise
`--max-concurrent-activities` alongside `--max-concurrent`.

`Init` logs the configured fan-out, so a run's log says what one iteration was worth in payloads.

## Payload paths the load covers

| Path | How it is driven |
| --- | --- |
| Schedule activity — input | `EncryptionEchoActivity` argument |
| Schedule activity — header | context propagator |
| Activity heartbeat — details | `activity.RecordHeartbeat` |
| Activity completion — result | echo activity return value |
| Activity failure — details | `EncryptionFailActivity` application error, with a cause carrying its own details |
| Activity — last heartbeat details | `EncryptionHeartbeatTimeoutActivity` heartbeat timeout |
| Activity cancellation — details | `EncryptionCancelActivity` canceled error, with `WaitForCancellation` |
| Start child — input, memo, search attributes | `ExecuteChildWorkflow` with `ChildWorkflowOptions` |
| Start child — header | context propagator |
| Signal external — input | `SignalExternalWorkflow` argument |
| Signal external — header | context propagator |
| Update-with-start — start memo, header, update argument, result | `UpdateWithStartWorkflow` starts `EncryptionTargetWorkflow` |
| Update — rejection failure | a plain `UpdateWorkflow` the handler's validator rejects |
| Upsert memo | `workflow.UpsertMemo` |
| Continue-as-new — input | `NewContinueAsNewError` argument |
| Continue-as-new — memo, search attributes | carried forward from the current run by the SDK, upserts included |
| Continue-as-new — header | context propagator |
| Continue-as-new — continued failure | `EncryptionRetryWorkflow` |
| Marker — details | `workflow.SideEffect`; failing local activity |
| Marker — failure | failing local activity (`LocalActivity` marker) |
| Version marker — details | `workflow.GetVersion`, under two change IDs |
| Complete workflow — result | return value of the run reached by continue-as-new |
| Fail workflow — failure and nested details | `EncryptionFailingWorkflow`, as a child of the coverage workflow |
| Fail workflow — message, stack trace | `EncodeCommonAttributes` moves them into `Failure.EncodedAttributes` |
| Nexus operation — input | synchronous `echo` operation, when an endpoint is configured |

Four of those paths do not go through the data converter, which is a property of the Go SDK rather
than of this app. They are still driven, because the load is about the paths existing, but no codec
work happens on them:

- **Headers.** `HeaderWriter.Set` takes an already-built `*commonpb.Payload`, and headers are written
  only by context propagators and interceptors — the client's data converter is not involved.
- **Search attributes.** The server indexes them, so no SDK routes them through a codec.
- **Marker headers.** `RecordMarkerCommandAttributes` has a `Header` field, but no Go SDK code path
  sets it, for any marker kind. There is no way to drive it.
- **Version markers.** `GetVersion` records the version the SDK needs for replay, not a user value,
  and the client's codec is not involved. 
