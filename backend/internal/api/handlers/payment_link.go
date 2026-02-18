package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/service"
	"github.com/ireuven89/routewise/pkg/utils"
)

// PaymentLinkHandler handles payment link related requests
type PaymentLinkHandler struct {
	service service.PaymentLinkService
}

// NewPaymentLinkHandler creates a new payment link handler
func NewPaymentLinkHandler(svc service.PaymentLinkService) *PaymentLinkHandler {
	return &PaymentLinkHandler{service: svc}
}

// SendPaymentLink manually sends payment link for a completed job
// POST /api/v1/jobs/:id/payment-link
func (h *PaymentLinkHandler) SendPaymentLink(c *gin.Context) {
	organizationID := c.GetUint("organization_id")
	userID := c.GetUint("organization_user_id")

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	notification, err := h.service.SendPaymentLinkForJob(organizationID, uint(jobID), userID)
	if err != nil {
		statusCode := http.StatusBadRequest
		if err == service.ErrPaymentNotEnabled {
			statusCode = http.StatusForbidden
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, notification)
}

// GetPaymentNotifications lists all payment notifications for a job
// GET /api/v1/jobs/:id/payment-notifications
func (h *PaymentLinkHandler) GetPaymentNotifications(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	notifications, err := h.service.GetPaymentNotifications(uint(jobID), organizationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch notifications"})
		return
	}

	c.JSON(http.StatusOK, notifications)
}

// GetPaymentSettings gets organization payment settings
// GET /api/v1/payment-settings
func (h *PaymentLinkHandler) GetPaymentSettings(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	settings, err := h.service.GetOrganizationSettings(organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdatePaymentSettings updates organization payment settings
// PUT /api/v1/payment-settings
func (h *PaymentLinkHandler) UpdatePaymentSettings(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	var req struct {
		BitPaymentEnabled    bool   `json:"bit_payment_enabled"`
		BitPhoneNumber       string `json:"bit_phone_number"`
		BitBusinessName      string `json:"bit_business_name"`
		AutoSendOnCompletion bool   `json:"auto_send_on_completion"`
		SMSTemplateHe        string `json:"sms_template_he"`
		SMSTemplateEn        string `json:"sms_template_en"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Basic validation
	if req.BitPaymentEnabled {
		if req.BitPhoneNumber == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bit phone number is required when payments are enabled"})
			return
		}

		// Validate and clean phone number
		cleanedPhone, err := utils.ValidatePhone(req.BitPhoneNumber)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid phone format. Use international format (e.g., +972501234567)"})
			return
		}
		req.BitPhoneNumber = cleanedPhone
	}

	// Get existing settings
	settings, err := h.service.GetOrganizationSettings(organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	// Update fields
	settings.BitPaymentEnabled = req.BitPaymentEnabled
	settings.BitPhoneNumber = req.BitPhoneNumber
	settings.BitBusinessName = req.BitBusinessName
	settings.AutoSendOnCompletion = req.AutoSendOnCompletion
	if req.SMSTemplateHe != "" {
		settings.SMSTemplateHe = req.SMSTemplateHe
	}
	if req.SMSTemplateEn != "" {
		settings.SMSTemplateEn = req.SMSTemplateEn
	}

	// Save to database
	if err := h.service.UpdateOrganizationSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}
