import { Connection, Client, ClientOptions } from '@temporalio/client';
import { parseArgs } from 'node:util';
import { WorkerContext, ConfigureWorkerFunction } from './common';

export class OmesWorkerStarter {
    private clientOptions: Partial<ClientOptions>;
    private configureFn?: ConfigureWorkerFunction;

    constructor(clientOptions: Partial<ClientOptions> = {}) {
        this.clientOptions = clientOptions;
    }

    configureWorker(fn: ConfigureWorkerFunction): void {
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

        const ctx: WorkerContext = {
            taskQueue: values['task-queue'],
            client,
            promListenAddress: values['prom-listen-address'],
        };

        const worker = await this.configureFn!(ctx);
        console.log(`Worker started on task queue: ${values['task-queue']}`);
        await worker.run();
    }
}
