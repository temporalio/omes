import assert from 'node:assert/strict';
import test from 'node:test';
import * as grpc from '@grpc/grpc-js';
import { Client } from '@temporalio/client';
import { TestWorkflowEnvironment } from '@temporalio/testing';
import { Worker } from '@temporalio/worker';
import type { ClientConfig, ClientFactory } from '../src/client.js';
import {
  ProjectServiceServer,
  type ProjectExecuteContext,
  type ProjectInitContext,
} from '../src/project.js';
import {
  bindGrpcServer,
  callUnary,
  createProjectServiceClient,
  defaultExecReq,
  makeClient,
  makeConnectOptions,
  makeExecuteRequest,
  makeInitRequest,
  shutdownGrpcServer,
  type ProjectServiceClient,
} from './test-helpers.js';
import { registerProjectService } from '../src/helpers.js';
import { InitRequest__Output } from '../src/generated/temporal/omes/projects/v1/InitRequest.js';
import { ExecuteRequest__Output } from '../src/generated/temporal/omes/projects/v1/ExecuteRequest.js';

void test('project server executes workflow against a real Temporal server', async () => {
  let initContext: ProjectInitContext | undefined;
  let executeContext: ProjectExecuteContext | undefined;
  let initClient: Client | undefined;
  let executeClient: Client | undefined;
  let executeResult: string | undefined;
  const eventKinds: string[] = [];
  const env = await TestWorkflowEnvironment.createLocal();
  const server = new grpc.Server();
  let client: ProjectServiceClient | undefined;

  try {
    const service = new ProjectServiceServer(
      {
        init: async (handlerClient, context) => {
          initClient = handlerClient;
          initContext = context;
          eventKinds.push('init');
        },
        execute: async (handlerClient, context) => {
          executeClient = handlerClient;
          const worker = await Worker.create({
            connection: env.nativeConnection,
            namespace: env.namespace,
            taskQueue: context.taskQueue,
            workflowsPath: require.resolve('./workflows.js'),
          });

          await worker.runUntil(async () => {
            const handle = await handlerClient.workflow.start('testWorkflow', {
              args: ['Temporal'],
              taskQueue: context.taskQueue,
              workflowId: `${context.run.executionId}-${context.iteration}`,
            });
            executeContext = context;
            executeResult = await handle.result();
            eventKinds.push('execute');
          });
        },
      },
      async (_config) => {
        return await new Client({
          connection: env.nativeConnection,
          namespace: env.namespace,
        });
      },
    );

    registerProjectService(server, service);
    client = createProjectServiceClient(await bindGrpcServer(server));

    await callUnary((callback) =>
      client!.Init(
        makeInitRequest({
          connectOptions: makeConnectOptions({
            serverAddress: env.address,
            namespace: env.namespace,
          }),
        }),
        callback,
      ),
    );

    await callUnary((callback) =>
      client!.Execute(
        makeExecuteRequest({
          iteration: 11,
        }),
        callback,
      ),
    );

    assert.deepEqual(eventKinds, ['init', 'execute']);
    assert.ok(initClient !== undefined && initClient === executeClient);
    assert.partialDeepStrictEqual(initContext, {
      run: {
        runId: 'run-id',
        executionId: 'exec-id',
      },
      taskQueue: 'task-queue',
      configJson: Buffer.from('{"hello":"world"}'),
    });
    assert.partialDeepStrictEqual(executeContext, {
      run: {
        runId: 'run-id',
        executionId: 'exec-id',
      },
      taskQueue: 'task-queue',
      iteration: 11,
      payload: Buffer.from('payload'),
    });
    assert.strictEqual(executeResult, 'Hello, Temporal!');
  } finally {
    if (client !== undefined) {
      client.close();
    }
    await shutdownGrpcServer(server);
    await env.teardown();
  }
});

