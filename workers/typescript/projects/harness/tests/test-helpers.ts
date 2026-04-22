import { promisify } from 'node:util';
import * as grpc from '@grpc/grpc-js';
import { Client, type ConnectionLike, type Metadata } from '@temporalio/client';
import * as apiGrpc from '../api/api/api_grpc_pb.js';
import * as apiPb from '../api/api/api_pb.js';
import type { ProjectServiceServer } from '../project.js';

type ConnectOptionsOverrides = Partial<apiPb.ConnectOptions.AsObject>;
type InitRequestOverrides = Partial<
  Omit<apiPb.InitRequest.AsObject, 'connectOptions' | 'configJson'>
> & {
  connectOptions?: apiPb.ConnectOptions;
  configJson?: Uint8Array;
};
type ExecuteRequestOverrides = Partial<Omit<apiPb.ExecuteRequest.AsObject, 'payload'>> & {
  payload?: Uint8Array;
};

export type ProjectServiceClient = InstanceType<typeof apiGrpc.ProjectServiceClient>;

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
    async withDeadline<ReturnType>(_: number | Date, fn: () => Promise<ReturnType>): Promise<ReturnType> {
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
}: ConnectOptionsOverrides = {}): apiPb.ConnectOptions {
  const connectOptions = new apiPb.ConnectOptions();
  connectOptions.setNamespace(namespace);
  connectOptions.setServerAddress(serverAddress);
  connectOptions.setAuthHeader(authHeader);
  connectOptions.setEnableTls(enableTls);
  connectOptions.setTlsCertPath(tlsCertPath);
  connectOptions.setTlsKeyPath(tlsKeyPath);
  connectOptions.setTlsServerName(tlsServerName);
  connectOptions.setDisableHostVerification(disableHostVerification);
  return connectOptions;
}

export function makeInitRequest({
  executionId = 'exec-id',
  runId = 'run-id',
  taskQueue = 'task-queue',
  connectOptions = makeConnectOptions(),
  configJson = Buffer.from('{"hello":"world"}'),
}: InitRequestOverrides = {}): apiPb.InitRequest {
  const request = new apiPb.InitRequest();
  request.setExecutionId(executionId);
  request.setRunId(runId);
  request.setTaskQueue(taskQueue);
  request.setConnectOptions(connectOptions);
  request.setConfigJson(configJson);
  return request;
}

export function makeExecuteRequest({
  iteration = 7,
  taskQueue = 'task-queue',
  payload = Buffer.from('payload'),
}: ExecuteRequestOverrides = {}): apiPb.ExecuteRequest {
  const request = new apiPb.ExecuteRequest();
  request.setIteration(iteration);
  request.setTaskQueue(taskQueue);
  request.setPayload(payload);
  return request;
}

function createUnaryHandler<Request, Response>(
  handler: (request: Request) => Promise<Response>,
): grpc.handleUnaryCall<Request, Response> {
  return async (call, callback) => {
    try {
      callback(null, await handler(call.request));
    } catch (error) {
      callback(error as grpc.ServiceError);
    }
  };
}

export function registerProjectService(server: grpc.Server, service: ProjectServiceServer): void {
  server.addService(apiGrpc.ProjectServiceService, {
    init: createUnaryHandler((request) => service.Init(request)),
    execute: createUnaryHandler((request) => service.Execute(request)),
  });
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
  return new apiGrpc.ProjectServiceClient(
    `127.0.0.1:${port}`,
    grpc.credentials.createInsecure(),
  );
}

export async function callUnary<Response>(
  invoke: (callback: (error: grpc.ServiceError | null, response: Response) => void) => void,
): Promise<Response> {
  return await promisify(invoke)();
}
