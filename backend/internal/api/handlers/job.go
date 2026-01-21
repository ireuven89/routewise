package handlers

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/getsentry/sentry-go"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

type JobHandler struct {
	jobRepo *repository.JobRepository
}

func NewJobHandler(db *sql.DB) *JobHandler {
	return &JobHandler{
		jobRepo: repository.NewJobRepository(db),
	}
}

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

func (h *JobHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job := &models.Job{
		UserID:          userID,
		CustomerID:      req.CustomerID,
		TechnicianID:    req.TechnicianID,
		Title:           req.Title,
		Description:     req.Description,
		Status:          models.StatusScheduled,
		ScheduledAt:     req.ScheduledAt,
		DurationMinutes: req.DurationMinutes,
		Price:           req.Price,
		Metadata:        req.Metadata,
	}

	if job.DurationMinutes == 0 {
		job.DurationMinutes = 60 // Default 1 hour
	}

	if err := h.jobRepo.Create(job); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	c.JSON(http.StatusCreated, job)
}

func (h *JobHandler) GetAll(c *gin.Context) {
	userID := c.GetUint("user_id")

	// Parse filters
	filters := make(map[string]interface{})

	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	if techIDStr := c.Query("technician_id"); techIDStr != "" {
		techID, err := strconv.ParseUint(techIDStr, 10, 32)
		if err == nil {
			filters["technician_id"] = uint(techID)
		}
	}

	if date := c.Query("date"); date != "" {
		filters["scheduled_date"] = date
	}

	sortBy := c.Query("sort")
	if sortBy == "" {
		sortBy = "created_at" // Default sort
	}

	jobs, err := h.jobRepo.FindAll(userID, filters, sortBy)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch jobs"})
		return
	}

	c.JSON(http.StatusOK, jobs)
}

func (h *JobHandler) GetByID(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	job, err := h.jobRepo.FindByID(uint(id), userID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	var req UpdateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch existing job
	job, err := h.jobRepo.FindByID(uint(id), userID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	// Update fields
	if req.Title != "" {
		job.Title = req.Title
	}
	job.Description = req.Description
	if !req.ScheduledAt.IsZero() {
		job.ScheduledAt = req.ScheduledAt
	}
	if req.DurationMinutes > 0 {
		job.DurationMinutes = req.DurationMinutes
	}
	job.Price = req.Price
	if req.Status != "" {
		job.Status = models.JobStatus(req.Status)
	}
	if req.Metadata != nil {
		job.Metadata = req.Metadata
	}

	if err := h.jobRepo.Update(job); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update job"})
		return
	}

	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) AssignTechnician(c *gin.Context) {
	userID := c.GetUint("user_id")

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

	if err := h.jobRepo.AssignTechnician(uint(id), userID, req.TechnicianID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign technician"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Technician assigned successfully"})
}

func (h *JobHandler) UpdateStatus(c *gin.Context) {
	fmt.Println("🔍 UpdateStatus called") // DEBUG

	userID := c.GetUint("user_id")
	fmt.Println("🔍 UserID:", userID) // DEBUG

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		fmt.Println("❌ Invalid job ID:", err) // DEBUG
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}
	fmt.Println("🔍 Job ID:", id) // DEBUG

	// DEBUG: Check request details
	fmt.Println("🔍 Content-Type:", c.GetHeader("Content-Type"))
	fmt.Println("🔍 Content-Length:", c.Request.ContentLength)
	fmt.Println("🔍 Method:", c.Request.Method)

	// DEBUG: Try to read body manually
	bodyBytes, _ := io.ReadAll(c.Request.Body)
	fmt.Println("🔍 Raw body:", string(bodyBytes))

	// IMPORTANT: Reset body so ShouldBindJSON can read it
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var req UpdateStatusRequest
	fmt.Println("🔍 About to bind JSON...") // DEBUG

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println("❌ Failed to bind JSON:", err) // DEBUG
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fmt.Println("🔍 Successfully bound JSON, status:", req.Status) // DEBUG

	status := models.JobStatus(req.Status)

	// Validate status
	validStatuses := map[models.JobStatus]bool{
		models.StatusScheduled:  true,
		models.StatusInProgress: true,
		models.StatusCompleted:  true,
		models.StatusCancelled:  true,
	}

	if !validStatuses[status] {
		fmt.Println("❌ Invalid status:", status) // DEBUG
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	if err := h.jobRepo.UpdateStatus(uint(id), userID, status); err != nil {
		fmt.Println("❌ Failed to update in DB:", err) // DEBUG
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update status"})
		return
	}

	fmt.Println("✅ Status updated successfully") // DEBUG
	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

func (h *JobHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	if err := h.jobRepo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Job deleted successfully"})
}
