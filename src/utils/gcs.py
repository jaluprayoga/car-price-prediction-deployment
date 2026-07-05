import os
import logging
from pathlib import Path
from dotenv import load_dotenv
from google.cloud import storage

logger = logging.getLogger("deployment.gcs")

# Resolve workspace root and load .env dynamically
current_dir = Path(__file__).resolve()
env_path = None
for parent in current_dir.parents:
    if (parent / ".env").exists():
        env_path = parent / ".env"
        break

if env_path:
    load_dotenv(env_path)
    logger.info(f"Loaded environment variables from: {env_path}")
else:
    logger.warning("No .env file found in parent directories. Using environment variables directly.")

def get_gcs_client():
    """Initializes the GCS client, setting credentials from GOOGLE_APPLICATION_CREDENTIALS if provided."""
    creds_path = os.environ.get("GOOGLE_APPLICATION_CREDENTIALS")
    if creds_path:
        creds_file = Path(creds_path)
        if not creds_file.is_absolute() and env_path:
            root_relative = env_path.parent / creds_path
            if root_relative.exists():
                creds_path = str(root_relative)
        
        if os.path.exists(creds_path):
            try:
                return storage.Client.from_service_account_json(creds_path)
            except Exception as e:
                logger.warning(f"Failed to initialize GCS client from service account json at {creds_path}: {e}. Falling back to default auth.")
        else:
            logger.warning(f"Credentials file specified in GOOGLE_APPLICATION_CREDENTIALS not found: {creds_path}")
            
    try:
        return storage.Client()
    except Exception as e:
        logger.warning(f"Could not initialize GCS client: {e}. GCS operations will be bypassed (local fallback).")
        return None

def download_from_gcs(local_file_path: str, gcs_blob_name: str, bucket_name: str = None) -> bool:
    """Downloads a file from GCS. Returns True on success, False on failure (e.g. if offline or not found)."""
    if not bucket_name:
        bucket_name = os.environ.get("GCS_BUCKET_NAME", "car-price-prediction-mlops")
        
    client = get_gcs_client()
    if not client:
        logger.info(f"GCS client unavailable. Skipping download of '{gcs_blob_name}' from cloud storage.")
        return False
        
    try:
        bucket = client.bucket(bucket_name)
        blob = bucket.blob(gcs_blob_name)
        
        os.makedirs(os.path.dirname(local_file_path), exist_ok=True)
        blob.download_to_filename(local_file_path)
        logger.info(f"Successfully downloaded '{gcs_blob_name}' from GCS to '{local_file_path}'")
        return True
    except Exception as e:
        logger.error(f"Failed to download '{gcs_blob_name}' from GCS: {e}")
        return False
