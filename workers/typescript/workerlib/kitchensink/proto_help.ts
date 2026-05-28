// Convert a protobuf duration to milliseconds
import type { google } from './protos/root';
import type Long from 'long';

type IDuration = google.protobuf.IDuration;

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
