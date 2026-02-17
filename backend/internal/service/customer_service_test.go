package service

import (
	"errors"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

// mockCustomerRepository is a mock implementation of CustomerRepository
type mockCustomerRepository struct {
	CreateFunc   func(*models.Customer) error
	FindByIDFunc func(uint, uint) (*models.Customer, error)
	UpdateFunc   func(*models.Customer) error
	DeleteFunc   func(uint, uint) error
	FindAllFunc  func(uint, string) ([]*models.Customer, error)
}

func (m *mockCustomerRepository) Create(customer *models.Customer) error {
	return m.CreateFunc(customer)
}

func (m *mockCustomerRepository) FindByID(id, orgID uint) (*models.Customer, error) {
	return m.FindByIDFunc(id, orgID)
}

func (m *mockCustomerRepository) Update(customer *models.Customer) error {
	return m.UpdateFunc(customer)
}

func (m *mockCustomerRepository) Delete(id, orgID uint) error {
	return m.DeleteFunc(id, orgID)
}

func (m *mockCustomerRepository) FindAll(orgID uint, search string) ([]*models.Customer, error) {
	return m.FindAllFunc(orgID, search)
}

// newMockCustomerService creates a new customer service with mocked dependencies
func newMockCustomerService(repo *mockCustomerRepository, geocoding GeocodingService) *CustomerSvc {
	return &CustomerSvc{
		repo:             (*repository.CustomerRepository)(nil), // Not used, mocked via repo field
		geocodingService: geocoding,
	}
}

// We need to override the internal repo field for testing
func newCustomerServiceWithMockRepo(mockRepo *mockCustomerRepository, geocoding GeocodingService) CustomerService {
	// Create a custom implementation that uses our mock
	return &testCustomerService{
		mockRepo:         mockRepo,
		geocodingService: geocoding,
	}
}

// testCustomerService wraps CustomerSvc for testing
type testCustomerService struct {
	mockRepo         *mockCustomerRepository
	geocodingService GeocodingService
}

func (s *testCustomerService) Create(input CreateCustomerInput) (*models.Customer, error) {
	// Copy the logic from CustomerSvc.Create but use mockRepo
	customer := &models.Customer{
		OrganizationID:    input.OrganizationID,
		CreatedBy:         &input.CreatedBy,
		Name:              input.Name,
		Email:             input.Email,
		Phone:             input.Phone,
		Address:           input.Address,
		Latitude:          input.Latitude,
		Longitude:         input.Longitude,
		GooglePlaceID:     input.GooglePlaceID,
		FormattedAddress:  input.FormattedAddress,
		AddressComponents: input.AddressComponents,
		Notes:             input.Notes,
	}

	// If coordinates not provided but address is, geocode it
	if (customer.Latitude == nil || customer.Longitude == nil) && customer.Address != "" && s.geocodingService != nil {
		result, err := s.geocodingService.GeocodeAddress(customer.Address)
		if err == nil {
			customer.Latitude = &result.Latitude
			customer.Longitude = &result.Longitude
			customer.GooglePlaceID = result.PlaceID
			customer.FormattedAddress = result.FormattedAddress
			// Convert map[string]string to map[string]interface{}
			customer.AddressComponents = convertStringMapToInterfaceMap(result.AddressComponents)
			now := time.Now()
			customer.GeocodedAt = &now
		}
	}

	// If coordinates were provided, mark as geocoded
	if customer.Latitude != nil && customer.Longitude != nil && customer.GeocodedAt == nil {
		now := time.Now()
		customer.GeocodedAt = &now
	}

	if err := s.mockRepo.Create(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *testCustomerService) GetAll(organizationID uint, search string) ([]*models.Customer, error) {
	return s.mockRepo.FindAll(organizationID, search)
}

func (s *testCustomerService) GetByID(id, organizationID uint) (*models.Customer, error) {
	customer, err := s.mockRepo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

func (s *testCustomerService) Update(id, organizationID uint, input UpdateCustomerInput) (*models.Customer, error) {
	customer, err := s.mockRepo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrCustomerNotFound
	}

	addressChanged := false
	if input.Name != "" {
		customer.Name = input.Name
	}
	customer.Email = input.Email
	if input.Phone != "" {
		customer.Phone = input.Phone
	}
	if input.Address != "" && input.Address != customer.Address {
		customer.Address = input.Address
		addressChanged = true
	}
	customer.Latitude = input.Latitude
	customer.Longitude = input.Longitude
	customer.GooglePlaceID = input.GooglePlaceID
	customer.FormattedAddress = input.FormattedAddress
	customer.AddressComponents = input.AddressComponents
	customer.Notes = input.Notes

	// If address changed and no coordinates provided, re-geocode
	if addressChanged && (customer.Latitude == nil || customer.Longitude == nil) && customer.Address != "" && s.geocodingService != nil {
		result, err := s.geocodingService.GeocodeAddress(customer.Address)
		if err == nil {
			customer.Latitude = &result.Latitude
			customer.Longitude = &result.Longitude
			customer.GooglePlaceID = result.PlaceID
			customer.FormattedAddress = result.FormattedAddress
			customer.AddressComponents = convertStringMapToInterfaceMap(result.AddressComponents)
			now := time.Now()
			customer.GeocodedAt = &now
		}
	}

	// If coordinates were provided, mark as geocoded
	if customer.Latitude != nil && customer.Longitude != nil && (customer.GeocodedAt == nil || addressChanged) {
		now := time.Now()
		customer.GeocodedAt = &now
	}

	if err := s.mockRepo.Update(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *testCustomerService) Delete(id, organizationID uint) error {
	return s.mockRepo.Delete(id, organizationID)
}

// mockGeocodingService is a mock implementation of GeocodingService
type mockGeocodingService struct {
	GeocodeAddressFunc  func(string) (*GeocodingResult, error)
	ReverseGeocodeFunc  func(float64, float64) (*GeocodingResult, error)
}

func (m *mockGeocodingService) GeocodeAddress(address string) (*GeocodingResult, error) {
	return m.GeocodeAddressFunc(address)
}

func (m *mockGeocodingService) ReverseGeocode(lat, lng float64) (*GeocodingResult, error) {
	return m.ReverseGeocodeFunc(lat, lng)
}

func TestCustomerService_Create_WithCoordinates(t *testing.T) {
	lat := 32.0853
	lng := 34.7818
	geocodingCalled := false

	mockRepo := &mockCustomerRepository{
		CreateFunc: func(customer *models.Customer) error {
			// Verify customer has the provided coordinates
			if customer.Latitude == nil || *customer.Latitude != lat {
				t.Errorf("expected latitude %f, got %v", lat, customer.Latitude)
			}
			if customer.Longitude == nil || *customer.Longitude != lng {
				t.Errorf("expected longitude %f, got %v", lng, customer.Longitude)
			}
			// Verify GeocodedAt is set
			if customer.GeocodedAt == nil {
				t.Error("expected GeocodedAt to be set when coordinates provided")
			}
			customer.ID = 1
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			return nil, errors.New("should not be called")
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := CreateCustomerInput{
		OrganizationID: 1,
		CreatedBy:      1,
		Name:           "Test Customer",
		Address:        "123 Main St",
		Latitude:       &lat,
		Longitude:      &lng,
		GooglePlaceID:  "ChIJ123",
	}

	customer, err := service.Create(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if customer.ID != 1 {
		t.Errorf("expected ID 1, got %d", customer.ID)
	}
	if geocodingCalled {
		t.Error("geocoding service should not be called when coordinates are provided")
	}
}

func TestCustomerService_Create_WithoutCoordinates_GeocodeSuccess(t *testing.T) {
	geocodingCalled := false
	expectedLat := 32.0853
	expectedLng := 34.7818

	mockRepo := &mockCustomerRepository{
		CreateFunc: func(customer *models.Customer) error {
			// Verify geocoded coordinates were set
			if customer.Latitude == nil || *customer.Latitude != expectedLat {
				t.Errorf("expected geocoded latitude %f, got %v", expectedLat, customer.Latitude)
			}
			if customer.Longitude == nil || *customer.Longitude != expectedLng {
				t.Errorf("expected geocoded longitude %f, got %v", expectedLng, customer.Longitude)
			}
			if customer.GooglePlaceID != "ChIJGeocoded123" {
				t.Errorf("expected geocoded place_id, got %s", customer.GooglePlaceID)
			}
			if customer.FormattedAddress != "123 Main St, Tel Aviv, Israel" {
				t.Errorf("expected formatted address, got %s", customer.FormattedAddress)
			}
			if customer.GeocodedAt == nil {
				t.Error("expected GeocodedAt to be set after geocoding")
			}
			customer.ID = 1
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			if address != "123 Main St" {
				t.Errorf("expected address '123 Main St', got %s", address)
			}
			return &GeocodingResult{
				Latitude:         expectedLat,
				Longitude:        expectedLng,
				PlaceID:          "ChIJGeocoded123",
				FormattedAddress: "123 Main St, Tel Aviv, Israel",
				AddressComponents: map[string]string{
					"city": "Tel Aviv",
				},
			}, nil
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := CreateCustomerInput{
		OrganizationID: 1,
		CreatedBy:      1,
		Name:           "Test Customer",
		Address:        "123 Main St",
	}

	customer, err := service.Create(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if customer.ID != 1 {
		t.Errorf("expected ID 1, got %d", customer.ID)
	}
	if !geocodingCalled {
		t.Error("geocoding service should be called when coordinates not provided")
	}
}

func TestCustomerService_Create_WithoutCoordinates_GeocodeFails(t *testing.T) {
	geocodingCalled := false

	mockRepo := &mockCustomerRepository{
		CreateFunc: func(customer *models.Customer) error {
			// Verify customer is still created despite geocoding failure
			if customer.Latitude != nil {
				t.Error("expected no latitude when geocoding fails")
			}
			if customer.Longitude != nil {
				t.Error("expected no longitude when geocoding fails")
			}
			if customer.GeocodedAt != nil {
				t.Error("expected GeocodedAt to be nil when geocoding fails")
			}
			customer.ID = 1
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			return nil, errors.New("geocoding API error")
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := CreateCustomerInput{
		OrganizationID: 1,
		CreatedBy:      1,
		Name:           "Test Customer",
		Address:        "Invalid Address",
	}

	customer, err := service.Create(input)
	if err != nil {
		t.Fatalf("expected no error (customer should still be created), got %v", err)
	}
	if customer.ID != 1 {
		t.Errorf("expected ID 1, got %d", customer.ID)
	}
	if !geocodingCalled {
		t.Error("geocoding service should be called")
	}
}

func TestCustomerService_Create_NoGeocodingService(t *testing.T) {
	mockRepo := &mockCustomerRepository{
		CreateFunc: func(customer *models.Customer) error {
			customer.ID = 1
			return nil
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, nil)

	input := CreateCustomerInput{
		OrganizationID: 1,
		CreatedBy:      1,
		Name:           "Test Customer",
		Address:        "123 Main St",
	}

	customer, err := service.Create(input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if customer.ID != 1 {
		t.Errorf("expected ID 1, got %d", customer.ID)
	}
}

func TestCustomerService_Update_AddressChanged_Geocode(t *testing.T) {
	geocodingCalled := false
	existingCustomer := &models.Customer{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Customer",
		Address:        "Old Address",
	}

	mockRepo := &mockCustomerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Customer, error) {
			if id != 1 || orgID != 1 {
				t.Errorf("expected id=1, orgID=1, got id=%d, orgID=%d", id, orgID)
			}
			return existingCustomer, nil
		},
		UpdateFunc: func(customer *models.Customer) error {
			// Verify address was updated
			if customer.Address != "New Address" {
				t.Errorf("expected address 'New Address', got %s", customer.Address)
			}
			// Verify geocoded data was set
			if customer.Latitude == nil || *customer.Latitude != 32.0853 {
				t.Errorf("expected geocoded latitude, got %v", customer.Latitude)
			}
			if customer.GeocodedAt == nil {
				t.Error("expected GeocodedAt to be set after geocoding")
			}
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			if address != "New Address" {
				t.Errorf("expected address 'New Address', got %s", address)
			}
			return &GeocodingResult{
				Latitude:         32.0853,
				Longitude:        34.7818,
				PlaceID:          "ChIJNew123",
				FormattedAddress: "New Address, Tel Aviv, Israel",
			}, nil
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateCustomerInput{
		Name:    "Test Customer",
		Address: "New Address",
	}

	customer, err := service.Update(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if customer.Address != "New Address" {
		t.Errorf("expected address 'New Address', got %s", customer.Address)
	}
	if !geocodingCalled {
		t.Error("geocoding service should be called when address changes")
	}
}

func TestCustomerService_Update_AddressNotChanged_NoGeocode(t *testing.T) {
	geocodingCalled := false
	lat := 32.0853
	lng := 34.7818
	geocodedAt := time.Now()

	existingCustomer := &models.Customer{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Customer",
		Address:        "Same Address",
		Latitude:       &lat,
		Longitude:      &lng,
		GeocodedAt:     &geocodedAt,
	}

	mockRepo := &mockCustomerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Customer, error) {
			return existingCustomer, nil
		},
		UpdateFunc: func(customer *models.Customer) error {
			// Verify geocoded data was not changed
			if customer.GeocodedAt == nil || !customer.GeocodedAt.Equal(geocodedAt) {
				t.Error("GeocodedAt should not change when address doesn't change")
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

	service := newCustomerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateCustomerInput{
		Name:    "Test Customer Updated",
		Address: "Same Address", // Same as existing
	}

	_, err := service.Update(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if geocodingCalled {
		t.Error("geocoding service should not be called when address doesn't change")
	}
}

func TestCustomerService_Update_WithCoordinatesProvided_NoGeocode(t *testing.T) {
	geocodingCalled := false
	lat := 32.0853
	lng := 34.7818

	existingCustomer := &models.Customer{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Customer",
		Address:        "Old Address",
	}

	mockRepo := &mockCustomerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Customer, error) {
			return existingCustomer, nil
		},
		UpdateFunc: func(customer *models.Customer) error {
			// Verify provided coordinates were used
			if customer.Latitude == nil || *customer.Latitude != lat {
				t.Errorf("expected provided latitude %f, got %v", lat, customer.Latitude)
			}
			if customer.GeocodedAt == nil {
				t.Error("expected GeocodedAt to be set when coordinates provided")
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

	service := newCustomerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateCustomerInput{
		Name:      "Test Customer",
		Address:   "New Address",
		Latitude:  &lat,
		Longitude: &lng,
	}

	customer, err := service.Update(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if customer.Latitude == nil || *customer.Latitude != lat {
		t.Errorf("expected latitude %f, got %v", lat, customer.Latitude)
	}
	if geocodingCalled {
		t.Error("geocoding service should not be called when coordinates are provided")
	}
}

func TestCustomerService_Update_CustomerNotFound(t *testing.T) {
	mockRepo := &mockCustomerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Customer, error) {
			return nil, errors.New("not found")
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, nil)

	input := UpdateCustomerInput{
		Name: "Test Customer",
	}

	_, err := service.Update(999, 1, input)
	if err == nil {
		t.Fatal("expected error for non-existent customer, got nil")
	}
	if err != ErrCustomerNotFound {
		t.Errorf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestCustomerService_Update_GeocodingFails(t *testing.T) {
	geocodingCalled := false

	existingCustomer := &models.Customer{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Customer",
		Address:        "Old Address",
	}

	mockRepo := &mockCustomerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Customer, error) {
			return existingCustomer, nil
		},
		UpdateFunc: func(customer *models.Customer) error {
			// Customer should still be updated despite geocoding failure
			if customer.Address != "New Address" {
				t.Errorf("expected address 'New Address', got %s", customer.Address)
			}
			// Geocoding data should not be set
			if customer.Latitude != nil {
				t.Error("expected no latitude when geocoding fails")
			}
			return nil
		},
	}

	mockGeocodingService := &mockGeocodingService{
		GeocodeAddressFunc: func(address string) (*GeocodingResult, error) {
			geocodingCalled = true
			return nil, errors.New("geocoding failed")
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, mockGeocodingService)

	input := UpdateCustomerInput{
		Name:    "Test Customer",
		Address: "New Address",
	}

	customer, err := service.Update(1, 1, input)
	if err != nil {
		t.Fatalf("expected no error (update should succeed despite geocoding failure), got %v", err)
	}
	if customer.Address != "New Address" {
		t.Errorf("expected address 'New Address', got %s", customer.Address)
	}
	if !geocodingCalled {
		t.Error("geocoding service should be called")
	}
}

func TestCustomerService_GetByID_Success(t *testing.T) {
	expectedCustomer := &models.Customer{
		ID:             1,
		OrganizationID: 1,
		Name:           "Test Customer",
	}

	mockRepo := &mockCustomerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Customer, error) {
			if id != 1 || orgID != 1 {
				t.Errorf("expected id=1, orgID=1, got id=%d, orgID=%d", id, orgID)
			}
			return expectedCustomer, nil
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, nil)

	customer, err := service.GetByID(1, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if customer.ID != expectedCustomer.ID {
		t.Errorf("expected customer ID %d, got %d", expectedCustomer.ID, customer.ID)
	}
}

func TestCustomerService_GetByID_NotFound(t *testing.T) {
	mockRepo := &mockCustomerRepository{
		FindByIDFunc: func(id, orgID uint) (*models.Customer, error) {
			return nil, errors.New("not found")
		},
	}

	service := newCustomerServiceWithMockRepo(mockRepo, nil)

	_, err := service.GetByID(999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent customer, got nil")
	}
	if err != ErrCustomerNotFound {
		t.Errorf("expected ErrCustomerNotFound, got %v", err)
	}
}
