import { temporal } from './protos/root';
import { isMainThread, Worker } from 'node:worker_threads';
import { activityInfo } from '@temporalio/activity';
import { Client, Backfill, ScheduleOptionsAction, ScheduleSpec } from '@temporalio/client';
import { ClientActionExecutor } from './client-action-executor';
import { payloadConverter } from './payload-converter';
import IResourcesActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity;
import IClientActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity;
import ICreateScheduleAction = temporal.omes.kitchen_sink.ICreateScheduleAction;
import IDescribeScheduleAction = temporal.omes.kitchen_sink.IDescribeScheduleAction;
import IUpdateScheduleAction = temporal.omes.kitchen_sink.IUpdateScheduleAction;
import IDeleteScheduleAction = temporal.omes.kitchen_sink.IDeleteScheduleAction;
import Payload = temporal.api.common.v1.Payload;

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

    const args: any[] = [];
    if (action.action.input && action.action.input.length > 0) {
      for (const payloadProto of action.action.input) {
        const payload = Payload.create(payloadProto);
        const decoded = payloadConverter.fromPayload(payload);
        args.push(decoded);
      }
    }

    const scheduleAction: ScheduleOptionsAction = {
      type: 'startWorkflow',
      workflowType: action.action.workflowType || 'kitchenSink',
      taskQueue,
      workflowId: uniqueWorkflowId,
      args,
      workflowRunTimeout: action.action.workflowExecutionTimeout?.seconds
        ? Number(action.action.workflowExecutionTimeout.seconds) * 1000
        : undefined,
      workflowTaskTimeout: action.action.workflowTaskTimeout?.seconds
        ? Number(action.action.workflowTaskTimeout.seconds) * 1000
        : undefined,
      retry: action.action.retryPolicy
        ? {
            initialInterval: action.action.retryPolicy.initialInterval?.seconds
              ? Number(action.action.retryPolicy.initialInterval.seconds) * 1000
              : undefined,
            maximumInterval: action.action.retryPolicy.maximumInterval?.seconds
              ? Number(action.action.retryPolicy.maximumInterval.seconds) * 1000
              : undefined,
            backoffCoefficient: action.action.retryPolicy.backoffCoefficient ?? undefined,
            maximumAttempts: action.action.retryPolicy.maximumAttempts ?? undefined,
            nonRetryableErrorTypes: action.action.retryPolicy.nonRetryableErrorTypes || undefined,
          }
        : undefined,
    };

    const spec: ScheduleSpec = {};
    if (action.spec) {
      spec.cronExpressions = action.spec.cronExpressions || [];
      spec.jitter = action.spec.jitter?.seconds
        ? Number(action.spec.jitter.seconds) * 1000
        : undefined;
    }

    const scheduleOptions: any = {
      scheduleId: uniqueScheduleId,
      action: scheduleAction,
      spec,
    };

    if (action.policies) {
      scheduleOptions.policies = {
        catchupWindow: action.policies.catchupWindow?.seconds
          ? `${action.policies.catchupWindow.seconds}s`
          : undefined,
      };
    }

    const stateOptions: any = {};
    if (action.policies?.remainingActions) {
      stateOptions.remainingActions = Number(action.policies.remainingActions);
    }
    if (action.policies?.triggerImmediately) {
      stateOptions.triggerImmediately = action.policies.triggerImmediately;
    }
    if (action.backfill && action.backfill.length > 0) {
      const backfills: Backfill[] = action.backfill.map((bf) => ({
        start: new Date(Number(bf.startTimestamp) * 1000),
        end: new Date(Number(bf.endTimestamp) * 1000),
      }));
      stateOptions.backfill = backfills;
    }
    if (Object.keys(stateOptions).length > 0) {
      scheduleOptions.state = stateOptions;
    }

    await client.schedule.create(scheduleOptions);
  },

  async DescribeScheduleActivity(action: IDescribeScheduleAction): Promise<void> {
    const activityContext = activityInfo();
    const info = activityContext.workflowExecution;

    if (!action.scheduleId) {
      throw new Error('scheduleId is required');
    }

    const uniqueScheduleId = makeScheduleIDUnique(action.scheduleId, info.workflowId);
    const handle = client.schedule.getHandle(uniqueScheduleId);
    await handle.describe();
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
        const newSpec: ScheduleSpec = {
          ...schedule.spec,
          cronExpressions: action.spec.cronExpressions || [],
          jitter: action.spec.jitter?.seconds
            ? Number(action.spec.jitter.seconds) * 1000
            : undefined,
        };
        schedule.spec = newSpec as any;
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
