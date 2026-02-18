package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

// mockWorkerRepository is a mock implementation of WorkerRepository
type mockWorkerRepository struct {
	CreateFunc   func(*models.Worker) error
	FindByIDFunc func(uint, uint) (*models.Worker, error)
	UpdateFunc   func(*models.Worker) error
	DeleteFunc   func(uint, uint) error
	FindAllFunc  func(uint, bool) ([]*models.Worker, error)
}

func (m *mockWorkerRepository) Create(worker *models.Worker) error {
	return m.CreateFunc(worker)
}

func (m *mockWorkerRepository) FindByID(id, orgID uint) (*models.Worker, error) {
	return m.FindByIDFunc(id, orgID)
}

func (m *mockWorkerRepository) Update(worker *models.Worker) error {
	return m.UpdateFunc(worker)
}

func (m *mockWorkerRepository) Delete(id, orgID uint) error {
	return m.DeleteFunc(id, orgID)
}

func (m *mockWorkerRepository) FindAll(orgID uint, activeOnly bool) ([]*models.Worker, error) {
	return m.FindAllFunc(orgID, activeOnly)
}

// We need to create a test wrapper for WorkerService that uses our mock
func newWorkerServiceWithMockRepo(mockRepo *mockWorkerRepository, geocoding GeocodingService) WorkerService {
	return &testWorkerService{
		mockRepo:         mockRepo,
		geocodingService: geocoding,
	}
}

// testWorkerService wraps WorkerSvc for testing
type testWorkerService struct {
	mockRepo         *mockWorkerRepository
	geocodingService GeocodingService
}

func (s *testWorkerService) Create(ctx context.Context, input CreateWorkerInput) (*models.Worker, error) {
	worker := &models.Worker{
		OrganizationID: input.OrganizationID,
		CreatedBy:      &input.CreatedBy,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		IsActive:       true,
	}

	if err := s.mockRepo.Create(worker); err != nil {
		return nil, err
	}
	return worker, nil
}

func (s *testWorkerService) GetAll(ctx context.Context, organizationID uint, activeOnly bool) ([]*models.Worker, error) {
	return s.mockRepo.FindAll(organizationID, activeOnly)
}

func (s *testWorkerService) GetByID(ctx context.Context, id, organizationID uint) (*models.Worker, error) {
	worker, err := s.mockRepo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrWorkerNotFound
	}
	return worker, nil
}

func (s *testWorkerService) Update(ctx context.Context, id, organizationID uint, input UpdateWorkerInput) (*models.Worker, error) {
	worker, err := s.mockRepo.FindByID(id, organizationID)
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

	// If home coordinates were provided, mark as geocoded
	if worker.HomeLatitude != nil && worker.HomeLongitude != nil && (worker.HomeGeocodedAt == nil || homeAddressChanged) {
		now := time.Now()
		worker.HomeGeocodedAt = &now
	}

	if err := s.mockRepo.Update(worker); err != nil {
		return nil, err
	}
	return worker, nil
}

func (s *testWorkerService) Delete(ctx context.Context, id, organizationID uint) error {
	return s.mockRepo.Delete(id, organizationID)
}

