import { NativeConnection, Runtime, Worker } from '@temporalio/worker';
import { Client } from '@temporalio/client';
import { createActivities } from '../../workerlib/kitchensink/activities';
import type { ClientConfig, App, WorkerContext } from '../../harness';

function payloadConverterPath(): string {
  return require.resolve('../../workerlib/kitchensink/payload-converter');
}

async function kitchenSinkClientFactory(config: ClientConfig): Promise<Client> {
  // Use native connection backed client. This client is also used
  // for the worker(s).
  Runtime.install(config.runtimeOptions);
  const connection = await NativeConnection.connect({
    address: config.targetHost,
    apiKey: config.apiKey,
    tls: config.tls,
  });

  return new Client({
    connection,
    namespace: config.namespace,
    dataConverter: {
      payloadConverterPath: payloadConverterPath(),
    },
  });
}

async function buildWorker(client: Client, context: WorkerContext): Promise<Worker> {
  const connection = client.connection;
  if (!(connection instanceof NativeConnection)) {
    throw new Error('Harness worker requires a NativeConnection-backed client');
  }

  return await Worker.create({
    connection,
    namespace: client.options.namespace,
    workflowsPath: require.resolve('../../workerlib/kitchensink/workflows'),
    activities: createActivities(client, context.errOnUnimplemented),
    taskQueue: context.taskQueue,
    ...context.workerOptions,
    dataConverter: {
      ...context.workerOptions.dataConverter,
      payloadConverterPath: payloadConverterPath(),
    },
  });
}

export const app: App = {
  worker: buildWorker,
  clientFactory: kitchenSinkClientFactory,
};
