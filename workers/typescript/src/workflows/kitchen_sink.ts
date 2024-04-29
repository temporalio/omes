import { temporal } from '../protos/root';
import {
  ActivityCancellationType as WFActivityCancellationType,
  ApplicationFailure,
  CancellationScope,
  ChildWorkflowHandle,
  ChildWorkflowOptions,
  condition,
  continueAsNew,
  defineQuery,
  defineSignal,
  defineUpdate,
  deprecatePatch,
  isCancellation,
  patched,
  scheduleActivity,
  scheduleLocalActivity,
  setHandler,
  sleep,
  startChild,
  upsertSearchAttributes,
  Workflow,
} from '@temporalio/workflow';
import {
  ActivityOptions,
  decompileRetryPolicy,
  LocalActivityOptions,
  SearchAttributes,
} from '@temporalio/common';
import WorkflowInput = temporal.omes.kitchen_sink.WorkflowInput;
import WorkflowState = temporal.omes.kitchen_sink.WorkflowState;
import Payload = temporal.api.common.v1.Payload;
import DoSignalActions = temporal.omes.kitchen_sink.DoSignal.DoSignalActions;
import IActionSet = temporal.omes.kitchen_sink.IActionSet;
import DoActionsUpdate = temporal.omes.kitchen_sink.DoActionsUpdate;
import IAction = temporal.omes.kitchen_sink.IAction;
import IPayload = temporal.api.common.v1.IPayload;
import IAwaitableChoice = temporal.omes.kitchen_sink.IAwaitableChoice;
import IExecuteActivityAction = temporal.omes.kitchen_sink.IExecuteActivityAction;
import ActivityCancellationType = temporal.omes.kitchen_sink.ActivityCancellationType;
import IWorkflowState = temporal.omes.kitchen_sink.IWorkflowState;
import { durationConvert, numify } from '../proto_help';

const reportStateQuery = defineQuery<IWorkflowState, [Payload]>('report_state');
const actionsSignal = defineSignal<[DoSignalActions]>('do_actions_signal');
const actionsUpdate = defineUpdate<IPayload | undefined, [DoActionsUpdate]>('do_actions_update');

