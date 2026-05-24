import { Command } from 'commander';
import type { App } from '../harness';
import { run } from '../harness';
import { app as workerApp } from './worker/app';

const defaultAppName = 'worker';

const registry: Record<string, App> = {
  worker: workerApp,
};

interface RegistryOptions {
  app: string;
}

function main(argv: string[] = process.argv.slice(2)): Promise<void> {
  const program = new Command()
    .passThroughOptions()
    .option('--app <app>', 'TypeScript worker app', defaultAppName);
  const options = program.parse(argv, { from: 'user' }).opts<RegistryOptions>();
  const appName = options.app;
  const app = registry[appName];
  if (app === undefined || appName === '') {
    throw new Error(`unknown TypeScript worker app ${JSON.stringify(appName)}`);
  }

  return run(app, program.args);
}

if (require.main === module) {
  void main().catch((err) => {
    console.error(err);
    process.exit(1);
  });
}
