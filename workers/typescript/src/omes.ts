import { NativeConnection, Runtime, Worker } from '@temporalio/worker';
import { Client } from '@temporalio/client';
import { createActivities } from './activities';
import type { ClientConfig, App, WorkerContext } from '@temporalio/omes-project-harness';
import { run as runApp } from '@temporalio/omes-project-harness';

const payloadConverterPath = require.resolve('./payload-converter');

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
      payloadConverterPath,
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
    workflowsPath: require.resolve('./workflows'),
    activities: createActivities(client, context.errOnUnimplemented),
    taskQueue: context.taskQueue,
    dataConverter: {
      payloadConverterPath,
    },
    ...context.workerOptions,
  });
}

async function main() {
  const app: App = {
    worker: buildWorker,
    clientFactory: kitchenSinkClientFactory,
  };
  await runApp(app);
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
