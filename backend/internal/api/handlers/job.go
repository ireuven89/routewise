package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/service"
)

type JobHandler struct {
	service service.JobService
}

func NewJobHandler(svc service.JobService) *JobHandler {
	return &JobHandler{service: svc}
}

// --- Request DTOs ---

type CreateJobRequest struct {
	CustomerID      uint        `json:"customer_id" binding:"required"`
	TechnicianID    *uint       `json:"technician_id"`
	Title           string      `json:"title" binding:"required"`
	Description     string      `json:"description"`
	ScheduledAt     time.Time   `json:"scheduled_at" binding:"required"`
	DurationMinutes int         `json:"duration_minutes"`
	Price           *float64    `json:"price"`
	Metadata        models.JSON `json:"metadata"`
}

type UpdateJobRequest struct {
	Title           string      `json:"title"`
	Description     string      `json:"description"`
	ScheduledAt     time.Time   `json:"scheduled_at"`
	DurationMinutes int         `json:"duration_minutes"`
	Price           *float64    `json:"price"`
	Status          string      `json:"status"`
	Metadata        models.JSON `json:"metadata"`
}

type AssignTechnicianRequest struct {
	TechnicianID *uint `json:"technician_id"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// --- Handlers ---

func (h *JobHandler) Create(c *gin.Context) {
	organizationID := c.GetUint("organization_id")
	organizationUserID := c.GetUint("organization_user_id")

	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.service.Create(service.CreateJobInput{
		OrganizationID:  organizationID,
		CreatedBy:       organizationUserID,
		CustomerID:      req.CustomerID,
		TechnicianID:    req.TechnicianID,
		Title:           req.Title,
		Description:     req.Description,
		ScheduledAt:     req.ScheduledAt,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
		Metadata:        req.Metadata,
	})
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(http.StatusCreated, job)
}

func (h *JobHandler) GetAll(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if techIDStr := c.Query("technician_id"); techIDStr != "" {
		if techID, err := strconv.ParseUint(techIDStr, 10, 32); err == nil {
			filters["technician_id"] = uint(techID)
		}
	}
	if date := c.Query("date"); date != "" {
		filters["scheduled_date"] = date
	}

	jobs, err := h.service.GetAll(organizationID, filters, c.Query("sort"))
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func (h *JobHandler) GetByID(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := h.service.GetByID(uint(id), organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) Update(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := h.service.Update(uint(id), organizationID, service.UpdateJobInput{
		Title:           req.Title,
		Description:     req.Description,
		ScheduledAt:     req.ScheduledAt,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
		Status:          req.Status,
		Metadata:        req.Metadata,
	})
	if err != nil {
		if errors.Is(err, service.ErrJobNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) AssignTechnician(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req AssignTechnicianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.AssignTechnician(uint(id), organizationID, req.TechnicianID); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign technician"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Worker assigned successfully"})
}

func (h *JobHandler) UpdateStatus(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.UpdateStatus(uint(id), organizationID, req.Status); err != nil {
		if errors.Is(err, service.ErrInvalidStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

func (h *JobHandler) Delete(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	if err := h.service.Delete(uint(id), organizationID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}
