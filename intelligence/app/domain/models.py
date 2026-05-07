from typing import TypedDict

from pydantic import BaseModel, Field


class InvestigationState(TypedDict, total=False):
    incident_id: str
    evidence: dict
    forecast: dict
    retrieval: dict
    diagnosis: dict
    action_id: str
    recovery: dict
    approved: bool
    attempts: int
    executable: bool


class ModelAnalysis(BaseModel):
    explanation: str = Field(
        description="Evidence-grounded explanation, distinguishing facts from hypotheses"
    )
    additional_checks: list[str]
    operational_recommendations: list[str]
