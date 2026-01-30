package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/pkg/utils"
	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type AuthService interface {
	Register(ctx context.Context, email string, password string) error
	Login(ctx context.Context, email string, password string) (string, error)
	RequestWorkerOTP(phone, companyCode string) error
	VerifyWorkerOTP(phone, companyCode, code string) (*models.Worker, string, error)
	sendSMS(phone, code string) error
}

type AuthServiceImpl struct {
	otpRepo              *repository.OTPRepository
	workerRepo           *repository.WorkerRepository
	organizationUserRepo *repository.OrganizationUserRepository
	twilioClient         *twilio.RestClient
}

func NewAuthService(workerRepository *repository.WorkerRepository, otpRepository *repository.OTPRepository) AuthService {

	return &AuthServiceImpl{
		workerRepo: workerRepository,
		otpRepo:    otpRepository,
	}
}

func (s *AuthServiceImpl) Register(ctx context.Context, email string, password string) error {
	//todo refactor implement
	return nil
}

func (s *AuthServiceImpl) Login(ctx context.Context, email string, password string) (string, error) {
	//todo refactor implement
	return "", nil
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
	params := &api.CreateMessageParams{}
	params.SetTo(phone)
	params.SetFrom(os.Getenv("TWILIO_PHONE_NUMBER"))
	params.SetBody(fmt.Sprintf("Your RouteWise code is: %s", code))

	_, err := s.twilioClient.Api.CreateMessage(params)

	return err
}
