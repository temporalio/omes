# Omes Starter Library (TypeScript)

Library for writing workflow load tests with Omes.

## Execution model

Your test package exports handlers and calls `run(...)` once:

```typescript
import { run } from '@temporalio/omes-starter';
import { clientMain } from './src/client';
import { workerMain } from './src/worker';

run({ client: clientMain, worker: workerMain });
```

At runtime, Omes invokes the same program with:

- `client`: starts HTTP server endpoints used by runner (`/execute`, `/info`)
- `worker`: starts the Temporal worker process

The worker is managed externally by the orchestrator/runtime. The starter does not expose `/shutdown`.

## Client handler API

`clientMain(config: ClientConfig)` is called once per `/execute`.

`ClientConfig` includes:

- `namespace`
- `connectionOptions` (native SDK options for `Connection.connect(...)`)
- `taskQueue`
- `runId`
- `iteration`

Core pattern for now is direct client construction per execute call:

```typescript
import { Client, Connection } from '@temporalio/client';
import type { ClientConfig } from '@temporalio/omes-starter';

export async function clientMain(config: ClientConfig): Promise<void> {
  const connection = await Connection.connect(config.connectionOptions);
  try {
    const client = new Client({ connection, namespace: config.namespace });
    const handle = await client.workflow.start('SimpleWorkflow', {
      workflowId: `wf-${config.runId}-${config.iteration}`,
      taskQueue: config.taskQueue,
    });
    await handle.result();
  } finally {
    await connection.close();
  }
}
```

Optional helper for sharing a connection across execute calls:

```typescript
import { Client, Connection } from '@temporalio/client';
import { ClientPool } from '@temporalio/omes-starter';
import type { ClientConfig } from '@temporalio/omes-starter';

const pool = new ClientPool();

export async function clientMain(config: ClientConfig): Promise<void> {
  const connection = await pool.getOrConnect('default', config.connectionOptions);
  const client = new Client({ connection, namespace: config.namespace });
  const handle = await client.workflow.start('SimpleWorkflow', {
    workflowId: `wf-${config.runId}-${config.iteration}`,
    taskQueue: config.taskQueue,
  });
  await handle.result();
}
```

Using `ClientPool` is optional; direct native connect remains the default pattern.

## Worker handler API

`workerMain(config: WorkerConfig) -> Promise<Worker>` receives:

- `namespace`
- `connectionOptions` (native SDK options for `NativeConnection.connect(...)`)
- `taskQueue`
- `promListenAddress`

When `promListenAddress` is set (via `--prom-listen-address`), starter installs worker runtime telemetry so Prometheus metrics are exposed at `<addr>/metrics`.

```typescript
import { NativeConnection, Worker } from '@temporalio/worker';
import type { WorkerConfig } from '@temporalio/omes-starter';

export async function workerMain(config: WorkerConfig): Promise<Worker> {
  const nativeConnection = await NativeConnection.connect(config.connectionOptions);
  return Worker.create({
    connection: nativeConnection,
    namespace: config.namespace,
    taskQueue: config.taskQueue,
    workflowsPath: require.resolve('./workflows'),
  });
}
```

## CLI arguments

Client mode:

- `--port` (default `8080`)
- `--task-queue` (required)
- `--server-address` (default `localhost:7233`)
- `--namespace` (default `default`)
- `--auth-header`
- `--tls`
- `--tls-server-name`

Worker mode:

- `--task-queue` (required)
- `--server-address` (default `localhost:7233`)
- `--namespace` (default `default`)
- `--auth-header`
- `--tls`
- `--tls-server-name`
- `--prom-listen-address`

## SDK version override

The starter is SDK-version agnostic. `omes exec` / `omes workflow` choose the effective SDK version:

- local path: sdkbuild generates `package.json` with `file:` SDK package references
- released version: sdkbuild pins `@temporalio/*` to that version

User code keeps normal imports:

```typescript
import { Client } from '@temporalio/client';
import { Worker } from '@temporalio/worker';
```

The override happens in generated build output; user project files are not mutated.

## Usage

```bash
# Run worker only
omes exec --language typescript --project-dir ./tests/simple-test -- \
  worker --task-queue my-queue --server-address localhost:7233 --namespace default

# Run load test (spawns client; add --spawn-worker to co-manage worker)
omes workflow --language typescript --project-dir ./tests/simple-test \
  --iterations 100 --task-queue my-queue --server-address localhost:7233 --namespace default
```
