import sys
import pandas as pd
from pathlib import Path
from typing import Literal, Dict, Any
from fastapi import APIRouter, Depends, HTTPException, Request, status
from pydantic import BaseModel, Field

# Add current module root to sys.path for absolute imports
module_root = Path(__file__).resolve().parent.parent.parent
if str(module_root) not in sys.path:
    sys.path.insert(0, str(module_root))

from src.models.predict import ModelPredictor
from src.api.dependencies import authenticate_request
from src.utils.logger import logger

try:
    from src.monitoring.metrics import PREDICTED_PRICE_HISTOGRAM, PREDICTION_COUNTER
    metrics_enabled = True
except ImportError:
    metrics_enabled = False

router = APIRouter(prefix="/api", tags=["Model Inference"])

class CarPredictionInput(BaseModel):
    """Pydantic validation model for single car prediction request."""
    year: int = Field(..., ge=1990, le=2026, description="Year of manufacture (1990-2026)")
    km_driven: int = Field(..., ge=0, description="Total kilometers driven")
    fuel: Literal["Petrol", "Diesel", "CNG", "LPG"] = Field(..., description="Fuel type used by the vehicle")
    seller_type: Literal["Individual", "Dealer", "Trustmark Dealer"] = Field(..., description="Type of seller listing the vehicle")
    transmission: Literal["Manual", "Automatic"] = Field(..., description="Transmission type")
    owner: Literal["First Owner", "Second Owner", "Third Owner", "Fourth & Above Owner", "Test Drive Car"] = Field(..., description="Number of previous owners")
    mileage: float = Field(..., ge=0.0, description="Mileage in kmpl or km/kg")
    engine: float = Field(..., ge=0.0, description="Engine displacement in CC")
    max_power: float = Field(..., ge=0.0, description="Max engine power in bhp")
    seats: int = Field(..., ge=2, le=10, description="Number of seats (2-10)")

    model_config = {
        "json_schema_extra": {
            "example": {
                "year": 2014,
                "km_driven": 27000,
                "fuel": "Petrol",
                "seller_type": "Dealer",
                "transmission": "Manual",
                "owner": "First Owner",
                "mileage": 23.4,
                "engine": 1248.0,
                "max_power": 74.0,
                "seats": 5
            }
        }
    }

def get_predictor(request: Request) -> ModelPredictor:
    """Dependency to retrieve the preloaded model predictor instance from application state."""
    return request.app.state.predictor

@router.post("/predict", status_code=status.HTTP_200_OK)
def predict_car_price(
    payload: CarPredictionInput,
    auth_info: str = Depends(authenticate_request),
    predictor: ModelPredictor = Depends(get_predictor)
) -> dict:
    """Predicts the selling price of a car based on physical/market input variables."""
    logger.info(f"Prediction requested by {auth_info}")
    
    car_age = 2026 - payload.year
    
    features_dict = {
        "km_driven": [payload.km_driven],
        "Age": [car_age],
        "mileage": [payload.mileage],
        "engine": [payload.engine],
        "max_power": [payload.max_power],
        "seats": [payload.seats],
        "fuel": [payload.fuel],
        "seller_type": [payload.seller_type],
        "transmission": [payload.transmission],
        "owner": [payload.owner]
    }
    
    input_df = pd.DataFrame(features_dict)
    
    try:
        predicted_values = predictor.predict(input_df)
        prediction = predicted_values[0]
        prediction = max(0.0, prediction)
        
        if metrics_enabled:
            PREDICTION_COUNTER.inc()
            PREDICTED_PRICE_HISTOGRAM.observe(prediction)
            
        logger.info(f"Inference completed. Predicted Selling Price: ${prediction:.2f} USD")
        
        return {
            "predicted_price_usd": round(prediction, 2),
            "currency": "USD (Dollars)",
            "authenticated_as": auth_info
        }
    except Exception as e:
        logger.error(f"Error during endpoint model prediction: {str(e)}")
        raise HTTPException(
            status_code=status.HTTP_500_INTERNAL_SERVER_ERROR,
            detail=f"Inference execution failed: {str(e)}"
        )
