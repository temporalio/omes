# Scheduler read recovery

`scheduler_stress` accepts `--option retry-schedule-reads=true`. Its default is
false. When enabled, a Describe that returns a typed RPC `DeadlineExceeded`
while its parent context is live gets one additional Describe on the same
schedule handle, after a cancelable 100ms delay. The additional call is bounded by
the earlier of the original schedule EndAt and parent deadline; neither the
scheduling window nor the scenario duration/drain is extended. If less than the
delay remains, no retry starts. A second failure still fails the iteration.

This is deliberately a read-only slice of operation recovery. Create, Update,
Delete, GenericExecutor iteration attempts and load-activity retries are
unchanged. Returned invalid-input, authorization, cancellation and
Unavailable/ResourceExhausted statuses do not enter a new general retry loop. An SDK child deadline can instead
convert prior retryable-status failures into a typed DeadlineExceeded; that
result qualifies while the parent remains live. The classifier cannot identify
server origin or establish how many SDK retries preceded it.
Existing Describe/Update/Delete NotFound handling remains compatible. Missing
resources are not evidence of acknowledged CRUD or scheduled action execution.

## Why recover this operation

In an OSS persistence-fault calibration, all eight throughput/scheduler loads
failed after repeated outer activity attempts. The earliest retained scheduler
failures included Create, Describe and Update persistence timeouts. Some later
error details were truncated; the loads eventually failed an expired-progress
guard after repeated activity retries. These are
observed failures, not proof of a server or heartbeat implementation defect.

The present scheduler records a failed Describe, continues the remaining
operations and cleanup, then returns that error. GenericExecutor can retry the
whole iteration, creating fresh schedule IDs and repeating unrelated work. A
transient read failure therefore discards the iteration even when another read
of the same acknowledged schedule could succeed. Whole-iteration retries are
not a substitute for that same-resource read.

The [pinned Go SDK transport retry classifier](https://github.com/temporalio/sdk-go/blob/f161b6560fe5/internal/common/retry/interceptor.go#L101-L111)
excludes DeadlineExceeded and Canceled. It already retries several other
statuses inside the public call;
a shorter SDK child deadline can still expire during a call or its retry
backoff and surface as the same typed DeadlineExceeded. Recovery can therefore
add another public call after earlier SDK transport attempts; it does not prove
that the server itself returned a deadline. The
[sdk/gRPC regression](../scenarios/scheduler_read_recovery_rpc_test.go) sends a
server DeadlineExceeded through the real SDK, plus SDK-child-deadline and
retryable-status/backoff exhaustion with a live parent. The default path fails;
the enabled path succeeds without replaying Create, Update or Delete. The
[stateful tests](../scenarios/scheduler_read_recovery_test.go) cover exhaustion,
cancellation, scheduling-window expiry, and committed writes with lost responses.
They reproduce the recovery boundary, not the original server fault's transport
origin or frequency. No new live FI result is claimed.

## Policy and limits

One extra read after 100ms limits new logical request fanout to two Describes for
an affected read. SDK-internal retries still occur within each call; two logical
calls do not mean two physical RPCs or a universal 20s bound. The original
scheduling window and context can cut the retry short. The delay is a small
bounded recovery policy, not a promise that a transient failure clears in 100ms.
This remains opt-in because it changes latency and load under fault.

This option does not replay writes or accept them by comparing matching state.
A create that committed before its response was lost still fails and can leave a schedule for
parent namespace cleanup. Update/delete response loss remains failure even if
the effect happened. Read/update failures retain the existing delete attempt;
cancellation can prevent it. EndAt stops scheduling and does not delete the
object. The repeated read retains the namespace/client and schedule ID of this
iteration's acknowledged create. Its result does not prove ownership or
authorize a new mutation. Write and cleanup authority are unchanged. It adds no
cleanup budget or detached context.

Leaving the default strict path unchanged is valid when extra reads are
undesired. Retrying ambiguous writes would need separate identity/protocol and
cleanup work; workflow-start reconciliation and bounded error envelopes are also
separate changes. This read slice does not fix the observed finite heartbeat,
retryable-error or timeout action failures, and cannot qualify FI or a full
longhaul run. It does not enable iteration-failure tolerance or alter verdicts.
