import { Client } from '@temporalio/client';
import type { Worker } from '@temporalio/worker';

/**
 * Config passed to client's execute function.
 */
export interface ClientConfig {
    /** Pre-created Temporal client connection */
    client: Client;
    /** Task queue for workflows */
    taskQueue: string;
    /** Unique ID for this load test run (from /execute request) */
    runId: string;
    /** Current iteration number (from /execute request) */
    iteration: number;
}

/**
 * Config passed to worker's configure function.
 */
export interface WorkerConfig {
    /** Pre-created Temporal client connection */
    client: Client;
    /** Task queue for the worker */
    taskQueue: string;
    /** Optional Prometheus metrics endpoint address */
    promListenAddress?: string;
}

// Backwards compatibility aliases
export type ExecuteContext = ClientConfig;
export type WorkerContext = WorkerConfig;

export type ClientFunction = (config: ClientConfig) => Promise<void>;
export type WorkerFunction = (config: WorkerConfig) => Promise<Worker>;

// Backwards compatibility function type aliases
export type ExecuteFunction = ClientFunction;
export type ConfigureWorkerFunction = WorkerFunction;
