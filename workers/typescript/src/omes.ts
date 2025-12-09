import { Command } from 'commander';
import {
  DefaultLogger,
  LogLevel,
  NativeConnection,
  Runtime,
  TelemetryOptions,
  Worker,
  WorkerOptions,
} from '@temporalio/worker';
import { Client, TLSConfig } from '@temporalio/client';
import * as fs from 'fs';
import { createActivities } from './activities';
import winston from 'winston';

async function run() {
  const program = new Command();
  program
    .option('-a, --server-address <address>', 'The host:port of the server', 'localhost:7233')
    .option('-q, --task-queue <taskQueue>', 'Task queue to use', 'omes')
    .option(
      '--task-queue-suffix-index-start <tqSufStart>',
      'Inclusive start for task queue suffix range',
      '0'
    )
    .option(
      '--task-queue-suffix-index-end <tqSufEnd>',
      'Inclusive end for task queue suffix range',
      '0'
    )
    .option('-n, --namespace <namespace>', 'The namespace to use', 'default')
    .option('--max-concurrent-activity-pollers <maxActPollers>', 'Max concurrent activity pollers')
    .option('--max-concurrent-workflow-pollers <maxWfPollers>', 'Max concurrent workflow pollers')
    .option(
      '--activity-poller-autoscale-max <actPollerAutoscaleMax>',
      'Max for activity poller autoscaling (overrides max-concurrent-activity-pollers)'
    )
    .option(
      '--workflow-poller-autoscale-max <wfPollerAutoscaleMax>',
      'Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers)'
    )
    .option('--max-concurrent-activities <maxActs>', 'Max concurrent activities')
    .option('--max-concurrent-workflow-tasks <maxWFTs>', 'Max concurrent workflow tasks')
    .option('--worker-activities-per-second <workerActivityRate>', 'Per-worker activity rate limit')
    .option('--log-level <logLevel>', '(debug info warn error panic fatal)', 'info')
    .option('--log-encoding <logEncoding>', '(console json)', 'console')
    .option('--tls', 'Enable TLS')
    .option('--tls-cert-path <clientCertPath>', 'Path to a client certificate for TLS')
    .option('--tls-key-path <clientKeyPath>', 'Path to a client key for TLS')
    .option('--prom-listen-address <promListenAddress>', 'Prometheus listen address')
    .option('--prom-handler-path <promHandlerPath>', 'Prometheus handler path', '/metrics')
    .option(
      '--err-on-unimplemented <errOnImplemented>',
      'Error when receiving unimplemented actions (currently only affects concurrent client actions)'
    );

  const opts = program.parse(process.argv).opts<{
    serverAddress: string;
    taskQueue: string;
    tqSufStart: number;
    tqSufEnd: number;
    namespace: string;

    maxActPollers: number;
    maxWfPollers: number;
    actPollerAutoscaleMax: number;
    wfPollerAutoscaleMax: number;
    maxActs: number;
    maxWFTs: number;

    logLevel: string;
    logEncoding: string;

    tls: boolean;
    clientCertPath: string;
    clientKeyPath: string;

    promListenAddress: string;
    promHandlerPath: string;
    workerActivityRate: number;
    errOnUnimplemented: boolean;
  }>();

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
        key,
      },
    };
  }

  // Configure logging (winston doesn't know about trace level which is obnoxious)
  const winstonLevel = opts.logLevel.toLowerCase() === 'trace' ? 'debug' : opts.logLevel;
  const winstonLogger = winston.createLogger({
    level: winstonLevel,
    format:
      opts.logEncoding === 'json'
        ? winston.format.json()
        : winston.format.combine(
            winston.format.colorize(),
            winston.format.simple(),
            winston.format.metadata()
          ),
    transports: [new winston.transports.Console()],
  });
  const logger = new DefaultLogger(coerceLogLevel(opts.logLevel), (entry) => {
    winstonLogger.log({
      level: entry.level.toLowerCase() === 'trace' ? 'debug' : entry.level.toLowerCase(),
      message: entry.message,
      timestamp: Number(entry.timestampNanos / 1_000_000n),
      meta: entry.meta,
    });
  });
  // Configure metrics
  const telemetryOptions: TelemetryOptions = {};
  if (opts.promListenAddress) {
    telemetryOptions.metrics = {
      prometheus: { bindAddress: opts.promListenAddress, useSecondsForDurations: true },
    };
  }

  Runtime.install({
    logger,
    telemetryOptions,
  });

  logger.info(`Connecting to server at ${opts.serverAddress}`);

  const connection = await NativeConnection.connect({
    address: opts.serverAddress,
    tls: tlsConfig,
  });

  const client = new Client({
    connection,
    namespace: opts.namespace,
    dataConverter: {
      payloadConverterPath: require.resolve('./payload-converter'),
    },
  });

  // Possibly create multiple workers if we are being asked to use multiple task queues
  const taskQueues = [];
  if (opts.tqSufEnd === 0 || opts.tqSufEnd === undefined) {
    logger.info('Running TypeScript worker on task queue ' + opts.taskQueue);
    taskQueues.push(opts.taskQueue);
  } else {
    for (let i = opts.tqSufStart; i <= opts.tqSufEnd; i++) {
      const taskQueue = opts.taskQueue + '-' + i;
      taskQueues.push(taskQueue);
    }
    logger.info(`Running TypeScript worker on ${taskQueues.length} task queues`);
  }

  const activities = createActivities(client, opts.errOnUnimplemented || false);

  const workerArgs: WorkerOptions = {
    connection,
    namespace: opts.namespace,
    workflowsPath: require.resolve('./workflows'),
    activities,
    taskQueue: opts.taskQueue,
    dataConverter: {
      payloadConverterPath: require.resolve('./payload-converter'),
    },
  };
  if (opts.actPollerAutoscaleMax) {
    workerArgs.activityTaskPollerBehavior = {
      type: 'autoscaling',
      maximum: opts.actPollerAutoscaleMax,
    };
  } else if (opts.maxActPollers) {
    workerArgs.maxConcurrentActivityTaskPolls = opts.maxActPollers;
  }
  if (opts.wfPollerAutoscaleMax) {
    workerArgs.workflowTaskPollerBehavior = {
      type: 'autoscaling',
      maximum: opts.wfPollerAutoscaleMax,
    };
  } else if (opts.maxWfPollers) {
    workerArgs.maxConcurrentWorkflowTaskPolls = opts.maxWfPollers;
  }
  if (opts.maxActs) {
    workerArgs.maxConcurrentActivityTaskExecutions = opts.maxActs;
  }
  if (opts.maxWFTs) {
    workerArgs.maxConcurrentWorkflowTaskExecutions = opts.maxWFTs;
  }
  if (opts.workerActivityRate) {
    workerArgs.maxActivitiesPerSecond = opts.workerActivityRate;
  }
  const workerPromises = [];
  for (const taskQueue of taskQueues) {
    workerArgs.taskQueue = taskQueue;
    const worker = await Worker.create(workerArgs);
    workerPromises.push(worker.run());
  }

  await Promise.all(workerPromises);
}

run()
  .then(() => {
    process.exit(0);
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });

function coerceLogLevel(value: string): LogLevel {
  const lowered = value.toLowerCase();
  if (lowered === 'trace') {
    return 'TRACE';
  } else if (lowered === 'debug') {
    return 'DEBUG';
  } else if (lowered === 'info') {
    return 'INFO';
  } else if (lowered === 'warn') {
    return 'WARN';
  } else if (lowered === 'error') {
    return 'ERROR';
  } else if (lowered === 'fatal') {
    return 'ERROR';
  }
  return 'INFO';
}
