import { Client } from '@temporalio/client';
import { ClientPool } from '@temporalio/omes-starter';
import type { ClientConfig } from '@temporalio/omes-starter';

const pool = new ClientPool();

/**
 * Called for each iteration - start a workflow and wait for result.
 */
export async function clientMain(config: ClientConfig): Promise<void> {
    const connection = await pool.getOrConnect('default', config.connectionOptions);
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
}

// Alternative: create a fresh connection per iteration using the SDK directly.
// This properly cleans up via connection.close(), but the pool above avoids
// redundant connection setup under load.
//
// import { Connection } from '@temporalio/client';
//
// export async function clientMain(config: ClientConfig): Promise<void> {
//     const connection = await Connection.connect(config.connectionOptions);
//     try {
//         const client = new Client({
//             connection,
//             namespace: config.namespace,
//         });
//         const handle = await client.workflow.start('SimpleWorkflow', {
//             workflowId: `wf-${config.runId}-${config.iteration}`,
//             taskQueue: config.taskQueue,
//         });
//         const result = await handle.result();
//         console.log(
//             `Workflow result (runId=${config.runId}, iteration=${config.iteration}): ${String(result)}`
//         );
//     } finally {
//         await connection.close();
//     }
// }
