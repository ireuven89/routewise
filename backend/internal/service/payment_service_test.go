package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

// -----------------------------------------------------------------------
// Mock repository dependencies (function-field mocks). PaymentService's
// fields are concrete *repository.XxxRepository types - not interfaces -
// so per this package's established convention (see provider_service_test.go,
// job_service_dashboard_test.go) we mirror the service logic against these
// mocks rather than trying to inject a fake into the real struct.
// -----------------------------------------------------------------------

type mockJobRepoForPayment struct {
	FindByIDFunc func(id, orgID uint) (*models.Job, error)
}

type mockCustomerRepoForPayment struct {
	FindByIDFunc func(id, orgID uint) (*models.Customer, error)
}

type mockOrgRepoForPayment struct {
	FindByIDFunc              func(id uint) (*models.Organization, error)
	UpdatePaymentSettingsFunc func(ctx context.Context, orgID uint, enabled bool, phone, businessName string, autoSend bool) error
}

type mockPaymentRepoForPayment struct {
	CreateIfNotActiveFunc func(ctx context.Context, n *models.PaymentNotification) error
	UpdateSMSResultFunc   func(ctx context.Context, id uint, smsStatus models.SMSStatus, paymentStatus models.PaymentStatus, sentAt *time.Time) error
	MarkPaidFunc          func(ctx context.Context, id, orgID uint) error
	ListByJobIDFunc       func(ctx context.Context, jobID, orgID uint) ([]*models.PaymentNotification, error)
}

// fakeNotifier is a hand-rolled test double for the NotificationService
// interface (an actual interface, so no wrapper is required - it can be
// injected directly, unlike the concrete repository dependencies above).
type fakeNotifier struct {
	SendSMSFunc func(phone, message string) error
	calls       []fakeNotifierCall
}

type fakeNotifierCall struct{ phone, message string }

func (f *fakeNotifier) SendSMS(phone, message string) error {
	f.calls = append(f.calls, fakeNotifierCall{phone, message})
	if f.SendSMSFunc != nil {
		return f.SendSMSFunc(phone, message)
	}
	return nil
}

var _ NotificationService = (*fakeNotifier)(nil)

// testPaymentService mirrors PaymentService's methods exactly, delegating to
// mocks in place of the concrete repository fields. It reuses the real
// package-level helper buildPaymentMessage and the real sentinel errors
// (ErrPaymentSettingsNotConfigured etc.) from payment_service.go so only the
// repository-wiring is duplicated, not the business logic itself.
type testPaymentService struct {
	jobRepo      *mockJobRepoForPayment
	customerRepo *mockCustomerRepoForPayment
	orgRepo      *mockOrgRepoForPayment
	paymentRepo  *mockPaymentRepoForPayment
	notifier     NotificationService
}

