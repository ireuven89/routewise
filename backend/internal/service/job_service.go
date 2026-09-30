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
	// ErrJobNotFound is the repository's sentinel, so "not found" from either layer matches.
	ErrJobNotFound   = repository.ErrJobNotFound
	ErrInvalidStatus = errors.New("invalid status")
	// ErrInvalidTechnician: the technician doesn't exist in this organization or is inactive.
	ErrInvalidTechnician = errors.New("invalid technician")
	// ErrInvalidAssignmentState: e.g. accepting a job that isn't a pending offer.
	ErrInvalidAssignmentState = errors.New("job is not in a state that allows this")
	// ErrMustAcceptFirst: a technician tried to work on a job they haven't accepted.
	ErrMustAcceptFirst = errors.New("accept the job first")
	// ErrInvalidTransition: a technician tried a status change their job's current status doesn't allow.
	ErrInvalidTransition = errors.New("job status does not allow this change")
)

// Statuses a technician may move their own job to (owners can set any valid status).
// Maps each status a technician may set to the status the job must currently be in.
var workerSettableStatuses = map[models.JobStatus]models.JobStatus{
	models.StatusInProgress: models.StatusScheduled,
	models.StatusCompleted:  models.StatusInProgress,
}

// workerLookup is the part of the worker repository the job service needs.
type workerLookup interface {
	FindByID(id uint, organizationID uint) (*models.Worker, error)
}

var validJobStatuses = map[models.JobStatus]bool{
	models.StatusScheduled:  true,
	models.StatusInProgress: true,
	models.StatusCompleted:  true,
	models.StatusCancelled:  true,
}

type JobService interface {
	CreateServiceCall(ctx context.Context, organizationID uint, request *models.CreateServiceCallRequest) (*models.CreateServiceCallResponse, error)
	Create(input CreateJobInput) (*models.Job, error)
	GetAll(organizationID uint, filters map[string]interface{}, sortBy string) ([]*models.Job, error)
	GetByID(id, organizationID uint) (*models.Job, error)
	Update(id, organizationID uint, input UpdateJobInput) (*models.Job, error)
	AssignTechnician(id, organizationID uint, technicianID *uint) error
	UpdateStatus(id, organizationID uint, status string) error

	// Technician-side (worker token) operations; all are limited to the worker's own jobs.
	GetByIDForWorker(id, organizationID, workerID uint) (*models.Job, error)
	UpdateStatusAsWorker(id, organizationID, workerID uint, status string) error
	AcceptJob(id, organizationID, workerID uint) error
	DeclineJob(id, organizationID, workerID uint, reason string) error
	Delete(id, organizationID uint) error
	GetDashboardStats(organizationID uint) (*models.DashboardStats, error)
}

type JobSvc struct {
	repo    *repository.JobRepository
	workers workerLookup
}

func NewJobService(repo *repository.JobRepository, workers workerLookup) *JobSvc {
	return &JobSvc{repo: repo, workers: workers}
}

// validateTechnician rejects assigning a technician from another organization or an inactive one.
func (s *JobSvc) validateTechnician(organizationID uint, technicianID *uint) error {
	if technicianID == nil {
		return nil
	}
	w, err := s.workers.FindByID(*technicianID, organizationID)
	if err != nil || w == nil || !w.IsActive {
		return ErrInvalidTechnician
	}
	return nil
}

type CreateJobInput struct {
	OrganizationID  uint
	CreatedBy       uint
	CustomerID      uint
	TechnicianID    *uint
	Title           string
	Description     string
	ScheduledAt     time.Time
	DurationMinutes int
	Price           *float64
	Metadata        models.JSON
}

type UpdateJobInput struct {
	Title           string
	Description     string
	ScheduledAt     time.Time
	DurationMinutes int
	Price           *float64
	Status          string
	Metadata        models.JSON
}

func (s *JobSvc) CreateServiceCall(ctx context.Context, organizationID uint, request *models.CreateServiceCallRequest) (*models.CreateServiceCallResponse, error) {
	if err := s.validateTechnician(organizationID, request.TechnicianID); err != nil {
		return nil, err
	}

	response, err := s.repo.CreateServiceCall(ctx, organizationID, request)

	if err != nil {
		fmt.Printf("CreateServiceCall error: %v\n", err)
		return nil, fmt.Errorf("CreateServiceCall: %w", err)
	}

	return response, nil
}

