For background context, please study the following documents carefully:

START_DOCUMENT------------------------------------------------------------------------------
# Temporal Activity Execution & saas-temporal Cloud Persistence: Implementation Overview

## Part 1: Activity Execution Models in Temporal Server

### 1.1 CHASM Standalone Activities (`chasm/lib/activity/`)

CHASM standalone activities are first-class, independently-scheduled executions outside workflow context. They use **mutable state only** -- no history events.

#### State Machine

States defined in `chasm/lib/activity/proto/v1/activity_state.proto`:

```
UNSPECIFIED
  → SCHEDULED
    → STARTED
      → COMPLETED (terminal)
      → FAILED (terminal)
      → CANCEL_REQUESTED → CANCELED (terminal)
      → TIMED_OUT (terminal)
      → TERMINATED (terminal)
    → CANCEL_REQUESTED → CANCELED (terminal)
    → TIMED_OUT (terminal)
    → TERMINATED (terminal)
    → SCHEDULED (retry path)
```

Lifecycle states (`activity.go:95-107`):
- `LifecycleStateRunning`: SCHEDULED, STARTED, CANCEL_REQUESTED
- `LifecycleStateCompleted`: COMPLETED
- `LifecycleStateFailed`: FAILED, TERMINATED, TIMED_OUT, CANCELED

#### State Transitions (`statemachine.go`)

| Transition | From | To | Trigger |
|---|---|---|---|
| TransitionScheduled (37-77) | UNSPECIFIED | SCHEDULED | Initial scheduling |
| TransitionRescheduled (87-127) | STARTED | SCHEDULED | Retry after failure |
| TransitionStarted (130-169) | SCHEDULED | STARTED | Worker accepts task |
| TransitionCompleted (177-202) | STARTED/CANCEL_REQUESTED | COMPLETED | Worker completes |
| TransitionFailed (210-237) | STARTED/CANCEL_REQUESTED | FAILED | Non-retryable failure |
| TransitionCancelRequested (278-295) | STARTED/SCHEDULED | CANCEL_REQUESTED | Cancel API called |
| TransitionCanceled (304-331) | CANCEL_REQUESTED | CANCELED | Worker acknowledges cancel |
| TransitionTerminated (246-275) | SCHEDULED/STARTED/CANCEL_REQUESTED | TERMINATED | Terminate API called |
| TransitionTimedOut (340-374) | SCHEDULED/STARTED/CANCEL_REQUESTED | TIMED_OUT | Timer task fires |

#### Mutable State Structures

**ActivityState** (proto):
- `activity_type`, `task_queue`, timeouts (`schedule_to_close`, `schedule_to_start`, `start_to_close`, `heartbeat`), `retry_policy`, `status`, `schedule_time`, `priority`, `cancel_state`, `terminate_state`

**Activity Go Component** (`activity.go:52-68`):
- `ActivityState` (embedded proto)
- `Visibility: chasm.Field[*chasm.Visibility]` -- search attributes
- `LastAttempt: chasm.Field[*ActivityAttemptState]` -- attempt count, stamp, started_time, failure details, worker identity
- `LastHeartbeat: chasm.Field[*ActivityHeartbeatState]` -- heartbeat details and recorded_time
- `RequestData: chasm.Field[*ActivityRequestData]` -- input, header, user_metadata
- `Outcome: chasm.Field[*ActivityOutcome]` -- successful (output) or failed (failure)
- `Store: chasm.ParentPtr[ActivityStore]` -- parent workflow (nil for standalone)

#### Task Flow

1. **Scheduling** (`handler.go:51-104`): `StartActivityExecution()` → creates Activity → applies TransitionScheduled
2. **Dispatch** (`activity_tasks.go:21-79`): `activityDispatchTaskExecutor` pushes to matching service via `AddActivityTask()`
3. **Start** (`activity.go:173-191`): `HandleStarted()` applies TransitionStarted, schedules start-to-close and heartbeat timeout tasks
4. **Completion** (`activity.go:259-280`): `HandleCompleted()` applies TransitionCompleted
5. **Failure** (`activity.go:284-323`): `HandleFailed()` checks retryability → either `tryReschedule()` or TransitionFailed
6. **Heartbeat** (`activity.go:559-586`): Updates LastHeartbeat, reschedules heartbeat timeout task

#### Timeout Tasks

- **ScheduleToStartTimeoutTask** (`activity_tasks.go:81-116`): Non-retryable → TIMED_OUT
- **ScheduleToCloseTimeoutTask** (`activity_tasks.go:118-150`): Non-retryable → TIMED_OUT
- **StartToCloseTimeoutTask** (`activity_tasks.go:152-198`): Attempts retry via `tryReschedule()`; if not retryable → TIMED_OUT
- **HeartbeatTimeoutTask** (`activity_tasks.go:200-276`): Validates heartbeat recency; attempts retry; if not retryable → TIMED_OUT

#### Retry Logic

- `shouldRetry()` (`activity.go:504-514`): Checks TransitionRescheduled possible, attempt < max, enough time remaining
- `hasEnoughTimeForRetry()` (`activity.go:518-534`): Exponential backoff calculation against schedule-to-close deadline
- `tryReschedule()` (`activity.go:489-502`): Applies TransitionRescheduled (increments attempt, schedules dispatch with backoff)

#### Cancellation

- `RequestCancelActivityExecution` (`handler.go:273-296`): Applies TransitionCancelRequested
  - If SCHEDULED: immediately applies TransitionCanceled (`activity.go:414-433`)
  - If STARTED: stays CANCEL_REQUESTED; worker receives cancellation on next interaction

---

### 1.2 Legacy Workflow Activities

Activities executed as part of a workflow use **mutable state (ActivityInfo) plus history events**.

#### History Events