func (s *testPaymentService) SendPaymentRequest(ctx context.Context, orgID, jobID uint, userID *uint) (*models.PaymentNotification, error) {
	org, err := s.orgRepo.FindByIDFunc(orgID)
	if err != nil {
		return nil, err
	}
	if !org.BitPaymentEnabled || org.BitPhoneNumber == "" {
		return nil, ErrPaymentSettingsNotConfigured
	}

	job, err := s.jobRepo.FindByIDFunc(jobID, orgID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	if job.Status != models.StatusCompleted || job.Price == nil || *job.Price <= 0 {
		return nil, ErrJobNotCompletedForPayment
	}

	customer, err := s.customerRepo.FindByIDFunc(job.CustomerID, orgID)
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
	if err := s.paymentRepo.CreateIfNotActiveFunc(ctx, notification); err != nil {
		return nil, err
	}

	if sendErr := s.notifier.SendSMS(customer.Phone, message); sendErr != nil {
		_ = s.paymentRepo.UpdateSMSResultFunc(ctx, notification.ID, models.SMSStatusFailed, models.PaymentStatusSendFailed, nil)
		notification.SMSStatus = models.SMSStatusFailed
		notification.PaymentStatus = models.PaymentStatusSendFailed
		return notification, errWrapSendFailure(sendErr)
	}

	now := time.Now()
	_ = s.paymentRepo.UpdateSMSResultFunc(ctx, notification.ID, models.SMSStatusSent, models.PaymentStatusSent, &now)
	notification.SMSStatus = models.SMSStatusSent
	notification.PaymentStatus = models.PaymentStatusSent
	notification.SentAt = &now
	return notification, nil
}

func (s *testPaymentService) TryAutoSendOnCompletion(ctx context.Context, orgID, jobID uint) {
	org, err := s.orgRepo.FindByIDFunc(orgID)
	if err != nil || !org.BitPaymentEnabled || !org.AutoSendPaymentSMS {
		return
	}
	_, _ = s.SendPaymentRequest(ctx, orgID, jobID, nil)
}

func (s *testPaymentService) MarkPaid(ctx context.Context, orgID, notificationID uint) error {
	return s.paymentRepo.MarkPaidFunc(ctx, notificationID, orgID)
}

func (s *testPaymentService) GetForJob(ctx context.Context, orgID, jobID uint) ([]*models.PaymentNotification, error) {
	return s.paymentRepo.ListByJobIDFunc(ctx, jobID, orgID)
}

func (s *testPaymentService) UpdateSettings(ctx context.Context, orgID uint, req UpdatePaymentSettingsRequest) error {
	return s.orgRepo.UpdatePaymentSettingsFunc(ctx, orgID, req.BitPaymentEnabled, req.BitPhoneNumber, req.BitBusinessName, req.AutoSendPaymentSMS)
}

func (s *testPaymentService) GetSettings(orgID uint) (*models.Organization, error) {
	return s.orgRepo.FindByIDFunc(orgID)
}

// errWrapSendFailure mirrors payment_service.go's fmt.Errorf("failed to send
// payment SMS: %w", sendErr) wrapping so errors.Is/errors.Unwrap behave the
// same way in tests as in production.
func errWrapSendFailure(sendErr error) error {
	return &sendFailureError{err: sendErr}
}

type sendFailureError struct{ err error }

func (e *sendFailureError) Error() string { return "failed to send payment SMS: " + e.err.Error() }
func (e *sendFailureError) Unwrap() error { return e.err }

// -----------------------------------------------------------------------
// Fixtures
// -----------------------------------------------------------------------

func configuredOrg(orgID uint) *models.Organization {
	return &models.Organization{
		ID:                 orgID,
		BitPaymentEnabled:  true,
		BitPhoneNumber:     "050-9999999",
		BitBusinessName:    "Cool Air Ltd",
		AutoSendPaymentSMS: false,
	}
}

func completedPricedJob(jobID, customerID uint, price float64) *models.Job {
	return &models.Job{
		ID:         jobID,
		CustomerID: customerID,
		Title:      "Fix AC unit",
		Status:     models.StatusCompleted,
		Price:      &price,
	}
}

func customerWithPhone(id uint, phone string) *models.Customer {
	return &models.Customer{ID: id, Name: "Jane Doe", Phone: phone}
}

// -----------------------------------------------------------------------
// SendPaymentRequest tests
// -----------------------------------------------------------------------

func TestPaymentService_SendPaymentRequest_OrgLookupError_Propagates(t *testing.T) {
	sentinel := errors.New("db unavailable")
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return nil, sentinel },
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_NotEnabled_ReturnsErrPaymentSettingsNotConfigured(t *testing.T) {
	jobRepoCalled := false
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) {
				return &models.Organization{BitPaymentEnabled: false, BitPhoneNumber: "050-9999999"}, nil
			},
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				jobRepoCalled = true
				return nil, nil
			},
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, ErrPaymentSettingsNotConfigured) {
		t.Errorf("expected ErrPaymentSettingsNotConfigured, got %v", err)
	}
	if jobRepoCalled {
		t.Error("job repo should not be consulted when payment settings are not configured")
	}
}

