# Rust worker

An Omes worker built on the [Temporal Rust SDK](https://github.com/temporalio/sdk-rust),
selected with `--language rust` (alias `rs`). This is separate from
`loadgen/kitchen-sink-gen`, the Rust tool that generates Kitchen Sink inputs.

## Running

```sh
go run ./cmd/dev install rust
go run ./cmd/omes prepare-worker --language rust --dir-name prepared
go run ./cmd/omes run-scenario-with-worker \
  --language rust --dir-name prepared \
  --scenario workflow_with_single_noop_activity \
  --iterations 2 --embedded-server
```

A prepared directory contains a single `program` executable, which serves
both the default Kitchen Sink `worker` app and project apps such as `helloworld`
(see [Running a project](../../README.md#running-a-project)).

To build against a local SDK checkout, pass `--version <path>` pointing at a
recursive `temporalio/sdk-rust` checkout. The override is applied only to the
prepared package's manifest, never to the tracked `Cargo.toml`.

## Limitations

Unsupported actions and options are skipped, unless
`--worker-err-on-unimplemented=true` is passed, in which case they fail.

- Search attribute upserts are not supported.
- Child workflow memos, cross-namespace children, activity/child versioning
  intents and Build ID compatibility-set versioning (`--build-id`) are not supported.
- The SDK cannot serve Nexus operations. Workflows and client actions can call
  Nexus operations on an endpoint served by another worker, but operations aimed
  at a run's auto-created endpoint fail immediately. See the Nexus note in
  [Running Omes](../../docs/running.md#gotchas).
- A nested canceled cause fails the workflow instead of canceling it.
- Mixing local and remote activities can hit a cache-eviction panic in the
  pinned SDK.

## Adding apps

Apps are compiled into the executable: register them in
[`src/apps/mod.rs`](src/apps/mod.rs), following the
[`helloworld` sample](src/apps/helloworld.rs).

## Protobuf generation

`build.rs` generates the Kitchen Sink and harness messages from `workers/proto`
at build time, so builds require `protoc` and the `workers/proto/api_upstream`
submodule. Set `OMES_API_PROTO_DIR` to use a different copy of the Temporal API
protos. Temporal API messages reuse the SDK's types.

To run the shared Kitchen Sink tests against this worker:

```sh
SDK=rust go test -v ./loadgen -run TestKitchenSink
```
