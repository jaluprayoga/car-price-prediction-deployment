import sys
import pandas as pd
import pytest
from pathlib import Path

# Add project root to sys.path
project_root = Path(__file__).resolve().parent.parent
if str(project_root) not in sys.path:
    sys.path.insert(0, str(project_root))


from src.models.predict import ModelPredictor
from src.constants.dummy_data import DUMMY_PAYLOAD

def test_predictor_inference() -> None:
    """Verifies the ModelPredictor can successfully load and execute inference."""
    predictor = ModelPredictor()
    
    # Adapt the dummy payload to the model feature schema (which expects 'Age' instead of 'year')
    payload = DUMMY_PAYLOAD.copy()
    payload["Age"] = 2026 - payload.pop("year")
    
    input_data = pd.DataFrame([payload])
    
    predictions = predictor.predict(input_data)
    assert len(predictions) == 1
    assert isinstance(predictions[0], float)
    assert predictions[0] >= 0.0
