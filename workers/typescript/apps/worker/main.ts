import { run as runApp } from '../../harness';
import { app } from './app';

async function main() {
  await runApp(app);
}

main()
  .then(() => {
    process.exit(0);
  })
  .catch((err) => {
    console.error(err);
    process.exit(1);
  });
