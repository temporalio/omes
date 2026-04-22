import { DefaultLogger, Logger, LogLevel } from '@temporalio/worker';
import winston from 'winston';

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
