import type { Client } from '@temporalio/client';
import { NativeConnection, Worker } from '@temporalio/worker';
import {
  defaultClientFactory,
  type App,
  type ProjectExecuteContext,
  type WorkerContext,
} from '@temporalio/omes-project-harness';

export function app(): App {
  return {
    worker: buildWorker,
    clientFactory: defaultClientFactory,
    project: {
      execute: executeProjectIteration,
    },
  };
}

async function buildWorker(client: Client, context: WorkerContext): Promise<Worker> {
  const connection = client.connection;
  if (!(connection instanceof NativeConnection)) {
    throw new Error('Helloworld project requires a NativeConnection-backed client');
  }

  return await Worker.create({
    connection,
    namespace: client.options.namespace,
    taskQueue: context.taskQueue,
    workflowsPath: require.resolve('./workflow'),
    ...context.workerOptions,
  });
}

async function executeProjectIteration(
  client: Client,
  context: ProjectExecuteContext,
): Promise<void> {
  const handle = await client.workflow.start('helloWorldWorkflow', {
    args: ['World'],
    taskQueue: context.taskQueue,
    workflowId: `${context.run.executionId}-${context.iteration}`,
  });
  const result = await handle.result();
  console.log(result);
}
