package io.temporal.omes;

import com.google.protobuf.util.Durations;
import io.temporal.activity.ActivityCancellationType;
import io.temporal.activity.ActivityOptions;
import io.temporal.activity.LocalActivityOptions;
import io.temporal.api.common.v1.Payload;
import io.temporal.api.common.v1.WorkflowExecution;
import io.temporal.api.enums.v1.ParentClosePolicy;
import io.temporal.common.Priority;
import io.temporal.common.RetryOptions;
import io.temporal.common.VersioningIntent;
import io.temporal.failure.ActivityFailure;
import io.temporal.failure.ApplicationFailure;
import io.temporal.failure.CanceledFailure;
import io.temporal.failure.ChildWorkflowFailure;
import io.temporal.workflow.*;
import java.time.Duration;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import javax.annotation.Nonnull;
import org.slf4j.Logger;

public class KitchenSinkWorkflowImpl implements KitchenSinkWorkflow {
  public static final Logger log = Workflow.getLogger(KitchenSinkWorkflowImpl.class);
  KitchenSink.WorkflowState state = KitchenSink.WorkflowState.getDefaultInstance();
  WorkflowQueue<KitchenSink.ActionSet> signalActionQueue = Workflow.newWorkflowQueue(1_000);

  @Override
  public Payload execute(KitchenSink.WorkflowInput input) {
    // Run all initial input actions
    if (input != null) {
      for (KitchenSink.ActionSet actionSet : input.getInitialActionsList()) {
        Payload result = handleActionSet(actionSet);
        if (result != null) {
          return result;
        }
      }
    }

    // Run all actions from signals
    while (true) {
      KitchenSink.ActionSet actionSet = signalActionQueue.cancellableTake();
      Payload result = handleActionSet(actionSet);
      if (result != null) {
        return result;
      }
    }
  }

  @Override
  public void doActionsSignal(KitchenSink.DoSignal.DoSignalActions signalActions) {
    if (signalActions.hasDoActionsInMain()) {
      signalActionQueue.put(signalActions.getDoActionsInMain());
    } else {
      handleActionSet(signalActions.getDoActions());
    }
  }

  @Override
  public Object doActionsUpdate(KitchenSink.DoActionsUpdate updateInput) {
    Payload result = handleActionSet(updateInput.getDoActions());
    if (result != null) {
      return result;
    }
    return this.state;
  }

  @Override
  public void doActionsUpdateValidator(KitchenSink.DoActionsUpdate updateInput) {
    if (updateInput.hasRejectMe()) {
      throw new RuntimeException("Rejected");
    }
  }

  @Override
  public KitchenSink.WorkflowState reportState(Object queryInput) {
    return state;
  }

  private Payload handleActionSet(KitchenSink.ActionSet actionSet) {
    if (actionSet == null) {
      return null;
    }
    // If these are non-concurrent, just execute and return if requested
    if (!actionSet.getConcurrent()) {
      for (KitchenSink.Action action : actionSet.getActionsList()) {
        Payload actionResult = handleAction(action);
        if (actionResult != null) {
          return actionResult;
        }
      }
      return null;
    }

    CompletablePromise<Payload> returnResult = Workflow.newPromise();
    List<Promise<Void>> results = new ArrayList<>(actionSet.getActionsList().size());
    for (KitchenSink.Action action : actionSet.getActionsList()) {
      results.add(
          Async.procedure(
              () -> {
                try {
                  Payload actionResult = handleAction(action);
                  if (actionResult != null) {
                    returnResult.complete(actionResult);
                  }
                } catch (Exception e) {
                  returnResult.completeExceptionally(new RuntimeException(e));
                  throw e;
                }
              }));
    }
    Promise.anyOf(
            Promise.allOf(results),
            returnResult,
            CancellationScope.current().getCancellationRequest())
        .get();
    CancellationScope.throwCanceled();

    if (returnResult.isCompleted()) {
      return returnResult.get();
    } else {
      return null;
    }
  }

