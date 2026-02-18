package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type GeocodingHandler struct {
	frontendAPIKey string
}

func NewGeocodingHandler(frontendAPIKey string) *GeocodingHandler {
	return &GeocodingHandler{
		frontendAPIKey: frontendAPIKey,
	}
}

// GetFrontendConfig returns the Google Maps configuration for the frontend
// GET /api/v1/config/google-maps
func (h *GeocodingHandler) GetFrontendConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_key": h.frontendAPIKey,
		"enabled": h.frontendAPIKey != "",
	})
}
