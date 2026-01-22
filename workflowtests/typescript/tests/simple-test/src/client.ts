import type { ClientConfig } from '@temporalio/omes-starter';

/**
 * Called for each iteration - start a workflow and wait for result.
 */
export async function clientMain(config: ClientConfig): Promise<void> {
    const handle = await config.client.workflow.start('SimpleWorkflow', {
        workflowId: `wf-${config.runId}-${config.iteration}`,
        taskQueue: config.taskQueue,
    });
    await handle.result();
}
