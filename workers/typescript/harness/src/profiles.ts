import type { WorkerOptions } from '@temporalio/worker';

export const WORKER_PROFILE_ENV_VAR = 'OMES_WORKER_PROFILE';
export const RESOURCE_BASED_DEFAULT_PROFILE = 'resource-based-default';

export type WorkerProfile = Readonly<Partial<WorkerOptions>>;

const profiles = new Map<string, WorkerProfile>();

function registerWorkerProfile(name: string, profile: WorkerProfile): void {
  profiles.set(name, profile);
}

export function lookupWorkerProfile(name: string): Partial<WorkerOptions> {
  const profile = profiles.get(name);
  if (profile === undefined) {
    throw new Error(`Unknown worker profile "${name}"`);
  }
  return { ...profile };
}

registerWorkerProfile(RESOURCE_BASED_DEFAULT_PROFILE, {
  tuner: {
    tunerOptions: {
      targetMemoryUsage: 0.8,
      targetCpuUsage: 0.8,
    },
  },
});
