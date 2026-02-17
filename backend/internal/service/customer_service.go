package service

import (
	"errors"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

var ErrCustomerNotFound = errors.New("customer not found")

type CustomerService interface {
	Create(input CreateCustomerInput) (*models.Customer, error)
	GetAll(organizationID uint, search string) ([]*models.Customer, error)
	GetByID(id, organizationID uint) (*models.Customer, error)
	Update(id, organizationID uint, input UpdateCustomerInput) (*models.Customer, error)
	Delete(id, organizationID uint) error
}

type CustomerSvc struct {
	repo             *repository.CustomerRepository
	geocodingService GeocodingService
}

func NewCustomerService(repo *repository.CustomerRepository, geocodingService GeocodingService) *CustomerSvc {
	return &CustomerSvc{
		repo:             repo,
		geocodingService: geocodingService,
	}
}

type CreateCustomerInput struct {
	OrganizationID    uint
	CreatedBy         uint
	Name              string
	Email             string
	Phone             string
	Address           string
	Latitude          *float64
	Longitude         *float64
	GooglePlaceID     string
	FormattedAddress  string
	AddressComponents map[string]interface{}
	Notes             string
}

type UpdateCustomerInput struct {
	Name              string
	Email             string
	Phone             string
	Address           string
	Latitude          *float64
	Longitude         *float64
	GooglePlaceID     string
	FormattedAddress  string
	AddressComponents map[string]interface{}
	Notes             string
}

func (s *CustomerSvc) Create(input CreateCustomerInput) (*models.Customer, error) {
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
			customer.AddressComponents = convertStringMapToInterfaceMap(result.AddressComponents)
			now := time.Now()
			customer.GeocodedAt = &now
		}
		// Don't fail customer creation if geocoding fails - log and continue
	}

	// If coordinates were provided (from frontend autocomplete), mark as geocoded
	if customer.Latitude != nil && customer.Longitude != nil && customer.GeocodedAt == nil {
		now := time.Now()
		customer.GeocodedAt = &now
	}

	if err := s.repo.Create(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *CustomerSvc) GetAll(organizationID uint, search string) ([]*models.Customer, error) {
	return s.repo.FindAll(organizationID, search)
}

func (s *CustomerSvc) GetByID(id, organizationID uint) (*models.Customer, error) {
	customer, err := s.repo.FindByID(id, organizationID)
	if err != nil {
		return nil, ErrCustomerNotFound
	}
	return customer, nil
}

func (s *CustomerSvc) Update(id, organizationID uint, input UpdateCustomerInput) (*models.Customer, error) {
	customer, err := s.repo.FindByID(id, organizationID)
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

	// If coordinates were provided (from frontend autocomplete), mark as geocoded
	if customer.Latitude != nil && customer.Longitude != nil && (customer.GeocodedAt == nil || addressChanged) {
		now := time.Now()
		customer.GeocodedAt = &now
	}

	if err := s.repo.Update(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *CustomerSvc) Delete(id, organizationID uint) error {
	return s.repo.Delete(id, organizationID)
}

// convertStringMapToInterfaceMap converts map[string]string to map[string]interface{}
func convertStringMapToInterfaceMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}
