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
  retryPolicy: IRetryPolicy | null | undefined,
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
// workflows it reverts to using number. Under protobufjs 8 a 64-bit field decoded inside the
// Workflow sandbox arrives as a plain {low, high, unsigned} object with no Long prototype,
// because the sandbox does not have the long library wired up, so toNumber() is missing. The
// bigint and string branches are defensive: protobufjs can represent 64-bit values either way
// depending on how the root is configured.
export function numify(
  n:
    | number
    | bigint
    | string
    | Long
    | { low: number; high: number; unsigned?: boolean }
    | undefined
    | null,
): number {
  if (!n) {
    return 0;
  }
  if (typeof n === 'number') {
    return n;
  }
  if (typeof n === 'bigint') {
    return Number(n);
  }
  if (typeof n === 'string') {
    return Number(n);
  }
  if (typeof (n as Long).toNumber === 'function') {
    return (n as Long).toNumber();
  }
  // Plain Long-shaped object: recombine the 32-bit halves.
  const { low, high, unsigned } = n as { low: number; high: number; unsigned?: boolean };
  const hi = unsigned ? high >>> 0 : high;
  return hi * 0x100000000 + (low >>> 0);
}
