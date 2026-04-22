import * as grpc from '@grpc/grpc-js';
import { Client, isGrpcServiceError } from '@temporalio/client';
import { DefaultLogger, Logger } from '@temporalio/worker';
import * as apiPb from './api/api/api_pb';
import {
  IProjectServiceServer,
  ProjectServiceService,
} from './api/api/api_grpc_pb';
import { buildClientConfig, ClientFactory } from './client';

export interface ProjectRunMetadata {
  runId: string;
  executionId: string;
}

export interface ProjectInitContext {
  logger: Logger;
  run: ProjectRunMetadata;
  taskQueue: string;
  configJson: Uint8Array;
}

export interface ProjectExecuteContext {
  logger: Logger;
  run: ProjectRunMetadata;
  taskQueue: string;
  iteration: number;
  payload: Uint8Array;
}

export type ProjectExecuteHandler = (
  client: Client,
  context: ProjectExecuteContext,
) => Promise<void>;

export type ProjectInitHandler = (
  client: Client,
  context: ProjectInitContext,
) => Promise<void>;

export interface ProjectHandlers {
  execute: ProjectExecuteHandler;
  init?: ProjectInitHandler;
}

const logger = new DefaultLogger('INFO');

export class ProjectServiceServer {
  private client: Client | undefined;
  private run: ProjectRunMetadata | undefined;

  constructor(
    private readonly handlers: ProjectHandlers,
    private readonly clientFactory: ClientFactory,
  ) {}

  async Init(request: apiPb.InitRequest): Promise<apiPb.InitResponse> {
    const taskQueue = request.getTaskQueue();
    const executionId = request.getExecutionId();
    const runId = request.getRunId();
    const connectOptions = request.getConnectOptions();

    if (!taskQueue) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'task_queue required');
    }
    if (!executionId) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'execution_id required');
    }
    if (!runId) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'run_id required');
    }
    if (connectOptions === undefined || !connectOptions.getServerAddress()) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'server_address required');
    }
    if (!connectOptions.getNamespace()) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'namespace required');
    }

    let config;
    try {
      config = buildClientConfig({
        server_address: connectOptions.getServerAddress(),
        namespace: connectOptions.getNamespace(),
        auth_header: connectOptions.getAuthHeader(),
        tls: connectOptions.getEnableTls(),
        tls_cert_path: connectOptions.getTlsCertPath(),
        tls_key_path: connectOptions.getTlsKeyPath(),
        tls_server_name: connectOptions.getTlsServerName() || undefined,
        disable_host_verification: connectOptions.getDisableHostVerification(),
      });
    } catch (error) {
      throw grpcError(
        grpc.status.INVALID_ARGUMENT,
        error instanceof Error ? error.message : String(error),
      );
    }

    let client;
    try {
      client = await this.clientFactory(config);
    } catch (error) {
      throw grpcError(
        grpc.status.INTERNAL,
        `failed to create client: ${error instanceof Error ? error.message : String(error)}`,
      );
    }

    const run = {
      runId,
      executionId,
    };

    if (this.handlers.init !== undefined) {
      try {
        await this.handlers.init(client, {
          logger,
          run,
          taskQueue,
          configJson: request.getConfigJson_asU8(),
        });
      } catch (error) {
        throw grpcError(
          grpc.status.INTERNAL,
          `init handler failed: ${error instanceof Error ? error.message : String(error)}`,
        );
      }
    }

    this.client = client;
    this.run = run;
    return new apiPb.InitResponse();
  }

  async Execute(request: apiPb.ExecuteRequest): Promise<apiPb.ExecuteResponse> {
    const taskQueue = request.getTaskQueue();

    if (!taskQueue) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'task_queue required');
    }
    if (this.client === undefined || this.run === undefined) {
      throw grpcError(grpc.status.FAILED_PRECONDITION, 'Init must be called before Execute');
    }

    try {
      await this.handlers.execute(this.client, {
        logger,
        run: this.run,
        taskQueue,
        iteration: request.getIteration(),
        payload: request.getPayload_asU8(),
      });
    } catch (error) {
      throw grpcError(
        grpc.status.INTERNAL,
        `execute handler failed: ${error instanceof Error ? error.message : String(error)}`,
      );
    }

    return new apiPb.ExecuteResponse();
  }
}

export async function runProjectServerCli(
  handlers: ProjectHandlers,
  clientFactory: ClientFactory,
  argv: readonly string[],
): Promise<void> {
  const port = parsePortArg(argv);
  const server = new grpc.Server();
  const service = new ProjectServiceServer(handlers, clientFactory);
  server.addService(ProjectServiceService, {
    init: unaryHandler((request) => service.Init(request)),
    execute: unaryHandler((request) => service.Execute(request)),
  } satisfies IProjectServiceServer);
  await new Promise((resolve, reject) => {
    server.bindAsync(`0.0.0.0:${port}`, grpc.ServerCredentials.createInsecure(), (error, boundPort) => {
      if (error) {
        reject(error);
        return;
      }
      resolve(boundPort);
    });
  });
  logger.info(`Project server listening on port ${port}`);

  await new Promise<void>((resolve, reject) => {
    const onInterrupt = () => {
      process.off('SIGINT', onInterrupt);
      server.tryShutdown((error) => {
        if (error) {
          reject(error);
          return;
        }
        resolve();
      });
    };
    process.once('SIGINT', onInterrupt);
  });
}

function parsePortArg(argv: readonly string[]): number {
  let port = 8080;
  for (let index = 0; index < argv.length; index += 1) {
    if (argv[index] === '--port') {
      port = Number.parseInt(argv[index + 1] ?? '', 10);
    }
  }
  return port;
}

function grpcError(code: grpc.status, details: string): grpc.ServiceError {
  return Object.assign(new Error(details), {
    code,
    details,
    metadata: new grpc.Metadata(),
    name: 'Error',
  });
}

function unaryHandler<Request, Response>(
  handler: (request: Request) => Promise<Response>,
): grpc.handleUnaryCall<Request, Response> {
  return async (call, callback) => {
    try {
      callback(null, await handler(call.request));
    } catch (error) {
      callback(isGrpcServiceError(error) ? error: grpcError(grpc.status.INTERNAL, error instanceof Error ? error.message : String(error)));
    }
  };
}