  @SuppressWarnings("deprecation")
  private Payload handleAction(KitchenSink.Action action) {
    if (action.hasReturnResult()) {
      KitchenSink.ReturnResultAction result = action.getReturnResult();
      return result.getReturnThis();
    } else if (action.hasReturnError()) {
      KitchenSink.ReturnErrorAction error = action.getReturnError();
      throw ApplicationFailure.newFailure(error.getFailure().getMessage(), "");
    } else if (action.hasContinueAsNew()) {
      KitchenSink.ContinueAsNewAction continueAsNew = action.getContinueAsNew();
      Workflow.continueAsNew(continueAsNew.getArgumentsList().get(0));
    } else if (action.hasTimer()) {
      KitchenSink.TimerAction timer = action.getTimer();
      CancellationScope scope =
          Workflow.newCancellationScope(
              () -> {
                Promise<Void> timerPromise =
                    Workflow.newTimer(Duration.ofMillis(timer.getMilliseconds()));
                handlePromise(timerPromise, timer.getAwaitableChoice());
              });
      scope.run();
    } else if (action.hasExecActivity()) {
      KitchenSink.ExecuteActivityAction activity = action.getExecActivity();
      launchActivity(activity);
    } else if (action.hasExecChildWorkflow()) {
      KitchenSink.ExecuteChildWorkflowAction childWorkflow = action.getExecChildWorkflow();
      launchChildWorkflow(childWorkflow);
    } else if (action.hasSetWorkflowState()) {
      KitchenSink.WorkflowState workflowState = action.getSetWorkflowState();
      state = workflowState;
    } else if (action.hasAwaitWorkflowState()) {
      KitchenSink.AwaitWorkflowState awaitWorkflowState = action.getAwaitWorkflowState();
      Workflow.await(
          () ->
              awaitWorkflowState
                  .getValue()
                  .equals(state.getKvsOrDefault(awaitWorkflowState.getKey(), "")));
    } else if (action.hasNestedActionSet()) {
      KitchenSink.ActionSet nestedActionSet = action.getNestedActionSet();
      return handleActionSet(nestedActionSet);
    } else if (action.hasSendSignal()) {
      KitchenSink.SendSignalAction sendSignal = action.getSendSignal();
      ExternalWorkflowStub stub =
          Workflow.newUntypedExternalWorkflowStub(
              WorkflowExecution.newBuilder()
                  .setWorkflowId(sendSignal.getWorkflowId())
                  .setRunId(sendSignal.getRunId())
                  .build());
      CancellationScope scope =
          Workflow.newCancellationScope(
              () -> {
                Promise promise =
                    Async.procedure(
                        stub::signal, sendSignal.getSignalName(), sendSignal.getArgsList());
                handlePromise(promise, sendSignal.getAwaitableChoice());
              });
      scope.run();
    } else if (action.hasUpsertSearchAttributes()) {
      KitchenSink.UpsertSearchAttributesAction upsertSearchAttributes =
          action.getUpsertSearchAttributes();
      Workflow.upsertSearchAttributes(upsertSearchAttributes.getSearchAttributes());
    } else if (action.hasSetPatchMarker()) {
      KitchenSink.SetPatchMarkerAction patchMarker = action.getSetPatchMarker();
      if (Workflow.getVersion(patchMarker.getPatchId(), -1, 1) == 1) {
        handleAction(patchMarker.getInnerAction());
      }
    } else if (action.hasUpsertMemo()) {
      KitchenSink.UpsertMemoAction upsertMemoAction = action.getUpsertMemo();
      Map<String, Object> memo = new HashMap();
      upsertMemoAction.getUpsertedMemo().getFieldsMap().forEach(memo::put);
      Workflow.upsertMemo(memo);
    } else {
      throw Workflow.wrap(new IllegalArgumentException("Unrecognized action type"));
    }
    return null;
  }

  private void launchChildWorkflow(KitchenSink.ExecuteChildWorkflowAction executeChildWorkflow) {
    String childWorkflowType =
        executeChildWorkflow.getWorkflowType().isEmpty()
            ? "kitchenSink"
            : executeChildWorkflow.getWorkflowType();

    CancellationScope scope =
        Workflow.newCancellationScope(
            () -> {
              ChildWorkflowStub stub = Workflow.newUntypedChildWorkflowStub(childWorkflowType);
              Promise result =
                  stub.executeAsync(Payload.class, executeChildWorkflow.getInputList().get(0));
              boolean expectCancelled = false;
              switch (executeChildWorkflow.getAwaitableChoice().getConditionCase()) {
                case ABANDON:
                  return;
                case CANCEL_BEFORE_STARTED:
                  CancellationScope.current().cancel();
                  expectCancelled = true;
                  break;
                case CANCEL_AFTER_STARTED:
                  stub.getExecution().get();
                  CancellationScope.current().cancel();
                  expectCancelled = true;
                  break;
                case CANCEL_AFTER_COMPLETED:
                  result.get();
                  CancellationScope.current().cancel();
                  expectCancelled = true;
                  break;
                case WAIT_FINISH:
                case CONDITION_NOT_SET:
                  result.get();
                  break;
              }

              if (expectCancelled) {
                try {
                  result.get();
                } catch (ChildWorkflowFailure e) {
                  if (!(e.getCause() instanceof CanceledFailure)) {
                    throw e;
                  }
                }
              }
            });
    scope.run();
  }

