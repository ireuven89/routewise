package handlers

import (
	"database/sql"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
	"github.com/ireuven89/routewise/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo    *repository.OrganizationUserRepository
	workerRepo  *repository.WorkerRepository
	authService service.AuthService
}

func NewAuthHandler(db *sql.DB, authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		userRepo:    repository.NewUserRepository(db),
		workerRepo:  repository.NewWorkerRepository(db),
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

	// Check if user already exists
	existingUser, _ := h.userRepo.FindByEmail(req.Email)
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create organization
	org := &models.Organization{
		Name:     req.CompanyName,
		Phone:    req.Phone,
		Industry: req.Industry,
	}

	if org.Industry == "" {
		org.Industry = "hvac"
	}

	// Create user
	user := &models.OrganizationUser{
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		Role:     "owner",
		Phone:    req.Phone,
	}

	// Create both in transaction
	if err := h.userRepo.CreateOrganizationWithUser(org, user); err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create account"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.OrganizationID, user.Email, user.Role, "user")
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Don't return password
	user.Password = ""

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

	// Find user by email
	user, err := h.userRepo.FindByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Get organization
	org, err := h.userRepo.FindOrganizationByID(user.OrganizationID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organization"})
		return
	}

	// Generate JWT token
	token, err := utils.GenerateToken(user.ID, user.OrganizationID, user.Email, user.Role, "user")
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Don't return password
	user.Password = ""

	c.JSON(http.StatusOK, AuthResponse{
		Token:        token,
		User:         user,
		Organization: org,
	})
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	organizationUserID := c.GetUint("organization_user_id")

	user, err := h.userRepo.FindByID(organizationUserID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	org, err := h.userRepo.FindOrganizationByID(user.OrganizationID)
	if err != nil {
		sentry.CaptureException(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch organization"})
		return
	}

	// Don't return password
	user.Password = ""

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

	// Call service
	err := h.authService.RequestWorkerOTP(req.Phone, req.CompanyCode)
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
