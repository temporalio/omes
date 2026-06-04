package io.temporal.omes;

import java.util.ArrayList;
import java.util.List;

// Maps an ExecuteActivityAction to its registered activity name and args.
// Shared by the workflow-scheduled path and the standalone-activity path.
public final class ActivityDispatch {
  final String type;
  final List<Object> args;

  private ActivityDispatch(String type, List<Object> args) {
    this.type = type;
    this.args = args;
  }

  static ActivityDispatch nameAndArgs(KitchenSink.ExecuteActivityAction act) {
    String type;
    List<Object> args = new ArrayList<>();
    if (act.hasDelay()) {
      type = "delay";
      args.add(act.getDelay());
    } else if (act.hasPayload()) {
      type = "payload";
      KitchenSink.ExecuteActivityAction.PayloadActivity payload = act.getPayload();
      byte[] inputData = new byte[payload.getBytesToReceive()];
      for (int i = 0; i < inputData.length; i++) {
        inputData[i] = (byte) (i % 256);
      }
      args.add(inputData);
      args.add(payload.getBytesToReturn());
    } else if (act.hasClient()) {
      type = "client";
      args.add(act.getClient());
    } else if (act.hasRetryableError()) {
      type = "retryable_error";
      args.add(act.getRetryableError());
    } else if (act.hasTimeout()) {
      type = "timeout";
      args.add(act.getTimeout());
    } else if (act.hasHeartbeat()) {
      type = "heartbeat";
      args.add(act.getHeartbeat());
    } else {
      type = "noop";
    }
    return new ActivityDispatch(type, args);
  }
}
