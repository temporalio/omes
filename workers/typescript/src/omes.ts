import { Command } from 'commander';
import { NativeConnection, Worker } from '@temporalio/worker';
import { TLSConfig } from '@temporalio/client';
import * as fs from 'fs';
import * as activities from './activities';

async function run() {
  const program = new Command();
  program
    .option('-a, --server-address <address>', 'The host:port of the server', 'localhost:7233')
    .option('-q, --task-queue <taskQueue>', 'Task queue to use', 'omes')
    .option('--task-queue-suffix-index-start <tqSufStart>',
      'Inclusive start for task queue suffix range', '0')
    .option('--task-queue-suffix-index-end <tqSufEnd>',
      'Inclusive end for task queue suffix range', '0')
    .option('-n, --namespace <namespace>', 'The namespace to use', 'default')
    .option('--max-concurrent-activity-pollers <maxActPollers>', 'Max concurrent activity pollers')
    .option('--max-concurrent-workflow-pollers <maxWfPollers>', 'Max concurrent workflow pollers')
    .option('--max-concurrent-activities <maxActs>', 'Max concurrent activities')
    .option('--max-concurrent-workflow-tasks <maxWFTs>', 'Max concurrent workflow tasks')
    .option('--log-level <logLevel>', '(debug info warn error panic fatal)', 'info')
    .option('--log-encoding <logEncoding>', '(console json)', 'console')
    .option('--tls', 'Enable TLS')
    .option('--tls-cert-path <clientCertPath>', 'Path to a client certificate for TLS')
    .option('--tls-key-path <clientKeyPath>', 'Path to a client key for TLS')
    .option('--prom-listen-address <promListenAddress>', 'Prometheus listen address')
    .option('--prom-handler-path <promHandlerPath>', 'Prometheus handler path', '/metrics');

  const opts = program.parse(process.argv).opts<{
    server: string;
    taskQueue: string;
    tqSufStart: number;
    tqSufEnd: number;
    namespace: string;

    maxActPollers: number;
    maxWfPollers: number;
    maxActs: number;
    maxWFTs: number;

    logLevel: string;
    logEncoding: string;

    tls: boolean;
    clientCertPath: string;
    clientKeyPath: string;

    promListenAddress: string;
    promHandlerPath: string;
  }>();

  console.log('Running TypeScript Omes');

  // Configure TLS
  let tlsConfig: TLSConfig | undefined;
  if (opts.tls) {
    if (!opts.clientCertPath) {
      throw new Error('Client cert path specified but no key path!');
    }
    if (!opts.clientKeyPath) {
      throw new Error('Client key path specified but no cert path!');
    }
    const crt = fs.readFileSync(opts.clientCertPath);
    const key = fs.readFileSync(opts.clientKeyPath);
    tlsConfig = {
      clientCertPair: {
        crt,
        key
      }
    };
  }


  const connection = await NativeConnection.connect({
    address: opts.server,
    tls: tlsConfig
  });

  const worker = await Worker.create({
    connection,
    namespace: opts.namespace,
    workflowsPath: require.resolve('./workflows'),
    activities,
    taskQueue: opts.taskQueue,
    dataConverter: {
      payloadConverterPath: require.resolve('./payload-converter')
    }
  });

  await worker.run();
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
}).then(() => {
  process.exit(0);
});