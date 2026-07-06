import { Client } from '@temporalio/client';
import { NativeConnection, Worker } from '@temporalio/worker';
import type { App, ProjectExecuteContext, WorkerContext } from '../../harness';
import { defaultClientFactory } from '../../harness';

async function buildWorker(client: Client, context: WorkerContext): Promise<Worker> {
  const connection = client.connection;
  if (!(connection instanceof NativeConnection)) {
    throw new Error('Helloworld worker requires a NativeConnection-backed client');
  }

  return await Worker.create({
    connection,
    namespace: client.options.namespace,
    workflowsPath: require.resolve('./workflows'),
    taskQueue: context.taskQueue,
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

export const app: App = {
  worker: buildWorker,
  clientFactory: defaultClientFactory,
  project: {
    execute: executeProjectIteration,
  },
};
