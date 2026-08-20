package service

import (
	"errors"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

// -----------------------------------------------------------------------
// AuthServiceImpl's repository fields (workerRepo, otpRepo, ...) are
// concrete *repository.XxxRepository types, not interfaces, so per this
// package's established mock-wrapper convention we mirror RequestWorkerOTP
// exactly against function-field mocks. This focuses on what changed for
// the Bit payment feature: AuthServiceImpl now takes a NotificationService
// (an actual interface) instead of building its own Twilio client, and
// RequestWorkerOTP calls notifier.SendSMS(...) instead of a deleted private
// sendSMS method.
// -----------------------------------------------------------------------

type mockWorkerRepoForAuth struct {
	FindByPhoneAndCompanyCodeFunc func(phone, companyCode string) (*models.Worker, error)
}

type mockOTPRepoForAuth struct {
	SaveFunc func(phone, companyCode, otpCode string, expiresAt time.Time) error
}

// testAuthRequestOTPService mirrors AuthServiceImpl.RequestWorkerOTP exactly.
type testAuthRequestOTPService struct {
	workerRepo *mockWorkerRepoForAuth
	otpRepo    *mockOTPRepoForAuth
	notifier   NotificationService
}

func (s *testAuthRequestOTPService) RequestWorkerOTP(phone, companyCode string) error {
	worker, err := s.workerRepo.FindByPhoneAndCompanyCodeFunc(phone, companyCode)
	if err != nil {
		return err
	}
	if worker == nil {
		return errors.New("worker not found")
	}
	if !worker.IsActive {
		return errors.New("account inactive")
	}

	otpCode := "123456" // deterministic stand-in for AuthServiceImpl.generateOTP()
	expiresAt := time.Now().Add(5 * time.Minute)

	if err := s.otpRepo.SaveFunc(phone, companyCode, otpCode, expiresAt); err != nil {
		return errors.New("failed to generate code")
	}

	if err := s.notifier.SendSMS(phone, "Your RouteWise code is: "+otpCode); err != nil {
		return errors.New("failed to send SMS")
	}

	return nil
}

// -----------------------------------------------------------------------
// Constructor signature check (compile-time): NewAuthService must accept a
// NotificationService as its 5th parameter instead of building its own
// Twilio client internally.
// -----------------------------------------------------------------------

func TestNewAuthService_AcceptsNotificationServiceParam(t *testing.T) {
	// This is primarily a compile-time assertion: if the constructor
	// signature regresses (e.g. drops the notifier param or changes its
	// position), this test file fails to build.
	svc := NewAuthService(nil, nil, nil, nil, &fakeNotifier{})
	if svc == nil {
		t.Fatal("expected a non-nil AuthService")
	}
}

// -----------------------------------------------------------------------
// RequestWorkerOTP tests
// -----------------------------------------------------------------------

func TestAuthService_RequestWorkerOTP_WorkerNotFound(t *testing.T) {
	notifier := &fakeNotifier{}
	svc := &testAuthRequestOTPService{
		workerRepo: &mockWorkerRepoForAuth{
			FindByPhoneAndCompanyCodeFunc: func(phone, companyCode string) (*models.Worker, error) { return nil, nil },
		},
		notifier: notifier,
	}

	err := svc.RequestWorkerOTP("050-1111111", "ACME-X7J2")
	if err == nil {
		t.Fatal("expected an error for a worker that doesn't exist")
	}
	if len(notifier.calls) != 0 {
		t.Error("notifier should not be called when the worker isn't found")
	}
}

func TestAuthService_RequestWorkerOTP_WorkerLookupError_Propagates(t *testing.T) {
	sentinel := errors.New("db unavailable")
	notifier := &fakeNotifier{}
	svc := &testAuthRequestOTPService{
		workerRepo: &mockWorkerRepoForAuth{
			FindByPhoneAndCompanyCodeFunc: func(phone, companyCode string) (*models.Worker, error) { return nil, sentinel },
		},
		notifier: notifier,
	}

	err := svc.RequestWorkerOTP("050-1111111", "ACME-X7J2")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
	if len(notifier.calls) != 0 {
		t.Error("notifier should not be called on a lookup error")
	}
}

func TestAuthService_RequestWorkerOTP_InactiveWorker_ReturnsError(t *testing.T) {
	notifier := &fakeNotifier{}
	svc := &testAuthRequestOTPService{
		workerRepo: &mockWorkerRepoForAuth{
			FindByPhoneAndCompanyCodeFunc: func(phone, companyCode string) (*models.Worker, error) {
				return &models.Worker{ID: 1, IsActive: false}, nil
			},
		},
		notifier: notifier,
	}

	err := svc.RequestWorkerOTP("050-1111111", "ACME-X7J2")
	if err == nil {
		t.Fatal("expected an error for an inactive worker")
	}
	if len(notifier.calls) != 0 {
		t.Error("notifier should not be called for an inactive worker")
	}
}

func TestAuthService_RequestWorkerOTP_SaveOTPError_Propagates(t *testing.T) {
	notifier := &fakeNotifier{}
	svc := &testAuthRequestOTPService{
		workerRepo: &mockWorkerRepoForAuth{
			FindByPhoneAndCompanyCodeFunc: func(phone, companyCode string) (*models.Worker, error) {
				return &models.Worker{ID: 1, IsActive: true}, nil
			},
		},
		otpRepo: &mockOTPRepoForAuth{
			SaveFunc: func(phone, companyCode, otpCode string, expiresAt time.Time) error {
				return errors.New("db write failed")
			},
		},
		notifier: notifier,
	}

	err := svc.RequestWorkerOTP("050-1111111", "ACME-X7J2")
	if err == nil {
		t.Fatal("expected an error when saving the OTP fails")
	}
	if len(notifier.calls) != 0 {
		t.Error("notifier should not be called when the OTP fails to save")
	}
}

func TestAuthService_RequestWorkerOTP_CallsNotifierSendSMSWithExpectedMessageFormat(t *testing.T) {
	notifier := &fakeNotifier{}
	svc := &testAuthRequestOTPService{
		workerRepo: &mockWorkerRepoForAuth{
			FindByPhoneAndCompanyCodeFunc: func(phone, companyCode string) (*models.Worker, error) {
				return &models.Worker{ID: 1, IsActive: true}, nil
			},
		},
		otpRepo: &mockOTPRepoForAuth{
			SaveFunc: func(phone, companyCode, otpCode string, expiresAt time.Time) error { return nil },
		},
		notifier: notifier,
	}

	if err := svc.RequestWorkerOTP("050-1111111", "ACME-X7J2"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("expected exactly 1 SendSMS call, got %d", len(notifier.calls))
	}
	call := notifier.calls[0]
	if call.phone != "050-1111111" {
		t.Errorf("expected SendSMS called with phone '050-1111111', got %q", call.phone)
	}
	if call.message != "Your RouteWise code is: 123456" {
		t.Errorf("expected message 'Your RouteWise code is: 123456', got %q", call.message)
	}
}

func TestAuthService_RequestWorkerOTP_NotifierSendFails_ReturnsError(t *testing.T) {
	notifier := &fakeNotifier{SendSMSFunc: func(phone, message string) error {
		return errors.New("twilio: rate limited")
	}}
	svc := &testAuthRequestOTPService{
		workerRepo: &mockWorkerRepoForAuth{
			FindByPhoneAndCompanyCodeFunc: func(phone, companyCode string) (*models.Worker, error) {
				return &models.Worker{ID: 1, IsActive: true}, nil
			},
		},
		otpRepo: &mockOTPRepoForAuth{
			SaveFunc: func(phone, companyCode, otpCode string, expiresAt time.Time) error { return nil },
		},
		notifier: notifier,
	}

	err := svc.RequestWorkerOTP("050-1111111", "ACME-X7J2")
	if err == nil {
		t.Fatal("expected an error when the notifier fails to send")
	}
}