void test('init rejects invalid TLS configuration', async () => {
  const server = new ProjectServiceServer({ execute: async () => undefined }, async () =>
    makeClient(),
  );

  const invalidInitReq = makeInitRequest({
    connectOptions: {
      enableTls: true,
      tlsCertPath: '/tmp/cert.pem',
    },
  });
  await assert.rejects(() => server.Init(invalidInitReq as InitRequest__Output), {
    code: grpc.status.INVALID_ARGUMENT,
    details: 'Client cert specified, but not client key!',
  });
});

void test('init passes run metadata to init handler', async () => {
  const client = makeClient();
  let receivedConfig: ClientConfig | undefined;
  let initCall:
    | {
        client: Client;
        context: ProjectInitContext;
      }
    | undefined;

  const clientFactory: ClientFactory = async (config) => {
    receivedConfig = config;
    return client;
  };
  const server = new ProjectServiceServer(
    {
      execute: async () => undefined,
      init: async (receivedClient, context) => {
        initCall = {
          client: receivedClient,
          context,
        };
      },
    },
    clientFactory,
  );

  await server.Init(makeInitRequest({}) as InitRequest__Output);
  assert.partialDeepStrictEqual(receivedConfig, {
    targetHost: 'localhost:7233',
    namespace: 'default',
  });
  assert.partialDeepStrictEqual(initCall, {
    client,
    context: {
      run: {
        runId: 'run-id',
        executionId: 'exec-id',
      },
      taskQueue: 'task-queue',
      configJson: Buffer.from('{"hello":"world"}'),
    },
  });
});

void test('Execute requires Init first', async () => {
  const server = new ProjectServiceServer({ execute: async () => undefined }, async () =>
    makeClient(),
  );

  await assert.rejects(() => server.Execute(makeExecuteRequest() as ExecuteRequest__Output), {
    code: grpc.status.FAILED_PRECONDITION,
    details: 'Init must be called before Execute',
  });
});

void test('execute passes iteration payload and run metadata', async () => {
  const client = makeClient();
  let executeCall:
    | {
        client: Client;
        context: ProjectExecuteContext;
      }
    | undefined;

  const server = new ProjectServiceServer(
    {
      execute: async (receivedClient, context) => {
        executeCall = {
          client: receivedClient,
          context,
        };
      },
    },
    async () => client,
  );

  await server.Init(makeInitRequest() as InitRequest__Output);
  await server.Execute(makeExecuteRequest() as ExecuteRequest__Output);

  assert.ok(executeCall !== undefined);
  assert.partialDeepStrictEqual(executeCall, {
    client,
    context: {
      run: {
        runId: 'run-id',
        executionId: 'exec-id',
      },
      ...defaultExecReq,
    },
  });
});

void test('client factory failure maps to internal error', async () => {
  const server = new ProjectServiceServer({ execute: async () => undefined }, async () => {
    throw new Error('boom');
  });

  await assert.rejects(() => server.Init(makeInitRequest() as InitRequest__Output), {
    code: grpc.status.INTERNAL,
    details: 'failed to create client: boom',
  });
});

void test('init handler failure does not leave server initialized', async () => {
  const server = new ProjectServiceServer(
    {
      execute: async () => undefined,
      init: async () => {
        throw new Error('bad init');
      },
    },
    async () => makeClient(),
  );

  await assert.rejects(() => server.Init(makeInitRequest() as InitRequest__Output), {
    code: grpc.status.INTERNAL,
    details: 'init handler failed: bad init',
  });

  await assert.rejects(() => server.Execute(makeExecuteRequest() as ExecuteRequest__Output), {
    code: grpc.status.FAILED_PRECONDITION,
    details: 'Init must be called before Execute',
  });
});

void test('execute handler failure maps to internal error', async () => {
  const server = new ProjectServiceServer(
    {
      execute: async () => {
        throw new Error('bad execute');
      },
    },
    async () => makeClient(),
  );

  await server.Init(makeInitRequest() as InitRequest__Output);
  await assert.rejects(() => server.Execute(makeExecuteRequest() as ExecuteRequest__Output), {
    code: grpc.status.INTERNAL,
    details: 'execute handler failed: bad execute',
  });
});
