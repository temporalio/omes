import express, { Request, Response, NextFunction } from 'express';
import { Connection, Client, ClientOptions } from '@temporalio/client';
import { parseArgs } from 'node:util';
import { ClientConfig, ClientFunction, ExecuteFunction } from './common';

export interface ClientStarterArgs {
    port: number;
    taskQueue: string;
    serverAddress: string;
    namespace: string;
}

export class OmesClientStarter {
    private clientOptions: Partial<ClientOptions>;
    private executeFn?: ClientFunction;
    private client?: Client;
    private taskQueue?: string;
    private server?: ReturnType<express.Application['listen']>;
    private activeRequests = 0;
    private shuttingDown = false;

    constructor(clientOptions: Partial<ClientOptions> = {}) {
        this.clientOptions = clientOptions;
    }

    onExecute(fn: ClientFunction): void {
        this.executeFn = fn;
    }

    private async handleExecute(req: Request, res: Response): Promise<void> {
        if (this.shuttingDown) {
            res.json({
                success: false,
                error: 'Starter is shutting down',
            });
            return;
        }

        this.activeRequests++;
        try {
            const config: ClientConfig = {
                client: this.client!,
                taskQueue: this.taskQueue!,
                runId: req.body.run_id,
                iteration: req.body.iteration,
            };

            await this.executeFn!(config);
            res.json({ success: true });
        } catch (error) {
            res.json({
                success: false,
                error: error instanceof Error ? error.message : String(error),
                traceback: error instanceof Error ? error.stack : undefined,
            });
        } finally {
            this.activeRequests--;
        }
    }

    private async handleShutdown(req: Request, res: Response): Promise<void> {
        const drainTimeoutMs = req.body?.drain_timeout_ms ?? 30000;
        res.json({ status: 'shutting_down' });
        this.shutdownWithDrain(drainTimeoutMs);
    }

    private async shutdownWithDrain(timeoutMs: number): Promise<void> {
        this.shuttingDown = true;
        const start = Date.now();

        // Wait for active requests to complete
        while (this.activeRequests > 0 && Date.now() - start < timeoutMs) {
            await new Promise((resolve) => setTimeout(resolve, 100));
        }

        if (this.activeRequests > 0) {
            console.warn(
                `Forcing shutdown with ${this.activeRequests} active requests`
            );
        }

        this.server?.close();
        process.exit(0);
    }

    private async handleInfo(_req: Request, res: Response): Promise<void> {
        // Get SDK version from the loaded module
        let sdkVersion = 'unknown';
        try {
            // eslint-disable-next-line @typescript-eslint/no-var-requires
            const pkg = require('@temporalio/client/package.json');
            sdkVersion = pkg.version;
        } catch {
            // Fallback if package.json not accessible
        }
        res.json({
            sdk_language: 'typescript',
            sdk_version: sdkVersion,
            starter_version: '0.1.0',
        });
    }

    async run(): Promise<void> {
        const { values } = parseArgs({
            options: {
                port: { type: 'string', default: '8080' },
                'task-queue': { type: 'string' },
                'server-address': { type: 'string', default: 'localhost:7233' },
                namespace: { type: 'string', default: 'default' },
            },
        });

        if (!values['task-queue']) {
            throw new Error('--task-queue is required');
        }

        await this._runWithArgs({
            port: parseInt(values.port!, 10),
            taskQueue: values['task-queue'],
            serverAddress: values['server-address']!,
            namespace: values.namespace!,
        });
    }

    /**
     * Start the HTTP server with pre-parsed args.
     * Used by cli.ts's run() function to pass already-parsed arguments.
     */
    async _runWithArgs(args: ClientStarterArgs): Promise<void> {
        this.taskQueue = args.taskQueue;

        // Create Temporal client at startup
        const connection = await Connection.connect({
            address: args.serverAddress,
        });

        this.client = new Client({
            connection,
            namespace: args.namespace,
            ...this.clientOptions,
        });

        const app = express();
        app.use(express.json());

        // Async error wrapper for all handlers
        const asyncHandler =
            (fn: (req: Request, res: Response) => Promise<void>) =>
            (req: Request, res: Response, _next: NextFunction) =>
                fn.call(this, req, res).catch((err) => {
                    res.json({ success: false, error: String(err) });
                });

        app.post('/execute', asyncHandler(this.handleExecute));
        app.post('/shutdown', asyncHandler(this.handleShutdown));
        app.get('/info', asyncHandler(this.handleInfo));

        this.server = app.listen(args.port, () => {
            console.log(`Client starter listening on port ${args.port}`);
        });
    }
}

/**
 * Convenience function to run a client with the given execute handler.
 *
 * Note: Prefer using run() from cli.ts with the new subcommand pattern.
 *
 * @example
 * // src/client.ts - user exports function
 * export async function clientMain(config: ClientConfig): Promise<void> {
 *     const handle = await config.client.workflow.start(...);
 *     await handle.result();
 * }
 *
 * // Call directly:
 * runClient(clientMain);
 *
 * @param handler - Async function called for each /execute request
 * @param clientOptions - Options passed to Client constructor
 */
export function runClient(
    handler: ClientFunction,
    clientOptions: Partial<ClientOptions> = {}
): void {
    const starter = new OmesClientStarter(clientOptions);
    starter.onExecute(handler);
    starter.run().catch((err) => {
        console.error(err);
        process.exit(1);
    });
}
