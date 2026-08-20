package service

import (
	"errors"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

// -----------------------------------------------------------------------
// JobSvc.repo is a concrete *repository.JobRepository and JobSvc.paymentSvc
// is a concrete *PaymentService - neither is an interface - so per this
// package's established convention (see provider_service_test.go,
// job_service_dashboard_test.go, payment_service_test.go) we mirror
// JobSvc.UpdateStatus/Update exactly against a function-field mock repo,
// replacing the paymentSvc.TryAutoSendOnCompletion call with a call-tracking
// slice so tests can assert whether/how often the auto-send hook fires
// without needing a real, fully-wired PaymentService.
// -----------------------------------------------------------------------

type mockJobRepoForStatus struct {
	UpdateStatusFunc func(id, orgID uint, status models.JobStatus) error
	FindByIDFunc     func(id, orgID uint) (*models.Job, error)
	UpdateFunc       func(job *models.Job) error
}

type autoSendCall struct{ orgID, jobID uint }

type testJobStatusService struct {
	repo          *mockJobRepoForStatus
	autoSendCalls []autoSendCall
}

// UpdateStatus mirrors JobSvc.UpdateStatus exactly.
func (s *testJobStatusService) UpdateStatus(id, organizationID uint, status string) error {
	jobStatus := models.JobStatus(status)
	if !validJobStatuses[jobStatus] {
		return ErrInvalidStatus
	}
	if err := s.repo.UpdateStatusFunc(id, organizationID, jobStatus); err != nil {
		return err
	}
	if jobStatus == models.StatusCompleted {
		s.autoSendCalls = append(s.autoSendCalls, autoSendCall{organizationID, id})
	}
	return nil
}

// Update mirrors JobSvc.Update exactly, including the fix under test: it
// never touches job.Status, regardless of what UpdateJobInput.Status holds.
func (s *testJobStatusService) Update(id, organizationID uint, input UpdateJobInput) (*models.Job, error) {
	job, err := s.repo.FindByIDFunc(id, organizationID)
	if err != nil {
		return nil, ErrJobNotFound
	}

	if input.Title != "" {
		job.Title = input.Title
	}
	job.Description = input.Description
	if !input.ScheduledAt.IsZero() {
		job.ScheduledAt = input.ScheduledAt
	}
	if input.DurationMinutes > 0 {
		job.DurationMinutes = input.DurationMinutes
	}
	job.Price = input.Price
	// input.Status is intentionally ignored - status changes must go through
	// UpdateStatus exclusively.
	if input.Metadata != nil {
		job.Metadata = input.Metadata
	}

	if err := s.repo.UpdateFunc(job); err != nil {
		return nil, err
	}
	return job, nil
}

// -----------------------------------------------------------------------
// UpdateStatus tests
// -----------------------------------------------------------------------

func TestJobService_UpdateStatus_InvalidStatus_ReturnsErrInvalidStatus(t *testing.T) {
	repoCalled := false
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		UpdateStatusFunc: func(id, orgID uint, status models.JobStatus) error {
			repoCalled = true
			return nil
		},
	}}

	err := svc.UpdateStatus(1, 1, "not_a_real_status")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
	if repoCalled {
		t.Error("repo should not be called for an invalid status")
	}
	if len(svc.autoSendCalls) != 0 {
		t.Error("auto-send hook should not fire for an invalid status")
	}
}

func TestJobService_UpdateStatus_ValidNonCompletedStatuses_DoNotTriggerAutoSend(t *testing.T) {
	tests := []string{"scheduled", "in_progress", "cancelled"}
	for _, status := range tests {
		t.Run(status, func(t *testing.T) {
			var capturedStatus models.JobStatus
			svc := &testJobStatusService{repo: &mockJobRepoForStatus{
				UpdateStatusFunc: func(id, orgID uint, s models.JobStatus) error {
					capturedStatus = s
					return nil
				},
			}}

			if err := svc.UpdateStatus(1, 1, status); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if string(capturedStatus) != status {
				t.Errorf("expected repo called with status %q, got %q", status, capturedStatus)
			}
			if len(svc.autoSendCalls) != 0 {
				t.Errorf("expected no auto-send hook call for status %q, got %d calls", status, len(svc.autoSendCalls))
			}
		})
	}
}

func TestJobService_UpdateStatus_Completed_TriggersAutoSendHookExactlyOnce(t *testing.T) {
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		UpdateStatusFunc: func(id, orgID uint, s models.JobStatus) error { return nil },
	}}

	if err := svc.UpdateStatus(7, 3, "completed"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(svc.autoSendCalls) != 1 {
		t.Fatalf("expected exactly 1 auto-send hook call, got %d", len(svc.autoSendCalls))
	}
	if svc.autoSendCalls[0].jobID != 7 || svc.autoSendCalls[0].orgID != 3 {
		t.Errorf("expected hook called with jobID=7 orgID=3, got %+v", svc.autoSendCalls[0])
	}
}

