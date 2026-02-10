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
	ErrJobNotFound   = errors.New("job not found")
	ErrInvalidStatus = errors.New("invalid status")
)

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
	Delete(id, organizationID uint) error
}

type JobSvc struct {
	repo *repository.JobRepository
}

func NewJobService(repo *repository.JobRepository) *JobSvc {
	return &JobSvc{repo: repo}
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

	response, err := s.repo.CreateServiceCall(ctx, organizationID, request)

	if err != nil {
		fmt.Printf("CreateServiceCall error: %v\n", err)
		return nil, fmt.Errorf("CreateServiceCall: %w", err)
	}

	return response, nil
}

func (s *JobSvc) Create(input CreateJobInput) (*models.Job, error) {
	duration := input.DurationMinutes
	if duration == 0 {
		duration = 60 // default 1 hour
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
	return s.repo.AssignTechnician(id, organizationID, technicianID)
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
