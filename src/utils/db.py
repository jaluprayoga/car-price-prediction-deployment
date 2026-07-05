import os
import sqlite3
from pathlib import Path
from src.utils.logger import logger

DB_PATH = Path(__file__).resolve().parent.parent.parent / "data" / "users.db"

def init_db() -> None:
    """Initializes the SQLite database and creates the users table if it does not exist."""
    DB_PATH.parent.mkdir(parents=True, exist_ok=True)
    
    logger.info(f"Initializing SQLite database at: {DB_PATH}")
    conn = sqlite3.connect(str(DB_PATH))
    cursor = conn.cursor()
    
    try:
        cursor.execute("""
            CREATE TABLE IF NOT EXISTS users (
                username TEXT PRIMARY KEY,
                hashed_password TEXT NOT NULL
            )
        """)
        conn.commit()
        logger.info("Users table verified/created successfully.")
    except Exception as e:
        logger.error(f"Failed to initialize database: {str(e)}")
        raise e
    finally:
        conn.close()

def get_db_connection() -> sqlite3.Connection:
    """Returns a connection to the SQLite database."""
    if not DB_PATH.exists():
        init_db()
    return sqlite3.connect(str(DB_PATH))

def create_user(username: str, hashed_password: str) -> bool:
    """Registers a new user in the database. Returns True if successful, False if user exists."""
    conn = get_db_connection()
    cursor = conn.cursor()
    try:
        cursor.execute(
            "INSERT INTO users (username, hashed_password) VALUES (?, ?)",
            (username, hashed_password)
        )
        conn.commit()
        logger.info(f"Successfully registered user: {username}")
        return True
    except sqlite3.IntegrityError:
        logger.warning(f"Registration failed. User already exists: {username}")
        return False
    except Exception as e:
        logger.error(f"Error writing to database: {str(e)}")
        raise e
    finally:
        conn.close()

def get_user(username: str) -> dict:
    """Retrieves user info from the database. Returns dict or None."""
    conn = get_db_connection()
    cursor = conn.cursor()
    try:
        cursor.execute("SELECT username, hashed_password FROM users WHERE username = ?", (username,))
        row = cursor.fetchone()
        if row:
            return {"username": row[0], "hashed_password": row[1]}
        return None
    except Exception as e:
        logger.error(f"Error querying database: {str(e)}")
        raise e
    finally:
        conn.close()
