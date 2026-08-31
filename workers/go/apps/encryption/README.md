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

**1. Two custom search attributes**, provisioned however your namespace does it (tcld, the Cloud UI):

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

Each iteration starts four workflows, sequentially: a message target (started by update-with-start,
then sent a rejected update and later a signal), the coverage workflow (which continues-as-new
once), a workflow that fails, and a two-attempt retry chain. An iteration is not a throughput benchmark — the load it generates is
*varied*, not fast.

## What is configured

[`client.go`](./client.go) sets three things on the client, and the Go harness routes both the worker
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
| Complete workflow — result | return value of the run reached by continue-as-new |
| Fail workflow — failure and nested details | `EncryptionFailingWorkflow`, run standalone and as a child |
| Fail workflow — message, stack trace | `EncodeCommonAttributes` moves them into `Failure.EncodedAttributes` |
| Nexus operation — input | synchronous `echo` operation, when an endpoint is configured |

Three of those paths do not go through the data converter, which is a property of the Go SDK rather
than of this app. They are still driven, because the load is about the paths existing, but no codec
work happens on them:

- **Headers.** `HeaderWriter.Set` takes an already-built `*commonpb.Payload`, and headers are written
  only by context propagators and interceptors — the client's data converter is not involved. Every
  outbound call also starts from a *fresh, empty* header map, so headers do not exist at all unless
  something writes them on each call. That is the only reason this app has a propagator.
- **Search attributes.** The server indexes them, so no SDK routes them through a codec;
  `NewPayloadCodecGRPCClientInterceptor` explicitly sets `SkipSearchAttributes`.
- **Marker headers.** `RecordMarkerCommandAttributes` has a `Header` field, but no Go SDK code path
  sets it, for any marker kind. There is no way to drive it.

## Notes on the awkward paths

**Heartbeat details.** Heartbeats live in mutable state, not history, so `RecordHeartbeat` produces
no history event — but the payload does go over the wire through the converter, which is the load
that matters here. `EncryptionHeartbeatTimeoutActivity` is the one path that also lands it in
history, as `TimeoutFailureInfo.LastHeartbeatDetails`.

**Search attributes.** These carry no codec load at all, since the server indexes them. They are
here so the start-child and continue-as-new commands have the field shape a real workflow would, and
they are the one part of the app with a namespace prerequisite. If that prerequisite is more trouble
than the paths are worth, deleting them is a contained change: two constants, two vars, one upsert
call, and the `TypedSearchAttributes` on the child and the start.

**Update-with-start.** The target workflow is started with `UpdateWithStartWorkflow` rather than
`ExecuteWorkflow`, so the client issues one `ExecuteMultiOperation` holding a
`StartWorkflowExecutionRequest` and an `UpdateWorkflowExecutionRequest` inside operation wrappers.
That is the deepest payload-bearing request the client sends, and this one carries the start memo,
the propagated header, and the update argument at three different nesting levels. The rejected
update is sent separately as a plain `UpdateWorkflow`, so both RPCs are covered.

Nothing in the SDK truncates at that depth — the `proxy` payload walker behind
`NewPayloadCodecGRPCClientInterceptor` has no production depth cap, and the last argument to the
SDK's own `visitProtoPayloads` is a concurrency limit rather than a depth. The value here is that
anything walking a request tree to find payloads has to descend the whole way, and no other request
this app sends is shaped like that.

**Continued failure.** The Go continue-as-new command sets no last-run fields, so a plain
continue-as-new produces no continued failure. The server does populate one across a retry chain, so
`EncryptionRetryWorkflow` is started with `MaximumAttempts: 2` and fails its first attempt with a
*retryable* error — a non-retryable one gets no second attempt. The server puts that failure on the
second attempt's started event as `ContinuedFailure`, where the SDK decodes it through the failure
converter. It is the one place in the app where a previously encoded failure comes back in as a new
run's input.

The sibling field `LastCompletionResult` is not covered: only cron and schedule chains produce it,
and driving one costs a poll loop and a chain to terminate for a payload that is an ordinary
data-converter value much like a workflow result.

**Steps that fail on purpose.** Several paths only produce a payload by failing, so the workflow
swallows their errors and keeps going. That those steps really do fail is covered by
[`workflow_test.go`](./workflow_test.go) rather than checked at run time.
