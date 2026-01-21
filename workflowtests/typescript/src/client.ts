import express, { Request, Response, NextFunction } from 'express';
import { Connection, Client, ClientOptions } from '@temporalio/client';
import { parseArgs } from 'node:util';
import { ExecuteContext, ExecuteFunction } from './common';

export class OmesClientStarter {
    private clientOptions: Partial<ClientOptions>;
    private executeFn?: ExecuteFunction;
    private client?: Client;
    private taskQueue?: string;
    private server?: ReturnType<express.Application['listen']>;
    private activeRequests = 0;
    private shuttingDown = false;

    constructor(clientOptions: Partial<ClientOptions> = {}) {
        this.clientOptions = clientOptions;
    }

    onExecute(fn: ExecuteFunction): void {
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
            const ctx: ExecuteContext = {
                iteration: req.body.iteration,
                runId: req.body.run_id,
                taskQueue: this.taskQueue!,
                client: this.client!,
            };

            await this.executeFn!(ctx);
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

        const port = parseInt(values.port!, 10);
        this.taskQueue = values['task-queue'];

        // Create Temporal client at startup
        const connection = await Connection.connect({
            address: values['server-address'],
        });

        this.client = new Client({
            connection,
            namespace: values.namespace,
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

        this.server = app.listen(port, () => {
            console.log(`Client starter listening on port ${port}`);
        });
    }
}
