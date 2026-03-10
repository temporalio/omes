import type { ConnectionOptions } from '@temporalio/client';
import type { NativeConnectionOptions, Worker } from '@temporalio/worker';

/**
 * Config passed to client execute function.
 */
export interface ClientConfig {
    /** Temporal namespace */
    namespace: string;
    /** Native SDK connection options for Connection.connect(...) */
    connectionOptions: ConnectionOptions;
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
    /** Temporal namespace */
    namespace: string;
    /** Native SDK connection options for NativeConnection.connect(...) */
    connectionOptions: NativeConnectionOptions;
    /** Task queue for the worker */
    taskQueue: string;
    /** Optional Prometheus metrics endpoint address */
    promListenAddress?: string;
}

export type ClientFunction = (config: ClientConfig) => Promise<void>;
export type WorkerFunction = (config: WorkerConfig) => Promise<Worker>;
