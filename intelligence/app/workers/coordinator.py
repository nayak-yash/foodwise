import asyncio
import logging

from langgraph.types import Command

from ..config import Settings
from ..infrastructure.backend import BackendClient

logger = logging.getLogger(__name__)


class InvestigationCoordinator:
    """Serializes local graph work and resumes durable incident checkpoints."""

    def __init__(self, settings: Settings, backend: BackendClient, graph, metrics):
        self.settings = settings
        self.backend = backend
        self.graph = graph
        self.metrics = metrics
        self.lock = asyncio.Lock()
        self.last_error = None

    async def reconcile_all(self):
        async with self.lock:
            await self._detect_short_deliveries()
            for incident in await self.backend.get("/incidents"):
                try:
                    await self._reconcile_incident(incident)
                except Exception as error:
                    self.last_error = str(error)
                    self.metrics.labels("investigation_retry").inc()
                    logger.exception("Investigation will retry: %s", incident["id"])

    async def run(self):
        while True:
            try:
                await self.reconcile_all()
            except Exception as error:
                self.last_error = str(error)
                logger.exception("Reconciliation failed; will retry")
            await asyncio.sleep(15)

    async def _detect_short_deliveries(self):
        for purchase_order in await self.backend.get("/purchase-orders"):
            if (
                purchase_order["external_status"] != "completed"
                or purchase_order["outstanding"] <= 0
            ):
                continue
            await self.backend.post(
                "/incidents",
                {
                    "purchase_order_id": purchase_order["id"],
                    "fingerprint": f"delivery-shortfall:{purchase_order['id']}",
                    "title": f"{purchase_order['ingredient']} delivery shortfall",
                    "severity": "high",
                },
                key=f"detect-{purchase_order['id']}",
            )

    async def _reconcile_incident(self, incident):
        config = {
            "configurable": {"thread_id": f"{self.settings.branch_id}:{incident['id']}"},
            "recursion_limit": 50,
        }
        snapshot = await self.graph.aget_state(config)
        if not snapshot.values:
            if incident["status"] == "detected":
                await self.graph.ainvoke({"incident_id": incident["id"]}, config)
            return
        if not snapshot.next:
            return
        interrupted = any(task.interrupts for task in snapshot.tasks)
        if interrupted:
            if "await_approval" in snapshot.next and incident["status"] not in (
                "approved",
                "escalated",
            ):
                return
            await self.graph.ainvoke(Command(resume=True), config)
        else:
            await self.graph.ainvoke(None, config)
