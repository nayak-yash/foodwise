import secrets
from datetime import date

from fastapi import APIRouter, Depends, Header, HTTPException, Request, Response
from prometheus_client import CONTENT_TYPE_LATEST, generate_latest

from .application import ApplicationServices

router = APIRouter()


def services(request: Request) -> ApplicationServices:
    return request.app.state.services


def authorized_services(
    application: ApplicationServices = Depends(services),
    authorization: str | None = Header(default=None),
) -> ApplicationServices:
    expected = f"Bearer {application.settings.agent_token}"
    if not authorization or not secrets.compare_digest(authorization, expected):
        raise HTTPException(401, "valid service token required")
    return application


@router.get("/healthz")
async def health():
    return {"status": "ok"}


@router.get("/readyz")
async def ready(response: Response, application: ApplicationServices = Depends(services)):
    response.status_code = 200 if application.consumer.ready else 503
    return {"status": "ready" if application.consumer.ready else "starting"}


@router.get("/metrics")
async def metrics(application: ApplicationServices = Depends(services)):
    return Response(generate_latest(application.metrics), media_type=CONTENT_TYPE_LATEST)


@router.get("/status")
async def status(application: ApplicationServices = Depends(authorized_services)):
    return {
        "ready": application.consumer.ready,
        "retrieval_mode": application.knowledge.mode,
        "embedding_error": application.knowledge.error,
        "reasoning_mode": "llm_assisted"
        if application.settings.openai_api_key
        else "deterministic",
        "last_error": application.consumer.last_error or application.coordinator.last_error,
        "verification_window_seconds": application.settings.verification_window_seconds,
    }


@router.get("/forecasts")
async def forecasts(
    target: date | None = None, application: ApplicationServices = Depends(authorized_services)
):
    history = await application.backend.get("/demand-history")
    inventory = await application.backend.get("/inventory")
    target = target or date.today()
    results = []
    for item in inventory:
        prediction = application.forecaster.forecast(history, item["id"], target)
        results.append(
            {
                **prediction,
                "ingredient": item["name"],
                "balance_kg": item["balance"],
                "recommended_reorder_kg": round(
                    max(0, prediction["planning_upper_kg"] - item["balance"]), 2
                ),
                "backtest": application.forecaster.backtest(history, item["id"]),
            }
        )
    return results
