import { promisify } from 'node:util';
import * as grpc from '@grpc/grpc-js';
import { Client } from '@temporalio/client';
import { DefaultLogger, Logger } from '@temporalio/worker';
import { buildClientConfig, ClientFactory } from './client';
import type { ExecuteRequest__Output } from './generated/temporal/omes/projects/v1/ExecuteRequest';
import type { ExecuteResponse } from './generated/temporal/omes/projects/v1/ExecuteResponse';
import type { InitRequest__Output } from './generated/temporal/omes/projects/v1/InitRequest';
import type { InitResponse } from './generated/temporal/omes/projects/v1/InitResponse';
import { grpcError, registerProjectService } from './helpers';

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

export type ProjectInitHandler = (client: Client, context: ProjectInitContext) => Promise<void>;

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

  async Init(request: InitRequest__Output): Promise<InitResponse> {
    const taskQueue = request.taskQueue ?? '';
    const executionId = request.executionId ?? '';
    const runId = request.runId ?? '';
    const connectOptions = request.connectOptions;

    if (!request.taskQueue) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'task_queue required');
    }
    if (!executionId) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'execution_id required');
    }
    if (!runId) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'run_id required');
    }
    if (connectOptions === undefined || !connectOptions.serverAddress) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'server_address required');
    }
    if (!connectOptions.namespace) {
      throw grpcError(grpc.status.INVALID_ARGUMENT, 'namespace required');
    }

    let config;
    try {
      config = buildClientConfig({
        server_address: connectOptions.serverAddress,
        namespace: connectOptions.namespace,
        auth_header: connectOptions.authHeader ?? '',
        tls: connectOptions.enableTls ?? false,
        tls_cert_path: connectOptions.tlsCertPath ?? '',
        tls_key_path: connectOptions.tlsKeyPath ?? '',
        tls_server_name: connectOptions.tlsServerName || undefined,
        disable_host_verification: connectOptions.disableHostVerification,
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
          configJson: request.configJson ?? Buffer.alloc(0),
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
    return {};
  }

  async Execute(request: ExecuteRequest__Output): Promise<ExecuteResponse> {
    const taskQueue = request.taskQueue ?? '';

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
        iteration: request.iteration ?? 0,
        payload: request.payload ?? Buffer.alloc(0),
      });
    } catch (error) {
      throw grpcError(
        grpc.status.INTERNAL,
        `execute handler failed: ${error instanceof Error ? error.message : String(error)}`,
      );
    }

    return {};
  }
}

export async function runProjectServerCli(
  handlers: ProjectHandlers,
  clientFactory: ClientFactory,
  argv: readonly string[],
): Promise<void> {
  const port = parsePortArg(argv);
  const server = new grpc.Server();
  const bindAsync = promisify(server.bindAsync.bind(server));
  const tryShutdown = promisify(server.tryShutdown.bind(server));
  const service = new ProjectServiceServer(handlers, clientFactory);
  registerProjectService(server, service);
  await bindAsync(`0.0.0.0:${port}`, grpc.ServerCredentials.createInsecure());
  logger.info(`Project server listening on port ${port}`);

  await new Promise<void>((resolve, reject) => {
    process.once('SIGINT', () => {
      void tryShutdown().then(resolve, reject);
    });
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
