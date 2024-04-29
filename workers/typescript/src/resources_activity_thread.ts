import { temporal } from './protos/root';
import { durationConvert } from './proto_help';
import { workerData } from 'node:worker_threads';
import Long from 'long';
import IResourcesActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.IResourcesActivity;
import ResourcesActivity = temporal.omes.kitchen_sink.ExecuteActivityAction.ResourcesActivity;

async function resourcesImpl(iinput: IResourcesActivity) {
  // If I don't do this somehow the longs inside the protos magically are not longs after being passed to the worker
  const input = ResourcesActivity.fromObject(iinput);
  const started = Date.now();
  const msToRun = durationConvert(input.runFor);
  console.log('Running resources activity for', msToRun);
  const buffer = Buffer.alloc((input.bytesToAllocate ?? new Long(0)).toNumber());
  let i = 0;
  while (Date.now() - started < msToRun) {
    if (input.cpuYieldEveryNIterations && i % input.cpuYieldEveryNIterations === 0) {
      await new Promise((resolve) => setTimeout(resolve, input.cpuYieldForMs ?? 1));
    }
    buffer[i % buffer.length] = Math.random() * 256;
    i++;
  }
  console.log(buffer[buffer.length]);
}

resourcesImpl(workerData)
  .then(() => undefined)
  .catch((err) => console.error('Resources worker failed:', err));
