import sys
from pathlib import Path
from fastapi import APIRouter, Depends, HTTPException, status
from fastapi.security import OAuth2PasswordRequestForm
from pydantic import BaseModel, Field

# Add current module root to sys.path for absolute imports
module_root = Path(__file__).resolve().parent.parent.parent
if str(module_root) not in sys.path:
    sys.path.insert(0, str(module_root))

from src.utils.db import create_user, get_user
from src.api.dependencies import get_password_hash, verify_password, create_access_token
from src.utils.logger import logger

router = APIRouter(prefix="/api/auth", tags=["Authentication"])

class UserRegister(BaseModel):
    """Pydantic model representing registration payload."""
    username: str = Field(..., min_length=3, max_length=50, description="Unique username")
    password: str = Field(..., min_length=6, max_length=100, description="Plain text password")

class Token(BaseModel):
    """Pydantic model representing OAuth2 JWT token response."""
    access_token: str
    token_type: str

@router.post("/register", status_code=status.HTTP_201_CREATED)
def register(user_data: UserRegister) -> dict:
    """Registers a new standard user in the system."""
    logger.info(f"Received registration request for username: {user_data.username}")
    hashed_pwd = get_password_hash(user_data.password)
    
    success = create_user(user_data.username, hashed_pwd)
    if not success:
        logger.warning(f"Registration aborted. Username already taken: {user_data.username}")
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="Username already registered."
        )
    return {"message": "User registered successfully."}

@router.post("/token", response_model=Token)
def login_for_access_token(form_data: OAuth2PasswordRequestForm = Depends()) -> dict:
    """OAuth2 compatible token login, yielding JWT token on success."""
    logger.info(f"Login attempt for username: {form_data.username}")
    user = get_user(form_data.username)
    
    if not user or not verify_password(form_data.password, user["hashed_password"]):
        logger.warning(f"Failed login attempt for username: {form_data.username}")
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Incorrect username or password.",
            headers={"WWW-Authenticate": "Bearer"},
        )
        
    access_token = create_access_token(data={"sub": user["username"]})
    logger.info(f"Successful login. Token generated for: {form_data.username}")
    return {"access_token": access_token, "token_type": "bearer"}
