import { Client } from '@temporalio/client';
import { Logger, Worker, WorkerOptions } from '@temporalio/worker';
import { Command } from 'commander';
import { buildClientConfig, ClientFactory } from './client';
import { configureLogger } from './helpers';

export interface WorkerContext {
  logger: Logger;
  taskQueue: string;
  errOnUnimplemented: boolean;
  workerOptions: Partial<WorkerOptions>;
}

export type WorkerFactory = (client: Client, context: WorkerContext) => Worker | Promise<Worker>;

interface WorkerCliOptions {
  taskQueue: string;
  taskQueueSuffixIndexStart: number;
  taskQueueSuffixIndexEnd: number;
  maxConcurrentActivityPollers?: number;
  maxConcurrentWorkflowPollers?: number;
  activityPollerAutoscaleMax?: number;
  workflowPollerAutoscaleMax?: number;
  maxConcurrentActivities?: number;
  maxConcurrentWorkflowTasks?: number;
  activitiesPerSecond?: number;
  errOnUnimplemented: boolean;
  logLevel: string;
  logEncoding: string;
  namespace: string;
  serverAddress: string;
  tls: boolean;
  tlsCertPath: string;
  tlsKeyPath: string;
  promListenAddress?: string;
  promHandlerPath: string;
  authHeader: string;
  buildId: string;
}

export async function runWorkerCli(
  workerFactory: WorkerFactory,
  clientFactory: ClientFactory,
  argv: readonly string[],
): Promise<void> {
  let resolveInterrupt!: () => void;
  const interruptPromise = new Promise<void>((resolve) => {
    resolveInterrupt = resolve;
  });
  const onInterrupt = () => resolveInterrupt();
  process.once('SIGINT', onInterrupt);
  try {
    await runWorker(workerFactory, clientFactory, interruptPromise, argv);
  } finally {
    process.off('SIGINT', onInterrupt);
  }
}

export async function runWorker(
  workerFactory: WorkerFactory,
  clientFactory: ClientFactory,
  interruptPromise: Promise<void>,
  argv: readonly string[],
): Promise<void> {
  const args = buildParser().parse(argv, { from: 'user' }).opts<WorkerCliOptions>();

  if (args.taskQueueSuffixIndexStart > args.taskQueueSuffixIndexEnd) {
    throw new Error('Task queue suffix start after end');
  }

  const logger = configureLogger(args.logLevel, args.logEncoding);
  const config = buildClientConfig({
    server_address: args.serverAddress,
    namespace: args.namespace,
    auth_header: args.authHeader,
    tls: args.tls,
    tls_cert_path: args.tlsCertPath,
    tls_key_path: args.tlsKeyPath,
    prom_listen_address: args.promListenAddress,
  });
  const client = await clientFactory({
    ...config,
    runtimeOptions: {
      ...config.runtimeOptions,
      logger,
    },
  });

  const taskQueues = buildTaskQueues(
    logger,
    args.taskQueue,
    args.taskQueueSuffixIndexStart,
    args.taskQueueSuffixIndexEnd,
  );
  const workerOptions = buildWorkerOptions(args);
  const workers = await Promise.all(
    taskQueues.map(
      async (taskQueue) =>
        await workerFactory(client, {
          logger,
          taskQueue,
          errOnUnimplemented: args.errOnUnimplemented,
          workerOptions,
        }),
    ),
  );

  await runWorkers(workers, interruptPromise);
}

export async function runWorkers(
  workers: readonly Worker[],
  interruptPromise: Promise<void>,
): Promise<void> {
  const allWorkersPromise = Promise.all(workers.map(async (worker) => await worker.run()));
  const completionPromise = allWorkersPromise.then(
    () => undefined,
    () => undefined,
  );
  await Promise.race([completionPromise, interruptPromise]);
  await Promise.all(workers.map(shutdownWorker));
  await allWorkersPromise;
}

async function shutdownWorker(worker: Worker): Promise<void> {
  try {
    await worker.shutdown();
  } catch (err) {
    if (!isAlreadyShuttingDown(err)) {
      throw err;
    }
  }
}

function isAlreadyShuttingDown(err: unknown): boolean {
  return (
    err instanceof Error &&
    err.name === 'IllegalStateError' &&
    err.message.startsWith('Not running. Current state:')
  );
}

function parseBooleanFlag(value: string | boolean): boolean {
  return typeof value === 'boolean' ? value : ['true', '1', 'yes'].includes(value.toLowerCase());
}