export async function kitchenSink(input: WorkflowInput | undefined): Promise<IPayload | undefined> {
  let workflowState: IWorkflowState = WorkflowState.create();
  const actionsQueue = new Array<IActionSet>();

  async function handleActionSet(actions: IActionSet): Promise<IPayload | undefined> {
    let rval: IPayload | undefined;

    if (!actions.concurrent) {
      for (const action of actions.actions ?? []) {
        const actionRval = await handleAction(action);
        if (actionRval) {
          rval = actionRval;
        }
      }
      return rval;
    }

    // Concurrent actions run concurrently but we return early if any return a value
    const promises = new Array<Promise<void>>();
    for (const action of actions.actions ?? []) {
      promises.push(
        handleAction(action).then((actionRval) => {
          if (actionRval) {
            rval = actionRval;
          }
        })
      );
    }
    const allComplete = Promise.all(promises);
    await Promise.race([allComplete, condition(() => rval !== undefined)]);

    return rval;
  }

  async function handleAction(action: IAction): Promise<IPayload | null | undefined> {
    async function handleAwaitableChoice<PR extends Promise<PRR>, PRR>(
      promise: () => PR,
      choice: IAwaitableChoice | null | undefined,
      afterStarted: (_: Promise<PRR | void>) => Promise<void> = async (_) => {
        await sleep(1);
      },
      afterCompleted: (_: Promise<PRR | void>) => Promise<void> = async (task) => {
        await task;
      }
    ) {
      const cancelScope = new CancellationScope();
      let didCancel = false;

      const cancellablePromise = cancelScope
        .run(() => promise())
        .catch((err) => {
          if (didCancel && isCancellation(err)) {
            return;
          }
          throw err;
        });

      if (choice?.abandon) {
        // Do nothing
      } else if (choice?.cancelBeforeStarted) {
        cancelScope.cancel();
        didCancel = true;
        await cancellablePromise;
      } else if (choice?.cancelAfterStarted) {
        await afterStarted(cancellablePromise);
        cancelScope.cancel();
        didCancel = true;
        await cancellablePromise;
      } else if (choice?.cancelAfterCompleted) {
        await afterCompleted(cancellablePromise);
        cancelScope.cancel();
      } else {
        await cancellablePromise;
      }
    }

    if (action.returnResult) {
      return action.returnResult.returnThis;
    } else if (action.returnError) {
      throw new ApplicationFailure(action.returnError.failure?.message);
    } else if (action.continueAsNew) {
      await continueAsNew(action.continueAsNew.arguments![0]);
    } else if (action.timer) {
      const ms = numify(action.timer.milliseconds);
      const sleeper = () => sleep(ms);
      await handleAwaitableChoice(sleeper, action.timer.awaitableChoice);
    } else if (action.execActivity) {
      const execAct = action.execActivity;
      await handleAwaitableChoice(
        () => launchActivity(execAct),
        action.execActivity.awaitableChoice
      );
    } else if (action.execChildWorkflow) {
      const opts: ChildWorkflowOptions = {};
      if (action.execChildWorkflow.workflowId) {
        opts.workflowId = action.execChildWorkflow.workflowId;
      }
      const execChild = action.execChildWorkflow;
      const childStarter = () => {
        return startChild(execChild.workflowType ?? 'kitchenSink', {
          args: execChild.input ?? [],
          ...opts,
        });
      };
      await handleAwaitableChoice(
        childStarter,
        action.execChildWorkflow.awaitableChoice,
        async (task) => {
          await task;
        },
        async (task: Promise<ChildWorkflowHandle<Workflow> | void>) => {
          const handle = await task;
          if (handle) {
            await handle.result();
          }
        }
      );
    } else if (action.setPatchMarker) {
      let wasPatched: boolean;
      if (action.setPatchMarker.deprecated) {
        deprecatePatch(action.setPatchMarker.patchId!);
        wasPatched = true;
      } else {
        wasPatched = patched(action.setPatchMarker.patchId!);
      }

      if (wasPatched && action.setPatchMarker.innerAction) {
        return await handleAction(action.setPatchMarker.innerAction);
      }
    } else if (action.setWorkflowState) {
      workflowState = WorkflowState.fromObject(action.setWorkflowState);
    } else if (action.awaitWorkflowState) {
      const key = action.awaitWorkflowState.key!;
      const value = action.awaitWorkflowState.value!;
      await condition(() => {
        return workflowState.kvs?.[key] === value;
      });
    } else if (action.upsertMemo) {
      // no upsert memo in ts
    } else if (action.upsertSearchAttributes) {
      const searchAttributes: SearchAttributes = {};
      for (const [key, value] of Object.entries(
        action.upsertSearchAttributes.searchAttributes ?? {}
      )) {
        if (key.includes('Keyword')) {
          searchAttributes[key] = [value.data![0].toString()];
        } else {
          searchAttributes[key] = [value.data![0]];
        }
      }
      upsertSearchAttributes(searchAttributes);
    } else if (action.nestedActionSet) {
      return await handleActionSet(action.nestedActionSet);
    } else {
      throw new ApplicationFailure('Unknown action ' + JSON.stringify(action));
    }
  }

  setHandler(reportStateQuery, (_) => workflowState);
  setHandler(actionsSignal, async (actions) => {
    if (actions.doActionsInMain) {
      actionsQueue.unshift(actions.doActionsInMain);
    } else if (actions.doActions) {
      await handleActionSet(actions.doActions);
    } else {
      throw new ApplicationFailure('Actions signal received with no actions!');
    }
  });
  setHandler(
    actionsUpdate,
    async (actions) => {
      const rval = await handleActionSet(actions.doActions!);
      if (rval) {
        return rval;
      }
    },
    {
      validator: (actions) => {
        if (actions.rejectMe) {
          throw new ApplicationFailure('Rejected');
        }
      },
    }
  );

  // Run all initial input actions
  if (input?.initialActions) {
    for (const actionSet of input.initialActions) {
      const rval = await handleActionSet(actionSet);
      if (rval) {
        return rval;
      }
    }
  }

  // Run all actions from signals
  for (;;) {
    await condition(() => actionsQueue.length > 0);
    const actions = actionsQueue.pop()!;
    const rval = await handleActionSet(actions);
    if (rval) {
      return rval;
    }
  }
}

function launchActivity(execActivity: IExecuteActivityAction): Promise<unknown> {
  let actType = 'noop';
  const args = [];
  if (execActivity.delay) {
    actType = 'delay';
    args.push(durationConvert(execActivity.delay));
  }
  if (execActivity.resources) {
    actType = 'resources';
    args.push(execActivity.resources);
  }

  const actArgs: ActivityOptions | LocalActivityOptions = {
    scheduleToCloseTimeout: durationConvert(execActivity.scheduleToCloseTimeout),
    startToCloseTimeout: durationConvert(execActivity.startToCloseTimeout),
    scheduleToStartTimeout: durationConvert(execActivity.scheduleToStartTimeout),
    retry: decompileRetryPolicy(execActivity.retryPolicy),
  };

  if (execActivity.isLocal) {
    return scheduleLocalActivity(actType, args, actArgs);
  } else {
    const remoteArgs = actArgs as ActivityOptions;
    remoteArgs.taskQueue = execActivity.taskQueue ?? undefined;
    remoteArgs.cancellationType = convertCancelType(execActivity.remote?.cancellationType);
    remoteArgs.heartbeatTimeout = durationConvert(execActivity.heartbeatTimeout);
    return scheduleActivity(actType, args, remoteArgs);
  }
}

function convertCancelType(
  ct: ActivityCancellationType | null | undefined
): WFActivityCancellationType | undefined {
  if (ct === ActivityCancellationType.TRY_CANCEL) {
    return WFActivityCancellationType.TRY_CANCEL;
  } else if (ct === ActivityCancellationType.WAIT_CANCELLATION_COMPLETED) {
    return WFActivityCancellationType.WAIT_CANCELLATION_COMPLETED;
  } else if (ct === ActivityCancellationType.ABANDON) {
    return WFActivityCancellationType.ABANDON;
  }
}