func (s *JobSvc) Create(input CreateJobInput) (*models.Job, error) {
	if err := s.validateTechnician(input.OrganizationID, input.TechnicianID); err != nil {
		return nil, err
	}
	duration := input.DurationMinutes
	if duration == 0 {
		duration = repository.DefaultJobDurationMinutes
	}

	job := &models.Job{
		OrganizationID:  input.OrganizationID,
		CreatedBy:       &input.CreatedBy,
		CustomerID:      input.CustomerID,
		TechnicianID:    input.TechnicianID,
		Title:           input.Title,
		Description:     input.Description,
		Status:          models.StatusScheduled,
		ScheduledAt:     input.ScheduledAt,
		DurationMinutes: duration,
		Price:           input.Price,
		Metadata:        input.Metadata,
	}

	if err := s.repo.Create(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobSvc) GetAll(organizationID uint, filters map[string]interface{}, sortBy string) ([]*models.Job, error) {
	if sortBy == "" {
		sortBy = "created_at"
	}
	return s.repo.FindAll(organizationID, filters, sortBy)
}

func (s *JobSvc) GetByID(id, organizationID uint) (*models.Job, error) {
	job, err := s.repo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrJobNotFound
	}
	return job, nil
}

func (s *JobSvc) Update(id, organizationID uint, input UpdateJobInput) (*models.Job, error) {
	job, err := s.repo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrJobNotFound
	}

	// Merge fields — only overwrite if the new value is non-zero
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
	if input.Status != "" {
		job.Status = models.JobStatus(input.Status)
	}
	if input.Metadata != nil {
		job.Metadata = input.Metadata
	}

	if err := s.repo.Update(job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *JobSvc) AssignTechnician(id, organizationID uint, technicianID *uint) error {
	if err := s.validateTechnician(organizationID, technicianID); err != nil {
		return err
	}
	return s.repo.AssignTechnician(id, organizationID, technicianID)
}

// GetByIDForWorker returns the job only if it's assigned to this worker; anything else is
// reported as not found so technicians can't probe other jobs.
func (s *JobSvc) GetByIDForWorker(id, organizationID, workerID uint) (*models.Job, error) {
	job, err := s.repo.FindByID(id, organizationID)
	if err != nil || job.TechnicianID == nil || *job.TechnicianID != workerID {
		return nil, ErrJobNotFound
	}
	return job, nil
}

// UpdateStatusAsWorker lets a technician start/complete their own job, but only after accepting it.
func (s *JobSvc) UpdateStatusAsWorker(id, organizationID, workerID uint, status string) error {
	jobStatus := models.JobStatus(status)
	from, ok := workerSettableStatuses[jobStatus]
	if !ok {
		return ErrInvalidStatus
	}
	job, err := s.GetByIDForWorker(id, organizationID, workerID)
	if err != nil {
		return err
	}
	if job.AssignmentStatus == nil || *job.AssignmentStatus != models.AssignmentAccepted {
		return ErrMustAcceptFirst
	}
	if job.Status != from {
		return ErrInvalidTransition
	}
	updated, err := s.repo.UpdateStatusAsWorker(id, organizationID, workerID, from, jobStatus)
	if err != nil {
		return err
	}
	if !updated { // changed underneath us, e.g. the owner cancelled it
		return ErrInvalidTransition
	}
	return nil
}

func (s *JobSvc) AcceptJob(id, organizationID, workerID uint) error {
	job, err := s.GetByIDForWorker(id, organizationID, workerID)
	if err != nil {
		return err
	}
	if job.AssignmentStatus == nil || *job.AssignmentStatus != models.AssignmentPending ||
		job.Status == models.StatusCancelled || job.Status == models.StatusCompleted {
		return ErrInvalidAssignmentState
	}
	ok, err := s.repo.AcceptAssignment(id, organizationID, workerID)
	if err != nil {
		return err
	}
	if !ok { // changed underneath us (reassigned/declined/cancelled)
		return ErrInvalidAssignmentState
	}
	return nil
}

// DeclineJob sends a scheduled job back to the unassigned pool, recording who declined and why.
func (s *JobSvc) DeclineJob(id, organizationID, workerID uint, reason string) error {
	job, err := s.GetByIDForWorker(id, organizationID, workerID)
	if err != nil {
		return err
	}
	if job.Status != models.StatusScheduled {
		return ErrInvalidAssignmentState
	}
	ok, err := s.repo.DeclineAssignment(id, organizationID, workerID, reason)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidAssignmentState
	}
	return nil
}

func (s *JobSvc) UpdateStatus(id, organizationID uint, status string) error {
	jobStatus := models.JobStatus(status)
	if !validJobStatuses[jobStatus] {
		return ErrInvalidStatus
	}
	return s.repo.UpdateStatus(id, organizationID, jobStatus)
}

func (s *JobSvc) Delete(id, organizationID uint) error {
	return s.repo.Delete(id, organizationID)
}

func (s *JobSvc) GetDashboardStats(organizationID uint) (*models.DashboardStats, error) {
	revenue, err := s.repo.GetRevenueStats(organizationID)
	if err != nil {
		return nil, err
	}
	return &models.DashboardStats{Revenue: revenue}, nil
}
