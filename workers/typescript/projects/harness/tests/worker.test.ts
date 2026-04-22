import assert from 'node:assert/strict';
import { Client } from '@temporalio/client';
import test from 'node:test';
import type { ClientConfig } from '../client.js';
import type { Worker } from '@temporalio/worker';
import { runWorker, runWorkers, type WorkerContext, type WorkerFactory } from '../worker.js';
import { makeClient } from './test-helpers.js';

interface WorkerFactoryCall {
  client: Client;
  context: WorkerContext;
}

interface TestWorker {
  runCalls: number;
  shutdownCalls: number;
  run(): Promise<void>;
  shutdown(): Promise<void>;
}

function neverInterrupt(): Promise<void> {
  return new Promise<void>(() => {});
}

function makeTestWorker(onRun: () => Promise<void>): TestWorker {
  return {
    runCalls: 0,
    shutdownCalls: 0,
    async run(): Promise<void> {
      this.runCalls += 1;
      await onRun();
    },
    async shutdown(): Promise<void> {
      this.shutdownCalls += 1;
    },
  };
}

void test('runWorker passes shared client and context to each worker factory', async () => {
  const client = makeClient();
  const clientConfigs: ClientConfig[] = [];
  const workerFactoryCalls: WorkerFactoryCall[] = [];

  const workerFactory: WorkerFactory = async (receivedClient, context) => {
    workerFactoryCalls.push({
      client: receivedClient,
      context,
    });
    return makeTestWorker(async () => undefined) as unknown as Worker;
  };
  const clientFactory = async (config: ClientConfig): Promise<Client> => {
    clientConfigs.push(config);
    return client;
  };

  await runWorker(workerFactory, clientFactory, neverInterrupt(), [
    '--task-queue',
    'omes',
    '--task-queue-suffix-index-start',
    '1',
    '--task-queue-suffix-index-end',
    '2',
  ]);

  assert.deepEqual(clientConfigs.map((config) => ({
    targetHost: config.targetHost,
    namespace: config.namespace,
    apiKey: config.apiKey,
    tls: config.tls,
  })), [
    {
      targetHost: 'localhost:7233',
      namespace: 'default',
      apiKey: undefined,
      tls: undefined,
    },
  ]);
  assert.strictEqual(workerFactoryCalls[0]?.client, client);
  assert.strictEqual(workerFactoryCalls[1]?.client, client);
  assert.deepEqual(workerFactoryCalls.map(({ context }) => ({
    taskQueue: context.taskQueue,
    errOnUnimplemented: context.errOnUnimplemented,
    workerOptions: context.workerOptions,
  })), [
    {
      taskQueue: 'omes-1',
      errOnUnimplemented: false,
      workerOptions: {},
    },
    {
      taskQueue: 'omes-2',
      errOnUnimplemented: false,
      workerOptions: {},
    },
  ]);
});

void test('runWorkers shuts down all workers when one fails', async () => {
  const failingWorker = makeTestWorker(async () => {
    throw new Error('boom');
  });
  const successfulWorker = makeTestWorker(async () => undefined);

  await assert.rejects(
    () =>
      runWorkers(
        [
          failingWorker as unknown as Worker,
          successfulWorker as unknown as Worker,
        ],
        neverInterrupt(),
      ),
    /boom/,
  );

  assert.deepEqual(
    {
      failingRunCalls: failingWorker.runCalls,
      successfulRunCalls: successfulWorker.runCalls,
      failingShutdownCalls: failingWorker.shutdownCalls,
      successfulShutdownCalls: successfulWorker.shutdownCalls,
    },
    {
      failingRunCalls: 1,
      successfulRunCalls: 1,
      failingShutdownCalls: 1,
      successfulShutdownCalls: 1,
    },
  );
});
