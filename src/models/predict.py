import sys
import time
import joblib
import shutil
import pandas as pd
from pathlib import Path
from typing import List

# Add current module root to sys.path for absolute imports
module_root = Path(__file__).resolve().parent.parent
if str(module_root) not in sys.path:
    sys.path.insert(0, str(module_root))

from src.utils.logger import logger

class ModelPredictor:
    """Singleton helper class to load the Scikit-learn model pipeline once and run inference."""
    
    _instance = None
    _model = None

    def __new__(cls, *args, **kwargs):
        if not cls._instance:
            cls._instance = super(ModelPredictor, cls).__new__(cls, *args, **kwargs)
        return cls._instance

    def __init__(self) -> None:
        if self._model is None:
            self.load_model()

    def load_model(self) -> None:
        """Loads the serialized model pipeline into memory, checking GCS first."""
        base_dir = Path(__file__).resolve().parent.parent.parent
        model_path = base_dir / "models" / "model.pkl"
        
        # 1. Attempt GCS download
        logger.info("Attempting to fetch latest model pipeline from GCS...")
        try:
            from src.utils.gcs import download_from_gcs
            gcs_success = download_from_gcs(str(model_path), "models/model.pkl")
            if gcs_success:
                logger.info("Model pipeline downloaded from GCS successfully.")
            else:
                logger.warning("GCS model download not successful or skipped. Checking local fallbacks...")
        except Exception as e:
            logger.warning(f"Error fetching model from GCS: {e}")
            
        # 2. Check local paths (fallback to sibling training folder if missing)
        if not model_path.exists():
            training_model_path = base_dir.parent / "training" / "models" / "model.pkl"
            if training_model_path.exists():
                logger.info(f"Copying model from local training folder: {training_model_path}")
                model_path.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy(training_model_path, model_path)
            else:
                # Check legacy joblib file
                joblib_model_path = base_dir / "models" / "model.joblib"
                if joblib_model_path.exists():
                    logger.info(f"Using legacy joblib model: {joblib_model_path}")
                    model_path = joblib_model_path
                else:
                    error_msg = f"Model artifact not found at {model_path} and sibling path. Please train a model first."
                    logger.critical(error_msg)
                    raise FileNotFoundError(error_msg)
                
        logger.info(f"Loading serialized model pipeline from: {model_path}")
        start_time = time.time()
        try:
            if model_path.suffix == ".pkl":
                import pickle
                with open(model_path, "rb") as f:
                    self._model = pickle.load(f)
            else:
                self._model = joblib.load(model_path)
            loading_duration = time.time() - start_time
            logger.info(f"Model pipeline successfully loaded in {loading_duration:.4f} seconds.")
        except Exception as e:
            logger.error(f"Failed to load model pipeline: {str(e)}")
            raise e

    def predict(self, input_df: pd.DataFrame) -> List[float]:
        """Generates predictions from a formatted DataFrame of features.
        
        Args:
            input_df (pd.DataFrame): Input features with columns matching training features.
            
        Returns:
            List[float]: Predicted car prices (in USD).
        """
        if self._model is None:
            error_msg = "Model is not loaded. Cannot run prediction."
            logger.error(error_msg)
            raise RuntimeError(error_msg)
            
        logger.info(f"Generating price prediction for {len(input_df)} records.")
        start_time = time.time()
        
        try:
            predictions = self._model.predict(input_df)
            inference_duration = time.time() - start_time
            logger.info(f"Generated {len(predictions)} predictions in {inference_duration:.4f} seconds.")
            return [float(p) for p in predictions]
        except Exception as e:
            logger.error(f"Prediction failed: {str(e)}")
            raise e

if __name__ == "__main__":
    try:
        predictor = ModelPredictor()
        print("Model predictor loaded successfully.")
    except Exception as err:
        print(f"Error initializing predictor: {err}")