function buildParser(): Command {
  return new Command()
    .option('-q, --task-queue <taskQueue>', 'Task queue to use', 'omes')
    .option(
      '--task-queue-suffix-index-start <tqSufStart>',
      'Inclusive start for task queue suffix range',
      (value) => Number.parseInt(value, 10),
      0,
    )
    .option(
      '--task-queue-suffix-index-end <tqSufEnd>',
      'Inclusive end for task queue suffix range',
      (value) => Number.parseInt(value, 10),
      0,
    )
    .option(
      '--max-concurrent-activity-pollers <maxActPollers>',
      'Max concurrent activity pollers',
      (value) => Number.parseInt(value, 10),
    )
    .option(
      '--max-concurrent-workflow-pollers <maxWfPollers>',
      'Max concurrent workflow pollers',
      (value) => Number.parseInt(value, 10),
    )
    .option(
      '--activity-poller-autoscale-max <actPollerAutoscaleMax>',
      'Max for activity poller autoscaling (overrides max-concurrent-activity-pollers)',
      (value) => Number.parseInt(value, 10),
    )
    .option(
      '--workflow-poller-autoscale-max <wfPollerAutoscaleMax>',
      'Max for workflow poller autoscaling (overrides max-concurrent-workflow-pollers)',
      (value) => Number.parseInt(value, 10),
    )
    .option('--max-concurrent-activities <maxActs>', 'Max concurrent activities', (value) =>
      Number.parseInt(value, 10),
    )
    .option('--max-concurrent-workflow-tasks <maxWFTs>', 'Max concurrent workflow tasks', (value) =>
      Number.parseInt(value, 10),
    )
    .option(
      '--activities-per-second <workerActivityRate>',
      'Per-worker activity rate limit',
      (value) => Number.parseFloat(value),
    )
    .option(
      '--err-on-unimplemented <errOnImplemented>',
      'Error when receiving unimplemented actions (currently only affects concurrent client actions)',
      parseBooleanFlag,
      false,
    )
    .option('--log-level <logLevel>', '(debug info warn error panic fatal)', 'info')
    .option('--log-encoding <logEncoding>', '(console json)', 'console')
    .option('-n, --namespace <namespace>', 'The namespace to use', 'default')
    .option('-a, --server-address <address>', 'The host:port of the server', 'localhost:7233')
    .option('--tls <tls>', 'Enable TLS (true/false)', parseBooleanFlag, false)
    .option('--tls-cert-path <clientCertPath>', 'Path to a client certificate for TLS', '')
    .option('--tls-key-path <clientKeyPath>', 'Path to a client key for TLS', '')
    .option('--prom-listen-address <promListenAddress>', 'Prometheus listen address')
    .option('--prom-handler-path <promHandlerPath>', 'Prometheus handler path', '/metrics')
    .option('--auth-header <authHeader>', 'Authorization header value', '')
    .option('--build-id <buildId>', 'Build ID', '');
}

function buildTaskQueues(
  logger: Logger,
  taskQueue: string,
  suffixStart: number,
  suffixEnd: number,
): string[] {
  if (suffixEnd === 0) {
    logger.info(`TypeScript worker will run on task queue ${taskQueue}`);
    return [taskQueue];
  }

  const taskQueues = [];
  for (let index = suffixStart; index <= suffixEnd; index += 1) {
    taskQueues.push(`${taskQueue}-${index}`);
  }
  logger.info(`TypeScript worker will run on ${taskQueues.length} task queues`);
  return taskQueues;
}

function buildWorkerOptions(options: WorkerCliOptions): Partial<WorkerOptions> {
  const workerOptions: Partial<WorkerOptions> = {};
  if (options.activityPollerAutoscaleMax !== undefined) {
    workerOptions.activityTaskPollerBehavior = {
      type: 'autoscaling',
      maximum: options.activityPollerAutoscaleMax,
    };
  } else if (options.maxConcurrentActivityPollers !== undefined) {
    workerOptions.maxConcurrentActivityTaskPolls = options.maxConcurrentActivityPollers;
  }

  if (options.workflowPollerAutoscaleMax !== undefined) {
    workerOptions.workflowTaskPollerBehavior = {
      type: 'autoscaling',
      maximum: options.workflowPollerAutoscaleMax,
    };
  } else if (options.maxConcurrentWorkflowPollers !== undefined) {
    workerOptions.maxConcurrentWorkflowTaskPolls = options.maxConcurrentWorkflowPollers;
  }

  if (options.maxConcurrentActivities !== undefined) {
    workerOptions.maxConcurrentActivityTaskExecutions = options.maxConcurrentActivities;
  }
  if (options.maxConcurrentWorkflowTasks !== undefined) {
    workerOptions.maxConcurrentWorkflowTaskExecutions = options.maxConcurrentWorkflowTasks;
  }
  if (options.activitiesPerSecond !== undefined) {
    workerOptions.maxActivitiesPerSecond = options.activitiesPerSecond;
  }
  return workerOptions;
}
