import { Client, Connection } from '@temporalio/client';
import type { ClientConfig } from '@temporalio/omes-starter';

// Optional ClientPool usage:
// import { ClientPool } from '@temporalio/omes-starter';
// const pool = new ClientPool();
//
// export async function clientMain(config: ClientConfig): Promise<void> {
//     const connection = await pool.getOrConnect('default', config.connectionOptions);
//     const client = new Client({
//         connection,
//         namespace: config.namespace,
//     });
//     ...
// }

/**
 * Called for each iteration - start a workflow and wait for result.
 */
export async function clientMain(config: ClientConfig): Promise<void> {
    const connection = await Connection.connect(config.connectionOptions);
    try {
        const client = new Client({
            connection,
            namespace: config.namespace,
        });
        const handle = await client.workflow.start('SimpleWorkflow', {
            workflowId: `wf-${config.runId}-${config.iteration}`,
            taskQueue: config.taskQueue,
        });
        const result = await handle.result();
        console.log(
            `Workflow result (runId=${config.runId}, iteration=${config.iteration}): ${String(result)}`
        );
    } finally {
        await connection.close();
    }
}
