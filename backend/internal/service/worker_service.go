package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

var ErrWorkerNotFound = errors.New("worker not found")

type WorkerSvc struct {
	repo             *repository.WorkerRepository
	geocodingService GeocodingService
}

type WorkerService interface {
	Create(ctx context.Context, worker CreateWorkerInput) (*models.Worker, error)
	Update(ctx context.Context, id, organizationID uint, input UpdateWorkerInput) (*models.Worker, error)
	Delete(ctx context.Context, organizationID, ID uint) error
	GetAll(ctx context.Context, organizationID uint, activeOnly bool) ([]*models.Worker, error)
	GetByID(ctx context.Context, id, organizationID uint) (*models.Worker, error)
}

func NewWorkerService(repo *repository.WorkerRepository, geocodingService GeocodingService) WorkerService {
	return &WorkerSvc{
		repo:             repo,
		geocodingService: geocodingService,
	}
}

type CreateWorkerInput struct {
	OrganizationID        uint
	CreatedBy             uint
	Name                  string
	Email                 string
	Phone                 string
	HomeAddress           string
	HomeLatitude          *float64
	HomeLongitude         *float64
	HomeGooglePlaceID     string
	HomeFormattedAddress  string
	HomeAddressComponents map[string]interface{}
}

type UpdateWorkerInput struct {
	Name                  string
	Email                 string
	Phone                 string
	IsActive              *bool
	HomeAddress           string
	HomeLatitude          *float64
	HomeLongitude         *float64
	HomeGooglePlaceID     string
	HomeFormattedAddress  string
	HomeAddressComponents map[string]interface{}
}

func (s *WorkerSvc) Create(ctx context.Context, input CreateWorkerInput) (*models.Worker, error) {
	worker := &models.Worker{
		OrganizationID:        input.OrganizationID,
		CreatedBy:             &input.CreatedBy,
		Name:                  input.Name,
		Email:                 input.Email,
		Phone:                 input.Phone,
		IsActive:              true, // new workers are always active
		HomeAddress:           input.HomeAddress,
		HomeLatitude:          input.HomeLatitude,
		HomeLongitude:         input.HomeLongitude,
		HomeGooglePlaceID:     input.HomeGooglePlaceID,
		HomeFormattedAddress:  input.HomeFormattedAddress,
		HomeAddressComponents: input.HomeAddressComponents,
	}

	// If home address provided but no coordinates, geocode it
	if worker.HomeAddress != "" && (worker.HomeLatitude == nil || worker.HomeLongitude == nil) && s.geocodingService != nil {
		result, err := s.geocodingService.GeocodeAddress(worker.HomeAddress)
		if err == nil {
			worker.HomeLatitude = &result.Latitude
			worker.HomeLongitude = &result.Longitude
			worker.HomeGooglePlaceID = result.PlaceID
			worker.HomeFormattedAddress = result.FormattedAddress
			worker.HomeAddressComponents = convertStringMapToInterfaceMap(result.AddressComponents)
		}
	}

	// If coordinates are set, mark as geocoded
	if worker.HomeLatitude != nil && worker.HomeLongitude != nil {
		now := time.Now()
		worker.HomeGeocodedAt = &now
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

	homeAddressChanged := false
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

	// Update home address fields
	if input.HomeAddress != "" && input.HomeAddress != worker.HomeAddress {
		worker.HomeAddress = input.HomeAddress
		homeAddressChanged = true
	}
	worker.HomeLatitude = input.HomeLatitude
	worker.HomeLongitude = input.HomeLongitude
	worker.HomeGooglePlaceID = input.HomeGooglePlaceID
	worker.HomeFormattedAddress = input.HomeFormattedAddress
	worker.HomeAddressComponents = input.HomeAddressComponents

	// If home address changed and no coordinates provided, geocode it
	if homeAddressChanged && (worker.HomeLatitude == nil || worker.HomeLongitude == nil) && worker.HomeAddress != "" && s.geocodingService != nil {
		result, err := s.geocodingService.GeocodeAddress(worker.HomeAddress)
		if err == nil {
			worker.HomeLatitude = &result.Latitude
			worker.HomeLongitude = &result.Longitude
			worker.HomeGooglePlaceID = result.PlaceID
			worker.HomeFormattedAddress = result.FormattedAddress
			worker.HomeAddressComponents = convertStringMapToInterfaceMap(result.AddressComponents)
			now := time.Now()
			worker.HomeGeocodedAt = &now
		}
	}

	// If home coordinates were provided (from frontend autocomplete), mark as geocoded
	if worker.HomeLatitude != nil && worker.HomeLongitude != nil && (worker.HomeGeocodedAt == nil || homeAddressChanged) {
		now := time.Now()
		worker.HomeGeocodedAt = &now
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
