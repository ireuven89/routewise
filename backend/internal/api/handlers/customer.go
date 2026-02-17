package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/service"
)

type CustomerHandler struct {
	service service.CustomerService
}

func NewCustomerHandler(svc service.CustomerService) *CustomerHandler {
	return &CustomerHandler{service: svc}
}

// --- Request DTOs ---

type CreateCustomerRequest struct {
	Name              string                 `json:"name" binding:"required"`
	Email             string                 `json:"email"`
	Phone             string                 `json:"phone" binding:"required"`
	Address           string                 `json:"address" binding:"required"`
	Latitude          *float64               `json:"latitude"`
	Longitude         *float64               `json:"longitude"`
	GooglePlaceID     string                 `json:"google_place_id"`
	FormattedAddress  string                 `json:"formatted_address"`
	AddressComponents map[string]interface{} `json:"address_components"`
	Notes             string                 `json:"notes"`
}

type UpdateCustomerRequest struct {
	Name              string                 `json:"name"`
	Email             string                 `json:"email"`
	Phone             string                 `json:"phone"`
	Address           string                 `json:"address"`
	Latitude          *float64               `json:"latitude"`
	Longitude         *float64               `json:"longitude"`
	GooglePlaceID     string                 `json:"google_place_id"`
	FormattedAddress  string                 `json:"formatted_address"`
	AddressComponents map[string]interface{} `json:"address_components"`
	Notes             string                 `json:"notes"`
}

// --- Handlers ---

func (h *CustomerHandler) Create(c *gin.Context) {
	organizationID := c.GetUint("organization_id")
	organizationUserID := c.GetUint("organization_user_id")

	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := h.service.Create(service.CreateCustomerInput{
		OrganizationID:    organizationID,
		CreatedBy:         organizationUserID,
		Name:              req.Name,
		Email:             req.Email,
		Phone:             req.Phone,
		Address:           req.Address,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		GooglePlaceID:     req.GooglePlaceID,
		FormattedAddress:  req.FormattedAddress,
		AddressComponents: req.AddressComponents,
		Notes:             req.Notes,
	})
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

func (h *CustomerHandler) GetAll(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	customers, err := h.service.GetAll(organizationID, c.Query("search"))
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers"})
		return
	}

	c.JSON(http.StatusOK, customers)
}

func (h *CustomerHandler) GetByID(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := h.service.GetByID(uint(id), organizationID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) Update(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer, err := h.service.Update(uint(id), organizationID, service.UpdateCustomerInput{
		Name:              req.Name,
		Email:             req.Email,
		Phone:             req.Phone,
		Address:           req.Address,
		Latitude:          req.Latitude,
		Longitude:         req.Longitude,
		GooglePlaceID:     req.GooglePlaceID,
		FormattedAddress:  req.FormattedAddress,
		AddressComponents: req.AddressComponents,
		Notes:             req.Notes,
	})
	if err != nil {
		if errors.Is(err, service.ErrCustomerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) Delete(c *gin.Context) {
	organizationID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	if err := h.service.Delete(uint(id), organizationID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted successfully"})
}
