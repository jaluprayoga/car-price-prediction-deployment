import sys
import jwt
from pathlib import Path
from datetime import datetime, timedelta, timezone
from fastapi import Depends, HTTPException, Security, status
from fastapi.security import APIKeyHeader, OAuth2PasswordBearer
from passlib.context import CryptContext

# Add current module root to sys.path for absolute imports
module_root = Path(__file__).resolve().parent.parent.parent
if str(module_root) not in sys.path:
    sys.path.insert(0, str(module_root))

from src.config.settings import settings
from src.utils.logger import logger
from src.utils.db import get_user

# CryptContext for password hashing
pwd_context = CryptContext(schemes=["pbkdf2_sha256"], deprecated="auto")

# JWT and API Key schemes
oauth2_scheme = OAuth2PasswordBearer(tokenUrl="/api/auth/token", auto_error=False)
api_key_header = APIKeyHeader(name="X-API-Key", auto_error=False)

def verify_password(plain_password: str, hashed_password: str) -> bool:
    """Verifies a plain text password against a hashed database entry."""
    return pwd_context.verify(plain_password, hashed_password)

def get_password_hash(password: str) -> str:
    """Generates a secure password hash."""
    return pwd_context.hash(password)

def create_access_token(data: dict, expires_delta: timedelta = None) -> str:
    """Generates a secure JWT access token."""
    to_encode = data.copy()
    if expires_delta:
        expire = datetime.now(timezone.utc) + expires_delta
    else:
        expire = datetime.now(timezone.utc) + timedelta(minutes=settings.ACCESS_TOKEN_EXPIRE_MINUTES)
    to_encode.update({"exp": expire})
    return jwt.encode(to_encode, settings.JWT_SECRET_KEY, algorithm=settings.JWT_ALGORITHM)

def get_current_user(token: str = Depends(oauth2_scheme)) -> str:
    """Dependency to validate JWT tokens and return the authenticated username."""
    if not token:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing JWT token.",
            headers={"WWW-Authenticate": "Bearer"},
        )
    credentials_exception = HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Could not validate credentials.",
        headers={"WWW-Authenticate": "Bearer"},
    )
    try:
        payload = jwt.decode(token, settings.JWT_SECRET_KEY, algorithms=[settings.JWT_ALGORITHM])
        username: str = payload.get("sub")
        if username is None:
            raise credentials_exception
    except jwt.PyJWTError:
        raise credentials_exception
        
    user = get_user(username)
    if user is None:
        raise credentials_exception
        
    return username

def verify_api_key(api_key: str = Depends(api_key_header)) -> str:
    """Dependency to validate X-API-Key headers."""
    if not api_key:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing X-API-Key header.",
        )
    if api_key not in settings.api_keys_list:
        logger.warning("Unsuccessful API Key verification attempt.")
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail="Invalid API Key.",
        )
    return api_key

def authenticate_request(
    token: str = Depends(oauth2_scheme),
    api_key: str = Depends(api_key_header)
) -> str:
    """Combined security dependency permitting EITHER JWT token OR X-API-Key header authentication."""
    if not token and not api_key:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Authentication required. Provide a valid JWT token or X-API-Key header.",
        )
    
    # Try validating API Key first
    if api_key:
        try:
            val = verify_api_key(api_key)
            return f"api_key:{val}"
        except HTTPException:
            if not token:
                raise
            
    # Fallback to JWT Bearer
    if token:
        try:
            usr = get_current_user(token)
            return f"user:{usr}"
        except HTTPException:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Invalid JWT token or API Key.",
            )
            
    raise HTTPException(
        status_code=status.HTTP_401_UNAUTHORIZED,
        detail="Could not validate authentication credentials.",
    )
