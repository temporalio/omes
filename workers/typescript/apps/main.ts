import type { App } from '../harness';
import { run as runApp } from '../harness';
import { app as workerApp } from './worker/app';

export const defaultAppName = 'worker';

const registry: Readonly<Record<string, App>> = {
  worker: workerApp,
};

export interface AppArgs {
  appName: string;
  args: readonly string[];
}

export type AppRunner = (app: App, argv: readonly string[]) => Promise<void>;

export function parseAppArgs(argv: readonly string[]): AppArgs {
  let appName = defaultAppName;
  let index = 0;

  while (index < argv.length) {
    const arg = argv[index];
    if (arg === '--app') {
      const value = argv[index + 1];
      if (!value) {
        throw new Error('--app requires an app name');
      }
      appName = value;
      index += 2;
      continue;
    }
    if (arg?.startsWith('--app=')) {
      const value = arg.slice('--app='.length);
      if (!value) {
        throw new Error('--app requires an app name');
      }
      appName = value;
      index += 1;
      continue;
    }
    break;
  }

  return {
    appName,
    args: argv.slice(index),
  };
}

export function resolveApp(appName: string): App {
  const app = registry[appName];
  if (app === undefined) {
    throw new Error(
      `Unknown TypeScript worker app ${JSON.stringify(appName)}. Available apps: ${availableApps().join(', ')}`,
    );
  }
  return app;
}

export function availableApps(): string[] {
  return Object.keys(registry).sort();
}

export async function run(
  argv: readonly string[] = process.argv.slice(2),
  appRunner: AppRunner = runApp,
): Promise<void> {
  const { appName, args } = parseAppArgs(argv);
  await appRunner(resolveApp(appName), args);
}

if (require.main === module) {
  void run()
    .then(() => {
      process.exit(0);
    })
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
}
