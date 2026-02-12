package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	uuid2 "github.com/google/uuid"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	RegisterOrganization(ctx context.Context, email, password, name, phone, companyName, industry string) (*models.OrganizationUser, *models.Organization, string, error)
	Login(ctx context.Context, email string, password string) (*models.OrganizationUser, *models.Organization, string, error)
	GetUserProfile(userID uint) (*models.OrganizationUser, *models.Organization, error)
	RequestWorkerOTP(phone, companyCode string) error
	VerifyWorkerOTP(phone, companyCode, code string) (*models.Worker, string, error)
	FindByEmail(email string) (*models.OrganizationUser, error)
	sendSMS(phone, code string) error
}

type AuthServiceImpl struct {
	otpRepo              *repository.OTPRepository
	workerRepo           *repository.WorkerRepository
	organizationUserRepo *repository.OrganizationUserRepository
	notificationService  NotificationService
}

func NewAuthService(workerRepository *repository.WorkerRepository, otpRepository *repository.OTPRepository, userRepository *repository.OrganizationUserRepository, notificationService NotificationService) AuthService {
	return &AuthServiceImpl{
		workerRepo:           workerRepository,
		organizationUserRepo: userRepository,
		otpRepo:              otpRepository,
		notificationService:  notificationService,
	}
}

func (s *AuthServiceImpl) RegisterOrganization(ctx context.Context, email, password, name, phone, companyName, industry string) (*models.OrganizationUser, *models.Organization, string, error) {
	// 1. Check if user exists
	existingUser, _ := s.organizationUserRepo.FindByEmail(email)
	if existingUser != nil {
		return nil, nil, "", errors.New("Email already registered")
	}

	// 2. Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, "", errors.New("Failed to process password")
	}

	// 3. Set default industry
	if industry == "" {
		industry = "hvac"
	}

	companyCode := s.generateCompanyCode(companyName)

	// 4. Create organization
	org := &models.Organization{
		Name:        companyName,
		Phone:       phone,
		Industry:    industry,
		CompanyCode: companyCode,
	}

	// 5. Create user
	user := &models.OrganizationUser{
		Email:    email,
		Password: string(hashedPassword),
		Name:     name,
		Role:     "owner",
		Phone:    phone,
	}

	// 6. Save both
	if err := s.organizationUserRepo.CreateOrganizationWithUser(org, user); err != nil {
		return nil, nil, "", errors.New("Failed to create account")
	}

	// 7. Generate token (business logic)
	token, err := utils.GenerateToken(user.ID, user.OrganizationID, user.Email, user.Role, "user")
	if err != nil {
		return nil, nil, "", errors.New("Failed to generate token")
	}

	// 8. Clear password before returning
	user.Password = ""

	return user, org, token, nil
}

func (s *AuthServiceImpl) FindByEmail(email string) (*models.OrganizationUser, error) {
	orgUser, err := s.organizationUserRepo.FindByEmail(email)

	if err != nil {
		return nil, err
	}

	return orgUser, nil
}

func (s *AuthServiceImpl) GetUserProfile(userID uint) (*models.OrganizationUser, *models.Organization, error) {
	// 1. Find user
	user, err := s.organizationUserRepo.FindByID(userID)
	if err != nil {
		return nil, nil, errors.New("User not found")
	}

	// 2. Get organization
	org, err := s.organizationUserRepo.FindOrganizationByID(user.OrganizationID)
	if err != nil {
		return nil, nil, errors.New("Failed to fetch organization")
	}

	// 3. Clear password before returning
	user.Password = ""

	return user, org, nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, email string, password string) (*models.OrganizationUser, *models.Organization, string, error) {
	// 1. Find user
	user, err := s.organizationUserRepo.FindByEmail(email)
	if err != nil {
		return nil, nil, "", errors.New("Invalid credentials")
	}

	// 2. Verify password (business logic)
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, nil, "", errors.New("Invalid credentials")
	}

	// 3. Get organization
	org, err := s.organizationUserRepo.FindOrganizationByID(user.OrganizationID)
	if err != nil {
		return nil, nil, "", errors.New("Failed to fetch organization")
	}

	// 4. Generate token (business logic)
	token, err := utils.GenerateToken(user.ID, user.OrganizationID, user.Email, user.Role, "user")
	if err != nil {
		return nil, nil, "", errors.New("Failed to generate token")
	}

	// 5. Clear password before returning
	user.Password = ""

	return user, org, token, nil
}

func (s *AuthServiceImpl) RequestWorkerOTP(phone, companyCode string) error {
	// 1. Find worker

	worker, err := s.workerRepo.FindByPhoneAndCompanyCode(phone, companyCode)
	if err != nil {
		return err
	}
	if worker == nil {
		return errors.New("worker not found")
	}
	if !worker.IsActive {
		return errors.New("account inactive")
	}

	// 2. Generate OTP
	otpCode := s.generateOTP()
	expiresAt := time.Now().Add(5 * time.Minute)

	// 3. Save OTP
	err = s.otpRepo.Save(phone, companyCode, otpCode, expiresAt)
	if err != nil {
		return errors.New("failed to generate code")
	}

	// 4. Send SMS
	err = s.sendSMS(phone, otpCode)
	if err != nil {
		return errors.New("failed to send SMS")
	}

	return nil
}

func (s *AuthServiceImpl) VerifyWorkerOTP(phone, companyCode, otpCode string) (*models.Worker, string, error) {
	// 1. Verify OTP
	otpID, err := s.otpRepo.Verify(phone, companyCode, otpCode)
	if err != nil {
		return nil, "", err
	}
	if otpID == 0 {
		return nil, "", errors.New("invalid or expired code")
	}

	// 2. Mark as verified
	if err = s.otpRepo.MarkVerified(otpID); err != nil {
		return nil, "", errors.New("failed to mark verified")
	}

	// 3. Get worker
	worker, err := s.workerRepo.FindByPhoneAndCompanyCode(phone, companyCode)
	if err != nil || worker == nil {
		return nil, "", errors.New("worker not found")
	}

	// 4. Generate token
	token, err := utils.GenerateToken(
		worker.ID,
		worker.OrganizationID,
		worker.Email,
		worker.Role,
		"worker",
	)
	if err != nil {
		return nil, "", errors.New("failed to generate token")
	}

	return worker, token, nil
}

func (s *AuthServiceImpl) generateOTP() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%06d", rand.Intn(1000000))
}

func (s *AuthServiceImpl) sendSMS(phone, code string) error {
	message := fmt.Sprintf("Your RouteWise code is: %s", code)
	return s.notificationService.SendSMS(phone, message)
}

func (s *AuthServiceImpl) generateUuid() (string, error) {
	uuid, err := uuid2.NewUUID()

	if err != nil {
		return "", err
	}

	return uuid.String(), nil
}

func (s *AuthServiceImpl) generateCompanyCode(name string) string {
	// Clean name: "ACME HVAC" → "ACME-HVAC"
	code := strings.ToUpper(strings.ReplaceAll(name, " ", "-"))

	// Add random suffix for uniqueness: "ACME-HVAC-X7J2"
	suffix := generateRandomString(4) // X7J2

	return fmt.Sprintf("%s-%s", code, suffix)
}

func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)

	if _, err := rand.Read(b); err != nil {
		// Fallback to simple random if crypto fails
		return fmt.Sprintf("%04d", time.Now().UnixNano()%10000)
	}

	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}

	return string(b)
}
