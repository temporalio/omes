import type { App } from '../harness';
import { run } from '../harness';
import { app as workerApp } from './worker/app';

const defaultAppName = 'worker';

const registry: Record<string, App> = {
  worker: workerApp,
};

function main(argv: readonly string[] = process.argv.slice(2)): Promise<void> {
  let appName = defaultAppName;
  let index = 0;

  if (argv[0] === '--app') {
    appName = argv[1] ?? '';
    index = 2;
  } else if (argv[0]?.startsWith('--app=')) {
    appName = argv[0].slice('--app='.length);
    index = 1;
  }

  const app = registry[appName];
  if (app === undefined || appName === '') {
    throw new Error(`unknown TypeScript worker app ${JSON.stringify(appName)}`);
  }

  return run(app, argv.slice(index));
}

if (require.main === module) {
  void main()
    .then(() => {
      process.exit(0);
    })
    .catch((err) => {
      console.error(err);
      process.exit(1);
    });
}