func TestPaymentService_SendPaymentRequest_EnabledButNoPhone_ReturnsErrPaymentSettingsNotConfigured(t *testing.T) {
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) {
				return &models.Organization{BitPaymentEnabled: true, BitPhoneNumber: ""}, nil
			},
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, ErrPaymentSettingsNotConfigured) {
		t.Errorf("expected ErrPaymentSettingsNotConfigured, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_JobLookupError_ReturnsErrJobNotFound(t *testing.T) {
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				return nil, errors.New("no rows")
			},
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_JobNotCompleted(t *testing.T) {
	price := 100.0
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				return &models.Job{ID: id, CustomerID: 3, Status: models.StatusInProgress, Price: &price}, nil
			},
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, ErrJobNotCompletedForPayment) {
		t.Errorf("expected ErrJobNotCompletedForPayment, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_JobCompletedButNoPrice(t *testing.T) {
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				return &models.Job{ID: id, CustomerID: 3, Status: models.StatusCompleted, Price: nil}, nil
			},
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, ErrJobNotCompletedForPayment) {
		t.Errorf("expected ErrJobNotCompletedForPayment, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_JobCompletedButZeroOrNegativePrice(t *testing.T) {
	tests := []struct {
		name  string
		price float64
	}{
		{name: "zero price", price: 0},
		{name: "negative price", price: -50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &testPaymentService{
				orgRepo: &mockOrgRepoForPayment{
					FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
				},
				jobRepo: &mockJobRepoForPayment{
					FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
						return &models.Job{ID: id, CustomerID: 3, Status: models.StatusCompleted, Price: &tt.price}, nil
					},
				},
			}

			_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
			if !errors.Is(err, ErrJobNotCompletedForPayment) {
				t.Errorf("expected ErrJobNotCompletedForPayment, got %v", err)
			}
		})
	}
}

func TestPaymentService_SendPaymentRequest_CustomerLookupError_Propagates(t *testing.T) {
	sentinel := errors.New("customer db error")
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 100), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return nil, sentinel },
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_CustomerNoPhone_ReturnsErrCustomerPhoneMissing(t *testing.T) {
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 100), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, ""), nil },
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, ErrCustomerPhoneMissing) {
		t.Errorf("expected ErrCustomerPhoneMissing, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_DuplicateActiveRequest_PropagatesRepoError(t *testing.T) {
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 100), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, "050-1111111"), nil },
		},
		paymentRepo: &mockPaymentRepoForPayment{
			CreateIfNotActiveFunc: func(ctx context.Context, n *models.PaymentNotification) error {
				return repository.ErrPaymentRequestAlreadyActive
			},
		},
		notifier: &fakeNotifier{
			SendSMSFunc: func(phone, message string) error {
				t.Error("notifier should not be called when CreateIfNotActive fails")
				return nil
			},
		},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if !errors.Is(err, repository.ErrPaymentRequestAlreadyActive) {
		t.Errorf("expected repository.ErrPaymentRequestAlreadyActive, got %v", err)
	}
}

