from prometheus_client import Counter, Histogram

PREDICTION_COUNTER = Counter(
    name="car_price_prediction_requests_total",
    documentation="Total number of prediction requests processed by the ML API"
)

PREDICTED_PRICE_HISTOGRAM = Histogram(
    name="car_price_predictions_price_usd",
    documentation="Distribution of predicted car prices in USD",
    buckets=(500, 1000, 2000, 4000, 6000, 8000, 10000, 15000, 20000, 30000, 50000, 75000, 100000)
)
