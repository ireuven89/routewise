package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

type WorkerHandler struct {
	workerRepo *repository.WorkerRepository
}

func NewWorkerHandler(db *sql.DB) *WorkerHandler {
	return &WorkerHandler{
		workerRepo: repository.NewWorkerRepository(db),
	}
}

type CreateWorkerRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email"`
	Phone    string `json:"phone" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type UpdateWorkerRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	IsActive *bool  `json:"is_active"`
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

	worker := &models.Worker{
		OrganizationID: organizationID,
		CreatedBy:      &organizationUserID,
		Name:           req.Name,
		Email:          req.Email,
		Phone:          req.Phone,
		IsActive:       true, // Default to active
	}

	if err := h.workerRepo.Create(worker); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create worker"})
		return
	}

	c.JSON(http.StatusCreated, worker)
}

func (h *WorkerHandler) GetAll(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	activeOnly := c.Query("active_only") == "true"

	technicians, err := h.workerRepo.FindAll(organizationID, activeOnly)
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

	technician, err := h.workerRepo.FindByID(uint(id), organizationID)
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

	// Fetch existing worker
	worker, err := h.workerRepo.FindByID(uint(id), organizationID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Worker not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		worker.Name = req.Name
	}
	worker.Email = req.Email
	if req.Phone != "" {
		worker.Phone = req.Phone
	}
	if req.IsActive != nil {
		worker.IsActive = *req.IsActive
	}

	if err := h.workerRepo.Update(worker); err != nil {
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

	if err := h.workerRepo.Delete(uint(id), organizationID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Technician not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Technician deleted successfully"})
}
