/**
 * CLI dispatch for omes-starter.
 *
 * This module provides the `run()` function that dispatches to either
 * client or worker mode based on the first command-line argument.
 */

import { Command } from 'commander';
import { Runtime } from '@temporalio/worker';
import type { ClientFunction, WorkerFunction, WorkerConfig } from './common';
import { OmesClientStarter } from './client';

export interface RunOptions {
    /** Async function called for each /execute request in client mode */
    client: ClientFunction;
    /** Async function that returns a configured Worker in worker mode */
    worker: WorkerFunction;
}

interface ClientCommandOptions {
    port: string;
    taskQueue: string;
    serverAddress: string;
    namespace: string;
    authHeader?: string;
    tls?: boolean;
    tlsServerName?: string;
}

interface WorkerCommandOptions {
    taskQueue: string;
    serverAddress: string;
    namespace: string;
    authHeader?: string;
    tls?: boolean;
    tlsServerName?: string;
    promListenAddress?: string;
}

/**
 * Main entry point - dispatches to client or worker based on first arg.
 *
 * User writes a main.ts that calls this function:
 *
 * ```typescript
 * import { run } from '@temporalio/omes-starter';
 * import { clientMain } from './src/client';
 * import { workerMain } from './src/worker';
 *
 * run({ client: clientMain, worker: workerMain });
 * ```
 *
 * The program is then invoked with a subcommand:
 * ```
 * node main.js client --port 8080 --server-address localhost:7233 ...
 * node main.js worker --task-queue my-queue --server-address localhost:7233 ...
 * ```
 */
export function run(options: RunOptions): void {
    const program = new Command();

    program
        .command('client')
        .description('Run as HTTP client starter')
        .option('--port <port>', 'HTTP port', '8080')
        .requiredOption('--task-queue <queue>', 'Task queue name')
        .option('--server-address <addr>', 'Temporal server address', 'localhost:7233')
        .option('--namespace <ns>', 'Temporal namespace', 'default')
        .option('--auth-header <header>', 'Authorization header value')
        .option('--tls', 'Enable TLS')
        .option('--tls-server-name <name>', 'TLS SNI override')
        .action((opts: ClientCommandOptions) => {
            const starter = new OmesClientStarter();
            starter.onExecute(options.client);
            starter._runWithArgs({
                port: parseInt(opts.port, 10),
                taskQueue: opts.taskQueue,
                serverAddress: opts.serverAddress,
                namespace: opts.namespace,
                authHeader: opts.authHeader,
                tls: opts.tls,
                tlsServerName: opts.tlsServerName,
            }).catch((err) => {
                console.error(err);
                process.exit(1);
            });
        });

    program
        .command('worker')
        .description('Run as Temporal worker')
        .requiredOption('--task-queue <queue>', 'Task queue name')
        .option('--server-address <addr>', 'Temporal server address', 'localhost:7233')
        .option('--namespace <ns>', 'Temporal namespace', 'default')
        .option('--auth-header <header>', 'Authorization header value')
        .option('--tls', 'Enable TLS')
        .option('--tls-server-name <name>', 'TLS SNI override')
        .option('--prom-listen-address <addr>', 'Prometheus metrics address')
        .action(async (opts: WorkerCommandOptions) => {
            try {
                if (opts.promListenAddress) {
                    Runtime.install({
                        telemetryOptions: {
                            metrics: {
                                prometheus: {
                                    bindAddress: opts.promListenAddress,
                                    useSecondsForDurations: true,
                                },
                            },
                        },
                    });
                }

                const connectionOptions: WorkerConfig['connectionOptions'] = {
                    address: opts.serverAddress,
                };
                if (opts.authHeader) {
                    connectionOptions.metadata = { Authorization: opts.authHeader };
                }
                if (opts.tlsServerName) {
                    connectionOptions.tls = { serverNameOverride: opts.tlsServerName };
                } else if (opts.tls) {
                    connectionOptions.tls = {};
                }

                const config: WorkerConfig = {
                    namespace: opts.namespace,
                    connectionOptions,
                    taskQueue: opts.taskQueue,
                    promListenAddress: opts.promListenAddress,
                };

                const worker = await options.worker(config);
                console.log(`Worker started on task queue: ${opts.taskQueue}`);
                await worker.run();
            } catch (err) {
                console.error(err);
                process.exit(1);
            }
        });

    program.parse(process.argv);
}
