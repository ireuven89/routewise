package handlers

import (
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/service"
)

type WorkerHandler struct {
	workerService service.WorkerService
}

func NewWorkerHandler(service service.WorkerService) *WorkerHandler {
	return &WorkerHandler{
		workerService: service,
	}
}

type CreateWorkerRequest struct {
	Name                  string                 `json:"name" binding:"required"`
	Email                 string                 `json:"email"`
	Phone                 string                 `json:"phone" binding:"required"`
	IsActive              bool                   `json:"is_active"`
	HomeAddress           string                 `json:"home_address"`
	HomeLatitude          *float64               `json:"home_latitude"`
	HomeLongitude         *float64               `json:"home_longitude"`
	HomeGooglePlaceID     string                 `json:"home_google_place_id"`
	HomeFormattedAddress  string                 `json:"home_formatted_address"`
	HomeAddressComponents map[string]interface{} `json:"home_address_components"`
}

type UpdateWorkerRequest struct {
	Name                  string                 `json:"name"`
	Email                 string                 `json:"email"`
	Phone                 string                 `json:"phone"`
	IsActive              *bool                  `json:"is_active"`
	HomeAddress           string                 `json:"home_address"`
	HomeLatitude          *float64               `json:"home_latitude"`
	HomeLongitude         *float64               `json:"home_longitude"`
	HomeGooglePlaceID     string                 `json:"home_google_place_id"`
	HomeFormattedAddress  string                 `json:"home_formatted_address"`
	HomeAddressComponents map[string]interface{} `json:"home_address_components"`
}

func (h *WorkerHandler) Create(c *gin.Context) {
	organizationID := c.GetUint("organization_id")
	organizationUserID := c.GetUint("organization_user_id")

	var req CreateWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	worker, err := h.workerService.Create(ctx, service.CreateWorkerInput{
		OrganizationID:        organizationID,
		CreatedBy:             organizationUserID,
		Name:                  req.Name,
		Email:                 req.Email,
		Phone:                 req.Phone,
		HomeAddress:           req.HomeAddress,
		HomeLatitude:          req.HomeLatitude,
		HomeLongitude:         req.HomeLongitude,
		HomeGooglePlaceID:     req.HomeGooglePlaceID,
		HomeFormattedAddress:  req.HomeFormattedAddress,
		HomeAddressComponents: req.HomeAddressComponents,
	})

	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create worker"})
		return
	}

	c.JSON(http.StatusCreated, worker)
}

func (h *WorkerHandler) GetAll(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	activeOnly := c.Query("active_only") == "true"

	ctx := c.Request.Context()

	technicians, err := h.workerService.GetAll(ctx, organizationID, activeOnly)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch workers"})
		return
	}

	c.JSON(http.StatusOK, technicians)
}

func (h *WorkerHandler) GetByID(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
		return
	}

	ctx := c.Request.Context()
	technician, err := h.workerService.GetByID(ctx, uint(id), organizationID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	c.JSON(http.StatusOK, technician)
}

func (h *WorkerHandler) Update(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid worker ID"})
		return
	}

	var req UpdateWorkerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	worker, err := h.workerService.Update(ctx, uint(id), organizationID, service.UpdateWorkerInput{
		Name:                  req.Name,
		Email:                 req.Email,
		Phone:                 req.Phone,
		IsActive:              req.IsActive,
		HomeAddress:           req.HomeAddress,
		HomeLatitude:          req.HomeLatitude,
		HomeLongitude:         req.HomeLongitude,
		HomeGooglePlaceID:     req.HomeGooglePlaceID,
		HomeFormattedAddress:  req.HomeFormattedAddress,
		HomeAddressComponents: req.HomeAddressComponents,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update worker"})
		return
	}

	c.JSON(http.StatusOK, worker)
}

func (h *WorkerHandler) Delete(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	ctx := c.Request.Context()

	if err := h.workerService.Delete(ctx, uint(id), organizationID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worker deleted successfully"})
}
