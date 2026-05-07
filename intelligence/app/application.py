"""Composition root and lifecycle ownership for the intelligence service."""

import asyncio
import contextlib
from contextlib import AsyncExitStack
from dataclasses import dataclass

from langgraph.checkpoint.postgres.aio import AsyncPostgresSaver
from prometheus_client import CollectorRegistry, Counter

from .config import Settings
from .infrastructure.backend import BackendClient
from .infrastructure.knowledge import KnowledgeRepository
from .services.diagnosis import DiagnosisService
from .services.forecasting import ForecastService
from .services.reasoning import ReasoningService
from .services.recovery import RecoveryVerifier
from .workers.consumer import OperationalEventConsumer
from .workers.coordinator import InvestigationCoordinator
from .workflows.investigation import InvestigationWorkflow


@dataclass
class ApplicationServices:
    settings: Settings
    backend: BackendClient
    knowledge: KnowledgeRepository
    forecaster: ForecastService
    coordinator: InvestigationCoordinator
    consumer: OperationalEventConsumer
    metrics: CollectorRegistry


class IntelligenceApplication:
    def __init__(self, settings: Settings):
        self.settings = settings
        self.resources = AsyncExitStack()
        self.tasks = []

    async def start(self) -> ApplicationServices:
        backend = BackendClient(self.settings)
        self.resources.push_async_callback(backend.close)
        knowledge = KnowledgeRepository(self.settings)
        forecaster = ForecastService()
        saver = await self.resources.enter_async_context(
            AsyncPostgresSaver.from_conn_string(self.settings.database_url)
        )
        await saver.setup()
        workflow = InvestigationWorkflow(
            self.settings,
            backend,
            knowledge,
            forecaster,
            DiagnosisService(),
            ReasoningService(self.settings, backend, knowledge),
            RecoveryVerifier(self.settings.verification_window_seconds),
        )
        registry = CollectorRegistry()
        events = Counter(
            "restaurantops_agent_events_total",
            "Agent event processing outcomes",
            ["outcome"],
            registry=registry,
        )
        coordinator = InvestigationCoordinator(
            self.settings, backend, workflow.compile(saver), events
        )
        consumer = OperationalEventConsumer(self.settings, coordinator, events)
        self.tasks = [
            asyncio.create_task(consumer.run(), name="operational-events"),
            asyncio.create_task(coordinator.run(), name="checkpoint-reconciliation"),
            asyncio.create_task(knowledge.initialize(), name="knowledge-index"),
        ]
        return ApplicationServices(
            self.settings, backend, knowledge, forecaster, coordinator, consumer, registry
        )

    async def close(self):
        for task in self.tasks:
            task.cancel()
        for task in self.tasks:
            with contextlib.suppress(asyncio.CancelledError):
                await task
        await self.resources.aclose()
