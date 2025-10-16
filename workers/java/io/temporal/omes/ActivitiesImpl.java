package io.temporal.omes;

import io.temporal.activity.Activity;
import io.temporal.activity.ActivityExecutionContext;
import io.temporal.client.WorkflowClient;
import io.temporal.client.WorkflowOptions;
import io.temporal.client.schedules.Schedule;
import io.temporal.client.schedules.ScheduleActionStartWorkflow;
import io.temporal.client.schedules.ScheduleBackfill;
import io.temporal.client.schedules.ScheduleClient;
import io.temporal.client.schedules.ScheduleHandle;
import io.temporal.client.schedules.ScheduleOptions;
import io.temporal.client.schedules.SchedulePolicy;
import io.temporal.client.schedules.ScheduleSpec;
import io.temporal.client.schedules.ScheduleUpdate;
import io.temporal.common.RetryOptions;
import java.time.Duration;
import java.time.Instant;
import java.util.ArrayList;
import java.util.List;
import java.util.Random;

public class ActivitiesImpl implements Activities {

  private final WorkflowClient client;

  public ActivitiesImpl(WorkflowClient client) {
    this.client = client;
  }

  @Override
  public void noop() {}

  @Override
  public void delay(com.google.protobuf.Duration d) throws InterruptedException {
    Thread.sleep(1000 * d.getSeconds() + d.getNanos() / 1_000_000);
  }

  @Override
  public byte[] payload(byte[] inputData, int bytesToReturn) {
    byte[] output = new byte[bytesToReturn];
    new Random().nextBytes(output);
    return output;
  }

  @Override
  public void client(
      io.temporal.omes.KitchenSink.ExecuteActivityAction.ClientActivity clientActivity) {
    var activityInfo = Activity.getExecutionContext().getInfo();
    String workflowId = activityInfo.getWorkflowId();
    String taskQueue = activityInfo.getActivityTaskQueue();
    ClientActionExecutor executor = new ClientActionExecutor(client, workflowId, taskQueue);
    executor.executeClientSequence(clientActivity.getClientSequence());
  }

  private String makeScheduleIdUnique(String baseScheduleId, String workflowExecutionId) {
    String sanitizedWorkflowId = workflowExecutionId.replace("/", "-");
    return baseScheduleId + "-" + sanitizedWorkflowId;
  }

  @Override
  public void createScheduleActivity(io.temporal.omes.KitchenSink.CreateScheduleAction action) {
    ActivityExecutionContext context = Activity.getExecutionContext();
    var activityInfo = context.getInfo();
    ScheduleClient scheduleClient = ScheduleClient.newInstance(client.getWorkflowServiceStubs());

    if (action.getScheduleId().isEmpty()) {
      throw new IllegalArgumentException("schedule_id is required");
    }
    if (!action.hasAction()) {
      throw new IllegalArgumentException("action is required");
    }

    String taskQueue =
        action.getAction().getTaskQueue().isEmpty()
            ? activityInfo.getActivityTaskQueue()
            : action.getAction().getTaskQueue();

    String uniqueScheduleId =
        makeScheduleIdUnique(action.getScheduleId(), activityInfo.getWorkflowId());

    String uniqueWorkflowId = action.getAction().getWorkflowId();
    if (!uniqueWorkflowId.isEmpty()) {
      uniqueWorkflowId = makeScheduleIdUnique(uniqueWorkflowId, activityInfo.getWorkflowId());
    }

    String workflowType =
        action.getAction().getWorkflowType().isEmpty()
            ? "kitchenSink"
            : action.getAction().getWorkflowType();

    Object[] args = action.getAction().getInputList().toArray();

    WorkflowOptions.Builder workflowOptionsBuilder =
        WorkflowOptions.newBuilder().setWorkflowId(uniqueWorkflowId).setTaskQueue(taskQueue);

    if (action.getAction().hasWorkflowExecutionTimeout()) {
      workflowOptionsBuilder.setWorkflowExecutionTimeout(
          Duration.ofSeconds(action.getAction().getWorkflowExecutionTimeout().getSeconds())
              .plusNanos(action.getAction().getWorkflowExecutionTimeout().getNanos()));
    }

    if (action.getAction().hasWorkflowTaskTimeout()) {
      workflowOptionsBuilder.setWorkflowTaskTimeout(
          Duration.ofSeconds(action.getAction().getWorkflowTaskTimeout().getSeconds())
              .plusNanos(action.getAction().getWorkflowTaskTimeout().getNanos()));
    }

    if (action.getAction().hasRetryPolicy()) {
      RetryOptions.Builder retryBuilder = RetryOptions.newBuilder();
      if (action.getAction().getRetryPolicy().getMaximumAttempts() > 0) {
        retryBuilder.setMaximumAttempts(action.getAction().getRetryPolicy().getMaximumAttempts());
      }
      if (action.getAction().getRetryPolicy().hasInitialInterval()) {
        retryBuilder.setInitialInterval(
            Duration.ofSeconds(
                    action.getAction().getRetryPolicy().getInitialInterval().getSeconds())
                .plusNanos(action.getAction().getRetryPolicy().getInitialInterval().getNanos()));
      }
      if (action.getAction().getRetryPolicy().hasMaximumInterval()) {
        retryBuilder.setMaximumInterval(
            Duration.ofSeconds(
                    action.getAction().getRetryPolicy().getMaximumInterval().getSeconds())
                .plusNanos(action.getAction().getRetryPolicy().getMaximumInterval().getNanos()));
      }
      if (action.getAction().getRetryPolicy().getBackoffCoefficient() > 0) {
        retryBuilder.setBackoffCoefficient(
            action.getAction().getRetryPolicy().getBackoffCoefficient());
      }
      if (!action.getAction().getRetryPolicy().getNonRetryableErrorTypesList().isEmpty()) {
        retryBuilder.setDoNotRetry(
            action
                .getAction()
                .getRetryPolicy()
                .getNonRetryableErrorTypesList()
                .toArray(new String[0]));
      }
      workflowOptionsBuilder.setRetryOptions(retryBuilder.build());
    }

    ScheduleActionStartWorkflow scheduleAction =
        ScheduleActionStartWorkflow.newBuilder()
            .setWorkflowType(workflowType)
            .setArguments(args)
            .setOptions(workflowOptionsBuilder.build())
            .build();

    ScheduleSpec.Builder specBuilder = ScheduleSpec.newBuilder();
    if (action.hasSpec()) {
      if (!action.getSpec().getCronExpressionsList().isEmpty()) {
        specBuilder.setCronExpressions(action.getSpec().getCronExpressionsList());
      }
      if (action.getSpec().hasJitter()) {
        specBuilder.setJitter(
            Duration.ofSeconds(action.getSpec().getJitter().getSeconds())
                .plusNanos(action.getSpec().getJitter().getNanos()));
      }
    }

    SchedulePolicy.Builder policyBuilder = SchedulePolicy.newBuilder();
    if (action.hasPolicies()) {
      if (action.getPolicies().hasCatchupWindow()) {
        policyBuilder.setCatchupWindow(
            Duration.ofSeconds(action.getPolicies().getCatchupWindow().getSeconds())
                .plusNanos(action.getPolicies().getCatchupWindow().getNanos()));
      }
    }

    Schedule schedule =
        Schedule.newBuilder()
            .setAction(scheduleAction)
            .setSpec(specBuilder.build())
            .setPolicy(policyBuilder.build())
            .build();

    ScheduleOptions.Builder scheduleOptionsBuilder = ScheduleOptions.newBuilder();
    if (action.hasPolicies()) {
      scheduleOptionsBuilder.setTriggerImmediately(action.getPolicies().getTriggerImmediately());
    }

    if (!action.getBackfillList().isEmpty()) {
      List<ScheduleBackfill> backfills = new ArrayList<>();
      for (var bf : action.getBackfillList()) {
        backfills.add(
            new ScheduleBackfill(
                Instant.ofEpochSecond(bf.getStartTimestamp()),
                Instant.ofEpochSecond(bf.getEndTimestamp())));
      }
      scheduleOptionsBuilder.setBackfills(backfills);
    }

    scheduleClient.createSchedule(uniqueScheduleId, schedule, scheduleOptionsBuilder.build());
  }

