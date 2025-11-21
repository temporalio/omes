import { temporal } from './protos/root';
import { isMainThread, Worker } from 'node:worker_threads';
import { activityInfo, heartbeat, sleep } from '@temporalio/activity';
import { Client } from '@temporalio/client';
import { ClientActionExecutor } from './client-action-executor';
import { ApplicationFailure } from '@temporalio/common';
import IResourcesActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity;
import IClientActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity;
import IRetryableErrorActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IRetryableErrorActivity;
import ITimeoutActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.ITimeoutActivity;
import IHeartbeatTimeoutActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IHeartbeatTimeoutActivity;
import { durationConvert } from './proto_help';

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

export async function retryableError(config: IRetryableErrorActivity): Promise<void> {
  const info = activityInfo();
  if (info.attempt <= (config.failAttempts || 0)) {
    throw ApplicationFailure.retryable('retryable error', 'RetryableError');
  }
}

export async function timeout(config: ITimeoutActivity): Promise<void> {
  const info = activityInfo();
  let duration = config.successDuration;
  if (info.attempt <= config.failAttempts!) {
    // Failure case: run failure duration (exceeds activity timeout)
    duration = config.failureDuration;
  }

  // Sleep for failure/success timeout duration.
  // In failure case, this will throw a cancellation error.
  await sleep(durationConvert(duration));
}

export async function heartbeatActivity(config: IHeartbeatTimeoutActivity): Promise<void> {
  const info = activityInfo();
  const shouldSendHeartbeats = info.attempt > (config.failAttempts || 0);
  let duration = config.successDuration;
  if (!shouldSendHeartbeats) {
    // Failure case: run failure duration (exceeds heartbeat timeout)
    duration = config.failureDuration;
  }
  // Sleep for failure/success timeout duration.
  // In failure case, this will throw a cancellation error.
  await sleep(durationConvert(duration));
  // On success, heartbeat
  heartbeat();
}

export const createActivities = (client: Client, errOnUnimplemented = false) => ({
  noop,
  resources,
  payload,
  retryable_error: retryableError,
  timeout,
  heartbeat: heartbeatActivity,
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  delay: require('@temporalio/activity').sleep,

  async client(clientActivity: IClientActivity): Promise<void> {
    const activityContext = activityInfo();
    const workflowId = activityContext.workflowExecution.workflowId;
    const taskQueue = activityContext.taskQueue;

    const executor = new ClientActionExecutor(client, workflowId, taskQueue, errOnUnimplemented);
    try {
      await executor.executeClientSequence(clientActivity.clientSequence);
    } catch (error) {
      console.error('Client activity failed:', error);
      throw error;
    }
  },
});
