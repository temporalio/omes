import { temporal } from './protos/root';
import { isMainThread, Worker } from 'node:worker_threads';
import { activityInfo } from '@temporalio/activity';
import { Client, ScheduleBackfill, ScheduleOptionsAction, ScheduleOptionsSpec } from '@temporalio/client';
import { ClientActionExecutor } from './client-action-executor';
import IResourcesActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity;
import IClientActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity;
import ICreateScheduleAction = temporal.omes.kitchen_sink.ICreateScheduleAction;
import IDescribeScheduleAction = temporal.omes.kitchen_sink.IDescribeScheduleAction;
import IUpdateScheduleAction = temporal.omes.kitchen_sink.IUpdateScheduleAction;
import IDeleteScheduleAction = temporal.omes.kitchen_sink.IDeleteScheduleAction;

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

function makeScheduleIDUnique(baseScheduleID: string, workflowExecutionID: string): string {
  const sanitizedWorkflowID = workflowExecutionID.replace(/\//g, '-');
  return `${baseScheduleID}-${sanitizedWorkflowID}`;
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

  async CreateScheduleActivity(action: ICreateScheduleAction): Promise<void> {
    const activityContext = activityInfo();
    const info = activityContext.workflowExecution;

    if (!action.scheduleId) {
      throw new Error('scheduleId is required');
    }
    if (!action.action) {
      throw new Error('action is required');
    }

    const taskQueue = action.action.taskQueue || activityContext.taskQueue;
    const uniqueScheduleId = makeScheduleIDUnique(action.scheduleId, info.workflowId);

    let uniqueWorkflowId = action.action.workflowId || '';
    if (uniqueWorkflowId) {
      uniqueWorkflowId = makeScheduleIDUnique(uniqueWorkflowId, info.workflowId);
    }

    const scheduleAction: ScheduleOptionsAction = {
      type: 'startWorkflow',
      workflowType: action.action.workflowType || 'kitchenSink',
      taskQueue: taskQueue,
      workflowId: uniqueWorkflowId,
      args: action.action.input || [],
      workflowRunTimeout: action.action.workflowExecutionTimeout?.seconds
        ? `${action.action.workflowExecutionTimeout.seconds}s`
        : undefined,
      workflowTaskTimeout: action.action.workflowTaskTimeout?.seconds
        ? `${action.action.workflowTaskTimeout.seconds}s`
        : undefined,
      retry: action.action.retryPolicy ? {
        initialInterval: action.action.retryPolicy.initialInterval?.seconds
          ? `${action.action.retryPolicy.initialInterval.seconds}s`
          : undefined,
        maximumInterval: action.action.retryPolicy.maximumInterval?.seconds
          ? `${action.action.retryPolicy.maximumInterval.seconds}s`
          : undefined,
        backoffCoefficient: action.action.retryPolicy.backoffCoefficient,
        maximumAttempts: action.action.retryPolicy.maximumAttempts,
        nonRetryableErrorTypes: action.action.retryPolicy.nonRetryableErrorTypes || undefined,
      } : undefined,
    };

    const spec: ScheduleOptionsSpec = {};
    if (action.spec) {
      spec.cronExpressions = action.spec.cronExpressions || [];
      spec.jitter = action.spec.jitter?.seconds ? `${action.spec.jitter.seconds}s` : undefined;
    }

    const scheduleOptions: any = {
      scheduleId: uniqueScheduleId,
      action: scheduleAction,
      spec: spec,
    };

    if (action.policies) {
      scheduleOptions.policies = {
        catchupWindow: action.policies.catchupWindow?.seconds
          ? `${action.policies.catchupWindow.seconds}s`
          : undefined,
      };
      if (action.policies.remainingActions) {
        scheduleOptions.remainingActions = Number(action.policies.remainingActions);
      }
      if (action.policies.triggerImmediately) {
        scheduleOptions.triggerImmediately = action.policies.triggerImmediately;
      }
    }

    if (action.backfill && action.backfill.length > 0) {
      const backfills: ScheduleBackfill[] = action.backfill.map(bf => ({
        start: new Date(Number(bf.startTimestamp) * 1000),
        end: new Date(Number(bf.endTimestamp) * 1000),
      }));
      scheduleOptions.backfill = backfills;
    }

    await client.schedule.create(scheduleOptions);
  },

  async DescribeScheduleActivity(action: IDescribeScheduleAction): Promise<any> {
    const activityContext = activityInfo();
    const info = activityContext.workflowExecution;

    if (!action.scheduleId) {
      throw new Error('scheduleId is required');
    }

    const uniqueScheduleId = makeScheduleIDUnique(action.scheduleId, info.workflowId);
    const handle = client.schedule.getHandle(uniqueScheduleId);
    return await handle.describe();
  },

  async UpdateScheduleActivity(action: IUpdateScheduleAction): Promise<void> {
    const activityContext = activityInfo();
    const info = activityContext.workflowExecution;

    if (!action.scheduleId) {
      throw new Error('scheduleId is required');
    }

    const uniqueScheduleId = makeScheduleIDUnique(action.scheduleId, info.workflowId);
    const handle = client.schedule.getHandle(uniqueScheduleId);

    await handle.update((schedule) => {
      if (action.spec) {
        schedule.spec.cronExpressions = action.spec.cronExpressions || [];
        if (action.spec.jitter?.seconds) {
          schedule.spec.jitter = `${action.spec.jitter.seconds}s`;
        }
      }
      return schedule;
    });
  },

  async DeleteScheduleActivity(action: IDeleteScheduleAction): Promise<void> {
    const activityContext = activityInfo();
    const info = activityContext.workflowExecution;

    if (!action.scheduleId) {
      throw new Error('scheduleId is required');
    }

    const uniqueScheduleId = makeScheduleIDUnique(action.scheduleId, info.workflowId);
    const handle = client.schedule.getHandle(uniqueScheduleId);
    await handle.delete();
  },
});