func TestPaymentService_SendPaymentRequest_SMSSendFails_NotificationStillCreatedWithSendFailedStatus(t *testing.T) {
	sendErr := errors.New("twilio: rate limited")
	var updateStatusCalled models.PaymentStatus
	var updateSMSStatusCalled models.SMSStatus
	var sentAtCaptured *time.Time
	sentAtCallbackInvoked := false

	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 250), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, "050-1111111"), nil },
		},
		paymentRepo: &mockPaymentRepoForPayment{
			CreateIfNotActiveFunc: func(ctx context.Context, n *models.PaymentNotification) error {
				n.ID = 77
				return nil
			},
			UpdateSMSResultFunc: func(ctx context.Context, id uint, smsStatus models.SMSStatus, paymentStatus models.PaymentStatus, sentAt *time.Time) error {
				updateSMSStatusCalled = smsStatus
				updateStatusCalled = paymentStatus
				sentAtCaptured = sentAt
				sentAtCallbackInvoked = true
				return nil
			},
		},
		notifier: &fakeNotifier{SendSMSFunc: func(phone, message string) error { return sendErr }},
	}

	notification, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if err == nil {
		t.Fatal("expected an error when SMS send fails")
	}
	if !errors.Is(err, sendErr) {
		t.Errorf("expected wrapped sendErr, got %v", err)
	}
	if notification == nil {
		t.Fatal("expected notification to still be returned on SMS failure (it was created in the DB)")
	}
	if notification.ID != 77 {
		t.Errorf("expected notification ID 77 from CreateIfNotActive, got %d", notification.ID)
	}
	if notification.SMSStatus != models.SMSStatusFailed {
		t.Errorf("expected notification.SMSStatus failed, got %v", notification.SMSStatus)
	}
	if notification.PaymentStatus != models.PaymentStatusSendFailed {
		t.Errorf("expected notification.PaymentStatus send_failed, got %v", notification.PaymentStatus)
	}
	if updateSMSStatusCalled != models.SMSStatusFailed {
		t.Errorf("expected UpdateSMSResult called with SMSStatusFailed, got %v", updateSMSStatusCalled)
	}
	if updateStatusCalled != models.PaymentStatusSendFailed {
		t.Errorf("expected UpdateSMSResult called with PaymentStatusSendFailed, got %v", updateStatusCalled)
	}
	if !sentAtCallbackInvoked {
		t.Fatal("expected UpdateSMSResult to be called")
	}
	if sentAtCaptured != nil {
		t.Errorf("expected UpdateSMSResult called with nil sentAt on failure, got %v", sentAtCaptured)
	}
}

func TestPaymentService_SendPaymentRequest_SMSSendSucceeds_MarksSent(t *testing.T) {
	var updateStatusCalled models.PaymentStatus
	var updateSMSStatusCalled models.SMSStatus
	var sentAtCaptured *time.Time
	var notifiedPhone, notifiedMessage string

	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 250), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, "050-1111111"), nil },
		},
		paymentRepo: &mockPaymentRepoForPayment{
			CreateIfNotActiveFunc: func(ctx context.Context, n *models.PaymentNotification) error {
				n.ID = 55
				return nil
			},
			UpdateSMSResultFunc: func(ctx context.Context, id uint, smsStatus models.SMSStatus, paymentStatus models.PaymentStatus, sentAt *time.Time) error {
				updateSMSStatusCalled = smsStatus
				updateStatusCalled = paymentStatus
				sentAtCaptured = sentAt
				return nil
			},
		},
		notifier: &fakeNotifier{SendSMSFunc: func(phone, message string) error {
			notifiedPhone = phone
			notifiedMessage = message
			return nil
		}},
	}

	notification, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if notification.SMSStatus != models.SMSStatusSent {
		t.Errorf("expected SMSStatus sent, got %v", notification.SMSStatus)
	}
	if notification.PaymentStatus != models.PaymentStatusSent {
		t.Errorf("expected PaymentStatus sent, got %v", notification.PaymentStatus)
	}
	if notification.SentAt == nil {
		t.Error("expected SentAt to be set")
	}
	if updateSMSStatusCalled != models.SMSStatusSent || updateStatusCalled != models.PaymentStatusSent {
		t.Errorf("expected UpdateSMSResult called with sent/sent, got %v/%v", updateSMSStatusCalled, updateStatusCalled)
	}
	if sentAtCaptured == nil {
		t.Error("expected UpdateSMSResult called with a non-nil sentAt")
	}
	if notifiedPhone != "050-1111111" {
		t.Errorf("expected notifier called with customer phone, got %q", notifiedPhone)
	}
	if notifiedMessage == "" {
		t.Error("expected a non-empty SMS message body")
	}
}

