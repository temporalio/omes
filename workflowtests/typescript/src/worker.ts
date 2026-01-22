import { Connection, Client, ClientOptions } from '@temporalio/client';
import { parseArgs } from 'node:util';
import { WorkerConfig, WorkerFunction } from './common';

export class OmesWorkerStarter {
    private clientOptions: Partial<ClientOptions>;
    private configureFn?: WorkerFunction;

    constructor(clientOptions: Partial<ClientOptions> = {}) {
        this.clientOptions = clientOptions;
    }

    configureWorker(fn: WorkerFunction): void {
        this.configureFn = fn;
    }

    async run(): Promise<void> {
        const { values } = parseArgs({
            options: {
                'task-queue': { type: 'string' },
                'server-address': { type: 'string', default: 'localhost:7233' },
                namespace: { type: 'string', default: 'default' },
                'prom-listen-address': { type: 'string' },
            },
        });

        if (!values['task-queue']) {
            throw new Error('--task-queue is required');
        }

        const connection = await Connection.connect({
            address: values['server-address'],
        });

        const client = new Client({
            connection,
            namespace: values.namespace,
            ...this.clientOptions,
        });

        const config: WorkerConfig = {
            client,
            taskQueue: values['task-queue'],
            promListenAddress: values['prom-listen-address'],
        };

        const worker = await this.configureFn!(config);
        console.log(`Worker started on task queue: ${values['task-queue']}`);
        await worker.run();
    }
}

/**
 * Convenience function to run a worker with the given configuration handler.
 *
 * Note: Prefer using run() from cli.ts with the new subcommand pattern.
 *
 * @example
 * // src/worker.ts - user exports function
 * export async function workerMain(config: WorkerConfig): Promise<Worker> {
 *     return Worker.create({
 *         connection: config.client.connection,
 *         taskQueue: config.taskQueue,
 *         workflowsPath: require.resolve('./workflows'),
 *     });
 * }
 *
 * // Call directly:
 * runWorker(workerMain);
 *
 * @param handler - Async function that returns a configured Worker
 * @param clientOptions - Options passed to Client constructor
 */
export function runWorker(
    handler: WorkerFunction,
    clientOptions: Partial<ClientOptions> = {}
): void {
    const starter = new OmesWorkerStarter(clientOptions);
    starter.configureWorker(handler);
    starter.run().catch((err) => {
        console.error(err);
        process.exit(1);
    });
}
