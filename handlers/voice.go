package handlers

import (
	"oba-twilio/client"
	"oba-twilio/handlers/voice"
	"oba-twilio/localization"
	"oba-twilio/middleware"
)

type VoiceHandler struct {
	*voice.Handler
}

func NewVoiceHandler(obaClient client.OneBusAwayClientInterface, locManager *localization.LocalizationManager) *VoiceHandler {
	return &VoiceHandler{
		Handler: voice.NewHandler(obaClient, locManager),
	}
}

func (h *VoiceHandler) Close() {
	if h.Handler != nil {
		h.Handler.Close()
	}
}

func (h *VoiceHandler) SetAnalytics(analyticsManager middleware.AnalyticsManager, hashSalt string) {
	if h.Handler != nil {
		h.Handler.SetAnalytics(analyticsManager, hashSalt)
	}
}