func TestPaymentService_SendPaymentRequest_MessageIncludesAmountAndBitDetails(t *testing.T) {
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) {
				return &models.Organization{
					BitPaymentEnabled: true,
					BitPhoneNumber:    "050-5555555",
					BitBusinessName:   "Acme HVAC",
				}, nil
			},
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 320), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, "050-1111111"), nil },
		},
		paymentRepo: &mockPaymentRepoForPayment{
			CreateIfNotActiveFunc: func(ctx context.Context, n *models.PaymentNotification) error { return nil },
			UpdateSMSResultFunc:   func(ctx context.Context, id uint, s models.SMSStatus, p models.PaymentStatus, sentAt *time.Time) error { return nil },
		},
		notifier: &fakeNotifier{},
	}

	notification, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	msg := notification.MessageBody
	for _, want := range []string{"320", "050-5555555", "Acme HVAC", "Fix AC unit", "Jane Doe"} {
		if !strings.Contains(msg, want) {
			t.Errorf("expected message to contain %q, got %q", want, msg)
		}
	}
}

func TestPaymentService_SendPaymentRequest_CreatedByPassedThrough(t *testing.T) {
	userID := uint(17)
	var capturedCreatedBy *uint
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 100), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, "050-1111111"), nil },
		},
		paymentRepo: &mockPaymentRepoForPayment{
			CreateIfNotActiveFunc: func(ctx context.Context, n *models.PaymentNotification) error {
				capturedCreatedBy = n.CreatedBy
				return nil
			},
			UpdateSMSResultFunc: func(ctx context.Context, id uint, s models.SMSStatus, p models.PaymentStatus, sentAt *time.Time) error { return nil },
		},
		notifier: &fakeNotifier{},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, &userID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedCreatedBy == nil || *capturedCreatedBy != 17 {
		t.Errorf("expected CreatedBy 17, got %v", capturedCreatedBy)
	}
}

func TestPaymentService_SendPaymentRequest_NilUserID_ForAutoSend(t *testing.T) {
	var capturedCreatedBy *uint
	sawNilAlready := false
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return configuredOrg(id), nil },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 100), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, "050-1111111"), nil },
		},
		paymentRepo: &mockPaymentRepoForPayment{
			CreateIfNotActiveFunc: func(ctx context.Context, n *models.PaymentNotification) error {
				capturedCreatedBy = n.CreatedBy
				sawNilAlready = n.CreatedBy == nil
				return nil
			},
			UpdateSMSResultFunc: func(ctx context.Context, id uint, s models.SMSStatus, p models.PaymentStatus, sentAt *time.Time) error { return nil },
		},
		notifier: &fakeNotifier{},
	}

	_, err := svc.SendPaymentRequest(context.Background(), 1, 2, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !sawNilAlready || capturedCreatedBy != nil {
		t.Errorf("expected nil CreatedBy for auto-send (userID=nil), got %v", capturedCreatedBy)
	}
}

// -----------------------------------------------------------------------
// TryAutoSendOnCompletion tests - must never panic or return an error.
// -----------------------------------------------------------------------

func TestPaymentService_TryAutoSendOnCompletion_OrgLookupError_SwallowedSilently(t *testing.T) {
	sendCalled := false
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return nil, errors.New("db down") },
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				sendCalled = true
				return nil, nil
			},
		},
	}

	// Must not panic.
	svc.TryAutoSendOnCompletion(context.Background(), 1, 2)
	if sendCalled {
		t.Error("expected SendPaymentRequest not to be attempted when org lookup fails")
	}
}

func TestPaymentService_TryAutoSendOnCompletion_BitPaymentDisabled_NoOp(t *testing.T) {
	sendCalled := false
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) {
				return &models.Organization{BitPaymentEnabled: false, AutoSendPaymentSMS: true}, nil
			},
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				sendCalled = true
				return nil, nil
			},
		},
	}

	svc.TryAutoSendOnCompletion(context.Background(), 1, 2)
	if sendCalled {
		t.Error("expected no send attempt when BitPaymentEnabled is false")
	}
}

