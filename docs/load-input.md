# Replaying a LoadInput

The `fuzzer` scenario can replay a whole workload described in kitchen sink proto. Pass the file with `--option load-input-file=<path>`. A path ending in `.json` is read as protojson, and any other path as binary proto. The option can't be combined with `input-file`, `seed`, `config`, `output-file`, or `kitchen-sink-gen`, which only shape the inputs a `LoadInput` replaces.

A `LoadInput` adds four things to a plain `TestInput` replay:

- Several weighted inputs, so the mix of shapes across iterations (update-with-start, payload sizes, and so on) follows the weights.
- Search attribute values drawn fresh on every iteration, both at workflow start and in upserts.
- Search attributes set when the workflow starts.
- Visibility reads (List and Count) at a fixed rate, independent of the workflow rate.

[`scenarios/testdata/load_input_example.json`](../scenarios/testdata/load_input_example.json) is a complete example. The messages are defined in [`kitchen_sink.proto`](../workers/proto/kitchen_sink/kitchen_sink.proto).

## Running one

```sh
go run ./cmd/omes run-scenario-with-worker --scenario fuzzer --language go --iterations 20 \
  --option load-input-file=scenarios/testdata/load_input_example.json
```

Only the Go worker passes typed upsert payloads to the server unchanged, so use `--language go` when inputs upsert generated attributes.

Unless `--do-not-register-search-attributes` is set, the run registers every attribute in `search_attributes` with its type before the first iteration, beside omes's default attributes. Attributes that already exist count as registered. The server skips an existing name even if its type differs, and upserts of that attribute then fail at runtime. With the flag set, register them some other way first.

The fuzzer creates a Nexus endpoint named `test-nexus-endpoint-<run-id>`, with the run ID reduced to letters, digits, and `-`, unless `--option nexus-endpoint=<name>` names an existing one. A `LoadInput`'s Nexus operations name their endpoint in `ExecuteNexusOperation.endpoint`, so they only use the created one if they name it.

## Format

```proto
message LoadInput {
  repeated WeightedTestInput inputs = 1;
  repeated SearchAttributeDefinition search_attributes = 2;
  repeated string start_search_attributes = 3;
  VisibilityReads visibility_reads = 4;
}
```

- `inputs`: each iteration picks one `TestInput` with probability `weight / total weight`. The weights must not all be 0.
- `search_attributes`: each definition has a `name`, an `IndexedValueType` (`type`), and a `generator`. Names must be unique and match `^[A-Za-z][A-Za-z0-9_]*$`.
- `start_search_attributes`: names from `search_attributes` that every iteration sets when it starts its workflow. They go into the start options beside `OmesExecutionID`.
- `visibility_reads`: reads to issue while iterations run. Leave it unset to issue none.

### Generated upserts

`UpsertSearchAttributesAction` has a `generated` field that lists names from `search_attributes`. Before an iteration starts its workflow, omes copies the chosen input, puts a fresh value for each name into the action's `search_attributes`, and clears `generated`. Workers never see the field.

The walk reaches every upsert in the input: nested action sets, client action sets, and payloads that hold an encoded `WorkflowInput`, such as continue-as-new arguments and child workflow inputs. Those payloads are decoded, filled in, and re-encoded. In protojson a payload's metadata and data are base64, so the example's continue-as-new argument is not readable as written. The easiest way to produce one is to build the `LoadInput` in Go with `kitchensink.ConvertToPayload` and marshal it with `protojson`.

### Generators

| Generator | Types | Values |
|---|---|---|
| `pool` | Keyword, Text, KeywordList | Value `i` of attribute `A` is `A-i`, for `i` in `[0, cardinality)`. `distribution` is `POOL_DISTRIBUTION_UNIFORM` (the default) or `POOL_DISTRIBUTION_ZIPF`, which draws with `rand.NewZipf(r, 1.1, 1, cardinality-1)`. For KeywordList, each value holds 1 to `max_items` distinct pool values; 0 means 3. |
| `int_range` | Int | Uniform in `[min, max]`, inclusive. |
| `double_range` | Double | Uniform in `[min, max)`. |
| `bool` | Bool | True with probability `true_fraction`. |
| `time_offset` | Datetime | The iteration's start time plus an offset uniform in `[min_offset, max_offset]`, in UTC. |

