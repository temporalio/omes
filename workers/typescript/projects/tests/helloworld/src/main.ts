import { run } from '@temporalio/omes-project-harness';
import { app } from './app';

async function main(): Promise<void> {
  await run(app());
}

void main().catch((err: unknown) => {
  console.error(err);
  process.exit(1);
});
