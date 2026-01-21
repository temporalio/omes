import { Client } from '@temporalio/client';
import type { Worker } from '@temporalio/worker';

export interface ExecuteContext {
    iteration: number;
    runId: string;
    taskQueue: string;
    client: Client;
}

export interface WorkerContext {
    taskQueue: string;
    client: Client;
    promListenAddress?: string;
}

export type ExecuteFunction = (ctx: ExecuteContext) => Promise<void>;
export type ConfigureWorkerFunction = (ctx: WorkerContext) => Promise<Worker>;
