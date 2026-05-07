"""Persistent incident workflow. Nodes are explicit methods with injected services."""

import json
from datetime import date, datetime, timezone

import httpx
from langgraph.graph import END, START, StateGraph
from langgraph.types import interrupt
from opentelemetry import trace

from ..config import Settings
from ..domain.models import InvestigationState
from ..infrastructure.backend import BackendClient
from ..infrastructure.knowledge import KnowledgeRepository
from ..services.diagnosis import DiagnosisService
from ..services.forecasting import ForecastService
from ..services.reasoning import ReasoningService
from ..services.recovery import RecoveryVerifier, parse_time

tracer = trace.get_tracer(__name__)


class InvestigationWorkflow:
    def __init__(
        self,
        settings: Settings,
        backend: BackendClient,
        knowledge: KnowledgeRepository,
        forecaster: ForecastService,
        diagnosis: DiagnosisService,
        reasoning: ReasoningService,
        verifier: RecoveryVerifier,
    ):
        self.settings = settings
        self.backend = backend
        self.knowledge = knowledge
        self.forecaster = forecaster
        self.diagnosis = diagnosis
        self.reasoning = reasoning
        self.verifier = verifier

    def compile(self, checkpointer):
        graph = StateGraph(InvestigationState)
        nodes = {
            "begin": self.begin,
            "collect": self.collect,
            "reason": self.reason,
            "publish": self.publish,
            "plan": self.plan,
            "await_approval": self.await_approval,
            "execute": self.execute,
            "verify": self.verify_recovery,
        }
        for name, node in nodes.items():
            graph.add_node(name, node)
        graph.add_edge(START, "begin")
        for source, target in [
            ("begin", "collect"),
            ("collect", "reason"),
            ("reason", "publish"),
            ("publish", "plan"),
        ]:
            graph.add_edge(source, target)
        graph.add_conditional_edges(
            "plan", lambda state: "await_approval" if state.get("action_id") else END
        )
        graph.add_conditional_edges(
            "await_approval", lambda state: "execute" if state.get("approved") else END
        )
        graph.add_conditional_edges(
            "execute", lambda state: "verify" if state.get("executable") else END
        )
        graph.add_conditional_edges("verify", self._after_verification)
        return graph.compile(checkpointer=checkpointer)

    @staticmethod
    def _after_verification(state):
        recovery = state.get("recovery", {})
        return END if recovery.get("recovered") or recovery.get("expired") else "verify"

    async def begin(self, state):
        await self.backend.post(
            f"/incidents/{state['incident_id']}/report",
            {"status": "investigating"},
            key=f"begin-{state['incident_id']}",
        )
        return {"attempts": 0}

    async def collect(self, state):
        with tracer.start_as_current_span("investigation.collect"):
            evidence = await self.backend.get(f"/incidents/{state['incident_id']}/evidence")
            history = await self.backend.get("/demand-history")
            prediction = self.forecaster.forecast(
                history, evidence["purchase_order"]["ingredient_id"], date.today()
            )
            retrieval = await self.knowledge.search(
                f"{evidence['purchase_order']['ingredient']} delivery shortage completed partial receipt stock recovery"
            )
            return {"evidence": evidence, "forecast": prediction, "retrieval": retrieval}

    async def reason(self, state):
        with tracer.start_as_current_span("investigation.reason"):
            diagnosis = self.diagnosis.diagnose(state["evidence"], state["forecast"])
            diagnosis["retrieval"] = state["retrieval"]
            diagnosis = await self.reasoning.enrich(
                diagnosis, state["incident_id"], state["forecast"]
            )
            return {"diagnosis": diagnosis}

    async def publish(self, state):
        await self.backend.post(
            f"/incidents/{state['incident_id']}/report",
            {"status": "investigating", "diagnosis": state["diagnosis"]},
            key=f"diagnosis-{state['incident_id']}",
        )
        return {}

    async def plan(self, state):
        if (
            state["diagnosis"]["confidence"] == "insufficient"
            or state["diagnosis"]["missing_kg"] <= 0
        ):
            await self._escalate(state, "Evidence insufficient for automatic replacement planning")
            return {"action_id": ""}
        try:
            action = await self.backend.post(
                f"/incidents/{state['incident_id']}/plan",
                {
                    "quantity": state["diagnosis"]["missing_kg"],
                    "reason": "Source replacement stock for the shortfall supported by receiving records.",
                },
                key=f"plan-{state['incident_id']}",
            )
        except httpx.HTTPStatusError as error:
            if error.response.status_code != 409:
                raise
            await self._escalate(
                state, "Receiving evidence changed during planning; manager review required."
            )
            return {"action_id": ""}
        return {"action_id": action["id"]}

    async def await_approval(self, state):
        interrupt({"kind": "manager_approval", "action_id": state["action_id"]})
        incident = await self.backend.get(f"/incidents/{state['incident_id']}")
        action = next(
            action for action in incident["actions"] if action["id"] == state["action_id"]
        )
        return {"approved": action["status"] in ("approved", "executed", "received")}

    async def execute(self, state):
        try:
            await self.backend.post(
                f"/actions/{state['action_id']}/execute", {}, key=f"execute-{state['action_id']}"
            )
        except httpx.HTTPStatusError as error:
            if error.response.status_code != 409:
                raise
            await self._escalate(
                state,
                "Approved action no longer matches current stock situation; manager review required.",
            )
            return {"executable": False}
        return {"executable": True}

    async def verify_recovery(self, state):
        evidence = await self.backend.get(f"/incidents/{state['incident_id']}/evidence")
        if evidence["incident"]["status"] == "escalated":
            return {"recovery": {**evidence["incident"]["recovery"], "expired": True}}
        if evidence["incident"]["status"] == "resolved":
            result = {**evidence["incident"]["recovery"], "recovered": True}
            await self._remember(state, result)
            return {"recovery": result}

        result = self.verifier.verify(evidence)
        if result["recovered"]:
            await self.backend.post(
                f"/incidents/{state['incident_id']}/report",
                {"status": "resolved", "recovery": result},
                key=f"resolved-{state['incident_id']}",
            )
            await self._remember(state, result)
            return {"recovery": result}
        if self._deadline_exceeded(evidence):
            return await self._reopen_and_escalate(state, evidence, result)

        previous = evidence["incident"].get("recovery", {})
        if any(
            previous.get(key) != result.get(key)
            for key in ("checks", "balance_kg", "sample_count", "observation_seconds")
        ):
            await self.backend.post(
                f"/incidents/{state['incident_id']}/report",
                {"status": "verifying", "recovery": result},
            )
        interrupt({"kind": "recovery_observation", "checks": result["checks"]})
        return {"recovery": result, "attempts": state.get("attempts", 0) + 1}

    def _deadline_exceeded(self, evidence):
        executed = [
            parse_time(action["executed_at"])
            for action in evidence["actions"]
            if action.get("executed_at")
        ]
        return (
            bool(executed)
            and (datetime.now(timezone.utc) - min(executed)).total_seconds()
            > self.settings.recovery_deadline_seconds
        )

    async def _reopen_and_escalate(self, state, evidence, result):
        result["expired"] = True
        await self.backend.post(
            f"/incidents/{state['incident_id']}/report",
            {
                "status": "investigating",
                "recovery": {
                    **result,
                    "reason": "Recovery deadline exceeded; investigation reopened.",
                },
            },
        )
        updated = self.diagnosis.diagnose(evidence, state["forecast"])
        updated["summary"] += (
            " Corrective action did not produce verified recovery within two hours. Manager intervention required."
        )
        await self.backend.post(
            f"/incidents/{state['incident_id']}/report",
            {"status": "escalated", "diagnosis": updated, "recovery": result},
        )
        return {"recovery": result}

    async def _escalate(self, state, reason):
        await self.backend.post(
            f"/incidents/{state['incident_id']}/report",
            {"status": "escalated", "recovery": {"reason": reason}},
        )

    async def _remember(self, state, recovery):
        diagnosis = state["diagnosis"]
        summary = {
            "root_cause": diagnosis["root_cause"],
            "summary": diagnosis["summary"],
            "missing_kg": diagnosis["missing_kg"],
            "action": "Replacement stock requested, approved, and actually received",
            "verified_recovery": recovery,
        }
        await self.knowledge.remember(
            state["incident_id"],
            json.dumps(summary),
        )
