package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

var (
	ErrPaymentSettingsNotConfigured = errors.New("bit payment is not configured for this organization")
	ErrJobNotCompletedForPayment    = errors.New("job must be completed with a price before requesting payment")
	ErrCustomerPhoneMissing         = errors.New("customer has no phone number on file")
)

type UpdatePaymentSettingsRequest struct {
	BitPaymentEnabled  bool   `json:"bit_payment_enabled"`
	BitPhoneNumber     string `json:"bit_phone_number"`
	BitBusinessName    string `json:"bit_business_name"`
	AutoSendPaymentSMS bool   `json:"auto_send_payment_sms"`
}

type PaymentService struct {
	jobRepo      *repository.JobRepository
	customerRepo *repository.CustomerRepository
	orgRepo      *repository.OrganizationRepository
	paymentRepo  *repository.PaymentNotificationRepository
	notifier     NotificationService
}

func NewPaymentService(jobRepo *repository.JobRepository, customerRepo *repository.CustomerRepository, orgRepo *repository.OrganizationRepository, paymentRepo *repository.PaymentNotificationRepository, notifier NotificationService) *PaymentService {
	return &PaymentService{
		jobRepo:      jobRepo,
		customerRepo: customerRepo,
		orgRepo:      orgRepo,
		paymentRepo:  paymentRepo,
		notifier:     notifier,
	}
}

// SendPaymentRequest sends a plain-text Bit payment-collection SMS for a completed, priced
// job. userID is nil for the auto-send-on-completion hook, non-nil for a manual button click.
func (s *PaymentService) SendPaymentRequest(ctx context.Context, orgID, jobID uint, userID *uint) (*models.PaymentNotification, error) {
	org, err := s.orgRepo.FindByID(orgID)
	if err != nil {
		return nil, err
	}
	if !org.BitPaymentEnabled || org.BitPhoneNumber == "" {
		return nil, ErrPaymentSettingsNotConfigured
	}

	job, err := s.jobRepo.FindByID(jobID, orgID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if job.Status != models.StatusCompleted || job.Price == nil || *job.Price <= 0 {
		return nil, ErrJobNotCompletedForPayment
	}

	customer, err := s.customerRepo.FindByID(job.CustomerID, orgID)
	if err != nil {
		return nil, err
	}
	if customer.Phone == "" {
		return nil, ErrCustomerPhoneMissing
	}

	message := buildPaymentMessage(org, job, customer)
	notification := &models.PaymentNotification{
		OrganizationID: orgID,
		JobID:          jobID,
		CustomerID:     customer.ID,
		Amount:         *job.Price,
		RecipientPhone: customer.Phone,
		MessageBody:    message,
		CreatedBy:      userID,
	}
	if err := s.paymentRepo.CreateIfNotActive(ctx, notification); err != nil {
		return nil, err // may be repository.ErrPaymentRequestAlreadyActive
	}

	if sendErr := s.notifier.SendSMS(customer.Phone, message); sendErr != nil {
		_ = s.paymentRepo.UpdateSMSResult(ctx, notification.ID, models.SMSStatusFailed, models.PaymentStatusSendFailed, nil)
		notification.SMSStatus = models.SMSStatusFailed
		notification.PaymentStatus = models.PaymentStatusSendFailed
		return notification, fmt.Errorf("failed to send payment SMS: %w", sendErr)
	}

	now := time.Now()
	_ = s.paymentRepo.UpdateSMSResult(ctx, notification.ID, models.SMSStatusSent, models.PaymentStatusSent, &now)
	notification.SMSStatus = models.SMSStatusSent
	notification.PaymentStatus = models.PaymentStatusSent
	notification.SentAt = &now
	return notification, nil
}

// TryAutoSendOnCompletion is the job-completion hook entry point. It never returns an error —
// a failed or skipped SMS send must not affect the job status update that triggered it.
func (s *PaymentService) TryAutoSendOnCompletion(ctx context.Context, orgID, jobID uint) {
	org, err := s.orgRepo.FindByID(orgID)
	if err != nil || !org.BitPaymentEnabled || !org.AutoSendPaymentSMS {
		return
	}
	_, _ = s.SendPaymentRequest(ctx, orgID, jobID, nil)
}

func (s *PaymentService) MarkPaid(ctx context.Context, orgID, notificationID uint) error {
	return s.paymentRepo.MarkPaid(ctx, notificationID, orgID)
}

func (s *PaymentService) GetForJob(ctx context.Context, orgID, jobID uint) ([]*models.PaymentNotification, error) {
	return s.paymentRepo.ListByJobID(ctx, jobID, orgID)
}

func (s *PaymentService) UpdateSettings(ctx context.Context, orgID uint, req UpdatePaymentSettingsRequest) error {
	return s.orgRepo.UpdatePaymentSettings(ctx, orgID, req.BitPaymentEnabled, req.BitPhoneNumber, req.BitBusinessName, req.AutoSendPaymentSMS)
}

func (s *PaymentService) GetSettings(orgID uint) (*models.Organization, error) {
	return s.orgRepo.FindByID(orgID)
}

func buildPaymentMessage(org *models.Organization, job *models.Job, customer *models.Customer) string {
	return fmt.Sprintf(
		`Hi %s, job "%s" is complete. Amount due: ₪%.0f. Pay via Bit to: %s (%s)`,
		customer.Name, job.Title, *job.Price, org.BitPhoneNumber, org.BitBusinessName,
	)
}
