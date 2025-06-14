package handlers

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"oba-twilio/client"
	"oba-twilio/formatters"
	"oba-twilio/models"
)

type VoiceHandler struct {
	OBAClient client.OneBusAwayClientInterface
}

func NewVoiceHandler(obaClient client.OneBusAwayClientInterface) *VoiceHandler {
	return &VoiceHandler{
		OBAClient: obaClient,
	}
}

func (h *VoiceHandler) HandleVoiceStart(c *gin.Context) {
	var req models.TwilioVoiceRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Failed to bind voice request: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLVoice("Invalid request format.")
		c.String(http.StatusBadRequest, twiml)
		return
	}

	log.Printf("Received voice call from %s", req.From)

	prompt := "Welcome to OneBusAway transit information. Please enter your stop ID followed by the pound key."
	
	c.Header("Content-Type", "text/xml")
	twiml, err := formatters.GenerateTwiMLGather(prompt, "/voice/input", 6)
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		twiml, _ = formatters.GenerateTwiMLVoice("Error generating response.")
	}

	c.String(http.StatusOK, twiml)
}

func (h *VoiceHandler) HandleVoiceInput(c *gin.Context) {
	var req models.TwilioVoiceRequest
	if err := c.ShouldBind(&req); err != nil {
		log.Printf("Failed to bind voice input request: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLVoice("Invalid request format.")
		c.String(http.StatusBadRequest, twiml)
		return
	}

	log.Printf("Received voice input from %s: %s", req.From, req.Digits)

	stopID := req.Digits
	if stopID == "" {
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLVoice("I didn't receive any digits. Please try calling again.")
		c.String(http.StatusOK, twiml)
		return
	}

	if len(stopID) < 3 || len(stopID) > 10 {
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLVoice("Invalid stop ID. Please try calling again with a valid stop ID.")
		c.String(http.StatusOK, twiml)
		return
	}

	obaResp, err := h.OBAClient.GetArrivalsAndDepartures(stopID)
	if err != nil {
		log.Printf("OneBusAway API error: %v", err)
		c.Header("Content-Type", "text/xml")
		twiml, _ := formatters.GenerateTwiMLVoice("Sorry, I couldn't find information for that stop. Please check the stop ID and try again.")
		c.String(http.StatusOK, twiml)
		return
	}

	arrivals := h.OBAClient.ProcessArrivals(obaResp)
	stopName := obaResp.Data.Entry.StopId // ABXOXO: FIXME // obaResp.Data.Entry.Stop.Name

	message := formatters.FormatVoiceResponse(arrivals, stopName)
	
	c.Header("Content-Type", "text/xml")
	twiml, err := formatters.GenerateTwiMLVoice(message)
	if err != nil {
		log.Printf("Failed to generate TwiML: %v", err)
		twiml, _ = formatters.GenerateTwiMLVoice("Error generating response.")
	}

	c.String(http.StatusOK, twiml)
}