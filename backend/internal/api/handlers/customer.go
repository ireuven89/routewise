package handlers

import (
	"database/sql"
	"fmt"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"net/http"
	"strconv"
)

type CustomerHandler struct {
	customerRepo *repository.CustomerRepository
}

func NewCustomerHandler(db *sql.DB) *CustomerHandler {
	return &CustomerHandler{
		customerRepo: repository.NewCustomerRepository(db),
	}
}

type CreateCustomerRequest struct {
	Name      string   `json:"name" binding:"required"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone" binding:"required"`
	Address   string   `json:"address" binding:"required"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Notes     string   `json:"notes"`
}

type UpdateCustomerRequest struct {
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Phone     string   `json:"phone"`
	Address   string   `json:"address"`
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Notes     string   `json:"notes"`
}

func (h *CustomerHandler) Create(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed parsing request", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	customer := &models.Customer{
		UserID:    userID,
		Name:      req.Name,
		Email:     req.Email,
		Phone:     req.Phone,
		Address:   req.Address,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Notes:     req.Notes,
	}

	if err := h.customerRepo.Create(customer); err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed creating customer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create customer"})
		return
	}

	c.JSON(http.StatusCreated, customer)
}

func (h *CustomerHandler) GetAll(c *gin.Context) {
	userID := c.GetUint("user_id")
	search := c.Query("search")

	customers, err := h.customerRepo.FindAll(userID, search)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed fetching customer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch customers"})
		return
	}

	c.JSON(http.StatusOK, customers)
}

func (h *CustomerHandler) GetByID(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed fetching customer", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	customer, err := h.customerRepo.FindByID(uint(id), userID)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed fetching customer", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) Update(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed updating customer", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed updating customer", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Fetch existing customer
	customer, err := h.customerRepo.FindByID(uint(id), userID)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed updating customer", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	// Update fields
	if req.Name != "" {
		customer.Name = req.Name
	}
	customer.Email = req.Email
	if req.Phone != "" {
		customer.Phone = req.Phone
	}
	if req.Address != "" {
		customer.Address = req.Address
	}
	customer.Latitude = req.Latitude
	customer.Longitude = req.Longitude
	customer.Notes = req.Notes

	if err := h.customerRepo.Update(customer); err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed updating customer", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update customer"})
		return
	}

	c.JSON(http.StatusOK, customer)
}

func (h *CustomerHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed deleting customer", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid customer ID"})
		return
	}

	if err := h.customerRepo.Delete(uint(id), userID); err != nil {
		sentry.CaptureException(err)
		fmt.Println("failed deleting customer", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Customer deleted successfully"})
}
