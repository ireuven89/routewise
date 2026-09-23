package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
)

type ServiceRequestHandler struct {
	service         *service.ServiceRequestService
	frontendBaseURL string
}

func NewServiceRequestHandler(svc *service.ServiceRequestService, frontendBaseURL string) *ServiceRequestHandler {
	return &ServiceRequestHandler{service: svc, frontendBaseURL: frontendBaseURL}
}

type createServiceRequestRequest struct {
	ServiceType   string     `json:"service_type" binding:"required"`
	Description   string     `json:"description"`
	CustomerName  string     `json:"customer_name" binding:"required"`
	CustomerPhone string     `json:"customer_phone" binding:"required"`
	Latitude      float64    `json:"latitude" binding:"required"`
	Longitude     float64    `json:"longitude" binding:"required"`
	Address       string     `json:"address"`
	PreferredTime *time.Time `json:"preferred_time"`
}

// Create handles POST /api/v1/public/service-requests (public, no auth).
func (h *ServiceRequestHandler) Create(c *gin.Context) {
	var req createServiceRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sr, err := h.service.CreateRequest(c.Request.Context(), service.CreateServiceRequestInput{
		ServiceType:   req.ServiceType,
		Description:   req.Description,
		CustomerName:  req.CustomerName,
		CustomerPhone: req.CustomerPhone,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		Address:       req.Address,
		PreferredTime: req.PreferredTime,
	})
	if err != nil {
		if errors.Is(err, service.ErrServiceRequestValidation) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create service request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":           sr.ID,
		"access_token": sr.AccessToken,
		"tracking_url": fmt.Sprintf("%s/find-service/requests/%s", h.frontendBaseURL, sr.AccessToken),
		"status":       sr.Status,
	})
}

// GetByToken handles GET /api/v1/public/service-requests/:token (public, no auth).
func (h *ServiceRequestHandler) GetByToken(c *gin.Context) {
	token := c.Param("token")

	sr, bids, err := h.service.GetByToken(c.Request.Context(), token)
	if err != nil {
		if errors.Is(err, repository.ErrServiceRequestNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load service request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"request": sr, "bids": bids})
}

type awardBidRequest struct {
	BidID uint `json:"bid_id" binding:"required"`
}

// Award handles POST /api/v1/public/service-requests/:token/award (public, no auth).
func (h *ServiceRequestHandler) Award(c *gin.Context) {
	token := c.Param("token")

	var req awardBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sr, err := h.service.AwardBid(c.Request.Context(), token, req.BidID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrServiceRequestNotFound), errors.Is(err, repository.ErrBidNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		case errors.Is(err, repository.ErrServiceRequestNotOpen):
			c.JSON(http.StatusConflict, gin.H{"error": "request is no longer open"})
		default:
			sentry.CaptureException(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to award bid"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"request": sr})
}

// ListLeads handles GET /api/v1/leads (protected, org-scoped).
func (h *ServiceRequestHandler) ListLeads(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	if orgID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	requests, bids, err := h.service.ListLeadsForOrg(c.Request.Context(), orgID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load leads"})
		return
	}

	type lead struct {
		Request interface{} `json:"request"`
		MyBid   interface{} `json:"my_bid"`
	}
	leads := make([]lead, len(requests))
	for i, r := range requests {
		leads[i] = lead{Request: r, MyBid: bids[i]}
	}

	c.JSON(http.StatusOK, gin.H{"leads": leads})
}

type upsertBidRequest struct {
	Price      float64 `json:"price" binding:"required"`
	EtaMinutes *int    `json:"eta_minutes"`
	Message    string  `json:"message"`
}

// UpsertBid handles PUT /api/v1/leads/:id/bid (protected, org-scoped).
func (h *ServiceRequestHandler) UpsertBid(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	if orgID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	requestID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lead id"})
		return
	}

	var req upsertBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bid, err := h.service.SubmitBid(c.Request.Context(), orgID, uint(requestID), service.SubmitBidInput{
		Price:      req.Price,
		EtaMinutes: req.EtaMinutes,
		Message:    req.Message,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrServiceRequestNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "lead not found"})
		case errors.Is(err, repository.ErrServiceRequestNotOpen), errors.Is(err, repository.ErrBidLocked):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrServiceRequestValidation):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			sentry.CaptureException(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit bid"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"bid": bid})
}
