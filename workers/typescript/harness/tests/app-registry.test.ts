import assert from 'node:assert/strict';
import test from 'node:test';
import type { App } from '../../harness';
import { app as workerApp } from '../../apps/worker/app';
import { availableApps, parseAppArgs, resolveApp, run } from '../../apps/main';

void test('parseAppArgs defaults to worker and forwards harness args', () => {
  assert.deepEqual(parseAppArgs(['worker', '--task-queue', 'omes']), {
    appName: 'worker',
    args: ['worker', '--task-queue', 'omes'],
  });
});

void test('parseAppArgs accepts space-separated app flag', () => {
  assert.deepEqual(parseAppArgs(['--app', 'worker', 'worker']), {
    appName: 'worker',
    args: ['worker'],
  });
});

void test('parseAppArgs accepts equals app flag', () => {
  assert.deepEqual(parseAppArgs(['--app=worker', 'project-server', '--port', '8080']), {
    appName: 'worker',
    args: ['project-server', '--port', '8080'],
  });
});

void test('resolveApp returns registered worker app', () => {
  assert.strictEqual(resolveApp('worker'), workerApp);
  assert.deepEqual(availableApps(), ['worker']);
});

void test('resolveApp rejects unknown app names', () => {
  assert.throws(() => resolveApp('missing'), /Unknown TypeScript worker app "missing"/);
});

void test('run dispatches selected app and forwards remaining args', async () => {
  let receivedApp: App | undefined;
  let receivedArgs: readonly string[] | undefined;

  await run(['--app=worker', 'worker', '--task-queue', 'omes'], async (app, argv) => {
    receivedApp = app;
    receivedArgs = argv;
  });

  assert.strictEqual(receivedApp, workerApp);
  assert.deepEqual(receivedArgs, ['worker', '--task-queue', 'omes']);
});
