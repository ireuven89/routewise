package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/api/middleware"
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

type DeclineJobRequest struct {
	Reason string `json:"reason"`
}

// workerID returns the technician's ID when the request comes from a technician (mobile) token.
// Worker tokens carry the worker ID in the organization_user_id claim.
func workerID(c *gin.Context) (uint, bool) {
	if c.GetString("user_type") != middleware.UserTypeWorker {
		return 0, false
	}
	return c.GetUint("organization_user_id"), true
}

func invalidTechnician(c *gin.Context, err error) bool {
	if errors.Is(err, service.ErrInvalidTechnician) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Technician not found or inactive"})
		return true
	}
	return false
}

// --- Handlers ---

func (h *JobHandler) CreateServiceCall(c *gin.Context) {
	organizationID := c.GetUint("organization_id")
	organizationUser := c.GetUint("organization_user_id")

	var req models.CreateServiceCallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Job.CreatedBy = organizationUser

	response, err := h.service.CreateServiceCall(c.Request.Context(), organizationID, &req)

	if err != nil {
		if invalidTechnician(c, err) {
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, response)
}

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
		if invalidTechnician(c, err) {
			return
		}
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
	// Technicians only ever see the jobs assigned to them, whatever they ask for.
	if wid, ok := workerID(c); ok {
		filters["technician_id"] = wid
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

	var job *models.Job
	if wid, ok := workerID(c); ok {
		job, err = h.service.GetByIDForWorker(uint(id), organizationID, wid)
	} else {
		job, err = h.service.GetByID(uint(id), organizationID)
	}
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
		if invalidTechnician(c, err) {
			return
		}
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

	// Owners can set any status (override); technicians only move their own accepted jobs forward.
	if wid, ok := workerID(c); ok {
		err = h.service.UpdateStatusAsWorker(uint(id), organizationID, wid, req.Status)
	} else {
		err = h.service.UpdateStatus(uint(id), organizationID, req.Status)
	}
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidStatus):
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		case errors.Is(err, service.ErrJobNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		case errors.Is(err, service.ErrMustAcceptFirst):
			c.JSON(http.StatusConflict, gin.H{"error": "Accept the job first"})
		case errors.Is(err, service.ErrInvalidTransition):
			c.JSON(http.StatusConflict, gin.H{"error": "Job status does not allow this change"})
		default:
			sentry.CaptureException(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

// Accept handles POST /jobs/:id/accept (technician accepts the job offered to them).
func (h *JobHandler) Accept(c *gin.Context) {
	h.respondToAssignment(c, func(id, orgID, wid uint) error {
		return h.service.AcceptJob(id, orgID, wid)
	}, "Job accepted")
}

// Decline handles POST /jobs/:id/decline {reason?}: the job goes back to the unassigned pool.
func (h *JobHandler) Decline(c *gin.Context) {
	var req DeclineJobRequest
	_ = c.ShouldBindJSON(&req) // body is optional
	h.respondToAssignment(c, func(id, orgID, wid uint) error {
		return h.service.DeclineJob(id, orgID, wid, req.Reason)
	}, "Job declined")
}

func (h *JobHandler) respondToAssignment(c *gin.Context, act func(id, orgID, wid uint) error, okMsg string) {
	wid, ok := workerID(c)
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only technicians can perform this action"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}
	if err := act(uint(id), c.GetUint("organization_id"), wid); err != nil {
		switch {
		case errors.Is(err, service.ErrJobNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		case errors.Is(err, service.ErrInvalidAssignmentState):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			sentry.CaptureException(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": okMsg})
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
