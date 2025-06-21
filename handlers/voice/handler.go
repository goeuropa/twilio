package voice

import (
	"log"

	"github.com/gin-gonic/gin"

	"oba-twilio/client"
	"oba-twilio/handlers/common"
	"oba-twilio/localization"
	"oba-twilio/middleware"
)

type Handler struct {
	OBAClient           client.OneBusAwayClientInterface
	SessionStore        *common.SessionStore
	TemplateManager     *TemplateManager
	LocalizationManager *localization.LocalizationManager
	ErrorHandler        *common.ErrorHandler
	analyticsManager    middleware.AnalyticsManager
	analyticsHashSalt   string
}

func NewHandler(obaClient client.OneBusAwayClientInterface, locManager *localization.LocalizationManager) *Handler {
	templateManager, err := NewTemplateManager()
	if err != nil {
		log.Fatalf("Failed to initialize voice template manager: %v", err)
	}

	return &Handler{
		OBAClient:           obaClient,
		SessionStore:        common.NewSessionStore(),
		TemplateManager:     templateManager,
		LocalizationManager: locManager,
		ErrorHandler:        common.NewErrorHandler(locManager),
	}
}

func (h *Handler) Close() {
	if h.SessionStore != nil {
		h.SessionStore.Close()
	}
}

func (h *Handler) SetAnalytics(analyticsManager middleware.AnalyticsManager, hashSalt string) {
	h.analyticsManager = analyticsManager
	h.analyticsHashSalt = hashSalt
}

// getLanguageFromRequest extracts language from URL parameter or defaults to primary language
func (h *Handler) getLanguageFromRequest(c *gin.Context) string {
	language := c.Query("lang")
	if language != "" && h.LocalizationManager.IsSupported(language) {
		return language
	}
	return h.LocalizationManager.GetPrimaryLanguage()
}
