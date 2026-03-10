import { Connection } from '@temporalio/client';
import type { ConnectionOptions } from '@temporalio/client';

/**
 * Caches Temporal client connections by user-provided key for the process lifetime.
 *
 * Reuses in-flight connects for the same key so concurrent callers do not create
 * duplicate connections.
 */
export class ClientPool {
    private readonly connections = new Map<string, Promise<Connection>>();

    async getOrConnect(key: string, connectionOptions: ConnectionOptions): Promise<Connection> {
        if (!key.trim()) {
            throw new Error('client pool key cannot be empty');
        }

        const existing = this.connections.get(key);
        if (existing) {
            return existing;
        }

        const connectPromise = Connection.connect(connectionOptions).catch((err) => {
            this.connections.delete(key);
            throw err;
        });
        this.connections.set(key, connectPromise);
        return connectPromise;
    }
}