func TestPaymentService_TryAutoSendOnCompletion_AutoSendDisabled_NoOp(t *testing.T) {
	sendCalled := false
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) {
				return &models.Organization{BitPaymentEnabled: true, BitPhoneNumber: "050-1111111", AutoSendPaymentSMS: false}, nil
			},
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				sendCalled = true
				return nil, nil
			},
		},
	}

	svc.TryAutoSendOnCompletion(context.Background(), 1, 2)
	if sendCalled {
		t.Error("expected no send attempt when AutoSendPaymentSMS is false")
	}
}

func TestPaymentService_TryAutoSendOnCompletion_DownstreamError_SwallowedSilently(t *testing.T) {
	// Org is enabled + auto-send, but the job isn't completed - SendPaymentRequest
	// will fail internally; TryAutoSendOnCompletion must not propagate or panic.
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) {
				return &models.Organization{BitPaymentEnabled: true, BitPhoneNumber: "050-1111111", AutoSendPaymentSMS: true}, nil
			},
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) {
				return &models.Job{ID: id, Status: models.StatusInProgress}, nil // not completed
			},
		},
	}

	// Must not panic; TryAutoSendOnCompletion has no return value to assert on.
	svc.TryAutoSendOnCompletion(context.Background(), 1, 2)
}

func TestPaymentService_TryAutoSendOnCompletion_HappyPath_SendsSMS(t *testing.T) {
	notifierCalled := false
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) {
				return &models.Organization{BitPaymentEnabled: true, BitPhoneNumber: "050-1111111", BitBusinessName: "Biz", AutoSendPaymentSMS: true}, nil
			},
		},
		jobRepo: &mockJobRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return completedPricedJob(id, 3, 100), nil },
		},
		customerRepo: &mockCustomerRepoForPayment{
			FindByIDFunc: func(id, orgID uint) (*models.Customer, error) { return customerWithPhone(id, "050-2222222"), nil },
		},
		paymentRepo: &mockPaymentRepoForPayment{
			CreateIfNotActiveFunc: func(ctx context.Context, n *models.PaymentNotification) error { return nil },
			UpdateSMSResultFunc:   func(ctx context.Context, id uint, s models.SMSStatus, p models.PaymentStatus, sentAt *time.Time) error { return nil },
		},
		notifier: &fakeNotifier{SendSMSFunc: func(phone, message string) error {
			notifierCalled = true
			return nil
		}},
	}

	svc.TryAutoSendOnCompletion(context.Background(), 1, 2)
	if !notifierCalled {
		t.Error("expected the notifier to be invoked on the happy path")
	}
}

// -----------------------------------------------------------------------
// MarkPaid / GetForJob / UpdateSettings / GetSettings delegation tests
// -----------------------------------------------------------------------

func TestPaymentService_MarkPaid_DelegatesWithSwappedArgOrder(t *testing.T) {
	var capturedID, capturedOrgID uint
	svc := &testPaymentService{
		paymentRepo: &mockPaymentRepoForPayment{
			MarkPaidFunc: func(ctx context.Context, id, orgID uint) error {
				capturedID = id
				capturedOrgID = orgID
				return nil
			},
		},
	}

	if err := svc.MarkPaid(context.Background(), 9, 55); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedID != 55 {
		t.Errorf("expected notificationID 55 passed as id, got %d", capturedID)
	}
	if capturedOrgID != 9 {
		t.Errorf("expected orgID 9, got %d", capturedOrgID)
	}
}

func TestPaymentService_MarkPaid_PropagatesNotFoundError(t *testing.T) {
	svc := &testPaymentService{
		paymentRepo: &mockPaymentRepoForPayment{
			MarkPaidFunc: func(ctx context.Context, id, orgID uint) error {
				return repository.ErrPaymentNotificationNotFound
			},
		},
	}

	err := svc.MarkPaid(context.Background(), 9, 55)
	if !errors.Is(err, repository.ErrPaymentNotificationNotFound) {
		t.Errorf("expected ErrPaymentNotificationNotFound, got %v", err)
	}
}

