package service

import (
	"errors"

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
	repo *repository.CustomerRepository
}

func NewCustomerService(repo *repository.CustomerRepository) *CustomerSvc {
	return &CustomerSvc{repo: repo}
}

type CreateCustomerInput struct {
	OrganizationID uint
	CreatedBy      uint
	Name           string
	Email          string
	Phone          string
	Address        string
	Latitude       *float64
	Longitude      *float64
	Notes          string
}

type UpdateCustomerInput struct {
	Name      string
	Email     string
	Phone     string
	Address   string
	Latitude  *float64
	Longitude *float64
	Notes     string
}

func (s *CustomerSvc) Create(input CreateCustomerInput) (*models.Customer, error) {
	customer := &models.Customer{
		OrganizationID: input.OrganizationID,
		CreatedBy:      &input.CreatedBy,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          input.Phone,
		Address:        input.Address,
		Latitude:       input.Latitude,
		Longitude:      input.Longitude,
		Notes:          input.Notes,
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

	if input.Name != "" {
		customer.Name = input.Name
	}
	customer.Email = input.Email
	if input.Phone != "" {
		customer.Phone = input.Phone
	}
	if input.Address != "" {
		customer.Address = input.Address
	}
	customer.Latitude = input.Latitude
	customer.Longitude = input.Longitude
	customer.Notes = input.Notes

	if err := s.repo.Update(customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *CustomerSvc) Delete(id, organizationID uint) error {
	return s.repo.Delete(id, organizationID)
}
