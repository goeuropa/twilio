package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"oba-twilio/client"
	"oba-twilio/formatters"
	"oba-twilio/models"
)

type SMSHandler struct {
	OBAClient client.OneBusAwayClientInterface
}

func NewSMSHandler(obaClient client.OneBusAwayClientInterface) *SMSHandler {
	return &SMSHandler{
		OBAClient: obaClient,
	}
}

func (h *SMSHandler) HandleSMS(c *gin.Context) {
	var req models.TwilioSMSRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Failed to bind SMS request: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLSMS("Invalid request format.")
		c.String(http.StatusBadRequest, twiml)
		return
	}

	log.Printf("Received SMS from %s: %s", req.From, req.Body)

	stopID := formatters.ExtractStopID(req.Body)
	if stopID == "" {
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLSMS("Please send a valid stop ID (e.g., 75403).")
		c.String(http.StatusOK, twiml)
		return
	}

	obaResp, err := h.OBAClient.GetArrivalsAndDepartures(stopID)
	if err != nil {
		log.Printf("OneBusAway API error: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLSMS("Sorry, I couldn't find information for that stop. Please check the stop ID and try again.")
		c.String(http.StatusOK, twiml)
		return
	}

	arrivals := h.OBAClient.ProcessArrivals(obaResp)
	stopName := obaResp.Data.Entry.Stop.Name

	message := formatters.FormatSMSResponse(arrivals, stopName)
	
	c.Header("Content-Type", "text/xml")
	twiml, err := formatters.GenerateTwiMLSMS(message)
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		twiml, _ = formatters.GenerateTwiMLSMS("Error generating response.")
	}

	c.String(http.StatusOK, twiml)
}