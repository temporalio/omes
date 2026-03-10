import { NativeConnection, Worker, bundleWorkflowCode } from '@temporalio/worker';
import type { WorkerConfig } from '@temporalio/omes-starter';

/**
 * Configure and return the worker.
 */
export async function workerMain(config: WorkerConfig): Promise<Worker> {
    const workflowsPath = require.resolve('./workflows');
    const workflowBundle = await bundleWorkflowCode({ workflowsPath });

    const nativeConnection = await NativeConnection.connect(config.connectionOptions);

    return Worker.create({
        connection: nativeConnection,
        namespace: config.namespace,
        taskQueue: config.taskQueue,
        workflowBundle,
    });
}
