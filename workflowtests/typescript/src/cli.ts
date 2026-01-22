/**
 * CLI dispatch for omes-starter.
 *
 * This module provides the `run()` function that dispatches to either
 * client or worker mode based on the first command-line argument.
 */

import { parseArgs } from 'node:util';
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
    const args = process.argv.slice(2);

    if (args.length === 0) {
        console.error('Usage: node main.js <client|worker> [options]');
        console.error('  client  - Run as HTTP client starter');
        console.error('  worker  - Run as Temporal worker');
        process.exit(1);
    }

    const mode = args[0];

    if (mode === 'client') {
        runClientMode(options.client, args.slice(1));
    } else if (mode === 'worker') {
        runWorkerMode(options.worker, args.slice(1));
    } else {
        console.error(`Unknown mode: ${mode}. Expected 'client' or 'worker'.`);
        process.exit(1);
    }
}

function runClientMode(handler: ClientFunction, argv: string[]): void {
    const { values } = parseArgs({
        args: argv,
        options: {
            port: { type: 'string', default: '8080' },
            'task-queue': { type: 'string' },
            'server-address': { type: 'string', default: 'localhost:7233' },
            namespace: { type: 'string', default: 'default' },
        },
    });

    if (!values['task-queue']) {
        console.error('Error: --task-queue is required');
        process.exit(1);
    }

    const starter = new OmesClientStarter();
    starter.onExecute(handler);
    starter._runWithArgs({
        port: parseInt(values.port!, 10),
        taskQueue: values['task-queue'],
        serverAddress: values['server-address']!,
        namespace: values.namespace!,
    }).catch((err) => {
        console.error(err);
        process.exit(1);
    });
}

async function runWorkerMode(
    handler: WorkerFunction,
    argv: string[]
): Promise<void> {
    const { values } = parseArgs({
        args: argv,
        options: {
            'task-queue': { type: 'string' },
            'server-address': { type: 'string', default: 'localhost:7233' },
            namespace: { type: 'string', default: 'default' },
            'prom-listen-address': { type: 'string' },
        },
    });

    if (!values['task-queue']) {
        console.error('Error: --task-queue is required');
        process.exit(1);
    }

    try {
        const connection = await Connection.connect({
            address: values['server-address'],
        });

        const client = new Client({
            connection,
            namespace: values.namespace,
        });

        const config: WorkerConfig = {
            client,
            taskQueue: values['task-queue'],
            promListenAddress: values['prom-listen-address'],
        };

        const worker = await handler(config);
        console.log(`Worker started on task queue: ${values['task-queue']}`);
        await worker.run();
    } catch (err) {
        console.error(err);
        process.exit(1);
    }
}
