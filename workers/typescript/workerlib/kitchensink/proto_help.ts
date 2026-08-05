// Convert a protobuf duration to milliseconds
import { decompileRetryPolicy, RetryPolicy } from '@temporalio/common';
import { google, temporal } from './protos/root';
import Long from 'long';

import IDuration = google.protobuf.IDuration;
import IExecuteActivityAction = temporal.omes.kitchen_sink.IExecuteActivityAction;
import IRetryPolicy = temporal.api.common.v1.IRetryPolicy;

// Convert a protobuf retry policy to the SDK's, treating an unset maximumAttempts as unlimited.
// decompileRetryPolicy passes proto's 0 (unlimited) straight through, but compileRetryPolicy then
// rejects 0 as not a positive integer. That throws a ValueError, which the client masks as
// "Unexpected error while making gRPC request" with no cause.
export function retryPolicyFromProto(
  retryPolicy: IRetryPolicy | null | undefined
): RetryPolicy | undefined {
  const policy = decompileRetryPolicy(retryPolicy);
  if (policy?.maximumAttempts === 0) {
    return { ...policy, maximumAttempts: undefined };
  }
  return policy;
}

// Map an ExecuteActivityAction to its registered activity name and args.
// Shared by the workflow-scheduled path and the standalone-activity path.
export function activityNameAndArgs(act: IExecuteActivityAction): [string, unknown[]] {
  if (act.delay) {
    return ['delay', [durationConvert(act.delay)]];
  } else if (act.resources) {
    return ['resources', [act.resources]];
  } else if (act.payload) {
    const inputData = new Uint8Array(act.payload.bytesToReceive || 0);
    for (let i = 0; i < inputData.length; i++) {
      inputData[i] = i % 256;
    }
    return ['payload', [inputData, act.payload.bytesToReturn]];
  } else if (act.client) {
    return ['client', [act.client]];
  } else if (act.retryableError) {
    return ['retryable_error', [act.retryableError]];
  } else if (act.timeout) {
    return ['timeout', [act.timeout]];
  } else if (act.heartbeat) {
    return ['heartbeat', [act.heartbeat]];
  }
  return ['noop', []];
}

export function durationConvertMaybeUndefined(d: IDuration | null | undefined): number | undefined {
  if (!d) {
    return undefined;
  }
  return durationConvert(d);
}
export function durationConvert(d: IDuration | null | undefined): number {
  if (!d) {
    return 0;
  }
  // convert to ms
  return Math.round(numify(d.seconds) * 1000 + (d.nanos ?? 0) / 1000000);
}

// I just cannot get protobuf to use Long consistently. For whatever insane reason for child
// workflows it reverts to using number.
export function numify(n: number | Long | undefined | null): number {
  if (!n) {
    return 0;
  }
  if (typeof n === 'number') {
    return n;
  }
  return n.toNumber();
}
