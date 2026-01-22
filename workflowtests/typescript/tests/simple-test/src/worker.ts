import { NativeConnection, Worker, bundleWorkflowCode } from '@temporalio/worker';
import type { WorkerConfig } from '@temporalio/omes-starter';

/**
 * Configure and return the worker.
 */
export async function workerMain(config: WorkerConfig): Promise<Worker> {
    const workflowsPath = require.resolve('./workflows');
    const workflowBundle = await bundleWorkflowCode({ workflowsPath });

    // Create native connection for the worker using the same address as the client
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const address = (config.client.connection as any).options?.address;
    const nativeConnection = await NativeConnection.connect({ address });

    return Worker.create({
        connection: nativeConnection,
        namespace: config.client.options.namespace,
        taskQueue: config.taskQueue,
        workflowBundle,
    });
}