  private void launchActivity(KitchenSink.ExecuteActivityAction executeActivity) {
    String activityType;
    List<Object> args = new ArrayList<>();

    if (executeActivity.hasDelay()) {
      activityType = "delay";
      args.add(executeActivity.getDelay());
    } else if (executeActivity.hasPayload()) {
      activityType = "payload";
      KitchenSink.ExecuteActivityAction.PayloadActivity payload = executeActivity.getPayload();
      byte[] inputData = new byte[payload.getBytesToReceive()];
      for (int i = 0; i < inputData.length; i++) {
        inputData[i] = (byte) (i % 256);
      }
      args.add(inputData);
      args.add(payload.getBytesToReturn());
    } else {
      activityType = "noop";
    }

    RetryOptions.Builder retryOptions =
        RetryOptions.newBuilder()
            .setDoNotRetry(
                executeActivity
                    .getRetryPolicy()
                    .getNonRetryableErrorTypesList()
                    .toArray(new String[0]))
            .setMaximumAttempts(executeActivity.getRetryPolicy().getMaximumAttempts());

    Duration initialInterval =
        toJavaDuration(executeActivity.getRetryPolicy().getInitialInterval());
    if (initialInterval != Duration.ZERO) {
      retryOptions.setInitialInterval(initialInterval);
    }

    Duration maximumInterval =
        toJavaDuration(executeActivity.getRetryPolicy().getMaximumInterval());
    if (maximumInterval != Duration.ZERO) {
      retryOptions.setMaximumInterval(maximumInterval);
    }

    double backoff = executeActivity.getRetryPolicy().getBackoffCoefficient();
    if (backoff != 0.0) {
      retryOptions.setBackoffCoefficient(backoff);
    }

    Priority.Builder prio = Priority.newBuilder();
    io.temporal.api.common.v1.Priority priority = executeActivity.getPriority();
    if (priority.getPriorityKey() > 0) {
      prio.setPriorityKey(priority.getPriorityKey());
    }
    if (executeActivity.getFairnessKey() != "") {
      throw new IllegalArgumentException("FairnessKey is not supported");
    }
    if (executeActivity.getFairnessWeight() > 0) {
      throw new IllegalArgumentException("FairnessWeight is not supported");
    }

    if (executeActivity.hasIsLocal()) {
      LocalActivityOptions.Builder builder =
          LocalActivityOptions.newBuilder()
              .setStartToCloseTimeout(toJavaDuration(executeActivity.getStartToCloseTimeout()))
              .setRetryOptions(retryOptions.build());

      if (executeActivity.hasScheduleToCloseTimeout()) {
        builder.setScheduleToCloseTimeout(
            toJavaDuration(executeActivity.getScheduleToCloseTimeout()));
      }
      if (executeActivity.hasScheduleToStartTimeout()) {
        builder.setScheduleToStartTimeout(
            toJavaDuration(executeActivity.getScheduleToStartTimeout()));
      }

      CancellationScope scope =
          Workflow.newCancellationScope(
              () -> {
                Promise<Void> activityResult =
                    Workflow.newUntypedLocalActivityStub(builder.build())
                        .executeAsync(activityType, void.class, args.toArray());
                handlePromise(activityResult, executeActivity.getAwaitableChoice());
              });
      scope.run();
    } else {
      KitchenSink.RemoteActivityOptions remoteOptions = executeActivity.getRemote();
      ActivityOptions.Builder builder =
          ActivityOptions.newBuilder()
              .setScheduleToStartTimeout(
                  toJavaDuration(executeActivity.getScheduleToStartTimeout()))
              .setStartToCloseTimeout(toJavaDuration(executeActivity.getStartToCloseTimeout()))
              .setHeartbeatTimeout(toJavaDuration(executeActivity.getHeartbeatTimeout()))
              .setDisableEagerExecution(remoteOptions.getDoNotEagerlyExecute())
              .setVersioningIntent(getVersioningIntent(remoteOptions.getVersioningIntent()))
              .setCancellationType(getActivityCancellationType(remoteOptions.getCancellationType()))
              .setRetryOptions(retryOptions.build())
              .setPriority(prio.build());

      if (executeActivity.hasScheduleToCloseTimeout()) {
        builder.setScheduleToCloseTimeout(
            toJavaDuration(executeActivity.getScheduleToCloseTimeout()));
      }

      String taskQueue = executeActivity.getTaskQueue();
      if (taskQueue != null && !taskQueue.isEmpty()) {
        builder.setTaskQueue(taskQueue);
      }

      CancellationScope scope =
          Workflow.newCancellationScope(
              () -> {
                Promise<Void> activityResult =
                    Workflow.newUntypedActivityStub(builder.build())
                        .executeAsync(activityType, void.class, args.toArray());
                handlePromise(activityResult, executeActivity.getAwaitableChoice());
              });
      scope.run();
    }
  }

