from typing import Dict, Any, List

DUMMY_PAYLOAD: Dict[str, Any] = {
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

INVALID_PAYLOAD: Dict[str, Any] = {
    "year": 1800,
    "km_driven": -100,
    "fuel": "Water",
    "seller_type": "Individual",
    "transmission": "Manual",
    "owner": "First Owner",
    "mileage": 23.4,
    "engine": 1248.0,
    "max_power": 74.0,
    "seats": 15
}

TEST_PAYLOADS: List[Dict[str, Any]] = [
    {
        "year": 2017,
        "km_driven": 6900,
        "fuel": "Petrol",
        "seller_type": "Dealer",
        "transmission": "Manual",
        "owner": "First Owner",
        "mileage": 21.14,
        "engine": 998.0,
        "max_power": 67.04,
        "seats": 5
    },
    {
        "year": 2015,
        "km_driven": 60000,
        "fuel": "Diesel",
        "seller_type": "Dealer",
        "transmission": "Automatic",
        "owner": "Second Owner",
        "mileage": 19.67,
        "engine": 1582.0,
        "max_power": 126.2,
        "seats": 5
    },
    {
        "year": 2017,
        "km_driven": 50000,
        "fuel": "Petrol",
        "seller_type": "Individual",
        "transmission": "Manual",
        "owner": "First Owner",
        "mileage": 17.7,
        "engine": 1197.0,
        "max_power": 81.86,
        "seats": 5
    }
]

CATEGORICAL_MAPPINGS = {
    "fuel": ["Petrol", "Diesel", "CNG", "LPG"],
    "seller_type": ["Dealer", "Individual", "Trustmark Dealer"],
    "transmission": ["Manual", "Automatic"],
    "owner": ["First Owner", "Second Owner", "Third Owner", "Fourth & Above Owner", "Test Drive Car"]
}
