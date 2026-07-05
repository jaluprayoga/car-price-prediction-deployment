package api

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jaluprayoga/car-price-prediction-deployment/internal/model"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

var (
	PredictionCounter = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "prediction_counter_total",
		Help: "Total number of car price predictions generated.",
	})
	PredictedPriceHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "predicted_price_usd",
		Help:    "Distribution of predicted car prices in USD.",
		Buckets: prometheus.ExponentialBuckets(1000, 2, 5),
	})
)

func init() {
	prometheus.MustRegister(PredictionCounter)
	prometheus.MustRegister(PredictedPriceHistogram)
}

// PredictorInterface abstracts the prediction logic for mock testing.
type PredictorInterface interface {
	Predict(features model.CarFeatures) (float32, error)
}

// Server encapsulates the endpoint routes configuration.
type Server struct {
	Predictor PredictorInterface
}

// SetupRouter registers routes on the Fiber app.
func (s *Server) SetupRouter(app *fiber.App) {
	// Expose prometheus metrics
	h := promhttp.Handler()
	app.Get("/metrics", func(c *fiber.Ctx) error {
		fasthttpadaptor.NewFastHTTPHandler(h)(c.Context())
		return nil
	})

	// Serve swagger.json
	app.Get("/swagger.json", func(c *fiber.Ctx) error {
		return c.SendFile("./docs/swagger.json")
	})

	// Serve Swagger UI
	app.Get("/docs", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/html")
		return c.SendString(swaggerUIHTML)
	})

	// Root status endpoint
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":        "healthy",
			"api_name":      "Car Price Prediction API",
			"documentation": "/docs",
			"metrics":       "/metrics",
			"endpoints": fiber.Map{
				"predict":  "/api/predict",
				"register": "/api/auth/register",
				"token":    "/api/auth/token",
			},
		})
	})

	apiGroup := app.Group("/api")

	// Auth group
	authGroup := apiGroup.Group("/auth")
	authGroup.Post("/register", s.handleRegister)
	authGroup.Post("/token", s.handleToken)

	// Prediction endpoint (authenticated)
	apiGroup.Post("/predict", AuthMiddleware(), s.handlePredict)
}
