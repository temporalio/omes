import { Client, WithStartWorkflowOperation } from '@temporalio/client';
import { ApplicationFailure } from '@temporalio/common';
import { WorkflowIdConflictPolicy } from '@temporalio/client';
import { temporal } from './protos/root';
import IClientSequence = temporal.omes.kitchen_sink.IClientSequence;
import IClientActionSet = temporal.omes.kitchen_sink.IClientActionSet;
import IClientAction = temporal.omes.kitchen_sink.IClientAction;
import IDoSignal = temporal.omes.kitchen_sink.IDoSignal;
import IDoUpdate = temporal.omes.kitchen_sink.IDoUpdate;
import IDoQuery = temporal.omes.kitchen_sink.IDoQuery;

export class ClientActionExecutor {
  private client: Client;
  private workflowId = '';
  private runId = '';
  private workflowType = 'kitchenSink';
  private workflowInput: any = null;
  private taskQueue;
  private errOnUnimplemented: boolean;

  constructor(client: Client, workflowId: string, taskQueue: string, errOnUnimplemented: boolean = false) {
    this.client = client;
    this.workflowId = workflowId;
    this.taskQueue = taskQueue;
    this.errOnUnimplemented = errOnUnimplemented;
  }

  async executeClientSequence(clientSeq?: IClientSequence | null): Promise<void> {
    for (const actionSet of clientSeq?.actionSets || []) {
      await this.executeClientActionSet(actionSet);
    }
  }

  private async executeClientActionSet(actionSet: IClientActionSet): Promise<void> {
    if (actionSet.concurrent) {
      if (this.errOnUnimplemented) {
        throw ApplicationFailure.nonRetryable('concurrent client actions are not supported');
      }
      // Skip concurrent actions when not erroring on unimplemented
      return;
    }

    for (const action of actionSet.actions || []) {
      await this.executeClientAction(action);
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
      signalArgs = [signal.doSignalActions];
    } else if (signal.custom) {
      signalName = signal.custom.name || '';
      signalArgs = signal.custom.args || [];
    } else {
      throw new Error('DoSignal must have a recognizable variant');
    }

    try {
      if (signal.withStart) {
        const handle = await this.client.workflow.signalWithStart(this.workflowType, {
          workflowId: this.workflowId,
          taskQueue: this.taskQueue,
          args: [this.workflowInput],
          signal: signalName,
          signalArgs,
          workflowIdConflictPolicy: WorkflowIdConflictPolicy.USE_EXISTING,
        });
        this.workflowId = handle.workflowId;
        this.runId = handle.signaledRunId;
      } else {
        const handle = this.client.workflow.getHandle(this.workflowId);
        await handle.signal(signalName, ...signalArgs);
      }
    } catch (error) {
      console.error(`Signal execution failed for ${signalName}:`, error);
      throw error;
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
        const startWorkflowOperation = new WithStartWorkflowOperation(this.workflowType, {
          workflowId: this.workflowId,
          taskQueue: this.taskQueue,
          args: [this.workflowInput],
          workflowIdConflictPolicy: WorkflowIdConflictPolicy.USE_EXISTING,
        });
        await this.client.workflow.executeUpdateWithStart(updateName, {
          args: [updateArgs],
          startWorkflowOperation,
        });
      } else {
        const handle = this.client.workflow.getHandle(this.workflowId);
        await handle.executeUpdate(updateName, { args: [updateArgs] });
      }
    } catch (error) {
      console.error(`Update execution failed for ${updateName}:`, error);
      if (!update.failureExpected) {
        throw error;
      }
    }
  }

  private async executeQueryAction(query: IDoQuery): Promise<void> {
    try {
      if (query.reportState) {
        const handle = this.client.workflow.getHandle(this.workflowId);
        await handle.query('report_state', null);
      } else if (query.custom) {
        const handle = this.client.workflow.getHandle(this.workflowId);
        const queryArgs = query.custom.args || [];
        await handle.query(query.custom.name || '', ...queryArgs);
      } else {
        throw new Error('DoQuery must have a recognizable variant');
      }
    } catch (error: unknown) {
      if (!query.failureExpected) {
        throw error;
      }
    }
  }
}
