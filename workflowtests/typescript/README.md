# Omes Starter Library (TypeScript)

Library for writing workflow load tests with Omes.

## Client Starter

The client starter provides an HTTP server that receives iteration requests from Omes. It receives runtime parameters via CLI arguments.

### CLI Arguments

| Argument | Required | Description |
|----------|----------|-------------|
| `--port` | No | HTTP server port (default: 8080) |
| `--task-queue` | Yes | Temporal task queue name |
| `--server-address` | No | Temporal server address (default: localhost:7233) |
| `--namespace` | No | Temporal namespace (default: default) |

**Note:** `run_id` is passed per-request in the `/execute` body, not as a CLI arg.

### Example

```typescript
import { OmesClientStarter, ExecuteContext } from '@temporalio/omes-starter';

const starter = new OmesClientStarter({
    // Optional client options
});

starter.onExecute(async (ctx: ExecuteContext) => {
    const handle = await ctx.client.workflow.start('MyWorkflow', {
        taskQueue: ctx.taskQueue,
        workflowId: `wf-${ctx.runId}-${ctx.iteration}`,
    });
    await handle.result();
});

starter.run();  // Handles CLI args internally
```

Run with:
```bash
npx ts-node client.ts --port 8080 --task-queue my-queue
```

## Worker Starter

```typescript
import { OmesWorkerStarter, WorkerContext } from '@temporalio/omes-starter';
import { Worker } from '@temporalio/worker';
import * as workflows from './workflows';

const starter = new OmesWorkerStarter({
    // Optional client options
});

starter.configureWorker(async (ctx: WorkerContext) => {
    return await Worker.create({
        connection: ctx.client.connection,
        namespace: ctx.client.options.namespace,
        taskQueue: ctx.taskQueue,
        workflowsPath: require.resolve('./workflows'),
    });
});

starter.run();  // Handles CLI args internally
```

Worker lifecycle is managed by `omes exec --remote-worker` which spawns this as a subprocess and provides HTTP endpoints for shutdown and metrics.

## SDK Version Override

The TypeScript starter library is SDK-version agnostic. When using `omes exec` or `omes workflow` commands, the SDK version is overridden transparently:

- **For local paths:** sdkbuild generates a `package.json` with `file:` protocol URLs pointing to local SDK packages
- **For released versions:** sdkbuild generates a `package.json` with npm version specifiers

User code writes normal TypeScript imports:
```typescript
import { Client } from '@temporalio/client';
import { Worker } from '@temporalio/worker';
```

The SDK override is invisible to user code - it happens via the generated `package.json` in the build directory. User's original files are never modified.
