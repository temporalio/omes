import {
  NativeConnection,
  Runtime,
  Worker,
} from '@temporalio/worker';
import { Client } from '@temporalio/client';
import { createActivities } from './activities';
import {
  ClientConfig,
  ClientFactory,
} from '../projects/harness/client';
import { run as runApp, App } from '../projects/harness/main';
import {
  WorkerContext,
} from '../projects/harness/worker';

const payloadConverterPath = require.resolve('./payload-converter');

export const kitchenSinkClientFactory: ClientFactory = async (
  config: ClientConfig,
): Promise<Client> => {
  // Use native connection backed client because resulting client
  // is also used for worker.
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
};

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