  private <V> void handlePromise(Promise<V> promise, KitchenSink.AwaitableChoice condition) {
    boolean expectCancelled = false;
    switch (condition.getConditionCase()) {
      case ABANDON:
        return;
      case CANCEL_BEFORE_STARTED:
        CancellationScope.current().cancel();
        expectCancelled = true;
        break;
      case CANCEL_AFTER_STARTED:
        // Wait a workflow task
        Workflow.sleep(Duration.ofMillis(1));
        CancellationScope.current().cancel();
        expectCancelled = true;
        break;
      case CANCEL_AFTER_COMPLETED:
        promise.get();
        CancellationScope.current().cancel();
        break;
      case WAIT_FINISH:
      case CONDITION_NOT_SET:
        promise.get();
        break;
    }

    if (expectCancelled) {
      try {
        promise.get();
      } catch (CanceledFailure e) {
      } catch (ActivityFailure e) {
        if (!(e.getCause() instanceof CanceledFailure)) {
          throw e;
        }
      }
    }
  }

  public static VersioningIntent getVersioningIntent(KitchenSink.VersioningIntent intent) {
    if (intent == null) {
      return null;
    }

    switch (intent) {
      case UNSPECIFIED:
        return VersioningIntent.VERSIONING_INTENT_UNSPECIFIED;
      case DEFAULT:
        return VersioningIntent.VERSIONING_INTENT_DEFAULT;
      case COMPATIBLE:
        return VersioningIntent.VERSIONING_INTENT_COMPATIBLE;
    }
    return null;
  }

  public static ActivityCancellationType getActivityCancellationType(
      KitchenSink.ActivityCancellationType cancellationType) {
    if (cancellationType == null) {
      return null;
    }

    switch (cancellationType) {
      case WAIT_CANCELLATION_COMPLETED:
        return ActivityCancellationType.WAIT_CANCELLATION_COMPLETED;
      case TRY_CANCEL:
        return ActivityCancellationType.TRY_CANCEL;
      case ABANDON:
        return ActivityCancellationType.ABANDON;
    }
    return null;
  }

  public static ParentClosePolicy getParentClosePolicy(
      KitchenSink.ParentClosePolicy parentClosePolicy) {
    if (parentClosePolicy == null) {
      return null;
    }

    switch (parentClosePolicy) {
      case PARENT_CLOSE_POLICY_UNSPECIFIED:
        return ParentClosePolicy.PARENT_CLOSE_POLICY_UNSPECIFIED;
      case PARENT_CLOSE_POLICY_ABANDON:
        return ParentClosePolicy.PARENT_CLOSE_POLICY_ABANDON;
      case PARENT_CLOSE_POLICY_TERMINATE:
        return ParentClosePolicy.PARENT_CLOSE_POLICY_TERMINATE;
      case PARENT_CLOSE_POLICY_REQUEST_CANCEL:
        return ParentClosePolicy.PARENT_CLOSE_POLICY_REQUEST_CANCEL;
    }
    return null;
  }

  @Nonnull
  public static Duration toJavaDuration(com.google.protobuf.Duration d) {
    if (d == null) {
      return Duration.ZERO;
    }
    return Duration.ofMillis(Durations.toMillis(d));
  }

  public static com.google.protobuf.Duration toProtoDuration(Duration d) {
    if (d == null) {
      return Durations.ZERO;
    }
    return Durations.fromMillis(d.toMillis());
  }
}
