import os
import logging
import sys

def setup_logger(name: str = "deployment") -> logging.Logger:
    """Configures and returns a unified logger for the deployment module."""
    logger = logging.getLogger(name)
    
    if not logger.handlers:
        log_level = os.environ.get("LOG_LEVEL", "INFO").upper()
        numeric_level = getattr(logging, log_level, logging.INFO)
        logger.setLevel(numeric_level)
        
        formatter = logging.Formatter(
            fmt='[%(asctime)s] %(levelname)s [%(name)s.%(funcName)s:%(lineno)d] - %(message)s',
            datefmt='%Y-%m-%d %H:%M:%S'
        )
        
        console_handler = logging.StreamHandler(sys.stdout)
        console_handler.setFormatter(formatter)
        logger.addHandler(console_handler)
        logger.propagate = False
        
    return logger

logger = setup_logger()
