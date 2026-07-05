import os
from pathlib import Path
from pydantic_settings import BaseSettings, SettingsConfigDict
from typing import List

# Resolve workspace root and load .env dynamically
current_dir = Path(__file__).resolve()
env_path = None
for parent in current_dir.parents:
    if (parent / ".env").exists():
        env_path = parent / ".env"
        break

class Settings(BaseSettings):
    """Application settings, loaded from environment variables and .env file."""
    
    LOG_LEVEL: str = "INFO"
    
    # JWT Authentication settings
    JWT_SECRET_KEY: str = "placeholder_jwt_secret_key_please_change_in_env_file"
    JWT_ALGORITHM: str = "HS256"
    ACCESS_TOKEN_EXPIRE_MINUTES: int = 30
    
    # API Keys settings (comma-separated list)
    API_KEYS: str = "placeholder_api_key_please_change_in_env_file"
    
    # MLflow settings
    MLFLOW_TRACKING_URI: str = "http://localhost:5000"
    
    model_config = SettingsConfigDict(
        env_file=str(env_path) if env_path else None,
        env_file_encoding="utf-8",
        extra="ignore"
    )

    @property
    def api_keys_list(self) -> List[str]:
        """Parses the comma-separated API_KEYS into a list of strings."""
        return [k.strip() for k in self.API_KEYS.split(",") if k.strip()]

# Global settings instance
settings = Settings()
