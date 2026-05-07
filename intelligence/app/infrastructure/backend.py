import hashlib
import json

import httpx

from ..config import Settings


class BackendClient:
    """Typed service boundary for authorized operational reads and mutations."""

    def __init__(self, settings: Settings):
        self.http = httpx.AsyncClient(
            base_url=settings.backend_url + "/api/v1",
            headers={"Authorization": f"Bearer {settings.agent_token}"},
            timeout=20,
        )

    async def close(self):
        await self.http.aclose()

    async def get(self, path):
        response = await self.http.get(path)
        response.raise_for_status()
        return response.json()

    async def post(self, path, body, key=None):
        encoded = json.dumps(body, sort_keys=True, separators=(",", ":"))
        key = key or "agent-" + hashlib.sha256((path + encoded).encode()).hexdigest()
        response = await self.http.post(
            path,
            content=encoded,
            headers={"Content-Type": "application/json", "Idempotency-Key": key},
        )
        response.raise_for_status()
        return response.json()