```
EVENT_TYPE_ACTIVITY_TASK_SCHEDULED (10)
EVENT_TYPE_ACTIVITY_TASK_STARTED (11)
EVENT_TYPE_ACTIVITY_TASK_COMPLETED (12)
EVENT_TYPE_ACTIVITY_TASK_FAILED (13)
EVENT_TYPE_ACTIVITY_TASK_TIMED_OUT (14)
EVENT_TYPE_ACTIVITY_TASK_CANCEL_REQUESTED (15)
EVENT_TYPE_ACTIVITY_TASK_CANCELED (16)
```

#### ActivityInfo Mutable State (`persistence/v1/executions.proto:524-661`)

Core: `activity_id`, `activity_type`, `task_queue`, `scheduled_time`, `started_time`, `started_event_id`, `scheduled_event_id`

Timeouts: `schedule_to_close_timeout`, `schedule_to_start_timeout`, `start_to_close_timeout`, `heartbeat_timeout`

Retry: `attempt`, `has_retry_policy`, `retry_initial_interval`, `retry_maximum_interval`, `retry_maximum_attempts`, `retry_backoff_coefficient`, `retry_expiration_time`, `retry_non_retryable_error_types`, `retry_last_failure`

State flags: `cancel_requested`, `cancel_request_id`, `timer_task_status` (bit flags), `stamp`, `paused`, `pause_info`

#### Pending Activity States (`activity.go:53-61`)

- SCHEDULED: `StartedEventId == 0`
- STARTED: `StartedEventId != 0 && !CancelRequested`
- CANCEL_REQUESTED: `CancelRequested`
- PAUSED: `Paused && Scheduled`
- PAUSE_REQUESTED: `Paused && Started`

#### Timer Task Status Flags

```go
TimerTaskStatusCreatedScheduleToStart = 1
TimerTaskStatusCreatedScheduleToClose = 2
TimerTaskStatusCreatedStartToClose    = 4
TimerTaskStatusCreatedHeartbeat       = 8
```

#### Pause/Unpause/Reset (unique to legacy model)

- **Pause** (`activity.go:254-284`): Sets `paused = true`, increments stamp if SCHEDULED
- **Unpause** (`activity.go:388-425`): Clears pause, regenerates retry task if SCHEDULED
- **Reset** (`activity.go:286-379`): Resets attempt to 1, optionally resets heartbeat/options

#### API Handlers (`service/history/api/`)

- `recordactivitytaskstarted/api.go`: Creates ActivityTaskStartedEvent
- `respondactivitytaskcompleted/api.go`: Creates ActivityTaskCompletedEvent
- `respondactivitytaskfailed/api.go`: Retry or ActivityTaskFailedEvent
- `respondactivitytaskcanceled/api.go`: Creates ActivityTaskCanceledEvent
- `recordactivitytaskheartbeat/api.go`: Updates heartbeat state, reschedules timeout

---

### 1.3 Activity Metrics (Both Models)

Defined in `common/metrics/metric_defs.go`. Both models emit the same metric names.

**Counters:**
| Metric | Description |
|---|---|
| `activity_success` | Successful completions (excludes retries) |
| `activity_fail` | Final failures (retries exhausted) |
| `activity_task_fail` | Per-attempt failures (includes retries) |
| `activity_cancel` | Canceled activities |
| `activity_terminate` | Terminated activities (CHASM only) |
| `activity_timeout` | Terminal timeouts |
| `activity_task_timeout` | Per-timeout events (includes retries) |

**Timers:**
| Metric | Description |
|---|---|
| `activity_start_to_close_latency` | StartedTime → completion/failure/timeout |
| `activity_schedule_to_close_latency` | ScheduleTime → completion/failure/timeout/cancel |

**Tags:** `namespace`, `task_queue_family`, `operation`, `activity_type`, `versioning_behavior`, `workflow_type` (set to `__temporal_standalone_activity__` for CHASM). Timeout metrics additionally tagged with `timeout_type` (SCHEDULE_TO_START, SCHEDULE_TO_CLOSE, START_TO_CLOSE, HEARTBEAT).

**Metric enrichment** (`activity.go:804-824`): `enrichMetricsHandler()` adds per-task-queue-family scoping via `metrics.GetPerTaskQueueFamilyScope()`.

---

### 1.4 Key Differences

| Aspect | CHASM Standalone | Legacy Workflow |
|---|---|---|
| Persistence | Mutable state only | Mutable state + history events |
| Parent context | Standalone execution | Part of workflow execution |
| State tracking | ActivityState + sub-components | ActivityInfo in workflow |
| Task dispatch | Direct to matching service | Via workflow task completion |
| Completion storage | Outcome field | History events |
| Cancellation | Explicit CANCEL_REQUESTED state | Boolean flag in ActivityInfo |
| Pause support | Not yet implemented | Full (pause, unpause, reset) |
| Search attributes | Visibility component (chasm) | Workflow search attributes |

---

## Part 2: saas-temporal Cloud Integration

### 2.1 Architecture Overview

saas-temporal wraps the Temporal server to run in Temporal Cloud cells by replacing core persistence with Cloud Data Storage (CDS), backed by:
- **Datastax Astra Cassandra** for durable storage
- **Write-Ahead Logs (WALs)** for durability before Cassandra persistence
- **OpenSearch/Elasticsearch** for workflow visibility
- **Tiered Storage** (S3/GCS/Azure) for history archival

### 2.2 Entry Point and Server Construction

**Main:** `cmd/temporal-service/main.go`

The `start` command:
1. Loads OSS Temporal configuration from YAML
2. Injects secrets (Astra, Elasticsearch credentials)
3. Sets up dynamic configuration
4. Optionally enables cloud metrics handler (Chronicle)
5. Configures authorization (SaaS Auth0 JWT + Temporal JWT)
6. Configures custom datastore with CDS
7. Creates server via `cds.NewServer()`

