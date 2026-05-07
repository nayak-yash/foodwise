import asyncio
import json
import logging

import psycopg
from aiokafka import AIOKafkaConsumer

from ..config import Settings
from .coordinator import InvestigationCoordinator

logger = logging.getLogger(__name__)


class OperationalEventConsumer:
    TRIGGER_EVENTS = {
        "delivery.received",
        "incident.detected",
        "action.approved",
        "action.rejected",
        "stock.received",
        "kitchen.measured",
    }
    CONSUMER_ID = "investigator-v1"

    def __init__(self, settings: Settings, coordinator: InvestigationCoordinator, metrics):
        self.settings = settings
        self.coordinator = coordinator
        self.metrics = metrics
        self.ready = False
        self.last_error = None

    async def run(self):
        consumer = AIOKafkaConsumer(
            "restaurantops.events",
            bootstrap_servers=self.settings.kafka_brokers,
            group_id="restaurantops-investigator-v1",
            enable_auto_commit=False,
            auto_offset_reset="earliest",
            max_poll_interval_ms=600000,
        )
        try:
            await consumer.start()
            self.ready = True
            async for message in consumer:
                await self._process_with_retry(message, consumer)
        finally:
            self.ready = False
            await consumer.stop()

    async def _process_with_retry(self, message, consumer):
        while True:
            try:
                event = json.loads(message.value)
                if event.get("branch_id") == self.settings.branch_id:
                    await self._process_event(event)
                await consumer.commit()
                self.metrics.labels("processed").inc()
                return
            except asyncio.CancelledError:
                raise
            except Exception as error:
                self.last_error = str(error)
                self.metrics.labels("retry").inc()
                logger.exception("Event not acknowledged; retrying same event")
                await asyncio.sleep(5)

    async def _process_event(self, event):
        async with await psycopg.AsyncConnection.connect(self.settings.database_url) as connection:
            existing = await (
                await connection.execute(
                    "SELECT 1 FROM processed_events WHERE consumer=%s AND event_id=%s",
                    (self.CONSUMER_ID, event["id"]),
                )
            ).fetchone()
            if existing:
                return
            if event["type"] in self.TRIGGER_EVENTS:
                await self.coordinator.reconcile_all()
            await connection.execute(
                "INSERT INTO processed_events(consumer,event_id) VALUES(%s,%s) ON CONFLICT DO NOTHING",
                (self.CONSUMER_ID, event["id"]),
            )
