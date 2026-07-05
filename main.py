import sys
from pathlib import Path
from contextlib import asynccontextmanager
from fastapi import FastAPI
from prometheus_fastapi_instrumentator import Instrumentator

# Add current workspace directory to python path for modular imports
project_root = Path(__file__).resolve().parent
if str(project_root) not in sys.path:
    sys.path.insert(0, str(project_root))

from src.utils.logger import logger
from src.utils.db import init_db
from src.models.predict import ModelPredictor
from src.api import routes_auth, routes_predict

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Context manager controlling lifespan startup and shutdown routines."""
    logger.info("Initializing API application lifecycle...")
    
    # 1. Initialize DB Table
    try:
        init_db()
    except Exception as db_err:
        logger.critical(f"Failed to initialize SQLite Database: {db_err}")
        
    # 2. Instantiate and cache model predictor singleton
    try:
        app.state.predictor = ModelPredictor()
        logger.info("ModelPredictor preloaded and stored in app state.")
    except Exception as model_err:
        logger.error(f"Could not load ML model on startup: {model_err}. Predictive endpoints will raise errors.")
        app.state.predictor = None
        
    yield
    
    logger.info("Tearing down API application lifecycle...")

app = FastAPI(
    title="Car Price Prediction ML API",
    description="A production-ready FastAPI Machine Learning API predicting car selling prices.",
    version="1.0.0",
    lifespan=lifespan
)

# Hook Prometheus Instrumentator and expose standard endpoints
Instrumentator().instrument(app).expose(app, endpoint="/metrics")

# Register routes
app.include_router(routes_auth.router)
app.include_router(routes_predict.router)

@app.get("/")
def api_status() -> dict:
    """Standard root status endpoint."""
    return {
        "status": "healthy",
        "api_name": "Car Price Prediction API",
        "documentation": "/docs",
        "metrics": "/metrics",
        "endpoints": {
            "predict": "/api/predict",
            "register": "/api/auth/register",
            "token": "/api/auth/token"
        }
    }
