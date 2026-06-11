import { ClientFactory } from './client';
import { ProjectHandlers, runProjectServerCli } from './project';
import { WorkerFactory, runWorkerCli } from './worker';

export interface App {
  worker: WorkerFactory;
  clientFactory: ClientFactory;
  project?: ProjectHandlers;
}

export async function run(
  app: App,
  argv: readonly string[] = process.argv.slice(2),
): Promise<void> {
  if (argv[0] === 'worker') {
    await runWorkerCli(app.worker, app.clientFactory, argv.slice(1));
    return;
  }

  if (argv[0] === 'project-server') {
    if (app.project === undefined) {
      throw new Error('Wanted project-server but no project handlers registered for this app');
    }
    await runProjectServerCli(app.project, app.clientFactory, argv.slice(1));
    return;
  }

  throw new Error(`Expected 'worker' or 'project-server', got ${argv.slice(0, 1)}`);
}
