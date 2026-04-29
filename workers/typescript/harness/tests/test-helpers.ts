import { promisify } from 'node:util';
import * as grpc from '@grpc/grpc-js';
import { Client, type ConnectionLike, type Metadata } from '@temporalio/client';
import type { ConnectOptions } from '../src/generated/temporal/omes/projects/v1/ConnectOptions';
import type { ExecuteRequest } from '../src/generated/temporal/omes/projects/v1/ExecuteRequest';
import type { InitRequest } from '../src/generated/temporal/omes/projects/v1/InitRequest';
import type { ProjectServiceClient } from '../src/generated/temporal/omes/projects/v1/ProjectService';
import { projectServiceClientConstructor } from '../src/grpc-helpers';

type ConnectOptionsOverrides = Partial<ConnectOptions>;
type InitRequestOverrides = Omit<Partial<InitRequest>, 'connectOptions'> & {
  connectOptions?: ConnectOptionsOverrides;
  configJson?: Uint8Array;
};
type ExecuteRequestOverrides = Partial<ExecuteRequest> & {
  payload?: Uint8Array;
};

export type { ProjectServiceClient };

function makeConnection(): ConnectionLike {
  return {
    workflowService: {} as never,
    operatorService: {} as never,
    plugins: [],
    async close(): Promise<void> {
      return undefined;
    },
    async ensureConnected(): Promise<void> {
      return undefined;
    },
    async withDeadline<ReturnType>(
      _: number | Date,
      fn: () => Promise<ReturnType>,
    ): Promise<ReturnType> {
      return await fn();
    },
    async withMetadata<ReturnType>(
      _: Metadata,
      fn: () => Promise<ReturnType>,
    ): Promise<ReturnType> {
      return await fn();
    },
    async withAbortSignal<ReturnType>(
      _: AbortSignal,
      fn: () => Promise<ReturnType>,
    ): Promise<ReturnType> {
      return await fn();
    },
  };
}

export function makeClient(): Client {
  return new Client({
    connection: makeConnection(),
  });
}

export function makeConnectOptions({
  namespace = 'default',
  serverAddress = 'localhost:7233',
  authHeader = '',
  enableTls = false,
  tlsCertPath = '',
  tlsKeyPath = '',
  tlsServerName = '',
  disableHostVerification = false,
}: ConnectOptionsOverrides = {}): ConnectOptions {
  return {
    namespace,
    serverAddress,
    authHeader,
    enableTls,
    tlsCertPath,
    tlsKeyPath,
    tlsServerName,
    disableHostVerification,
  };
}

export const defaultInitReq = {
  executionId: 'exec-id',
  runId: 'run-id',
  taskQueue: 'task-queue',
  connectOptions: makeConnectOptions(),
  configJson: Buffer.from('{"hello":"world"}'),
};

export function makeInitRequest({
  connectOptions,
  ...overrides
}: InitRequestOverrides = {}): InitRequest {
  return {
    ...defaultInitReq,
    ...overrides,
    connectOptions: makeConnectOptions(connectOptions),
  };
}

export const defaultExecReq = {
  iteration: 7,
  taskQueue: 'task-queue',
  payload: Buffer.from('payload'),
};

export function makeExecuteRequest(overrides: ExecuteRequestOverrides = {}): ExecuteRequest {
  return { ...defaultExecReq, ...overrides };
}

export async function callUnary<Response>(
  invoke: (callback: (error: grpc.ServiceError | null, response: Response) => void) => void,
): Promise<Response> {
  return await promisify(invoke)();
}

export async function bindGrpcServer(server: grpc.Server): Promise<number> {
  return await promisify(server.bindAsync.bind(server))(
    '127.0.0.1:0',
    grpc.ServerCredentials.createInsecure(),
  );
}

export async function shutdownGrpcServer(server: grpc.Server): Promise<void> {
  await promisify(server.tryShutdown.bind(server))();
}

export function createProjectServiceClient(port: number): ProjectServiceClient {
  const ProjectServiceClientCtor = projectServiceClientConstructor;
  return new ProjectServiceClientCtor(`127.0.0.1:${port}`, grpc.credentials.createInsecure());
}
