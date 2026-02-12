package service

import (
	"errors"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockPaymentLinkService implements PaymentLinkService for testing
type MockPaymentLinkService struct {
	mock.Mock
}

func (m *MockPaymentLinkService) SendPaymentLinkForJob(organizationID, jobID, userID uint) (*models.PaymentNotification, error) {
	args := m.Called(organizationID, jobID, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PaymentNotification), args.Error(1)
}

func (m *MockPaymentLinkService) GetPaymentNotifications(jobID, organizationID uint) ([]*models.PaymentNotification, error) {
	args := m.Called(jobID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.PaymentNotification), args.Error(1)
}

func (m *MockPaymentLinkService) GetOrganizationSettings(organizationID uint) (*models.OrganizationPaymentSettings, error) {
	args := m.Called(organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.OrganizationPaymentSettings), args.Error(1)
}

// Tests for maybeAutoSendPaymentLink function
// Note: UpdateStatus repository integration is tested separately in integration tests
// These tests focus on the payment link auto-send business logic

func TestJobService_MaybeAutoSendPaymentLink_Enabled(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	// Mock settings with auto-send enabled
	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    true,
		AutoSendOnCompletion: true,
		BitPhoneNumber:       "+972501234567",
	}

	// Mock settings retrieval
	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil)

	// Mock successful payment link send
	notification := &models.PaymentNotification{
		ID:             1,
		OrganizationID: organizationID,
		JobID:          jobID,
		Amount:         500.00,
	}
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, jobID, uint(0)).
		Return(notification, nil)

	// Execute the maybeAutoSendPaymentLink directly
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	// Give goroutine time to execute (maybeAutoSendPaymentLink is synchronous, but keeping for safety)
	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
}

func TestJobService_MaybeAutoSendPaymentLink_Disabled(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	// Mock settings with auto-send disabled
	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    true,
		AutoSendOnCompletion: false, // Disabled
	}

	// Mock settings retrieval
	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil)

	// SendPaymentLinkForJob should NOT be called

	// Execute
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	// Give time for any potential execution
	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
	// Verify SendPaymentLinkForJob was NOT called
	mockPaymentLinkService.AssertNotCalled(t, "SendPaymentLinkForJob")
}

func TestJobService_MaybeAutoSendPaymentLink_PaymentLinkServiceNil(t *testing.T) {
	// Create job service without payment link service (nil)
	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: nil,
	}

	organizationID := uint(1)
	jobID := uint(100)

	// Execute - should not panic
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	// Success if no panic
	assert.True(t, true)
}

func TestJobService_MaybeAutoSendPaymentLink_SettingsError(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	// Mock settings error
	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).
		Return(nil, errors.New("settings not found"))

	// Execute
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
	// SendPaymentLinkForJob should NOT be called if settings retrieval fails
	mockPaymentLinkService.AssertNotCalled(t, "SendPaymentLinkForJob")
}

func TestJobService_MaybeAutoSendPaymentLink_SendError(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    true,
		AutoSendOnCompletion: true,
	}

	// Mock settings retrieval
	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil)

	// Mock send error (e.g., job has no price)
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, jobID, uint(0)).
		Return(nil, ErrJobHasNoPrice)

	// Execute - should not panic, error is logged
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
}

func TestJobService_MaybeAutoSendPaymentLink_PaymentNotEnabled(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	// Mock settings with payment disabled
	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    false, // Disabled
		AutoSendOnCompletion: true,
	}

	// Mock settings retrieval
	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil)

	// Mock send error
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, jobID, uint(0)).
		Return(nil, ErrPaymentNotEnabled)

	// Execute
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
}

func TestJobService_MaybeAutoSendPaymentLink_UserIDIsZero(t *testing.T) {
	// Verify that auto-send uses userID=0 to distinguish from manual sends
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    true,
		AutoSendOnCompletion: true,
	}

	notification := &models.PaymentNotification{ID: 1, JobID: jobID, CreatedBy: uintPtr(0)}

	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil)

	// IMPORTANT: Verify userID is 0 for auto-send
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, jobID, uint(0)).
		Return(notification, nil)

	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
}

func TestJobService_UpdateStatus_InvalidStatus(t *testing.T) {
	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: nil,
	}

	organizationID := uint(1)
	jobID := uint(100)

	// Execute with invalid status
	err := jobService.UpdateStatus(jobID, organizationID, "invalid_status")

	// Assert
	assert.Error(t, err)
	assert.Equal(t, ErrInvalidStatus, err)
}

func TestJobService_UpdateStatus_ValidStatuses(t *testing.T) {
	tests := []struct {
		name   string
		status string
		valid  bool
	}{
		{"valid - scheduled", "scheduled", true},
		{"valid - in_progress", "in_progress", true},
		{"valid - completed", "completed", true},
		{"valid - cancelled", "cancelled", true},
		{"invalid - pending", "pending", false},
		{"invalid - draft", "draft", false},
		{"invalid - empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We're only testing validation, not repository interaction
			jobStatus := models.JobStatus(tt.status)
			_, isValid := validJobStatuses[jobStatus]

			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestJobService_MaybeAutoSendPaymentLink_MultipleJobs(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	job1ID := uint(100)
	job2ID := uint(101)

	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    true,
		AutoSendOnCompletion: true,
	}

	notification1 := &models.PaymentNotification{ID: 1, JobID: job1ID}
	notification2 := &models.PaymentNotification{ID: 2, JobID: job2ID}

	// Mock for job 1
	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil).Once()
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, job1ID, uint(0)).
		Return(notification1, nil).Once()

	// Mock for job 2
	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil).Once()
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, job2ID, uint(0)).
		Return(notification2, nil).Once()

	// Execute both
	jobService.maybeAutoSendPaymentLink(organizationID, job1ID)
	jobService.maybeAutoSendPaymentLink(organizationID, job2ID)

	// Give time for execution
	time.Sleep(100 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
}

func TestJobService_MaybeAutoSendPaymentLink_PaymentAlreadySent(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    true,
		AutoSendOnCompletion: true,
	}

	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil)
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, jobID, uint(0)).
		Return(nil, ErrPaymentAlreadySent)

	// Execute - should handle error gracefully
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
}

func TestJobService_MaybeAutoSendPaymentLink_CustomerHasNoPhone(t *testing.T) {
	mockPaymentLinkService := new(MockPaymentLinkService)

	jobService := &JobSvc{
		repo:               nil,
		paymentLinkService: mockPaymentLinkService,
	}

	organizationID := uint(1)
	jobID := uint(100)

	settings := &models.OrganizationPaymentSettings{
		ID:                   1,
		OrganizationID:       organizationID,
		BitPaymentEnabled:    true,
		AutoSendOnCompletion: true,
	}

	mockPaymentLinkService.On("GetOrganizationSettings", organizationID).Return(settings, nil)
	mockPaymentLinkService.On("SendPaymentLinkForJob", organizationID, jobID, uint(0)).
		Return(nil, ErrCustomerHasNoPhone)

	// Execute - should handle error gracefully (logged, not returned)
	jobService.maybeAutoSendPaymentLink(organizationID, jobID)

	time.Sleep(50 * time.Millisecond)

	mockPaymentLinkService.AssertExpectations(t)
}

// Helper function
func uintPtr(u uint) *uint {
	return &u
}