**Server creation:** `cds/export/cds/server.go`:
```go
func NewServer(serviceFxOpts FxOptions, opts ...temporal.ServerOption) (temporal.Server, error) {
    return newServerFx(TopLevelModule, serviceFxOpts, opts...)
}
```

Uses Uber FX dependency injection with modules for persistence factory, dynamic config, serialization, and per-service modules (history, matching, frontend, worker).

### 2.3 CDS Factory Architecture (`cds/export/cds/factory.go`)

**FactoryProvider** (lines 51-65): Implements `client.AbstractDataStoreFactory`
- `NumberOfShards`, `OrderedDatastoreConfigs` (shards → datastores)
- `HistoryDatastoreConfigs` (weighted distribution)
- `WALFollowerProviders` for WAL followers
- `Clock`, `DynamicConfig`, `ChasmRegistry`

**Factory**: Manages three WAL pools:
- **MS WAL** (MutableState): Records mutable state mutations
- **HE WAL** (HistoryEvent): Records history events
- **LP WAL** (LargePayload): Records oversized payloads

Plus store providers: `MultiDBStoreProvider` for ordinal datastores, separate history store provider with tiered storage, optional Walker integration.

### 2.4 Astra Cassandra Integration (`cds/storage/cassandra/astra/`)

**Session creation** (`gocql.go`): Wraps gocql with Astra-specific config (TLS, connection pooling, retry policies) via Datastax `gocql-astra`.

**Query instrumentation** (`gocql_metrics.go:48-100`): `queryMetricsObserver` instruments every query with 150-entry LRU statement cache.

**Cassandra Metrics:**
| Metric | Description |
|---|---|
| `CassandraConns` | Connection count |
| `CassandraQueryTotalLatency` | Query latency |
| `CassandraBatchTotalLatency` | Batch latency |
| `CassandraQuery` | Query count |
| `CassandraBytesTx` / `CassandraBytesTx` | Network bytes |
| `CassandraLargeResponse` / `CassandraLargeRequest` | Large payload detection |
| `CassandraRetries` | Retry histogram |
| `CassandraErrors` | Error counters |

Tags: `OperationType` (INSERT/UPDATE/DELETE/SELECT), `TableName`, `CasTag` (CAS operation)

### 2.5 Write-Ahead Logs (`cds/export/wal/`, `cds/stream/`)

WALs provide durability guarantees before data reaches Cassandra.

**WAL Client Interface** (`cds/export/wal/crud.go`):
```go
WriteMS(), WriteHE(), WriteLP()  // Write operations per pool
ReadMS(), ReadHE(), ReadLP()     // Read operations per pool
```

**Configuration** (`cds/config/configs.go:46-140`):
- Rate limiting: `WALReadsRate`, `WALReadsBurst`
- Timeouts: `WALDialTimeout`, `WALReadTimeout`, `WALWriteTimeout`
- Ledger rotation: `WALLedgerRotationBytesThreshold`, `WALLedgerRotationAgeThreshold`
- Retention: `WALLedgerLifetime`
- Parallelism: `WALMaxParallelReads`
- Feature flags: `WALReadV2Enabled`, `WALV2EncodingEnabled`

**WAL Metrics** (`cds/metrics/metrics.go:34-56`):
| Metric | Description |
|---|---|
| `wal_latency` | Operation latency |
| `wal_stream_dial_attempt/success/error` | Connection establishment |
| `wal_stream_dns_latency` | DNS resolution |
| `wal_stream_connect_latency` | TCP connect |
| `wal_stream_handshake_latency` | TLS handshake |
| `wal_stream_send/receive_latency` | I/O latency |
| `wal_health_check_failed_count` | Connection health |
| `wal_write_timeout_count` | Timeout tracking |
| `wal_reader_page_latency` | Page read latency |
| `wal_entries_per_read` | Batch size histogram |
| `wal_compression_count` | Compression events |

**Flush Metrics** (lines 13-27):
| Metric | Description |
|---|---|
| `flush_latency` | Time to flush to persistence |
| `flush_error` | Flush failures |
| `flush_snapshot_aborts` | Snapshot abort count |
| `flush_persistence_behindness_bytes/count/time` | Persistence lag |
| `flush_time_since_last_persist` | Staleness |
| `flush_reason_count` | Flush trigger reasons (by namespace) |

**Recovery Metrics** (lines 57-70):
| Metric | Description |
|---|---|
| `recovery_total_latency` | Full recovery duration |
| `recovery_open_reader_latency` | Snapshot reader open |
| `recovery_rate_limiter_latency` | Rate limiting delay |
| `recovery_first_read_latency/bytes` | Initial WAL read |
| `recovery_takeover_latency` | Takeover phase |
| `recovery_wal_update_latency` | WAL update during recovery |

**Ledger Metrics** (lines 77-82):
| Metric | Description |
|---|---|
| `ledger_rotation_count` | Rotations |
| `logs_per_ledger` | Logs per ledger histogram |
| `segments_per_shard` | Segments per shard histogram |
| `segment_too_old_count` | GC candidates |
| `active_segment_too_old_count` | Rotation delay |

### 2.6 Execution Store Wrapper (`cds/export/cds/execution_store.go`)

Wraps the Cassandra execution store to:
- Convert mutable state mutations to WAL records (`NewMSWALRecord()`)
- Convert history events to WAL records (`NewHEWALRecord()`)
- Calculate storage metering
- Manage snapshot trimming
- Implement history event caching

Implements `persistence.ExecutionStore` and `persistence.ShardStore`.

### 2.7 How Activity State Flows Through CDS

