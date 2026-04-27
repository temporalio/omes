import { DefaultLogger, Logger, LogLevel } from '@temporalio/worker';
import winston from 'winston';
import { type ProjectServiceServer } from '../src/project';
import * as grpc from '@grpc/grpc-js';
import { projectServiceDefinition } from './grpc-helpers';

const temporalLogLevels: Record<string, LogLevel> = {
  trace: 'TRACE',
  debug: 'DEBUG',
  info: 'INFO',
  warn: 'WARN',
  error: 'ERROR',
  fatal: 'ERROR',
};

const winstonLogLevels: Record<string, string> = {
  trace: 'debug',
  debug: 'debug',
  info: 'info',
  warn: 'warn',
  error: 'error',
  fatal: 'error',
};

export function configureLogger(logLevel: string, logEncoding: string): Logger {
  const normalized = logLevel.toLowerCase();
  const temporalLevel = temporalLogLevels[normalized] ?? 'INFO';
  const winstonLevel = winstonLogLevels[normalized] ?? 'info';

  const winstonLogger = winston.createLogger({
    level: winstonLevel,
    format:
      logEncoding === 'json'
        ? winston.format.json()
        : winston.format.combine(
            winston.format.colorize(),
            winston.format.simple(),
            winston.format.metadata(),
          ),
    transports: [new winston.transports.Console()],
  });

  return new DefaultLogger(temporalLevel, (entry) => {
    winstonLogger.log({
      level: winstonLogLevels[entry.level.toLowerCase()] ?? 'info',
      message: entry.message,
      timestamp: Number(entry.timestampNanos / 1_000_000n),
      meta: entry.meta,
    });
  });
}

export function unaryHandler<Request, Response>(
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
  server.addService(projectServiceDefinition, {
    Init: unaryHandler((request) => service.Init(request)),
    Execute: unaryHandler((request) => service.Execute(request)),
  });
}

export function grpcError(code: grpc.status, details: string): grpc.ServiceError {
  return Object.assign(new Error(details), {
    code,
    details,
    metadata: new grpc.Metadata(),
    name: 'Error',
  });
}
