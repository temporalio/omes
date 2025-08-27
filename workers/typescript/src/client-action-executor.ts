import { ApplicationFailure } from '@temporalio/common';
import { temporal } from './protos/root';
import IClientSequence = temporal.omes.kitchen_sink.IClientSequence;

export class ClientActionExecutor {
  // eslint-disable-next-line @typescript-eslint/no-empty-function
  constructor(_client: unknown, _workflowId: string, _taskQueue: string) {}

  async executeClientSequence(_seq?: IClientSequence | null): Promise<void> {
    throw ApplicationFailure.nonRetryable('client actions activity is not supported');
  }
}
