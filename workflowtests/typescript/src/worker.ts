import { Command } from 'commander';
import { Connection, Client, ClientOptions } from '@temporalio/client';
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
        const program = new Command();
        program
            .requiredOption('--task-queue <queue>', 'Task queue name')
            .option('--server-address <addr>', 'Temporal server address', 'localhost:7233')
            .option('--namespace <ns>', 'Temporal namespace', 'default')
            .option('--prom-listen-address <addr>', 'Prometheus metrics address')
            .parse();

        const opts = program.opts();

        const connection = await Connection.connect({
            address: opts.serverAddress,
        });

        const client = new Client({
            connection,
            namespace: opts.namespace,
            ...this.clientOptions,
        });

        const config: WorkerConfig = {
            client,
            taskQueue: opts.taskQueue,
            promListenAddress: opts.promListenAddress,
        };

        const worker = await this.configureFn!(config);
        console.log(`Worker started on task queue: ${opts.taskQueue}`);
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
