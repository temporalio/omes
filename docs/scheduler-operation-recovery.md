# Scheduler operation recovery

`--option recover-schedule-operations=true` enables operation recovery for the
whole scheduler workload. It defaults to false. It does not change GenericExecutor or activity retries.

Each operation shares a budget of four logical calls, including probes, and
five seconds including SDK calls and cancelable 100ms inter-call waits. Create,
read and update are also bounded by the original EndAt; delete may clean up
after EndAt while the parent remains live. SDK Update itself performs Describe
and Update, and SDK transport retries may send multiple physical RPCs per logical
call. This budget is not four RPCs or four independent five-second timeouts.

Create freezes its ID, spec, action, overlap and EndAt before the first call.
An ownership Memo identifies the create intent. After an ambiguous error, a
matching owned object is recorded as observed creation, not an acknowledged
Create RPC. An absent object permits another Create with the same ID and frozen
intent; unresolved or foreign objects remain failures.

Update gives each intended mutation a fresh State.Note witness, preserving the
schedule/action window and payload. A failure in SDK's initial Describe can retry
the SDK call. After a mutation may have been sent, recovery only probes: the
fresh witness must be observed with the owned intent. Matching the old payload
or passing EndAt is not proof of update execution. If the witness is not observed
within budget, the outcome remains unknown and fails. No ambiguous Update is
replayed by this layer. A server overwrite of Note can prevent reconciliation.

Delete probes ownership before writing and after ambiguous failure. Only the
same owned ID can be retried. Observed absence and direct NotFound keep existing
compatibility without claiming a Delete or Update acknowledgment. Unknown
objects remain the native namespace cleanup owner's responsibility; cancellation
never starts detached cleanup.

Ownership assumes Omes' fresh random IDs are not reused by another writer.
Memo, action identity/payload, overlap and EndAt are checked; canonicalized cron
specs are not compared byte-for-byte. The SDK has no conditional Delete fence
against concurrent replacement. Default payload encoding is used for validation;
incompatible encodings fail closed. This is not a general multi-writer schedule
manager or an exactly-once protocol.

Info-level operation summaries distinguish acknowledgments, observed effects,
compatible NotFound and unresolved errors, with schedule/mutation identity,
logical-call count, error codes and elapsed time. No per-ID metrics are added. Fault settings, iteration
acceptance, unavailable-progress handling and coverage remain strict.
