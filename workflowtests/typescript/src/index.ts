// Main entry point
export { run, RunOptions } from './cli';

// Config types
export { ClientConfig, WorkerConfig, ClientFunction, WorkerFunction } from './common';

// Client starter (used by cli.ts)
export { OmesClientStarter, ClientStarterArgs } from './client';

// Optional helper for sharing client connections across execute calls
export { ClientPool } from './client_pool';
