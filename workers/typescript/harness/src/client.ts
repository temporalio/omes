import * as fs from 'node:fs';
import { Client, TLSConfig } from '@temporalio/client';
import { NativeConnection, Runtime, RuntimeOptions } from '@temporalio/worker';

export interface ClientConfig {
  targetHost: string;
  namespace: string;
  apiKey?: string;
  tls?: TLSConfig;
  runtimeOptions: RuntimeOptions;
}

export interface BuildClientConfigOptions {
  server_address: string;
  namespace: string;
  auth_header: string;
  tls: boolean;
  tls_cert_path: string;
  tls_key_path: string;
  tls_server_name?: string;
  disable_host_verification?: boolean;
  prom_listen_address?: string;
}

export type ClientFactory = (config: ClientConfig) => Promise<Client>;

export async function defaultClientFactory(config: ClientConfig): Promise<Client> {
  Runtime.install(config.runtimeOptions);
  // Use native connection backed client. This client is also used for the worker(s).
  const connection = await NativeConnection.connect({
    address: config.targetHost,
    apiKey: config.apiKey,
    tls: config.tls,
  });

  return new Client({
    connection,
    namespace: config.namespace,
  });
}

export function buildClientConfig(options: BuildClientConfigOptions): ClientConfig {
  return {
    targetHost: options.server_address,
    namespace: options.namespace,
    apiKey: buildApiKey(options.auth_header),
    tls: buildTlsConfig(options),
    runtimeOptions: buildRuntimeOptions(options.prom_listen_address),
  };
}

function buildApiKey(authHeader: string): string | undefined {
  if (!authHeader) {
    return undefined;
  }
  return authHeader.startsWith('Bearer ') ? authHeader.slice('Bearer '.length) : authHeader;
}

function buildTlsConfig(options: BuildClientConfigOptions): TLSConfig | undefined {
  if (options.disable_host_verification) {
    process.emitWarning(
      'disable_host_verification is not supported by the TypeScript SDK; ignoring',
    );
  }

  if (options.tls_cert_path) {
    if (!options.tls_key_path) {
      throw new Error('Client cert specified, but not client key!');
    }
    return {
      clientCertPair: {
        crt: fs.readFileSync(options.tls_cert_path),
        key: fs.readFileSync(options.tls_key_path),
      },
      serverNameOverride: options.tls_server_name,
    };
  }

  if (options.tls_key_path) {
    throw new Error('Client key specified, but not client cert!');
  }

  if (!options.tls) {
    return undefined;
  }

  return options.tls_server_name ? { serverNameOverride: options.tls_server_name } : {};
}

function buildRuntimeOptions(promListenAddress: string | undefined): RuntimeOptions {
  const coreLogLevelEnv = process.env.TEMPORAL_CORE_LOG_LEVEL?.toUpperCase();
  const coreLogLevel =
    coreLogLevelEnv === 'TRACE' ||
    coreLogLevelEnv === 'DEBUG' ||
    coreLogLevelEnv === 'INFO' ||
    coreLogLevelEnv === 'WARN' ||
    coreLogLevelEnv === 'ERROR'
      ? coreLogLevelEnv
      : 'INFO';
  const telemetryOptions: RuntimeOptions['telemetryOptions'] = {};
  telemetryOptions.logging = {
    filter: {
      core: coreLogLevel,
      other: 'WARN',
    },
  };

  if (promListenAddress) {
    telemetryOptions.metrics = {
      prometheus: {
        bindAddress: promListenAddress,
        useSecondsForDurations: true,
      },
    };
  }

  return {
    telemetryOptions,
  };
}
