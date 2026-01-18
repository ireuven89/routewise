package handlers

import (
	"database/sql"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"net/http"
	"strconv"
)

type TechnicianHandler struct {
	technicianRepo *repository.TechnicianRepository
}

func NewTechnicianHandler(db *sql.DB) *TechnicianHandler {
	return &TechnicianHandler{
		technicianRepo: repository.NewTechnicianRepository(db),
	}
}

type CreateTechnicianRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email"`
	Phone    string `json:"phone" binding:"required"`
	IsActive bool   `json:"is_active"`
}

type UpdateTechnicianRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	IsActive *bool  `json:"is_active"`
}

func (h *TechnicianHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req CreateTechnicianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	technician := &models.Technician{
		UserID:   userID,
		Name:     req.Name,
		Email:    req.Email,
		Phone:    req.Phone,
		IsActive: true, // Default to active
	}

	if err := h.technicianRepo.Create(technician); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create technician"})
		return
	}

	c.JSON(http.StatusCreated, technician)
}

func (h *TechnicianHandler) GetAll(c *gin.Context) {
	userID := c.GetUint("user_id")

	activeOnly := c.Query("active_only") == "true"

	technicians, err := h.technicianRepo.FindAll(userID, activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch technicians"})
		return
	}

	c.JSON(http.StatusOK, technicians)
}

func (h *TechnicianHandler) GetByID(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	technician, err := h.technicianRepo.FindByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Technician not found"})
		return
	}

	c.JSON(http.StatusOK, technician)
}

func (h *TechnicianHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	var req UpdateTechnicianRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch existing technician
	technician, err := h.technicianRepo.FindByID(uint(id), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Technician not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		technician.Name = req.Name
	}
	technician.Email = req.Email
	if req.Phone != "" {
		technician.Phone = req.Phone
	}
	if req.IsActive != nil {
		technician.IsActive = *req.IsActive
	}

	if err := h.technicianRepo.Update(technician); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update technician"})
		return
	}

	c.JSON(http.StatusOK, technician)
}

func (h *TechnicianHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid technician ID"})
		return
	}

	if err := h.technicianRepo.Delete(uint(id), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Technician not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Technician deleted successfully"})
}
