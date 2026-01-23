/**
 * CLI dispatch for omes-starter.
 *
 * This module provides the `run()` function that dispatches to either
 * client or worker mode based on the first command-line argument.
 */

import { Command } from 'commander';
import { Connection, Client } from '@temporalio/client';
import type { ClientFunction, WorkerFunction, WorkerConfig } from './common';
import { OmesClientStarter } from './client';

export interface RunOptions {
    /** Async function called for each /execute request in client mode */
    client: ClientFunction;
    /** Async function that returns a configured Worker in worker mode */
    worker: WorkerFunction;
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
        .action((opts) => {
            const starter = new OmesClientStarter();
            starter.onExecute(options.client);
            starter._runWithArgs({
                port: parseInt(opts.port, 10),
                taskQueue: opts.taskQueue,
                serverAddress: opts.serverAddress,
                namespace: opts.namespace,
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
        .option('--prom-listen-address <addr>', 'Prometheus metrics address')
        .action(async (opts) => {
            try {
                const connection = await Connection.connect({
                    address: opts.serverAddress,
                });

                const client = new Client({
                    connection,
                    namespace: opts.namespace,
                });

                const config: WorkerConfig = {
                    client,
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

    program.parse();
}