  @Override
  public Object describeScheduleActivity(
      io.temporal.omes.KitchenSink.DescribeScheduleAction action) {
    ActivityExecutionContext context = Activity.getExecutionContext();
    var activityInfo = context.getInfo();
    ScheduleClient scheduleClient = ScheduleClient.newInstance(client.getWorkflowServiceStubs());

    if (action.getScheduleId().isEmpty()) {
      throw new IllegalArgumentException("schedule_id is required");
    }

    String uniqueScheduleId =
        makeScheduleIdUnique(action.getScheduleId(), activityInfo.getWorkflowId());

    ScheduleHandle handle = scheduleClient.getHandle(uniqueScheduleId);
    handle.describe();
    return null;
  }

  @Override
  public void updateScheduleActivity(io.temporal.omes.KitchenSink.UpdateScheduleAction action) {
    ActivityExecutionContext context = Activity.getExecutionContext();
    var activityInfo = context.getInfo();
    ScheduleClient scheduleClient = ScheduleClient.newInstance(client.getWorkflowServiceStubs());

    if (action.getScheduleId().isEmpty()) {
      throw new IllegalArgumentException("schedule_id is required");
    }

    String uniqueScheduleId =
        makeScheduleIdUnique(action.getScheduleId(), activityInfo.getWorkflowId());

    ScheduleHandle handle = scheduleClient.getHandle(uniqueScheduleId);

    handle.update(
        input -> {
          Schedule.Builder scheduleBuilder =
              Schedule.newBuilder(input.getDescription().getSchedule());

          if (action.hasSpec()) {
            ScheduleSpec.Builder specBuilder = ScheduleSpec.newBuilder();
            if (!action.getSpec().getCronExpressionsList().isEmpty()) {
              specBuilder.setCronExpressions(action.getSpec().getCronExpressionsList());
            }
            if (action.getSpec().hasJitter()) {
              specBuilder.setJitter(
                  Duration.ofSeconds(action.getSpec().getJitter().getSeconds())
                      .plusNanos(action.getSpec().getJitter().getNanos()));
            }
            scheduleBuilder.setSpec(specBuilder.build());
          }

          return new ScheduleUpdate(scheduleBuilder.build());
        });
  }

  @Override
  public void deleteScheduleActivity(io.temporal.omes.KitchenSink.DeleteScheduleAction action) {
    ActivityExecutionContext context = Activity.getExecutionContext();
    var activityInfo = context.getInfo();
    ScheduleClient scheduleClient = ScheduleClient.newInstance(client.getWorkflowServiceStubs());

    if (action.getScheduleId().isEmpty()) {
      throw new IllegalArgumentException("schedule_id is required");
    }

    String uniqueScheduleId =
        makeScheduleIdUnique(action.getScheduleId(), activityInfo.getWorkflowId());

    ScheduleHandle handle = scheduleClient.getHandle(uniqueScheduleId);
    handle.delete();
  }
}
