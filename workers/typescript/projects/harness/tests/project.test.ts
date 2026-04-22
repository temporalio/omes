import assert from 'node:assert/strict';
import { join } from 'node:path';
import test from 'node:test';
import * as grpc from '@grpc/grpc-js';
import { Client } from '@temporalio/client';
import { TestWorkflowEnvironment } from '@temporalio/testing';
import { Worker } from '@temporalio/worker';
import type { ClientConfig, ClientFactory } from '../client.js';
import {
  ProjectServiceServer,
  type ProjectExecuteContext,
  type ProjectInitContext,
} from '../project.js';
import {
  bindGrpcServer,
  callUnary,
  createProjectServiceClient,
  makeClient,
  makeConnectOptions,
  makeExecuteRequest,
  makeInitRequest,
  registerProjectService,
  shutdownGrpcServer,
  type ProjectServiceClient,
} from './test-helpers.js';

type ProjectEvent =
  | {
      kind: 'init';
      client: Client;
      context: Pick<ProjectInitContext, 'run' | 'taskQueue'> & { configJson: string };
    }
  | {
      kind: 'execute';
      client: Client;
      context: Pick<ProjectExecuteContext, 'run' | 'taskQueue' | 'iteration'> & { payload: string };
      result: string;
    };

test('Init rejects invalid TLS configuration', async () => {
  const server = new ProjectServiceServer(
    { execute: async () => undefined },
    async () => makeClient(),
  );

  await assert.rejects(
    () =>
      server.Init(
        makeInitRequest({
          connectOptions: makeConnectOptions({
            enableTls: true,
            tlsCertPath: '/tmp/cert.pem',
          }),
        }),
      ),
    {
      code: grpc.status.INVALID_ARGUMENT,
      details: 'Client cert specified, but not client key!',
    },
  );
});

test('Init passes run metadata to the init handler', async () => {
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

  await server.Init(makeInitRequest());

  assert.ok(receivedConfig !== undefined);
  assert.ok(initCall !== undefined);
  assert.deepEqual({
    targetHost: receivedConfig.targetHost,
    namespace: receivedConfig.namespace,
    apiKey: receivedConfig.apiKey,
    tls: receivedConfig.tls,
  }, {
    targetHost: 'localhost:7233',
    namespace: 'default',
    apiKey: undefined,
    tls: undefined,
  });
  assert.strictEqual(initCall.client, client);
  assert.deepEqual({
    run: initCall.context.run,
    taskQueue: initCall.context.taskQueue,
    configJson: Buffer.from(initCall.context.configJson).toString('utf8'),
  }, {
    run: {
      runId: 'run-id',
      executionId: 'exec-id',
    },
    taskQueue: 'task-queue',
    configJson: '{"hello":"world"}',
  });
});

void test('Execute requires Init first', async () => {
  const server = new ProjectServiceServer(
    { execute: async () => undefined },
    async () => makeClient(),
  );

  await assert.rejects(
    () => server.Execute(makeExecuteRequest()),
    {
      code: grpc.status.FAILED_PRECONDITION,
      details: 'Init must be called before Execute',
    },
  );
});

void test('Execute passes iteration payload and run metadata', async () => {
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

  await server.Init(makeInitRequest());
  await server.Execute(makeExecuteRequest());

  assert.ok(executeCall !== undefined);
  assert.strictEqual(executeCall.client, client);
  assert.deepEqual({
    run: executeCall.context.run,
    taskQueue: executeCall.context.taskQueue,
    iteration: executeCall.context.iteration,
    payload: Buffer.from(executeCall.context.payload).toString('utf8'),
  }, {
    run: {
      runId: 'run-id',
      executionId: 'exec-id',
    },
    taskQueue: 'task-queue',
    iteration: 7,
    payload: 'payload',
  });
});

void test('client factory failure maps to internal error', async () => {
  const server = new ProjectServiceServer(
    { execute: async () => undefined },
    async () => {
      throw new Error('boom');
    },
  );

  await assert.rejects(
    () => server.Init(makeInitRequest()),
    {
      code: grpc.status.INTERNAL,
      details: 'failed to create client: boom',
    },
  );
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

  await assert.rejects(
    () => server.Init(makeInitRequest()),
    {
      code: grpc.status.INTERNAL,
      details: 'init handler failed: bad init',
    },
  );

  await assert.rejects(
    () => server.Execute(makeExecuteRequest()),
    {
      code: grpc.status.FAILED_PRECONDITION,
      details: 'Init must be called before Execute',
    },
  );
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

  await server.Init(makeInitRequest());
  await assert.rejects(
    () => server.Execute(makeExecuteRequest()),
    {
      code: grpc.status.INTERNAL,
      details: 'execute handler failed: bad execute',
    },
  );
});

void test('project server executes workflow against a real Temporal server', async () => {
  const events: ProjectEvent[] = [];
  const taskQueue = 'project-harness-e2e';
  const env = await TestWorkflowEnvironment.createLocal();
  const server = new grpc.Server();
  let client: ProjectServiceClient | undefined;

  try {
    const service = new ProjectServiceServer(
      {
        init: async (handlerClient, context) => {
          events.push({
            kind: 'init',
            client: handlerClient,
            context: {
              run: context.run,
              taskQueue: context.taskQueue,
              configJson: Buffer.from(context.configJson).toString('utf8'),
            },
          });
        },
        execute: async (handlerClient, context) => {
          const result = await handlerClient.workflow.execute('projectHarnessEchoWorkflow', {
            args: [Buffer.from(context.payload).toString('utf8')],
            workflowId: `${context.run.executionId}-${context.iteration}`,
            taskQueue: context.taskQueue,
          });
          events.push({
            kind: 'execute',
            client: handlerClient,
            context: {
              run: context.run,
              taskQueue: context.taskQueue,
              iteration: context.iteration,
              payload: Buffer.from(context.payload).toString('utf8'),
            },
            result,
          });
        },
      },
      async (config) =>
        new Client({
          connection: env.nativeConnection,
          namespace: config.namespace,
        }),
    );
    registerProjectService(server, service);

    client = createProjectServiceClient(await bindGrpcServer(server));
    assert.ok(client !== undefined);
    const projectClient = client;
    const worker = await Worker.create({
      connection: env.nativeConnection,
      taskQueue,
      workflowsPath: join(__dirname, 'workflows/project_echo_workflow.js'),
    });

    await callUnary((callback) =>
      projectClient.init(
        makeInitRequest({
          taskQueue,
          connectOptions: makeConnectOptions({
            namespace: env.namespace ?? 'default',
            serverAddress: env.address,
          }),
        }),
        callback,
      ),
    );
    await worker.runUntil(async () => {
      await callUnary((callback) =>
        projectClient.execute(
          makeExecuteRequest({ taskQueue }),
          callback,
        ),
      );
    });
  } finally {
    client?.close();
    await shutdownGrpcServer(server);
    await env.teardown();
  }

  assert.equal(events.length, 2);
  assert.strictEqual(events[0]?.client, events[1]?.client);
  assert.deepEqual(events[0], {
    kind: 'init',
    client: events[0]?.client,
    context: {
      run: {
        runId: 'run-id',
        executionId: 'exec-id',
      },
      taskQueue,
      configJson: '{"hello":"world"}',
    },
  });
  assert.deepEqual(events[1], {
    kind: 'execute',
    client: events[0]?.client,
    context: {
      run: {
        runId: 'run-id',
        executionId: 'exec-id',
      },
      taskQueue,
      iteration: 7,
      payload: 'payload',
    },
    result: 'payload',
  });
});
