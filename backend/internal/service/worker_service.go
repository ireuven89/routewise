package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

var ErrWorkerNotFound = errors.New("worker not found")

type WorkerSvc struct {
	repo *repository.WorkerRepository
}

type WorkerService interface {
	Create(ctx context.Context, worker CreateWorkerInput) (*models.Worker, error)
	Update(ctx context.Context, id, organizationID uint, input UpdateWorkerInput) (*models.Worker, error)
	Delete(ctx context.Context, organizationID, ID uint) error
	GetAll(ctx context.Context, organizationID uint, activeOnly bool) ([]*models.Worker, error)
	GetByID(ctx context.Context, id, organizationID uint) (*models.Worker, error)
}

func NewWorkerService(repo *repository.WorkerRepository) WorkerService {
	return &WorkerSvc{repo: repo}
}

type CreateWorkerInput struct {
	OrganizationID uint
	CreatedBy      uint
	Name           string
	Email          string
	Phone          string
}

type UpdateWorkerInput struct {
	Name     string
	Email    string
	Phone    string
	IsActive *bool
}

func (s *WorkerSvc) Create(ctx context.Context, input CreateWorkerInput) (*models.Worker, error) {
	worker := &models.Worker{
		OrganizationID: input.OrganizationID,
		CreatedBy:      &input.CreatedBy,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		IsActive:       true, // new workers are always active
	}

	if err := s.repo.Create(worker); err != nil {
		fmt.Println("WorkerSvc.Create Error creating worker", err)
		return nil, err
	}
	return worker, nil
}

func (s *WorkerSvc) GetAll(ctx context.Context, organizationID uint, activeOnly bool) ([]*models.Worker, error) {
	return s.repo.FindAll(organizationID, activeOnly)
}

func (s *WorkerSvc) GetByID(ctx context.Context, id, organizationID uint) (*models.Worker, error) {
	worker, err := s.repo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrWorkerNotFound
	}
	return worker, nil
}

func (s *WorkerSvc) Update(ctx context.Context, id, organizationID uint, input UpdateWorkerInput) (*models.Worker, error) {
	worker, err := s.repo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrWorkerNotFound
	}

	if input.Name != "" {
		worker.Name = input.Name
	}
	worker.Email = input.Email
	if input.Phone != "" {
		worker.Phone = input.Phone
	}
	if input.IsActive != nil {
		worker.IsActive = *input.IsActive
	}

	if err := s.repo.Update(worker); err != nil {
		fmt.Println("WorkerSvc.Update Error updating worker", err)
		return nil, err
	}
	return worker, nil
}

func (s *WorkerSvc) Delete(ctx context.Context, id, organizationID uint) error {
	return s.repo.Delete(id, organizationID)
}
