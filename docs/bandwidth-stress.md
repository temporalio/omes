# Bandwidth stress scenario

`bandwidth_stress` generates sustained, incompressible activity input and result payloads. It is intended
for calibrating namespace bandwidth limits against downstream storage traffic.

Each iteration starts one workflow and runs one or more remote payload activities. The default is one
activity with a 100 KiB input and a 100 KiB result. Use standard run flags to control workflow rate,
concurrency, and duration.

The default shape produces two billable Actions: the workflow start and the activity schedule. It also
writes two user payloads: the activity input and activity result. At the default size, this is 200 KiB
of user payload per workflow and 100 KiB per Action. This is the initial replay hypothesis, not a claim
that the original OMES workflow had this exact shape.

The following command models the initial INC-1725 workload delta. The cell rose from approximately
8,000 to 8,500 Actions per second before the incident to approximately 11,500 to 12,000 Actions per
second during the incident. This scenario produces two Actions per workflow, so 1,500 workflow starts
per second represents the lower bound of the observed increase. Each workflow carries 200 KiB of user
payload across the activity input and result.

```sh
go run ./cmd/omes run-scenario-with-worker \
  --scenario bandwidth_stress \
  --language go \
  --server-address s-server-test-bwrl-bandwidth.e2e.tmprl-test.cloud:7233 \
  --namespace s-server-test-bwrl-bandwidth.e2e \
  --tls \
  --tls-server-name s-server-test-bwrl-bandwidth.e2e.tmprl-test.cloud \
  --tls-cert-path <client-certificate-path> \
  --tls-key-path <client-private-key-path> \
  --run-id bandwidth-inc-1725 \
  --duration 1h \
  --max-iterations-per-second 1500 \
  --max-concurrent 3000 \
  --option activities-per-workflow=1 \
  --option 'payload-distribution-json={"size":{"type":"fixed","value":"102400"}}'
```

Do not begin calibration at the incident rate. Use separate runs for each plateau so every stage has a
distinct run ID and metric interval. Wait at least five minutes between stages, or until Cassandra query
latency and errors return to the pretest baseline.

| Stage | Iterations/s | Expected Actions/s | User payload MiB/s | Duration |
| --- | ---: | ---: | ---: | ---: |
| 25% | 375 | 750 | 73 | 10m |
| 50% | 750 | 1,500 | 146 | 10m |
| 65% | 975 | 1,950 | 190 | 10m |
| 75% | 1,125 | 2,250 | 220 | 10m |
| 85% | 1,275 | 2,550 | 249 | 10m |
| 100% | 1,500 | 3,000 | 293 | 15m |

For each stage, change `--max-iterations-per-second`, `--duration`, and `--run-id`. Keep the scenario
options and worker configuration unchanged. The expected values assume one activity per workflow and
two 100 KiB payloads. Confirm the Action rate from metrics because retries and failed starts can change
the observed rate.

The Cassandra transmitted write rate rose from approximately 10 to 20 MB/s to approximately 1.2 to
1.4 GB/s. At 1,500 workflows per second, this scenario supplies approximately 293 MiB/s of serialized
user payload. The replay must measure whether persistence amplifies that payload to the observed
Cassandra delta. Do not derive the offered workflow rate directly from Cassandra transmitted bytes.

Do not use the incident replay profile on a small scaffold cell. The initial `s-server-test-bwrl` cell has
a 1,500 RPS cell limit and the namespace inherits a 500 APS limit. Without a deliberate capacity
change, keep the offered rate at or below 250 workflows per second. A small-cell functional ramp can
use 62, 125, 187, and 250 workflows per second. These stages validate payload charging and throttling,
but they do not reproduce the incident's storage pressure.

Stop the ramp when any signal approaches the incident condition instead of continuing automatically:

| Signal | Incident condition |
| --- | ---: |
| Cassandra transmitted write bytes | 1.2 to 1.4 GB/s |
| Cassandra query p99 | 100 to 120 ms |
| Storage latency p99 | 48 to 55 ms |
| Cassandra query errors | 15 to 35/s |
| StartWorkflowExecution p99 | 175 to 185 ms |
| RespondActivityTaskCompleted p99 | 80 to 95 ms |
| RespondWorkflowTaskCompleted p99 | 50 to 60 ms |

The first sustained latency or error increase identifies the pressure knee. Repeat the stage immediately
below that knee for at least 30 minutes before selecting a limit. The complete incident profile is only
needed after lower stages establish a safe operating range.

The historical cell used one 14 CU C40i Astra database. Modeled capacity was 69K STPS for the cell,
84K for Astra, 84K for History, 100K for WAL, and 120K for shards. History had 28 replicas. Storage
grew from approximately 3.6 TB to 7.4 TB during the incident. Prefer a test cell with the same storage
shape. If an exact match is unavailable, record every component capacity and compare pressure as a
fraction of Astra capacity rather than comparing raw throughput alone.

## Options

`activities-per-workflow` controls how many remote payload activities each workflow runs sequentially.
Its default is `1`.

`payload-distribution-json` controls the payload size of each activity input and result. It accepts the
shared OMES distribution format. The default is a fixed 100 KiB payload:

```json
{"size":{"type":"fixed","value":"102400"}}
```

A mixed distribution can represent a workload with several payload sizes:

```json
{"size":{"type":"discrete","weights":{"10240":1,"102400":9}}}
```

Measure the replay using the same stable interval for each ratio:

```text
charged History request bytes / Actions
Cassandra transmitted bytes / charged History request bytes
Cassandra transmitted bytes / Actions
```

Record these values for every plateau:

```text
run ID and exact start/end time
offered iterations/s and observed Actions/s
charged History request bytes/s
Cassandra transmitted write bytes/s
Astra storage growth/s
Cassandra and storage p50, p95, and p99 latency
Cassandra query errors/s
frontend p50, p95, and p99 latency for the three payload RPCs
bandwidth throttled requests/s
```
