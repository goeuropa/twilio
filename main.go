package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"oba-twilio/client"
	"oba-twilio/handlers"
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

	smsHandler := handlers.NewSMSHandler(obaClient)
	voiceHandler := handlers.NewVoiceHandler(obaClient)

	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "OneBusAway Twilio Integration",
			"status":  "healthy",
		})
	})

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
		})
	})

	r.POST("/sms", smsHandler.HandleSMS)
	r.POST("/voice", voiceHandler.HandleVoiceStart)
	r.POST("/voice/find_stop", voiceHandler.HandleFindStop)
	r.POST("/voice/menu_action", voiceHandler.HandleVoiceMenuAction)

	log.Printf("Starting server on port %s", port)
	log.Printf("OneBusAway API: %s", obaBaseURL)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
