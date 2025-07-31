import {temporal} from './protos/root';
import {isMainThread, Worker} from 'node:worker_threads';
// import { activityInfo } from '@temporalio/activity';
import {Client} from '@temporalio/client';
import IResourcesActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity;
import IClientActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IClientActivity;
import IClientSequence = temporal.omes.kitchen_sink.IClientSequence;
import IClientActionSet = temporal.omes.kitchen_sink.IClientActionSet;
import IClientAction = temporal.omes.kitchen_sink.IClientAction;
import IDoSignal = temporal.omes.kitchen_sink.IDoSignal;
import IDoUpdate = temporal.omes.kitchen_sink.IDoUpdate;
import IDoQuery = temporal.omes.kitchen_sink.IDoQuery;

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

export async function client(clientActivity: IClientActivity, context: any) {
  const client = context.context?.client as Client;

  const executor = new ClientActionsExecutor(client);
  await executor.executeClientSequence(clientActivity.clientSequence!);
}

class ClientActionsExecutor {
  private client: Client;
  private workflowId = '';
  private runId = '';
  private workflowType = 'kitchenSink';
  private workflowInput: any = null;

  constructor(client: Client) {
    this.client = client;
  }

  async executeClientSequence(clientSeq: IClientSequence): Promise<void> {
    for (const actionSet of clientSeq.actionSets || []) {
      await this.executeClientActionSet(actionSet);
    }
  }

  private async executeClientActionSet(actionSet: IClientActionSet): Promise<void> {
    if (actionSet.concurrent) {
      throw new Error('Concurrent client actions are not supported in TypeScript worker');
    }

    for (const action of actionSet.actions || []) {
      await this.executeClientAction(action);
    }

    if (actionSet.waitForCurrentRunToFinishAtEnd) {
      if (this.workflowId && this.runId) {
        const handle = this.client.workflow.getHandle(this.workflowId, this.runId);
        try {
          await handle.result();
        } catch (error) {
          // Ignore continue-as-new errors
        }
      }
    }
  }

  private async executeClientAction(action: IClientAction): Promise<void> {
    if (action.doSignal) {
      await this.executeSignalAction(action.doSignal);
    } else if (action.doUpdate) {
      await this.executeUpdateAction(action.doUpdate);
    } else if (action.doQuery) {
      await this.executeQueryAction(action.doQuery);
    } else if (action.nestedActions) {
      await this.executeClientActionSet(action.nestedActions);
    } else {
      throw new Error('Client action must have a recognized variant');
    }
  }

  private async executeSignalAction(signal: IDoSignal): Promise<void> {
    let signalName: string;
    let signalArgs: any;

    if (signal.doSignalActions) {
      signalName = 'do_actions_signal';
      signalArgs = signal.doSignalActions;
    } else if (signal.custom) {
      signalName = signal.custom.name || '';
      signalArgs = signal.custom.args || [];
    } else {
      throw new Error('DoSignal must have a recognizable variant');
    }

    if (signal.withStart) {
      const workflowId = this.workflowId || (globalThis as any).crypto.randomUUID();
      const handle = await this.client.workflow.start(this.workflowType, {
        workflowId,
        taskQueue: 'default',
        args: [this.workflowInput],
      });
      await handle.signal(signalName, signalArgs);
      this.workflowId = handle.workflowId;
      this.runId = handle.firstExecutionRunId;
    } else {
      const handle = this.client.workflow.getHandle(this.workflowId);
      await handle.signal(signalName, signalArgs);
    }
  }

  private async executeUpdateAction(update: IDoUpdate): Promise<void> {
    let updateName: string;
    let updateArgs: any;

    if (update.doActions) {
      updateName = 'do_actions_update';
      updateArgs = update.doActions;
    } else if (update.custom) {
      updateName = update.custom.name || '';
      updateArgs = update.custom.args || [];
    } else {
      throw new Error('DoUpdate must have a recognizable variant');
    }

    try {
      if (update.withStart) {
        const workflowId = this.workflowId || (globalThis as any).crypto.randomUUID();
        const handle = await this.client.workflow.start(this.workflowType, {
          workflowId,
          taskQueue: 'default',
          args: [this.workflowInput],
          workflowIdReusePolicy: 'ALLOW_DUPLICATE_FAILED_ONLY',
        });
        await handle.executeUpdate(updateName, updateArgs);
        this.workflowId = handle.workflowId;
        this.runId = handle.firstExecutionRunId;
      } else {
        const handle = this.client.workflow.getHandle(this.workflowId);
        await handle.executeUpdate(updateName, updateArgs);
      }
    } catch (error) {
      if (!update.failureExpected) {
        throw error;
      }
    }
  }

  private async executeQueryAction(query: IDoQuery): Promise<void> {
    try {
      if (query.reportState) {
        const handle = this.client.workflow.getHandle(this.workflowId);
        await handle.query('report_state');
      } else if (query.custom) {
        const handle = this.client.workflow.getHandle(this.workflowId);
        const queryArgs = query.custom.args || [];
        await handle.query(query.custom.name || '', ...queryArgs);
      } else {
        throw new Error('DoQuery must have a recognizable variant');
      }
    } catch (error) {
      if (!query.failureExpected) {
        throw error;
      }
    }
  }
}
