package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
)

type PaymentHandler struct {
	service *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: svc}
}

// SendPaymentRequest handles POST /api/v1/jobs/:id/payment-requests
func (h *PaymentHandler) SendPaymentRequest(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	userID := c.GetUint("organization_user_id")

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	notification, err := h.service.SendPaymentRequest(c.Request.Context(), orgID, uint(jobID), &userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPaymentSettingsNotConfigured),
			errors.Is(err, service.ErrJobNotCompletedForPayment),
			errors.Is(err, service.ErrCustomerPhoneMissing):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrJobNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		case errors.Is(err, repository.ErrPaymentRequestAlreadyActive):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			if notification != nil {
				// Notification row was created but the SMS send failed.
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "notification": notification})
				return
			}
			sentry.CaptureException(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send payment request"})
		}
		return
	}

	c.JSON(http.StatusCreated, notification)
}

// GetPaymentNotifications handles GET /api/v1/jobs/:id/payment-notifications
func (h *PaymentHandler) GetPaymentNotifications(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	notifications, err := h.service.GetForJob(c.Request.Context(), orgID, uint(jobID))
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment notifications"})
		return
	}
	if notifications == nil {
		notifications = []*models.PaymentNotification{}
	}

	c.JSON(http.StatusOK, notifications)
}

// MarkPaid handles PATCH /api/v1/payment-notifications/:id/paid
func (h *PaymentHandler) MarkPaid(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.service.MarkPaid(c.Request.Context(), orgID, uint(id)); err != nil {
		if errors.Is(err, repository.ErrPaymentNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark payment as paid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment marked as paid"})
}

// GetPaymentSettings handles GET /api/v1/organization/payment-settings
func (h *PaymentHandler) GetPaymentSettings(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	org, err := h.service.GetSettings(orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bit_payment_enabled":   org.BitPaymentEnabled,
		"bit_phone_number":      org.BitPhoneNumber,
		"bit_business_name":     org.BitBusinessName,
		"auto_send_payment_sms": org.AutoSendPaymentSMS,
	})
}

// UpdatePaymentSettings handles PUT /api/v1/organization/payment-settings
func (h *PaymentHandler) UpdatePaymentSettings(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	var req service.UpdatePaymentSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BitPaymentEnabled && req.BitPhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone number is required when Bit payment is enabled"})
		return
	}

	if err := h.service.UpdateSettings(c.Request.Context(), orgID, req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment settings updated"})
}
