import os
import sys
import time
import pytest
from pathlib import Path
from fastapi.testclient import TestClient

# Add project root to sys.path
project_root = Path(__file__).resolve().parent.parent
if str(project_root) not in sys.path:
    sys.path.insert(0, str(project_root))


# Override database path before any other imports load database
import src.utils.db as db
TEST_DB_PATH = Path(__file__).resolve().parent.parent / "data" / "test_users.db"
db.DB_PATH = TEST_DB_PATH

from main import app
from src.config.settings import settings
from src.constants.dummy_data import DUMMY_PAYLOAD, INVALID_PAYLOAD

@pytest.fixture(scope="module", autouse=True)
def setup_test_db():
    """Sets up a clean test database and deletes it after tests run."""
    TEST_DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    if TEST_DB_PATH.exists():
        os.remove(TEST_DB_PATH)
    
    db.init_db()
    yield
    
    if TEST_DB_PATH.exists():
        try:
            os.remove(TEST_DB_PATH)
        except OSError:
            pass

@pytest.fixture
def client():
    """FastAPI TestClient fixture that injects a Mock Predictor into app state."""
    class MockPredictor:
        def predict(self, df):
            return [5000.0]
            
    with TestClient(app) as c:
        c.app.state.predictor = MockPredictor()
        yield c

def test_root_endpoint(client) -> None:
    """Tests GET / returns healthy status."""
    response = client.get("/")
    assert response.status_code == 200
    assert response.json()["status"] == "healthy"

def test_user_flow_auth_and_prediction(client) -> None:
    """Tests the full user flow: registration -> login -> token retrieval -> predict with JWT."""
    username = f"user_{int(time.time())}"
    password = "secretpassword"
    
    # 1. Register user
    reg_response = client.post("/api/auth/register", json={
        "username": username,
        "password": password
    })
    assert reg_response.status_code == 201
    assert "User registered successfully" in reg_response.json()["message"]
    
    # Try duplicate registration
    dup_response = client.post("/api/auth/register", json={
        "username": username,
        "password": password
    })
    assert dup_response.status_code == 400
    
    # 2. Get Access Token (Login)
    login_response = client.post("/api/auth/token", data={
        "username": username,
        "password": password
    })
    assert login_response.status_code == 200
    token_data = login_response.json()
    assert "access_token" in token_data
    assert token_data["token_type"] == "bearer"
    token = token_data["access_token"]
    
    # Try invalid password
    bad_login = client.post("/api/auth/token", data={
        "username": username,
        "password": "wrongpassword"
    })
    assert bad_login.status_code == 401
    
    # 3. Request Prediction with valid token
    headers = {"Authorization": f"Bearer {token}"}
    pred_response = client.post("/api/predict", json=DUMMY_PAYLOAD, headers=headers)
    assert pred_response.status_code == 200
    res_data = pred_response.json()
    assert "predicted_price_usd" in res_data
    assert isinstance(res_data["predicted_price_usd"], float)
    assert res_data["predicted_price_usd"] >= 0.0

def test_predict_api_key_auth(client) -> None:
    """Tests predicting using machine-to-machine X-API-Key authentication."""
    valid_key = settings.api_keys_list[0]
    headers = {"X-API-Key": valid_key}
    
    response = client.post("/api/predict", json=DUMMY_PAYLOAD, headers=headers)
    assert response.status_code == 200
    res_data = response.json()
    assert isinstance(res_data["predicted_price_usd"], float)
    assert res_data["predicted_price_usd"] >= 0.0
    assert "api_key" in response.json()["authenticated_as"]

def test_predict_unauthorized(client) -> None:
    """Tests that accessing prediction without valid auth returns 401."""
    response = client.post("/api/predict", json=DUMMY_PAYLOAD)
    assert response.status_code == 401
    
    response = client.post("/api/predict", json=DUMMY_PAYLOAD, headers={"X-API-Key": "invalid-key-here"})
    assert response.status_code == 403

def test_predict_invalid_payload(client) -> None:
    """Tests Pydantic validation for malformed input payloads."""
    valid_key = settings.api_keys_list[0]
    headers = {"X-API-Key": valid_key}
    
    response = client.post("/api/predict", json=INVALID_PAYLOAD, headers=headers)
    assert response.status_code == 422
    
    errors = response.json()["detail"]
    assert len(errors) > 0
