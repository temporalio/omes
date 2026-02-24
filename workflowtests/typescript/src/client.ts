import express, { Request, Response } from 'express';
import type { ConnectionOptions } from '@temporalio/client';
import { ClientConfig, ClientFunction } from './common';

export interface ClientStarterArgs {
    port: number;
    taskQueue: string;
    serverAddress: string;
    namespace: string;
    authHeader?: string;
    tls?: boolean;
    tlsServerName?: string;
}

export class OmesClientStarter {
    private executeFn?: ClientFunction;
    private baseConfig?: Omit<ClientConfig, 'runId' | 'iteration'>;

    onExecute(fn: ClientFunction): void {
        this.executeFn = fn;
    }

    private async handleExecute(req: Request, res: Response): Promise<void> {
        try {
            const config: ClientConfig = {
                ...this.baseConfig!,
                runId: req.body.run_id,
                iteration: req.body.iteration,
            };
            await this.executeFn!(config);
            res.json({ success: true });
        } catch (error) {
            res.json({
                success: false,
                error: error instanceof Error ? error.message : String(error),
            });
        }
    }

    /**
     * Start the HTTP server with pre-parsed args.
     * Used by cli.ts's run() function to pass already-parsed arguments.
     */
    async _runWithArgs(args: ClientStarterArgs): Promise<void> {
        const connectionOptions: ConnectionOptions = { address: args.serverAddress };
        if (args.authHeader) {
            connectionOptions.metadata = { Authorization: args.authHeader };
        }
        if (args.tlsServerName) {
            connectionOptions.tls = { serverNameOverride: args.tlsServerName };
        } else if (args.tls) {
            connectionOptions.tls = {};
        }
        this.baseConfig = {
            namespace: args.namespace,
            connectionOptions,
            taskQueue: args.taskQueue,
        };

        const app = express();
        app.use(express.json());

        app.post('/execute', (req: Request, res: Response) => {
            void this.handleExecute(req, res);
        });
        app.get('/info', (_req: Request, res: Response) => {
            res.json({});
        });

        await new Promise<void>((resolve, reject) => {
            const server = app.listen(args.port, () => {
                console.log(`Client starter listening on port ${args.port}`);
                resolve();
            });
            server.on('error', reject);
        });
    }
}
