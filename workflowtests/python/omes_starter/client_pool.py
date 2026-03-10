import asyncio

from temporalio.client import Client


class ClientPool:
    """Caches Temporal clients by user-provided key for the process lifetime."""

    def __init__(self):
        self._clients: dict[str, Client] = {}
        self._lock = asyncio.Lock()

    async def get_or_connect(self, key: str, target_host: str, **kwargs) -> Client:
        if not key:
            raise ValueError("client pool key cannot be empty")

        async with self._lock:
            existing = self._clients.get(key)
            if existing is not None:
                return existing

            client = await Client.connect(target_host, **kwargs)
            self._clients[key] = client
            return client
