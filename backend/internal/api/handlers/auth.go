package handlers

import (
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/service"
	"github.com/ireuven89/routewise/pkg/utils"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Password    string `json:"password" binding:"required,min=6"`
	Name        string `json:"name" binding:"required"`
	Phone       string `json:"phone" binding:"required"`
	CompanyName string `json:"company_name" binding:"required"`
	Industry    string `json:"industry"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token        string                   `json:"token"`
	User         *models.OrganizationUser `json:"user"`
	Organization *models.Organization     `json:"organization"`
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	user, org, token, err := h.authService.RegisterOrganization(ctx,
		req.Email,
		req.Password,
		req.Name,
		req.Phone,
		req.CompanyName,
		req.Industry)

	if err != nil {
		// Check error type for appropriate status code
		if err.Error() == "Email already registered" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		Token:        token,
		User:         user,
		Organization: org,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Call service (all logic is there)
	ctx := c.Request.Context()
	user, org, token, err := h.authService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if err.Error() == "Invalid credentials" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	c.JSON(http.StatusOK, AuthResponse{
		Token:        token,
		User:         user,
		Organization: org,
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	organizationUserID := c.GetUint("organization_user_id")

	user, org, err := h.authService.GetUserProfile(organizationUserID)
	if err != nil {
		if err.Error() == "User not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		}
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":         user,
		"organization": org,
	})
}

// RequestWorkerOTP handles POST /api/v1/workers/request-otp
func (h *AuthHandler) RequestWorkerOTP(c *gin.Context) {
	var req struct {
		CompanyCode string `json:"company_code" binding:"required"`
		Phone       string `json:"phone" binding:"required"`
	}

	// Parse JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "company_code and phone are required"})
		return
	}

	validatedPhone, err := utils.ValidatePhone(req.Phone)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Call service
	err = h.authService.RequestWorkerOTP(validatedPhone, req.CompanyCode)
	if err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// Success
	c.JSON(200, gin.H{
		"message":    "Code sent to your phone",
		"expires_in": 300, // 5 minutes
	})
}

// VerifyWorkerOTP handles POST /api/v1/workers/verify-otp
func (h *AuthHandler) VerifyWorkerOTP(c *gin.Context) {
	var req struct {
		CompanyCode string `json:"company_code" binding:"required"`
		Phone       string `json:"phone" binding:"required"`
		Code        string `json:"code" binding:"required"`
	}

	// Parse JSON
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "company_code, phone, and code are required"})
		return
	}

	// Call service
	worker, token, err := h.authService.VerifyWorkerOTP(req.Phone, req.CompanyCode, req.Code)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	// Success
	c.JSON(200, gin.H{
		"token": token,
		"worker": gin.H{
			"id":              worker.ID,
			"organization_id": worker.OrganizationID,
			"name":            worker.Name,
			"phone":           worker.Phone,
			"email":           worker.Email,
			"role":            worker.Role,
		},
	})
}