**CHASM activities**: Activity mutable state → MS WAL write → Cassandra persistence. No HE WAL involvement (no history events). State transitions are persisted as mutable state mutations via the execution store wrapper.

**Legacy workflow activities**: ActivityInfo mutable state → MS WAL write → Cassandra. History events (Scheduled, Started, Completed, etc.) → HE WAL write → Cassandra. Both paths go through the execution store wrapper's WAL record conversion.

### 2.8 OpenSearch/Elasticsearch Visibility (`visibility/`)

**Factory:** `visibility/factory.go` -- `VisibilityStoreFactory` creates visibility stores configured per cloud cell.

**Batch processor metrics** (`visibility/common/metrics_defs.go`):
| Metric | Description |
|---|---|
| `visibility_batch_processor_request_add_latency` | Enqueue time |
| `visibility_batch_processor_request_latency` | Total request latency |
| `visibility_batch_processor_request_errors` | Failed requests |
| `visibility_batch_processor_commit_latency` | Batch commit time |
| `visibility_batch_processor_batch_size` | Items per batch histogram |
| `visibility_batch_processor_batch_requests` | Requests per batch histogram |
| `visibility_batch_processor_queued_requests` | Queue depth histogram |
| `visibility_batch_processor_corrupted_data` | Data integrity failures |
| `visibility_batch_processor_duplicate_request` | Deduplication events |

### 2.9 Tiered Storage (`cds/persistence/tieredstorage/`)

Long-term history archival to cloud object stores:
- S3 (AWS): `s3_store.go`
- GCS (Google Cloud): `gcs_store.go`
- Azure Blob: `azure_client.go`

Interface: `Upload()`, `Read()`, `Delete()`, `List()`, `PluginName()`

Metrics: `ReadWorkflowHistory`, `UploadWorkflowHistory`, `DeleteWorkflowHistory`, `ListTieredStorageObjects`

### 2.10 Persistence Store Metrics (`cds/persistence/metrics/defs.go`)

**Store layer** (lines 70-85):
| Metric | Description |
|---|---|
| `store_requests` | Request count by operation |
| `store_latency` | Operation latency |
| `store_errors` | Errors: shard_exists, shard_ownership_lost, condition_failed, timeout, unavailable |

**Manager layer** (lines 89-102):
| Metric | Description |
|---|---|
| `saas_persistence_requests` | High-level request count |
| `saas_persistence_latency` | High-level latency |
| `saas_persistence_errors` | Error tracking |

Tags: `operation` (CreateShard, UpdateShard, GetWorkflowExecution, etc.), `component`, `cass_cluster`

### 2.11 Cloud Metrics Infrastructure

**Handler chain** (`cloudmetricshandler/delegating_recorders.go`):
1. `allowlistedRecorder`: Filters through allowlist
2. `multiRecorder`: Sends to multiple backends

**Chronicle integration** (`cloudmetricshandler/chronicle_recorder.go`):
- Enabled by `TEMPORAL_ENABLE_CLOUDMETRICSHANDLER`
- Config: `/etc/temporal/cloudmetricshandler`
- Kubernetes enrichment: pod name, namespace, labels
- Backends: S3 writer, HTTP writer (to Chronicle service)
- Batch config: 50K queue, 25K batch, 100ms flush

**Action metering** (`actionmetering/metrics.go`):
- `billable_action_count` with tags: namespace, action_type, workflow_type, workflow_task_queue
- Activity type/task queue currently placeholder `"_unknown_"` with TODOs for standalone activity support

### 2.12 Additional Cloud Features

- **Authorization**: SaaS Auth0 JWT + Temporal JWT, TLS client certs
- **Quotas/Flow Control** (`quotas/`, `flowcontrol/`): Request-level and task-queue quotas
- **Multi-region replication** (`cds/service/history/replication/`): Custom replication filters
- **Metering V3**: S3/GCS/Azure bucket metering
- **SMS (etcd)**: Secondary Metadata Store for namespace/cluster metadata
- **Dynamic config**: 150+ hot-reloadable properties (`cds/config/configs.go`)
END_DOCUMENT--------------------------------------------------------------------------------------

START_DOCUMENT------------------------------------------------------------------------------
# Standalone Activity COGS and margins

@Dan Davison March 17, 2026

We want to ensure that we are billing in a way that meets our target margins for new product features in cloud, such as new CHASM execution types. To do this, we need to know certain things about COGS (cost of goods sold) for these features. This document outlines how to estimate COGS for Standalone Activity relative to Workflow and the implications of this for margins.

# Motivation: avoiding cannibalization