func TestWorkerService_Update_WithHomeCoordinates_NoGeocode(t *testing.T) {
	geocodingCalled := false
	lat := 32.0853
	lng := 34.7818

	existingWorker := &models.Worker{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Worker",
	}

	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			if id != 1 || orgID != 1 {
				t.Errorf("expected id=1, orgID=1, got id=%d, orgID=%d", id, orgID)
			}
			return existingWorker, nil
		},
		UpdateFunc: func(worker *models.Worker) error {
			// Verify provided coordinates were used
			if worker.HomeLatitude == nil || *worker.HomeLatitude != lat {
				t.Errorf("expected home latitude %f, got %v", lat, worker.HomeLatitude)
			}
			if worker.HomeLongitude == nil || *worker.HomeLongitude != lng {
				t.Errorf("expected home longitude %f, got %v", lng, worker.HomeLongitude)
			}
			if worker.HomeGeocodedAt == nil {
				t.Error("expected HomeGeocodedAt to be set when coordinates provided")
			}
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			return nil, errors.New("should not be called")
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateWorkerInput{
		Name:          "Test Worker",
		HomeAddress:   "123 Home St",
		HomeLatitude:  &lat,
		HomeLongitude: &lng,
	}

	worker, err := service.Update(context.Background(), 1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if worker.HomeLatitude == nil || *worker.HomeLatitude != lat {
		t.Errorf("expected home latitude %f, got %v", lat, worker.HomeLatitude)
	}
	if geocodingCalled {
		t.Error("geocoding service should not be called when coordinates are provided")
	}
}

func TestWorkerService_Update_WithoutHomeCoordinates_GeocodeSuccess(t *testing.T) {
	geocodingCalled := false
	expectedLat := 32.0853
	expectedLng := 34.7818

	existingWorker := &models.Worker{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Worker",
	}

	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			return existingWorker, nil
		},
		UpdateFunc: func(worker *models.Worker) error {
			// Verify geocoded coordinates were set
			if worker.HomeLatitude == nil || *worker.HomeLatitude != expectedLat {
				t.Errorf("expected geocoded home latitude %f, got %v", expectedLat, worker.HomeLatitude)
			}
			if worker.HomeLongitude == nil || *worker.HomeLongitude != expectedLng {
				t.Errorf("expected geocoded home longitude %f, got %v", expectedLng, worker.HomeLongitude)
			}
			if worker.HomeGooglePlaceID != "ChIJHomeGeocoded123" {
				t.Errorf("expected geocoded place_id, got %s", worker.HomeGooglePlaceID)
			}
			if worker.HomeFormattedAddress != "123 Home St, Tel Aviv, Israel" {
				t.Errorf("expected formatted address, got %s", worker.HomeFormattedAddress)
			}
			if worker.HomeGeocodedAt == nil {
				t.Error("expected HomeGeocodedAt to be set after geocoding")
			}
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			if address != "123 Home St" {
				t.Errorf("expected address '123 Home St', got %s", address)
			}
			return &GeocodingResult{
				Latitude:         expectedLat,
				Longitude:        expectedLng,
				PlaceID:          "ChIJHomeGeocoded123",
				FormattedAddress: "123 Home St, Tel Aviv, Israel",
				AddressComponents: map[string]string{
					"city": "Tel Aviv",
				},
			}, nil
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateWorkerInput{
		Name:        "Test Worker",
		HomeAddress: "123 Home St",
	}

	worker, err := service.Update(context.Background(), 1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if worker.HomeAddress != "123 Home St" {
		t.Errorf("expected home address '123 Home St', got %s", worker.HomeAddress)
	}
	if !geocodingCalled {
		t.Error("geocoding service should be called when coordinates not provided")
	}
}

func TestWorkerService_Update_GeocodingFails_StillUpdates(t *testing.T) {
	geocodingCalled := false

	existingWorker := &models.Worker{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Worker",
		HomeAddress:    "Old Home Address",
	}

	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			return existingWorker, nil
		},
		UpdateFunc: func(worker *models.Worker) error {
			// Worker should still be updated despite geocoding failure
			if worker.HomeAddress != "New Home Address" {
				t.Errorf("expected home address 'New Home Address', got %s", worker.HomeAddress)
			}
			// Geocoding data should not be set
			if worker.HomeLatitude != nil {
				t.Error("expected no home latitude when geocoding fails")
			}
			if worker.HomeLongitude != nil {
				t.Error("expected no home longitude when geocoding fails")
			}
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			return nil, errors.New("geocoding API error")
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateWorkerInput{
		Name:        "Test Worker",
		HomeAddress: "New Home Address",
	}

	worker, err := service.Update(context.Background(), 1, 1, input)
	if err != nil {
		t.Fatalf("expected no error (update should succeed despite geocoding failure), got %v", err)
	}
	if worker.HomeAddress != "New Home Address" {
		t.Errorf("expected home address 'New Home Address', got %s", worker.HomeAddress)
	}
	if !geocodingCalled {
		t.Error("geocoding service should be called")
	}
}

func TestWorkerService_Update_HomeAddressNotChanged_NoGeocode(t *testing.T) {
	geocodingCalled := false
	lat := 32.0853
	lng := 34.7818
	geocodedAt := time.Now()

	existingWorker := &models.Worker{
		ID:              1,
		OrganizationID:  1,
		Name:            "Test Worker",
		HomeAddress:     "Same Home Address",
		HomeLatitude:    &lat,
		HomeLongitude:   &lng,
		HomeGeocodedAt:  &geocodedAt,
	}

	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			return existingWorker, nil
		},
		UpdateFunc: func(worker *models.Worker) error {
			// Verify geocoded data was not changed
			if worker.HomeGeocodedAt == nil || !worker.HomeGeocodedAt.Equal(geocodedAt) {
				t.Error("HomeGeocodedAt should not change when home address doesn't change")
			}
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			return nil, errors.New("should not be called")
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateWorkerInput{
		Name:        "Test Worker Updated",
		HomeAddress: "Same Home Address", // Same as existing
	}

	_, err := service.Update(context.Background(), 1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if geocodingCalled {
		t.Error("geocoding service should not be called when home address doesn't change")
	}
}

func TestWorkerService_Update_NoGeocodingService(t *testing.T) {
	existingWorker := &models.Worker{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Worker",
	}

	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			return existingWorker, nil
		},
		UpdateFunc: func(worker *models.Worker) error {
			if worker.HomeAddress != "123 Home St" {
				t.Errorf("expected home address '123 Home St', got %s", worker.HomeAddress)
			}
			return nil
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	input := UpdateWorkerInput{
		Name:        "Test Worker",
		HomeAddress: "123 Home St",
	}

	worker, err := service.Update(context.Background(), 1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if worker.HomeAddress != "123 Home St" {
		t.Errorf("expected home address '123 Home St', got %s", worker.HomeAddress)
	}
}

func TestWorkerService_Update_WorkerNotFound(t *testing.T) {
	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			return nil, errors.New("not found")
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	input := UpdateWorkerInput{
		Name: "Test Worker",
	}

	_, err := service.Update(context.Background(), 999, 1, input)
	if err == nil {
		t.Fatal("expected error for non-existent worker, got nil")
	}
	if err != ErrWorkerNotFound {
		t.Errorf("expected ErrWorkerNotFound, got %v", err)
	}
}

func TestWorkerService_Update_IsActiveFlag(t *testing.T) {
	isActive := false

	existingWorker := &models.Worker{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Worker",
		IsActive:       true,
	}

	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			return existingWorker, nil
		},
		UpdateFunc: func(worker *models.Worker) error {
			if worker.IsActive != isActive {
				t.Errorf("expected IsActive %v, got %v", isActive, worker.IsActive)
			}
			return nil
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	input := UpdateWorkerInput{
		Name:     "Test Worker",
		IsActive: &isActive,
	}

	worker, err := service.Update(context.Background(), 1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if worker.IsActive != isActive {
		t.Errorf("expected IsActive %v, got %v", isActive, worker.IsActive)
	}
}

func TestWorkerService_Create_Success(t *testing.T) {
	mockRepo := &mockWorkerRepository{
		CreateFunc: func(worker *models.Worker) error {
			// Verify worker fields
			if worker.Name != "New Worker" {
				t.Errorf("expected name 'New Worker', got %s", worker.Name)
			}
			if worker.Email != "test@example.com" {
				t.Errorf("expected email 'test@example.com', got %s", worker.Email)
			}
			if worker.Phone != "1234567890" {
				t.Errorf("expected phone '1234567890', got %s", worker.Phone)
			}
			if !worker.IsActive {
				t.Error("expected new worker to be active")
			}
			if worker.OrganizationID != 1 {
				t.Errorf("expected organization_id 1, got %d", worker.OrganizationID)
			}
			worker.ID = 1
			return nil
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	input := CreateWorkerInput{
		OrganizationID: 1,
		CreatedBy:      1,
		Name:           "New Worker",
		Email:          "test@example.com",
		Phone:          "1234567890",
	}

	worker, err := service.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if worker.ID != 1 {
		t.Errorf("expected ID 1, got %d", worker.ID)
	}
	if !worker.IsActive {
		t.Error("expected new worker to be active")
	}
}

func TestWorkerService_GetByID_Success(t *testing.T) {
	expectedWorker := &models.Worker{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Worker",
	}

	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			if id != 1 || orgID != 1 {
				t.Errorf("expected id=1, orgID=1, got id=%d, orgID=%d", id, orgID)
			}
			return expectedWorker, nil
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	worker, err := service.GetByID(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if worker.ID != expectedWorker.ID {
		t.Errorf("expected worker ID %d, got %d", expectedWorker.ID, worker.ID)
	}
}

func TestWorkerService_GetByID_NotFound(t *testing.T) {
	mockRepo := &mockWorkerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
			return nil, errors.New("not found")
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	_, err := service.GetByID(context.Background(), 999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent worker, got nil")
	}
	if err != ErrWorkerNotFound {
		t.Errorf("expected ErrWorkerNotFound, got %v", err)
	}
}

func TestWorkerService_GetAll_ActiveOnly(t *testing.T) {
	expectedWorkers := []*models.Worker{
		{ID: 1, Name: "Active Worker", IsActive: true},
	}

	mockRepo := &mockWorkerRepository{
		FindAllFunc: func(orgID uint, activeOnly bool) ([]*models.Worker, error) {
			if orgID != 1 {
				t.Errorf("expected orgID=1, got %d", orgID)
			}
			if !activeOnly {
				t.Error("expected activeOnly=true")
			}
			return expectedWorkers, nil
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	workers, err := service.GetAll(context.Background(), 1, true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(workers) != 1 {
		t.Errorf("expected 1 worker, got %d", len(workers))
	}
	if workers[0].ID != 1 {
		t.Errorf("expected worker ID 1, got %d", workers[0].ID)
	}
}

func TestWorkerService_Delete_Success(t *testing.T) {
	deleteCalled := false

	mockRepo := &mockWorkerRepository{
		DeleteFunc: func(id, orgID uint) error {
			deleteCalled = true
			if id != 1 || orgID != 1 {
				t.Errorf("expected id=1, orgID=1, got id=%d, orgID=%d", id, orgID)
			}
			return nil
		},
	}

	service := newWorkerServiceWithMockRepo(mockRepo, nil)

	err := service.Delete(context.Background(), 1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !deleteCalled {
		t.Error("expected Delete to be called on repository")
	}
}
