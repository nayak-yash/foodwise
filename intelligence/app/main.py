import logging
from contextlib import asynccontextmanager

from fastapi import FastAPI

from .api import router
from .application import IntelligenceApplication
from .config import Settings
from .telemetry import configure_telemetry

logging.basicConfig(level=logging.INFO, format="%(asctime)s %(levelname)s %(name)s %(message)s")


def create_app(settings: Settings | None = None) -> FastAPI:
    settings = settings or Settings.from_environment()

    @asynccontextmanager
    async def lifespan(app: FastAPI):
        application = IntelligenceApplication(settings)
        try:
            app.state.services = await application.start()
            yield
        finally:
            await application.close()

    app = FastAPI(title="Foodwise Intelligence", lifespan=lifespan)
    app.include_router(router)
    configure_telemetry(app, settings)
    return app


app = create_app()
