/**
 * Entry point for simple-test workflow load testing.
 *
 * This file calls run() with the client and worker handlers.
 * It is invoked by omes with a subcommand:
 *     node dist/main.js client --port 8080 --task-queue omes-xxx ...
 *     node dist/main.js worker --task-queue omes-xxx ...
 */

import { run } from '@temporalio/omes-starter';
import { clientMain } from './src/client';
import { workerMain } from './src/worker';

run({ client: clientMain, worker: workerMain });