We have rules (see [temporalio/action](https://github.com/temporalio/action)) specifying how customer operations map to billable Actions. For example, suppose a customer executes a Workflow that executes a single Activity, which succeeds on first attempt without heartbeating. This incurs 2 Actions (StartWorkflow and ScheduleActivity). We’ll call this a “Single Activity Workflow” (SAW).

We haven’t yet decided how we will bill for Standalone Activity (SAA). But suppose that we decide that executing a single SAA (no retries, no heartbeating) is 1 Action (StartStandaloneActivity).

If we want SAA margins to match SAW margins, then we want the COGS of SAA (no retries, no heartbeating) to be ≤ 1/2 that of SAW (because we get half as much revenue for the SAA). If it is not, then there would be some degree of cannibalization (customers switch their single-activity workloads to SAA, but our margins there are worse). We’d hope it would be offset by increased volume, but we’d still prefer SAA margins to match SAW.

### What about retries and heartbeating?

SAW (no retries and no heartbeating) is 2 Actions. If the activity retries once it becomes 3 Actions (ScheduleActivity now happens twice); if it heartbeats once during each attempt it becomes 5 Actions.

Let’s assume (as we currently intend) that we apply the same billing rules to Standalone Activity retries and heartbeating. Then, as long as SAA is not worse than Workflow Activity with respect to COGS of retries and heartbeating, our margins from those customer operations will be at least as good under SAA as when they are done in the context of a pre-CHASM workflow. CHASM has been designed for efficiency; we have reason to be optimistic that it’s not *worse* than the legacy workflow activity implementation.

# Problem statement

The above suggests that we should focus on estimating the ratio of COGS for Standalone Activity (SAA) relative to Single-activity Workflow (SAW) in the no retries, no heartbeating case:

$$
R = \frac{C_{SAA}}{C_{SAW}}.
$$

We expect $R < 1$ because SAA achieves execution of an activity with fewer RPCs, persistence operations, etc, than SAW. We are hoping that it is less than 1/2 since then our SAA margins are as good or better than our workflow margins, assuming we bill 1 Action for SAA.

# Estimating the COGS ratio

We’ll assume that the COGS for a SAA or SAW execution results solely from invoices from third parties relating to cloud compute resources. COGS for an execution type (SAA or SAW) is the sum of price ($p$) times quantity consumed ($q$) over all resources:

$$
C = \sum_{i} p_i q_i.
$$

We want the COGS ratio $R$. We can write that as a weighted average of per-resource usage ratios:

$$
R = \frac{C_{SAA}}{C_{SAW}} = \sum_i f_i r_i.
$$

This allows us to calculate $R$ as a function of two things that we can estimate:

- $f_i = p_i q_{i}(SAW) / \sum_j p_j q_{j}(SAW)$ is the fraction of SAW COGS attributable to resource $i$ (“spend share”). We’ll use our current cloud spend for this.
- $r_i = q_i(SAA) / q_i(SAW)$ is the per-resource usage ratio. We will estimate these by comparing the implementations or by running experiments in cloud cells.

The resources ($i$) potentially include:
1. Data egress
2. CPU usage
3. Memory usage
4. Persistence operations against our WALs
5. Persistence operations against Astra (to be replaced by Walker)
6. Persistence operations against OpenSearch (visibility)
7. Metrics/logs processing and storage costs, Clickhouse

*At-rest data storage is excluded: we bill customers separately for storage on a GB/h basis, so it does not need to be subsidized by Actions. (Tangentially, it’s worth noting that we expect SAA storage to cost users half what they’d pay for SAW since SAW stores the input and output payloads in both workflow scheduled/complete events and activity scheduled/complete events.)*

# Per-resource usage ratios

To proceed, we need to estimate the SAW vs SAA usage ratio ($r_i$) for each resource.

The following table summarizes the two implementations. It describes the simplest possible happy-path scenario: an activity that succeeds on first attempt without heartbeating, via sync matches.

| # | Single-activity Workflow | Standalone Activity |
| --- | --- | --- |
| 1 | RPC: `StartWorkflowExecution` => HEWAL, MSWAL; Vis&; Cassandra& | RPC: `StartActivityExecution` => MSWAL; Vis&; Cassandra& |
| 2 | Task => RPC: `AddWorkflowTask` |  |
| 3 | RPC: `RecordWorkflowTaskStarted` => HEWAL, MSWAL; Cassandra& |  |
| 4 | RPC: `RespondWorkflowTaskCompleted` => HEWAL, MSWAL; Cassandra& |  |
| 5 | Task => RPC: `AddActivityTask` | Task => RPC: `AddActivityTask` |
| 6 | RPC: `RecordActivityTaskStarted` => HEWAL, MSWAL; Cassandra& | RPC: `RecordActivityTaskStarted` => MSWAL; Cassandra& |
| 7 | RPC: `RespondActivityTaskCompleted` => HEWAL, MSWAL; Cassandra& | RPC: `RespondActivityTaskCompleted` => MSWAL; Vis&; Cassandra& |
| 8 | Task => RPC: `AddWorkflowTask` |  |
| 9 | RPC: `RecordWorkflowTaskStarted` => HEWAL, MSWAL; Cassandra& |  |
| 10 | RPC: `RespondWorkflowTaskCompleted` => HEWAL, MSWAL; Vis&; Cassandra& |  |
- `&` indicates a write that’s not on the sync response path
- `AddWorkflowTask` and `AddActivityTask` involve inter-service RPCs but no persistence writes in the happy path (“sync match”).
- The table does not show worker poll requests
- An additional `Vis&` is incurred in both cases when the execution is deleted.

Comparing the implementations in the table gives

$$
r_{\text{WAL}} = \frac{3}{14} = 0.21,~~~~
r_{\text{Cass}} = \frac{3}{7} = 0.43,~~~~
r_{\text{Vis}} = \frac{3}{3} = 1.0.~~~~
$$

These ratios count writes only. Cassandra reads are not expected to differ much between SAW and SAA since they use similar caching mechanics with the result that a high proportiion of both SAW and SAA executions incur ~1 read (on execution creation);.

In addition, we can estimate data transfer costs by comparing the implementations. These are likely dominated by egress to customer infra (ingress is free on AWS and GCP; data transfers to Astra, OpenSearch, and Grafana are in-VPC or via PrivateLink). Let the activity input and output payload sizes be $S_I$ and $S_O$. Payload egress for SAW is $2S_I + 2S_O$ (input payload sent to workflow and activity workers; output payload sent to workflow worker and client). For SAA this is $S_I + S_O$ since there is no workflow worker detour. This gives

$$
r_\text{data\_transfer} = 0.5.
$$

# COGS ratio estimate

Using approximate/preliminary cloud spend share numbers (thanks @Stephen Chan ) we have:

| **Resource** | **Spend share $f_i$ (preliminary)** | **Usage ratio $r_i$** | **Notes** |
| --- | --- | --- | --- |
| **Astra writes** | 40% | $\frac{3}{7}$ = 0.43 | SAW does 2 additional writes for each WFT |
| **Visibility** (OpenSearch) | 20% | $\frac{3}{3}$ = 1.00 | Equal — both SAA and SAW produce exactly ~~2~~ 3 visibility updates |
| **WAL writes** | 10% | $\frac{3}{14}$ = 0.21 | Half of Astra ratio: SAA writes only to MSWAL, whereas SAW writes to both HEWAL and MSWAL |
| **EC2 compute** | 10% | ? | Would need cloud cell experiment |
| **Data transfer** | 10% | $\frac{1}{2}$ = 0.50 | SAW sends payloads via workflow worker round-trip; SAA does not |
| **Overheads** (incl. Clickhouse) | 10% | ? |  |

This gives the following estimate of the COGS ratio:

$$
\begin{align*}
R &=
\underbrace{0.4 \times 0.43}_{\text{Astra}:~0.17} +
\underbrace{0.2 \times 1.0}_{\text{Vis}:~0.20} +
\underbrace{0.1 \times 0.21}_{\text{WAL}:~0.02} +
\underbrace{0.1 \times 0.50}_{\text{Tx}:~0.05} +
0.1 \cdot r_\text{compute} + 0.1 \cdot r_\text{overhead} \\\\
&=
0.44 + 0.1(r_\text{compute} + r_\text{overhead}).
\end{align*}
$$

# Sensitivity analysis

Before thinking about the implications of this for billing and margins, the next steps are:

1. Refine the cloud spend estimates (Cloud Capacity team; does not involve load experiments)
2. Decide whether we want to do load experiments to estimate $r_\text{compute}$
3. Decide how we will address $r_\text{overhead}$

For (2) and (3) we can do some initial sensitivity analysis:

SAW does 10 RPCs vs SAA’s 4 (with 7 vs 3 of them doing persistence writes in the sync-match case). If services are CPU-bound then this suggests that $0.4 < r_\text{compute} < 1.0$ might be reasonable.

The other overheads include (per @Stephen Chan ) Clickhouse, observability cells, and Envoy proxies. Since these costs should also scale with RPC count, let’s assume the same bounds: $0.4 < r_\text{overhead} < 1.0$. This gives:

$$
0.52 \leq R \leq 0.64.
$$

![image.png](.task/sensitivity.png)

For example, if SAW margins were 70%, SAA margins would be 62% - 69%. This margin reduction would affect at maximum the ~3% of workflows that are SAW.

- COGS ratio to margins conversion formula

     $\text{margin}_{\text{SAA}} = 1 - 2R(1 - \text{margin}_{\text{SAW}})$.


# Discussion

- **Visibility limits SAA margins**. Visibility is expensive (20%), but SAA and SAW perform the same number of visibility writes, so it combines a large weight with the worst possible ratio.
- **(Unfavorable) Over-provisioning would push $R$ up.** The usage ratios above for persistence are derived from write counts, which only translate to cost savings if capacity tracks usage. But e.g. Astra is bought in fixed hardware units (“Astra Classic”). If any resource component is over-provisioned then SAA and SAW would pay the same cost per execution and $r_i \to 1.0$, making SAA margins less attractive relative to workflow.
- **Cloud spend share**. We could attempt to separate fixed costs and renormalize (see [Next steps](https://www.notion.so/Next-steps-3268fc567738805e82ddd9c1e1d4c9d1?pvs=21)). This would be favorable to SAA margins if it decreases the visibility share, but unfavorable if it decreases Astra share.

    We’re estimating $f_i$ from cloud spend, so we’re assuming that the spend distribution for single-activity workflows would be similar to the spend distribution for the real mix of customer workflows. I suspect this is a reasonable modeling assumption since in both cases the application is performing the same state transitions in response to workflow and activity task processing.

- **(Mixed) Effect of migration to Walker**. Walker replaces Astra with storage that is under our own control, making right-sizing easier. This may mean that the 3/7 write ratio is more fully realized under Walker, moving SAA COGS away from SAW. However, Walker will be cheaper than Astra, so persistence’s share of spend shrinks. Since persistence is where SAA has its largest advantage, this would bring SAA COGS closer to SAW.

    These two effects act in opposite directions and the net result will depends on their relative magnitudes. This suggests that we should monitor COGS calculations as the Walker migration proceeds.

- **(Future) A visibility backend migration would improve SAA margins.** There has been [movement](https://www.notion.so/Visibility-CDS-2a98fc567738807e9ee0f318edc4c16f?pvs=21) toward replacing OpenSearch. As discussed above, any reduction in visibility spend share would make SAA COGS more attractive relative to workflow.

# Conclusion

- [We are planning to bill SAA at 1/2 the price of SAW](https://www.notion.so/PRD-Standalone-Activities-for-durable-job-processing-1ee8fc567738806d8b6fe8e2eeae0fc4?pvs=21). Although there are various assumptions involved, at this point it looks like SAA COGS will be more than 1/2 SAW COGS: the estimated range above is $0.52 \leq R \leq 0.64$. This implies that some degree of cannibalization is likely. The extent of cannibalization would be bounded by the proportion of current workloads that are SAW, which is 3% per @Phil Prasek. It may be offset by volume growth attributable to SAA.

# Next steps

- **Refine cloud spend share estimates.**

    The cloud spend share weights used in this analysis are supposed to be marginal costs. We could attempt to separate marginal vs fixed costs and renormalize our spend share percentages. This would be favorable to SAA margins if it decreases the visibility share, but unfavorable if it decreases Astra share.

- **Investigate any impact of over-provisioning.**

    SAA margins may be less favorable than the calculations suggest if some resources are over-provisioned. See discussion [above](https://www.notion.so/Standalone-Activity-COGS-and-margins-3268fc567738803cb63fd9397ffd351c?pvs=21).

- **Decide whether to do cloud cell experiments**.

    Unlike the other resource categories, we lack any obvious theoretical basis for estimating  $r_\text{compute}$ and $r_\text{overhead}$. Estimating $r_\text{compute}$ via cloud cell experiments would require perhaps one engineer-week.  If this were to show a value close to 0.4 then it would suggest that the upper bound on $R$ is 0.56, as opposed to the current 0.64. This would however still be subject to all the assumptions discussed above. We could also attempt to tighten our estimated bounds on $r_\text{overhead}$ via experiment.

    If we decide to do this, the $r_\text{compute}$ experiment would be something like the following: choose a reference activity (e.g. sleeps for 10s, no heartbeating, never fails) and run SAA and SAW workloads on a cloud cell at a fixed start rate (e.g. 10/s) for a sustained period (e.g. 1hr). Fixing start rate rather than concurrency naturally controls for end-to-end latency differences between SAA and SAW.  $r_\text{cpu}$ and $r_\text{memory}$ can then be estimated from metrics as the ratio of mean utilization above the idle baseline. The analysis will need to decide how to combine them, e.g. based on which is more often limiting; alternatively, using the larger of the two would yield a conservative calculation.
END_DOCUMENT------------------------------------------------------------------------------

START_DOCUMENT------------------------------------------------------------------------------
# Test plan for SAA COGS measurement

@Dan Davison March 19, 2026

The [SAA COGS proposal](.task/saa-cogs.md) made an initial estimate of the SAA/SAW COGS ratio based on estimating persistence, visibility, and data transfer usage ratios directly from the implementation. But for compute and overheads we have no analytical estimate. We plan to run an experiment to:

1. Estimate the missing $r_\text{compute}$.
2. Validate the analytical $r_i$ against observed metrics

For comparison, the Fairness COGS experiment docs:

- [Test plan](https://www.notion.so/temporalio/Test-plan-for-COGS-measurement-28c8fc56773880169cdcc4087a98ceaf)
- [Fairness COGS Impact](https://www.notion.so/temporalio/Fairness-COGS-Impact-2c58fc567738808f806cfbf09b771b2c)
- [Pricing Council doc](https://www.notion.so/temporalio/WIP-Pricing-Council-Fairness-COGS-Impact-2cc8fc56773880dcb3efe435623edd9a)




# Proposed SAA experiment


## Workloads

Two workloads, run sequentially on the same cell:

1. **SAW**: execute workflow with one activity (no heartbeat, no retry).
2. **SAA**: execute standalone activity (no heartbeat, no retry).

## Parameters

**Start rate.** I think that we should fix start rate rather than concurrency, since this naturally controls for end-to-end latency differences between SAA and SAW (i.e. a cell running SAW will see higher load because the concurrency will be higher because the SAW end-to-end latency is higher). The fairness experiment used 4k tasks/s. Is starting 4k executions/s reasonable for us?

**Activity.** Immediate successful return; no heartbeat, no retry. We could compare with a 1s sleep to see if result differ?

**Sync match.** Do one run such that sync match should be 100%, and another tuned such that sync match is lower? Verify sync match from metrics (`syncmatch_latency`, `asyncmatch_latency`)

**Duration and repetitions.** Steady-state load; we need long enough for stable CPU averages. The
fairness experiment used 6h per scenario but this was maybe because of their more sophisticated
sinusoidal load design? 1h more than enough for the SAA experiment? ≥2 runs per workload to check
variance/reproducibility.

## Infrastructure

- Anything special about test cell sizing?
- Workers should run outside the cell (how did fairness experiment do this?)

## Metrics

Initial dashboard content https://grafana.tmprl-internal.cloud/d/saacogs/saa-cogs:


- **CPU per service** (frontend, history, matching). `node_namespace_pod_container:container_cpu_usage_seconds_total:sum_rate` — a k8s recording rule over cAdvisor container metrics (defined in saas-components prometheus rules).
- **Memory per service**. `container_memory_working_set_bytes` — also k8s/cAdvisor (defined in saas-components alert rules).
- **RPC rate by method**, one panel per service (frontend, history, matching). `service_requests` counter ([temporal:common/metrics/metric_defs.go:615](https://github.com/temporalio/temporal/blob/main/common/metrics/metric_defs.go)), tagged with `operation` (the RPC method name). Recorded by a gRPC server-side interceptor ([telemetry.go:177](https://github.com/temporalio/temporal/blob/main/common/rpc/interceptor/telemetry.go)), so it captures inter-service RPCs (e.g. history→matching `AddActivityTask`).
- **Astra writes by table**. `cassandra_query` counter with `verb!="select"`, plus `cassandra_batch` counter, both broken down by `table`. Tags include `operation`, `table`, `verb`, `cas` ([saas-temporal:cds/metrics/metrics.go:233,238](https://github.com/temporalio/saas-temporal/blob/main/cds/metrics/metrics.go)).
- **Astra reads by table**. `cassandra_query` with `verb="select"`, broken down by `table`.
- **WAL operation rate by type**. `wal_latency_count` ([saas-temporal:cds/metrics/metrics.go:35](https://github.com/temporalio/saas-temporal/blob/main/cds/metrics/metrics.go)) broken down by `walType` label (values: `MUTABLE_STATE_WAL`, `HISTORY_EVENT_WAL`, `LARGE_PAYLOAD_WAL` — see [saas-temporal:cds/common/tag/tag.go:11-24](https://github.com/temporalio/saas-temporal/blob/main/cds/common/tag/tag.go)). Note: this metric covers both reads and writes; there is no separate write-only WAL metric. This is arguably more relevant to COGS since WAL reads also cost something.
- **Visibility persistence rate by operation**. `visibility_persistence_requests` counter ([temporal:common/metrics/metric_defs.go:1398](https://github.com/temporalio/temporal/blob/main/common/metrics/metric_defs.go)), tagged with `operation` (values include `RecordWorkflowExecutionStarted`, `RecordWorkflowExecutionClosed`, `UpsertWorkflowExecution`, `DeleteWorkflowExecution` — see [visiblity_manager_metrics.go](https://github.com/temporalio/temporal/blob/main/common/persistence/visibility/visiblity_manager_metrics.go)).
- **Sync vs async match rate**. `syncmatch_latency_count` and `asyncmatch_latency_count` ([temporal:common/metrics/metric_defs.go:1119-1120](https://github.com/temporalio/temporal/blob/main/common/metrics/metric_defs.go)).


## Load generator (omes)

- Add a new scenario that starts standalone activities directly from the load generator, not from within a workflow.
- Build the omes Go worker Docker image and deploy it as a pod on k8s, configured to poll the test cell. Do we have implementation we can borrow from the fairness experiment?




<details>
<summary>Appendix: Comparison with fairness experiment (see commits by David Reiss)</summary>

| | Fairness | SAA |
|---|---|---|
| **Treatments** | Same workload, two matcher modes | Two execution types (SAW vs SAA) |
| **Quantity computed** | $\Delta C / C$ | Ratio $r_i = q_i(\text{SAA}) / q_i(\text{SAW})$ |
| **Load shape** | Sinusoidal backlog (exercises matcher) | Steady-state at fixed start rate (our model assumes sync match) |
| **What is measured** | CPU per service, Astra operation rates | CPU per service, memory per service, Astra operation rates by table and verb, WAL write rates, visibility write rates, RPC handling rates per service per method |
| **Predictions to validate** | None — purely empirical | $r_\text{Cass} = 3/7$, $r_\text{WAL} = 3/14$, $r_\text{Vis} = 3/3$, per-method RPC rates matching proposal table |

Fixed start rate (not fixed task throughput) because SAA and SAW generate different numbers of tasks per execution.

**Question**: what is the incremental COGS of enabling the fairness matcher vs the classic matcher?

**COGS components**: (1) Astra queries (~35% of total COGS), (2) EC2 compute (~9%, split across frontend+matching and history). Ignored: data transfer, Astra storage, non-AWS costs (Clickhouse <3%).

**Setup**: dedicated test cell `s-oss-dnr-faircogs3` (64 partitions). Load generator: Omes Ebb and Flow — sinusoidal activity task backlog. 5 scenarios (classic, fairness with 0/1k/100k keys, priority), each 6 hours. Measured via [dedicated Grafana dashboard](https://grafana.tmprl-internal.cloud/d/df6pldpkiy1vka/faircogs).

**Results**: Astra showed no significant increase. CPU increased up to 23% (frontend) and 36% (history) in the worst case (1k fairness keys). COGS impact: $(0.035 \times 0.23) + (0.057 \times 0.36) = 2.8\%$. Pricing council recommendation: price fairness on value to customer, not COGS.





</details>

<details>
<summary>Appendix: possible experimental outcomes</summary>

- **Analytical predictions confirmed, $R$ in predicted range.** Observed $r_\text{Cass}$, $r_\text{WAL}$, $r_\text{Vis}$, and per-method RPC rates match the analytical derivations. $r_\text{compute}$ lands in $[0.4, 1.0]$, giving $R$ in roughly $0.52$–$0.64$. We present $R$ with a tighter confidence interval than the proposal (because $r_\text{compute}$ is now estimated, not bounded).
- **$r_\text{compute}$ is low, pushing $R$ toward 0.5.** If $r_\text{compute} \approx 0.4$ and analytical predictions hold, $R \approx 0.52$. Cannibalization is near-zero.
- **Observed $r_i$ diverge from analytical predictions.** Some assumption is wrong (e.g. sync match doesn't hold at test load, or there are unaccounted persistence writes). We recompute $R$ using observed values and identify which assumption failed and whether it reflects production conditions or a test artifact.
- **$R$ is higher than predicted.** $R > 0.64$ would mean worse cannibalization than estimated. Options: accept the margin reduction (bounded by ~3% SAW share), adjust billing, or identify engineering work to reduce SAA COGS.

</details>

END_DOCUMENT------------------------------------------------------------------------------


Your task is to help me design and build the omes-based tooling that we will use to perform the experiments outlined above to learn about COGS of SAA an SAW. We are in the omes repo; study it carefully. Our work will broadly break into the following phases that we must design holistically:

(1) Add any missing omes functionality that will be needed in order to be able to use omes to generate the SAA and SAW load for the experiments.
(2) Run the experiments against the cloud cell that Stephen has prepared: its name is s-saa-cogs.

I am not familiar with performing operations against cloud cells, so you will need to resarch and help me during this. But we have several good resources: study the contents of the 'oncall' and 'runbooks' repos, and also use the /agent-slack skill. You also have Notion and Temporal Docs MCP. Use the more modern 'ct' rather than its alias 'omni'.

Initial grafana dashboard JSON is at .task/saacogs.json.

Important: I'd like an early aim to be to get an end-to-end proof-of-principle of this working. Therefore let's not make the omes component sophisticated initially; just the bare minimum to run an SAW and SAA workload. But I am a bit intimidated by doing anything with the cloud cell since I don't know how. So I guess one early aim is to be able to point our metrics dashboard at s-saa-cogs, and see idle state, then run one of our omes commands, and see activity increase in the dashboard. Please maintain a file of useful shell commands with terse comments where necessary. I will run them and show you the outut. Don't do operations against cloud or observability yourself unless I explicitly ask you to.