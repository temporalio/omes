import { temporal } from './protos/root';
import { isMainThread, Worker } from 'node:worker_threads';
import { activityInfo } from '@temporalio/activity';
import { Client } from '@temporalio/client';
import { ClientActionExecutor } from './client-action-executor';
import IResourcesActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity;
import IClientActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity;

export { sleep as delay } from '@temporalio/activity';

export async function noop() {
  return undefined;
}

export async function resources(input: IResourcesActivity) {
  if (isMainThread) {
    return new Promise<void>((resolve, reject) => {
      const worker = new Worker(__dirname + '/resources_activity_thread.js', { workerData: input });
      worker.on('message', (message) => console.log('Worker got message', message));
      worker.on('error', (err) => console.error('Worker error:', err));
      worker.on('exit', (code) => {
        if (code !== 0) {
          reject(new Error(`Worker stopped with exit code ${code}`));
        } else {
          resolve();
        }
      });
    });
  }
}

export async function payload(inputData: Uint8Array, bytesToReturn: number): Promise<Uint8Array> {
  const output = new Uint8Array(bytesToReturn);
  for (let i = 0; i < bytesToReturn; i++) {
    output[i] = Math.floor(Math.random() * 256);
  }
  return output;
}

export const createActivities = (client: Client) => ({
  noop,
  resources,
  payload,
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  delay: require('@temporalio/activity').sleep,

  async client(clientActivity: IClientActivity): Promise<void> {
    const activityContext = activityInfo();
    const workflowId = activityContext.workflowExecution.workflowId;
    const taskQueue = activityContext.taskQueue;

    const executor = new ClientActionExecutor(client, workflowId, taskQueue);
    try {
      await executor.executeClientSequence(clientActivity.clientSequence);
    } catch (error) {
      console.error('Client activity failed:', error);
      throw error;
    }
  },
});