func TestPaymentService_GetForJob_Delegates(t *testing.T) {
	expected := []*models.PaymentNotification{{ID: 1}, {ID: 2}}
	var capturedJobID, capturedOrgID uint
	svc := &testPaymentService{
		paymentRepo: &mockPaymentRepoForPayment{
			ListByJobIDFunc: func(ctx context.Context, jobID, orgID uint) ([]*models.PaymentNotification, error) {
				capturedJobID = jobID
				capturedOrgID = orgID
				return expected, nil
			},
		},
	}

	out, err := svc.GetForJob(context.Background(), 9, 3)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(out))
	}
	if capturedJobID != 3 || capturedOrgID != 9 {
		t.Errorf("expected jobID=3 orgID=9, got jobID=%d orgID=%d", capturedJobID, capturedOrgID)
	}
}

func TestPaymentService_GetForJob_PropagatesError(t *testing.T) {
	sentinel := errors.New("query failed")
	svc := &testPaymentService{
		paymentRepo: &mockPaymentRepoForPayment{
			ListByJobIDFunc: func(ctx context.Context, jobID, orgID uint) ([]*models.PaymentNotification, error) {
				return nil, sentinel
			},
		},
	}

	_, err := svc.GetForJob(context.Background(), 9, 3)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestPaymentService_UpdateSettings_Delegates(t *testing.T) {
	var capturedOrgID uint
	var capturedEnabled, capturedAutoSend bool
	var capturedPhone, capturedBusinessName string
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			UpdatePaymentSettingsFunc: func(ctx context.Context, orgID uint, enabled bool, phone, businessName string, autoSend bool) error {
				capturedOrgID = orgID
				capturedEnabled = enabled
				capturedPhone = phone
				capturedBusinessName = businessName
				capturedAutoSend = autoSend
				return nil
			},
		},
	}

	req := UpdatePaymentSettingsRequest{
		BitPaymentEnabled:  true,
		BitPhoneNumber:     "050-1234567",
		BitBusinessName:    "Acme",
		AutoSendPaymentSMS: true,
	}
	if err := svc.UpdateSettings(context.Background(), 4, req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedOrgID != 4 || !capturedEnabled || capturedPhone != "050-1234567" ||
		capturedBusinessName != "Acme" || !capturedAutoSend {
		t.Errorf("unexpected captured args: orgID=%d enabled=%v phone=%q biz=%q autoSend=%v",
			capturedOrgID, capturedEnabled, capturedPhone, capturedBusinessName, capturedAutoSend)
	}
}

func TestPaymentService_UpdateSettings_PropagatesError(t *testing.T) {
	sentinel := errors.New("write failed")
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			UpdatePaymentSettingsFunc: func(ctx context.Context, orgID uint, enabled bool, phone, businessName string, autoSend bool) error {
				return sentinel
			},
		},
	}

	err := svc.UpdateSettings(context.Background(), 4, UpdatePaymentSettingsRequest{})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestPaymentService_GetSettings_Delegates(t *testing.T) {
	expected := &models.Organization{ID: 4, BitPaymentEnabled: true}
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return expected, nil },
		},
	}

	org, err := svc.GetSettings(4)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if org != expected {
		t.Errorf("expected the repo's organization value, got %+v", org)
	}
}

func TestPaymentService_GetSettings_PropagatesError(t *testing.T) {
	sentinel := errors.New("not found")
	svc := &testPaymentService{
		orgRepo: &mockOrgRepoForPayment{
			FindByIDFunc: func(id uint) (*models.Organization, error) { return nil, sentinel },
		},
	}

	_, err := svc.GetSettings(4)
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}