A generator that doesn't fit its type, a name in `generated` or `start_search_attributes` without a definition, and a bad read template all fail the run at startup, before any workflow starts. The error names the attribute.

Each iteration's values come from a random source seeded by the run ID, the execution ID, and the iteration number. Every omes process gets its own execution ID, so processes started with the same `--run-id` write different values, while a retried iteration writes the same values it did the first time. Datetime values are the exception, since they depend on the clock.

## Visibility reads

```proto
message VisibilityReads {
  double rate_per_second = 1;
  uint32 max_in_flight = 2;
  repeated VisibilityReadTemplate templates = 3;
}

message VisibilityReadTemplate {
  string name = 1;
  VisibilityReadKind kind = 2;  // VISIBILITY_READ_KIND_LIST (default) or VISIBILITY_READ_KIND_COUNT
  string query = 3;
  uint32 weight = 4;
  uint32 page_size = 5;         // LIST only. 0 means 100.
}
```

The reader runs open-loop beside the iterations. Each tick of a `rate_per_second` limiter picks a template by weight and issues one read in its own goroutine, so a slow server doesn't lower the offered rate. A tick that finds `max_in_flight` reads outstanding is skipped and counted. `max_in_flight` of 0 means `max(64, rate_per_second)`. Each read times out after 10 seconds. When the iterations end, the reader waits up to 5 seconds for reads in flight and then cancels them.

A failed read doesn't fail the run. Errors are logged at most once per template every 30 seconds, with a count of the ones suppressed.

### Placeholders

Each read renders its template's query with fresh values:

| Placeholder | Renders as |
|---|---|
| `{Attr}` | One generated value of `Attr` as a query literal. A KeywordList renders one element, since `=` on a list attribute matches any element. |
| `{Attr:in:N}` | `N` distinct values of a Keyword or KeywordList attribute, joined with commas, for an `IN (...)` list. The values are drawn uniformly, even from a zipf pool. `N` can't exceed the pool's cardinality. |
| `{ago:<duration>}` | The quoted RFC3339 time `<duration>` before now, such as `{ago:15m}`. |
| `{wfid:prefix:broad}` | The quoted prefix `w-<run-id>-`, shared by every workflow omes starts in the run. It also matches runs whose ID begins with `<run-id>-`. Child workflows don't share it, since the SDK names them `<parent-run-id>_<seq>`. |
| `{wfid:prefix:narrow}` | A quoted prefix that matches at most about eleven started workflow IDs. |
| `{wfid:prefix-end:narrow}` | The same narrow prefix with `~` appended, for the upper bound of a range. Every narrow placeholder in one query shares one draw. |
| `{wfid:ids:N}` | `N` quoted workflow IDs of started iterations, joined with commas. |

Workflow IDs follow `Run.DefaultStartWorkflowOptions`: `w-<run-id>-<execution-id>-<iteration>`. Templates are parsed and their attribute names checked against `search_attributes` at startup. A template has no way to escape `{` or `}`, so neither can appear outside a placeholder, even in a string literal.

### Metrics

Every series is tagged `template`, `kind` (`list` or `count`), and the run's `namespace`:

| Metric | Type | Meaning |
|---|---|---|
| `omes_visibility_read_latency_seconds` | timer | Latency of each successful read. |
| `omes_visibility_reads_total` | counter | Successful reads. |
| `omes_visibility_read_errors_total` | counter | Failed reads, also tagged `error` as `timeout` or `other`. |
| `omes_visibility_reads_skipped_total` | counter | Ticks skipped because `max_in_flight` reads were outstanding. |

When the run ends, the reader logs a summary of reads completed, failed, skipped, and abandoned after the drain timeout.
