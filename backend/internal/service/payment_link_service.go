package service

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

var (
	ErrJobNotCompleted       = errors.New("job must be completed to send payment link")
	ErrJobHasNoPrice         = errors.New("job has no price set")
	ErrCustomerHasNoPhone    = errors.New("customer has no phone number")
	ErrPaymentNotEnabled     = errors.New("payment links not enabled for organization")
	ErrPaymentAlreadySent    = errors.New("payment link already sent for this job")
	ErrBitPhoneNotConfigured = errors.New("Bit phone number not configured for organization")
)

// PaymentLinkService handles payment link generation and sending
type PaymentLinkService interface {
	SendPaymentLinkForJob(organizationID, jobID, userID uint) (*models.PaymentNotification, error)
	GetPaymentNotifications(jobID, organizationID uint) ([]*models.PaymentNotification, error)
	GetOrganizationSettings(organizationID uint) (*models.OrganizationPaymentSettings, error)
	UpdateOrganizationSettings(settings *models.OrganizationPaymentSettings) error
}

// PaymentLinkServiceImpl implements PaymentLinkService
type PaymentLinkServiceImpl struct {
	paymentNotifRepo    *repository.PaymentNotificationRepository
	paymentSettingsRepo *repository.PaymentSettingsRepository
	jobRepo             *repository.JobRepository
	customerRepo        *repository.CustomerRepository
	notificationSvc     NotificationService
}

// NewPaymentLinkService creates a new payment link service instance
func NewPaymentLinkService(
	paymentNotifRepo *repository.PaymentNotificationRepository,
	paymentSettingsRepo *repository.PaymentSettingsRepository,
	jobRepo *repository.JobRepository,
	customerRepo *repository.CustomerRepository,
	notificationSvc NotificationService,
) PaymentLinkService {
	return &PaymentLinkServiceImpl{
		paymentNotifRepo:    paymentNotifRepo,
		paymentSettingsRepo: paymentSettingsRepo,
		jobRepo:             jobRepo,
		customerRepo:        customerRepo,
		notificationSvc:     notificationSvc,
	}
}

// SendPaymentLinkForJob sends a payment link to the customer for a completed job
func (s *PaymentLinkServiceImpl) SendPaymentLinkForJob(organizationID, jobID, userID uint) (*models.PaymentNotification, error) {
	// 1. Get organization settings
	settings, err := s.paymentSettingsRepo.GetByOrganizationID(organizationID)
	if err != nil || !settings.BitPaymentEnabled {
		return nil, ErrPaymentNotEnabled
	}

	// 2. Validate Bit configuration
	if settings.BitPhoneNumber == "" {
		return nil, ErrBitPhoneNotConfigured
	}

	// 3. Get job details
	job, err := s.jobRepo.FindByID(jobID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("job not found: %w", err)
	}

	// 4. Validate job is completed
	if job.Status != models.StatusCompleted {
		return nil, ErrJobNotCompleted
	}

	// 5. Validate price exists
	if job.Price == nil || *job.Price <= 0 {
		return nil, ErrJobHasNoPrice
	}

	// 6. Get customer
	customer, err := s.customerRepo.FindByID(job.CustomerID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	// 7. Validate customer has phone
	if customer.Phone == "" {
		return nil, ErrCustomerHasNoPhone
	}

	// 8. Check if already sent
	existing, _ := s.paymentNotifRepo.FindByJobID(jobID, organizationID)
	for _, notif := range existing {
		if notif.PaymentStatus == models.PaymentStatusSent ||
			notif.PaymentStatus == models.PaymentStatusPaid {
			return nil, ErrPaymentAlreadySent
		}
	}

	// 9. Generate Bit payment link
	paymentLink := s.generateBitLink(settings, *job.Price, job.Title)

	// 10. Format SMS message
	message := s.formatSMSMessage(settings, customer, job, *job.Price, paymentLink)

	// 11. Create notification record
	notification := &models.PaymentNotification{
		OrganizationID: organizationID,
		JobID:          jobID,
		CustomerID:     customer.ID,
		Amount:         *job.Price,
		PaymentLinkURL: paymentLink,
		PaymentMethod:  "bit",
		SentVia:        "sms",
		RecipientPhone: customer.Phone,
		SMSStatus:      models.SMSStatusPending,
		PaymentStatus:  models.PaymentStatusPending,
		CreatedBy:      &userID,
	}

	err = s.paymentNotifRepo.Create(notification)
	if err != nil {
		return nil, fmt.Errorf("failed to create notification: %w", err)
	}

	// 12. Send SMS (non-blocking)
	go func() {
		err := s.notificationSvc.SendSMS(customer.Phone, message)
		now := time.Now()

		if err != nil {
			s.paymentNotifRepo.UpdateSMSStatus(notification.ID, organizationID, models.SMSStatusFailed, nil)
		} else {
			s.paymentNotifRepo.UpdateSMSStatus(notification.ID, organizationID, models.SMSStatusSent, &now)
			s.paymentNotifRepo.UpdatePaymentStatus(notification.ID, organizationID, models.PaymentStatusSent, nil)
		}
	}()

	return notification, nil
}

// GetPaymentNotifications retrieves all payment notifications for a job
func (s *PaymentLinkServiceImpl) GetPaymentNotifications(jobID, organizationID uint) ([]*models.PaymentNotification, error) {
	notifications, err := s.paymentNotifRepo.FindByJobID(jobID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment notifications: %w", err)
	}
	return notifications, nil
}

// GetOrganizationSettings retrieves payment settings for an organization
func (s *PaymentLinkServiceImpl) GetOrganizationSettings(organizationID uint) (*models.OrganizationPaymentSettings, error) {
	settings, err := s.paymentSettingsRepo.GetByOrganizationID(organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment settings: %w", err)
	}
	return settings, nil
}

// UpdateOrganizationSettings updates payment settings for an organization
func (s *PaymentLinkServiceImpl) UpdateOrganizationSettings(settings *models.OrganizationPaymentSettings) error {
	err := s.paymentSettingsRepo.Update(settings)
	if err != nil {
		return fmt.Errorf("failed to update payment settings: %w", err)
	}
	return nil
}

// generateBitLink creates a Bit payment link
func (s *PaymentLinkServiceImpl) generateBitLink(settings *models.OrganizationPaymentSettings, amount float64, description string) string {
	// Bit deep link format: bit://pay?phone=<phone>&amount=<amount>&description=<desc>
	// Web fallback: https://bit.app.link/pay?phone=<phone>&amount=<amount>&description=<desc>

	link := fmt.Sprintf(
		"https://bit.app.link/pay?phone=%s&amount=%.2f&description=%s",
		url.QueryEscape(settings.BitPhoneNumber),
		amount,
		url.QueryEscape(description),
	)

	return link
}

// formatSMSMessage formats an SMS message using template variables
func (s *PaymentLinkServiceImpl) formatSMSMessage(
	settings *models.OrganizationPaymentSettings,
	customer *models.Customer,
	job *models.Job,
	amount float64,
	link string,
) string {
	// Use Hebrew template by default (primary market)
	template := settings.SMSTemplateHe

	// Simple template variable replacement
	message := strings.ReplaceAll(template, "{{customer_name}}", customer.Name)
	message = strings.ReplaceAll(message, "{{job_title}}", job.Title)
	message = strings.ReplaceAll(message, "{{amount}}", fmt.Sprintf("%.2f", amount))
	message = strings.ReplaceAll(message, "{{link}}", link)

	return message
}
