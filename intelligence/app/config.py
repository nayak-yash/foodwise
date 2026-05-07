"""Validated process configuration, injected into application services."""

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    database_url: str
    agent_token: str
    backend_url: str = "http://backend:8080"
    kafka_brokers: str = "kafka:9092"
    branch_id: str = "11111111-1111-1111-1111-111111111111"
    retrieval_mode: str = "fastembed"
    openai_model: str = "gpt-4o-mini"
    openai_api_key: str = ""
    verification_window_seconds: float = 600
    recovery_deadline_seconds: float = 7200
    otel_endpoint: str = ""

    @classmethod
    def from_environment(cls) -> "Settings":
        return cls(
            database_url=os.environ["DATABASE_URL"],
            agent_token=os.environ["AGENT_API_TOKEN"],
            backend_url=os.getenv("BACKEND_URL", "http://backend:8080"),
            kafka_brokers=os.getenv("KAFKA_BROKERS", "kafka:9092"),
            retrieval_mode=os.getenv("RETRIEVAL_MODE", "fastembed"),
            openai_model=os.getenv("OPENAI_MODEL", "gpt-4o-mini"),
            openai_api_key=os.getenv("OPENAI_API_KEY", ""),
            verification_window_seconds=float(os.getenv("VERIFY_WINDOW_SECONDS", "600")),
            otel_endpoint=os.getenv("OTEL_EXPORTER_OTLP_ENDPOINT", ""),
        )
