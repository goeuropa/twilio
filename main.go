package main

import (
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"oba-twilio/client"
	"oba-twilio/handlers"
	"oba-twilio/health"
	"oba-twilio/localization"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	obaAPIKey := os.Getenv("ONEBUSAWAY_API_KEY")
	if obaAPIKey == "" {
		log.Fatal("ONEBUSAWAY_API_KEY environment variable is required but not set")
	}
	if obaAPIKey == "test" || obaAPIKey == "TEST" || obaAPIKey == "placeholder" {
		log.Fatal("Invalid API key detected. Please set a valid ONEBUSAWAY_API_KEY environment variable")
	}

	obaBaseURL := os.Getenv("ONEBUSAWAY_BASE_URL")
	if obaBaseURL == "" {
		obaBaseURL = "https://api.pugetsound.onebusaway.org"
	}

	supportedLanguages := os.Getenv("SUPPORTED_LANGUAGES")
	if supportedLanguages == "" {
		supportedLanguages = "en-US"
	}

	locManager, err := localization.NewManager(supportedLanguages)
	if err != nil {
		log.Fatal("Failed to initialize localization manager:", err)
	}

	log.Printf("Localization initialized with languages: %s", supportedLanguages)

	obaClient := client.NewOneBusAwayClient(obaBaseURL, obaAPIKey)

	log.Printf("Initializing coverage area for OneBusAway server...")
	if err := obaClient.InitializeCoverage(); err != nil {
		log.Printf("Warning: Failed to initialize coverage area: %v", err)
		log.Printf("SearchStops functionality may not work properly")
	} else {
		coverage := obaClient.GetCoverageArea()
		log.Printf("Coverage area initialized: center=(%.4f,%.4f), radius=%.0fm",
			coverage.CenterLat, coverage.CenterLon, coverage.Radius)
	}

	// Initialize session store
	sessionStore := handlers.NewImprovedSessionStore()
	defer sessionStore.Close()

	smsHandler := handlers.NewSMSHandler(obaClient, locManager)
	voiceHandler := handlers.NewVoiceHandler(obaClient, locManager)

	// Initialize health check system
	healthManager := health.NewManager(
		health.WithTimeout(10*time.Second),
		health.WithCacheTTL(30*time.Second),
		health.WithMaxConcurrentChecks(5),
		health.WithSystemInfo(true),
		health.WithMetrics(true),
	)

	// Register health checkers
	healthManager.AddChecker(&health.SystemHealthChecker{})
	healthManager.AddChecker(health.NewOneBusAwayHealthChecker(obaClient))
	healthManager.AddChecker(health.NewSessionStoreHealthChecker(sessionStore))
	healthManager.AddChecker(health.NewLocalizationHealthChecker(locManager))
	healthManager.AddChecker(health.NewHTTPServerHealthChecker(port))

	// Create health handler
	healthHandler := health.NewHandler(healthManager)

	r := gin.Default()

	// Add health check middleware
	r.Use(healthHandler.HealthMiddleware())
	r.Use(healthHandler.HealthResponseMiddleware())

	// Application info endpoint
	r.GET("/", func(c *gin.Context) {
		coverage := obaClient.GetCoverageArea()
		response := gin.H{
			"message": "OneBusAway Twilio Integration",
			"status":  "healthy",
			"version": "1.0.0",
		}

		if coverage != nil {
			response["coverage"] = gin.H{
				"center_lat": coverage.CenterLat,
				"center_lon": coverage.CenterLon,
				"radius":     coverage.Radius,
			}
		}

		c.JSON(200, response)
	})

	// Setup comprehensive health check endpoints
	healthHandler.SetupRoutes(r)

	r.POST("/sms", smsHandler.HandleSMS)
	r.POST("/voice", voiceHandler.HandleVoiceStart)
	r.POST("/voice/find_stop", voiceHandler.HandleFindStop)
	r.POST("/voice/menu_action", voiceHandler.HandleVoiceMenuAction)

	log.Printf("Starting server on port %s", port)
	log.Printf("OneBusAway API: %s", obaBaseURL)
	log.Printf("Health check endpoints configured:")
	log.Printf("  - GET /health (liveness probe)")
	log.Printf("  - GET /health/ready (readiness probe)")
	log.Printf("  - GET /health/detailed (comprehensive status)")
	log.Printf("  - GET /metrics (Prometheus metrics)")
	log.Printf("  - GET /health/stats (basic statistics)")
	log.Printf("  - GET /health/config (configuration info)")

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
