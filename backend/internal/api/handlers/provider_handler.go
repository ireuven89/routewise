package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
)

type ProviderHandler struct {
	providerService *service.ProviderService
	frontendAPIKey  string
}

func NewProviderHandler(providerService *service.ProviderService, frontendAPIKey string) *ProviderHandler {
	return &ProviderHandler{
		providerService: providerService,
		frontendAPIKey:  frontendAPIKey,
	}
}

// SearchProviders handles GET /api/v1/public/providers
// Query params: lat, lng, service_type
func (h *ProviderHandler) SearchProviders(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	serviceType := c.DefaultQuery("service_type", "hvac")

	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng are required"})
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lat"})
		return
	}
	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lng"})
		return
	}

	providers, err := h.providerService.SearchProviders(c.Request.Context(), lat, lng, serviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if providers == nil {
		providers = make([]*repository.ProviderResult, 0)
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// GetPublicGoogleMapsConfig handles GET /api/v1/public/config/google-maps (no auth)
func (h *ProviderHandler) GetPublicGoogleMapsConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_key": h.frontendAPIKey,
		"enabled": h.frontendAPIKey != "",
	})
}

// UpdateServiceArea handles PUT /api/v1/organization/service-area
func (h *ProviderHandler) UpdateServiceArea(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	if orgID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req service.UpdateServiceAreaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Latitude == 0 && req.Longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude are required"})
		return
	}

	if err := h.providerService.UpdateServiceArea(c.Request.Context(), orgID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "service area updated"})
}

// UpdateServiceOffer handles PUT /api/v1/organization/service-offer
func (h *ProviderHandler) UpdateServiceOffer(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	if orgID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req service.UpdateServiceOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.providerService.UpdateServiceOffer(c.Request.Context(), orgID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "service offer updated"})
}
