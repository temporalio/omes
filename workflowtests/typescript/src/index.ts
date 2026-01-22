// Main entry point
export { run, RunOptions } from './cli';

// Config types
export { ClientConfig, WorkerConfig, ClientFunction, WorkerFunction } from './common';

// Backwards compatibility aliases
export { ExecuteContext, WorkerContext, ExecuteFunction, ConfigureWorkerFunction } from './common';

// Class-based API (less preferred)
export { OmesClientStarter, runClient, ClientStarterArgs } from './client';
export { OmesWorkerStarter, runWorker } from './worker';