func TestJobService_UpdateStatus_RepoError_PropagatesAndSkipsAutoSend(t *testing.T) {
	sentinel := errors.New("db write failed")
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		UpdateStatusFunc: func(id, orgID uint, s models.JobStatus) error { return sentinel },
	}}

	err := svc.UpdateStatus(1, 1, "completed")
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
	if len(svc.autoSendCalls) != 0 {
		t.Error("auto-send hook should not fire when the repo update itself fails")
	}
}

// -----------------------------------------------------------------------
// Update tests - including the regression test for the fixed status-bypass
// bug: Update() must never mutate job.Status, even if UpdateJobInput.Status
// is set to something else. Only UpdateStatus may change status.
// -----------------------------------------------------------------------

func TestJobService_Update_NeverMutatesStatus_RegressionTest(t *testing.T) {
	existing := &models.Job{
		ID:     1,
		Status: models.StatusInProgress,
		Title:  "Original title",
	}
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return existing, nil },
		UpdateFunc:   func(job *models.Job) error { return nil },
	}}

	// Attempt to bypass validation/the auto-send hook by setting Status directly
	// via Update() instead of going through UpdateStatus.
	input := UpdateJobInput{Status: "completed"}
	job, err := svc.Update(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if job.Status != models.StatusInProgress {
		t.Errorf("expected Update() to leave Status untouched (still in_progress), got %q", job.Status)
	}
	if len(svc.autoSendCalls) != 0 {
		t.Error("Update() must never trigger the auto-send-on-completion hook - only UpdateStatus may")
	}
}

func TestJobService_Update_JobNotFound_ReturnsErrJobNotFound(t *testing.T) {
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return nil, errors.New("no rows") },
	}}

	_, err := svc.Update(1, 1, UpdateJobInput{})
	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf("expected ErrJobNotFound, got %v", err)
	}
}

func TestJobService_Update_MergesFieldsWithZeroValueSkipLogic(t *testing.T) {
	existing := &models.Job{
		ID:              1,
		Title:           "Old title",
		Description:     "Old description",
		ScheduledAt:     time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
		DurationMinutes: 60,
		Price:           nil,
		Metadata:        models.JSON{"old": "value"},
	}
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return existing, nil },
		UpdateFunc:   func(job *models.Job) error { return nil },
	}}

	newScheduledAt := time.Date(2026, 6, 15, 14, 0, 0, 0, time.UTC)
	price := 250.0
	input := UpdateJobInput{
		Title:           "New title",
		Description:     "New description", // description is always overwritten (no zero-skip)
		ScheduledAt:     newScheduledAt,
		DurationMinutes: 90,
		Price:           &price,
		Metadata:        models.JSON{"new": "value"},
	}

	job, err := svc.Update(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if job.Title != "New title" {
		t.Errorf("expected title updated, got %q", job.Title)
	}
	if job.Description != "New description" {
		t.Errorf("expected description updated, got %q", job.Description)
	}
	if !job.ScheduledAt.Equal(newScheduledAt) {
		t.Errorf("expected scheduledAt updated, got %v", job.ScheduledAt)
	}
	if job.DurationMinutes != 90 {
		t.Errorf("expected duration updated to 90, got %d", job.DurationMinutes)
	}
	if job.Price == nil || *job.Price != 250.0 {
		t.Errorf("expected price updated to 250.0, got %v", job.Price)
	}
	if job.Metadata["new"] != "value" {
		t.Errorf("expected metadata updated, got %v", job.Metadata)
	}
}

func TestJobService_Update_EmptyOrZeroFields_KeepExistingValues(t *testing.T) {
	original := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	existing := &models.Job{
		ID:              1,
		Title:           "Keep this title",
		ScheduledAt:     original,
		DurationMinutes: 60,
	}
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return existing, nil },
		UpdateFunc:   func(job *models.Job) error { return nil },
	}}

	// Empty title, zero-value ScheduledAt, and zero DurationMinutes must NOT
	// overwrite the existing values.
	input := UpdateJobInput{Title: "", ScheduledAt: time.Time{}, DurationMinutes: 0}
	job, err := svc.Update(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if job.Title != "Keep this title" {
		t.Errorf("expected title unchanged, got %q", job.Title)
	}
	if !job.ScheduledAt.Equal(original) {
		t.Errorf("expected scheduledAt unchanged, got %v", job.ScheduledAt)
	}
	if job.DurationMinutes != 60 {
		t.Errorf("expected duration unchanged, got %d", job.DurationMinutes)
	}
}

func TestJobService_Update_RepoUpdateError_Propagates(t *testing.T) {
	sentinel := errors.New("db write failed")
	existing := &models.Job{ID: 1}
	svc := &testJobStatusService{repo: &mockJobRepoForStatus{
		FindByIDFunc: func(id, orgID uint) (*models.Job, error) { return existing, nil },
		UpdateFunc:   func(job *models.Job) error { return sentinel },
	}}

	_, err := svc.Update(1, 1, UpdateJobInput{})
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}
