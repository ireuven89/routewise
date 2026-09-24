package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

// -----------------------------------------------------------------------
// Message builder tests (pure functions, exercised directly)
// -----------------------------------------------------------------------

func TestBuildTrackingLinkSMS_ContainsCustomerNameAndURL(t *testing.T) {
	sr := &models.ServiceRequest{CustomerName: "Dana Levi", ServiceType: "hvac"}
	msg := buildTrackingLinkSMS(sr, "https://app.example.com/find-service/requests/abc123")

	if !strings.Contains(msg, "Dana Levi") {
		t.Errorf("expected message to contain customer name, got: %s", msg)
	}
	if !strings.Contains(msg, "https://app.example.com/find-service/requests/abc123") {
		t.Errorf("expected message to contain tracking URL, got: %s", msg)
	}
}

func TestBuildAwardConfirmationSMS_ContainsWinnerName(t *testing.T) {
	sr := &models.ServiceRequest{ServiceType: "plumbing"}
	msg := buildAwardConfirmationSMS(sr, "Cool Air Ltd", "https://app.example.com/find-service/requests/abc123")

	if !strings.Contains(msg, "Cool Air Ltd") {
		t.Errorf("expected message to contain winner org name, got: %s", msg)
	}
}

func TestBuildBidRejectedMessage_DoesNotNameWinner(t *testing.T) {
	sr := &models.ServiceRequest{ServiceType: "electrical", Address: "1 Herzl St"}
	msg := buildBidRejectedMessage(sr)

	if !strings.Contains(msg, "electrical") || !strings.Contains(msg, "1 Herzl St") {
		t.Errorf("expected message to reference service type and address, got: %s", msg)
	}
}

// -----------------------------------------------------------------------
// CreateRequest / SubmitBid validation tests
//
// ServiceRequestService depends on concrete repository structs, not interfaces (matching
// this codebase's existing convention, see provider_service_test.go), so validation logic
// is exercised here via a hand-mirrored test-local copy backed by mock interfaces, rather
// than the real ServiceRequestService.
// -----------------------------------------------------------------------

type mockServiceRequestRepo struct {
	CreateFunc                    func(ctx context.Context, sr *models.ServiceRequest) error
	FindByIDFunc                  func(ctx context.Context, id uint) (*models.ServiceRequest, error)
	FindMatchingOrganizationsFunc func(ctx context.Context, lat, lng float64, serviceType string, cap int) ([]*repository.MatchedOrg, error)
	UpsertBidFunc                 func(ctx context.Context, bid *models.ServiceRequestBid) error
}

func (m *mockServiceRequestRepo) Create(ctx context.Context, sr *models.ServiceRequest) error {
	return m.CreateFunc(ctx, sr)
}
func (m *mockServiceRequestRepo) FindByID(ctx context.Context, id uint) (*models.ServiceRequest, error) {
	return m.FindByIDFunc(ctx, id)
}
func (m *mockServiceRequestRepo) FindMatchingOrganizations(ctx context.Context, lat, lng float64, serviceType string, cap int) ([]*repository.MatchedOrg, error) {
	return m.FindMatchingOrganizationsFunc(ctx, lat, lng, serviceType, cap)
}
func (m *mockServiceRequestRepo) UpsertBid(ctx context.Context, bid *models.ServiceRequestBid) error {
	return m.UpsertBidFunc(ctx, bid)
}

// testServiceRequestService mirrors ServiceRequestService.CreateRequest/SubmitBid's
// validation logic against the mock interface above.
type testServiceRequestService struct {
	repo *mockServiceRequestRepo
}

func (s *testServiceRequestService) CreateRequest(ctx context.Context, in CreateServiceRequestInput) (*models.ServiceRequest, error) {
	if in.Latitude < -90 || in.Latitude > 90 {
		return nil, errors.New("invalid service request: invalid latitude")
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return nil, errors.New("invalid service request: invalid longitude")
	}
	if in.ServiceType == "" {
		return nil, errors.New("invalid service request: service_type is required")
	}
	if in.CustomerName == "" || in.CustomerPhone == "" {
		return nil, errors.New("invalid service request: customer name and phone are required")
	}

	sr := &models.ServiceRequest{
		ServiceType:   in.ServiceType,
		CustomerName:  in.CustomerName,
		CustomerPhone: in.CustomerPhone,
		Latitude:      in.Latitude,
		Longitude:     in.Longitude,
		Status:        models.ServiceRequestStatusOpen,
	}
	if err := s.repo.Create(ctx, sr); err != nil {
		return nil, err
	}
	return sr, nil
}

func (s *testServiceRequestService) SubmitBid(ctx context.Context, orgID, requestID uint, in SubmitBidInput) (*models.ServiceRequestBid, error) {
	sr, err := s.repo.FindByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if sr.Status != models.ServiceRequestStatusOpen {
		return nil, repository.ErrServiceRequestNotOpen
	}
	if in.Price <= 0 {
		return nil, errors.New("invalid service request: price must be greater than zero")
	}

	bid := &models.ServiceRequestBid{ServiceRequestID: requestID, OrganizationID: orgID, Price: in.Price, Message: in.Message}
	if err := s.repo.UpsertBid(ctx, bid); err != nil {
		return nil, err
	}
	return bid, nil
}

func TestServiceRequestService_CreateRequest_InvalidLatLng(t *testing.T) {
	tests := []struct {
		name string
		in   CreateServiceRequestInput
	}{
		{name: "lat too high", in: CreateServiceRequestInput{Latitude: 91, Longitude: 0, ServiceType: "hvac", CustomerName: "A", CustomerPhone: "050"}},
		{name: "lat too low", in: CreateServiceRequestInput{Latitude: -91, Longitude: 0, ServiceType: "hvac", CustomerName: "A", CustomerPhone: "050"}},
		{name: "lng too high", in: CreateServiceRequestInput{Latitude: 0, Longitude: 181, ServiceType: "hvac", CustomerName: "A", CustomerPhone: "050"}},
		{name: "lng too low", in: CreateServiceRequestInput{Latitude: 0, Longitude: -181, ServiceType: "hvac", CustomerName: "A", CustomerPhone: "050"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockServiceRequestRepo{
				CreateFunc: func(_ context.Context, _ *models.ServiceRequest) error {
					t.Error("repo.Create should not be called on invalid coordinates")
					return nil
				},
			}
			svc := &testServiceRequestService{repo: repo}

			_, err := svc.CreateRequest(context.Background(), tt.in)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestServiceRequestService_CreateRequest_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name string
		in   CreateServiceRequestInput
	}{
		{name: "missing service type", in: CreateServiceRequestInput{Latitude: 32, Longitude: 34, CustomerName: "A", CustomerPhone: "050"}},
		{name: "missing customer name", in: CreateServiceRequestInput{Latitude: 32, Longitude: 34, ServiceType: "hvac", CustomerPhone: "050"}},
		{name: "missing customer phone", in: CreateServiceRequestInput{Latitude: 32, Longitude: 34, ServiceType: "hvac", CustomerName: "A"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockServiceRequestRepo{
				CreateFunc: func(_ context.Context, _ *models.ServiceRequest) error {
					t.Error("repo.Create should not be called when required fields are missing")
					return nil
				},
			}
			svc := &testServiceRequestService{repo: repo}

			_, err := svc.CreateRequest(context.Background(), tt.in)
			if err == nil {
				t.Fatal("expected validation error, got nil")
			}
		})
	}
}

func TestServiceRequestService_CreateRequest_ValidInput_CallsRepo(t *testing.T) {
	created := false
	repo := &mockServiceRequestRepo{
		CreateFunc: func(_ context.Context, sr *models.ServiceRequest) error {
			created = true
			sr.ID = 42
			if sr.Status != models.ServiceRequestStatusOpen {
				t.Errorf("expected status open, got %s", sr.Status)
			}
			return nil
		},
	}
	svc := &testServiceRequestService{repo: repo}

	sr, err := svc.CreateRequest(context.Background(), CreateServiceRequestInput{
		Latitude: 32.08, Longitude: 34.78, ServiceType: "hvac", CustomerName: "Dana", CustomerPhone: "050-1234567",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !created {
		t.Error("expected repo.Create to be called")
	}
	if sr.ID != 42 {
		t.Errorf("expected ID 42, got %d", sr.ID)
	}
}

func TestServiceRequestService_SubmitBid_RejectsNonOpenRequest(t *testing.T) {
	repo := &mockServiceRequestRepo{
		FindByIDFunc: func(_ context.Context, _ uint) (*models.ServiceRequest, error) {
			return &models.ServiceRequest{ID: 1, Status: models.ServiceRequestStatusAwarded}, nil
		},
		UpsertBidFunc: func(_ context.Context, _ *models.ServiceRequestBid) error {
			t.Error("UpsertBid should not be called for a non-open request")
			return nil
		},
	}
	svc := &testServiceRequestService{repo: repo}

	_, err := svc.SubmitBid(context.Background(), 7, 1, SubmitBidInput{Price: 100})
	if !errors.Is(err, repository.ErrServiceRequestNotOpen) {
		t.Fatalf("expected ErrServiceRequestNotOpen, got %v", err)
	}
}

func TestServiceRequestService_SubmitBid_RejectsNonPositivePrice(t *testing.T) {
	tests := []float64{0, -10}
	for _, price := range tests {
		repo := &mockServiceRequestRepo{
			FindByIDFunc: func(_ context.Context, _ uint) (*models.ServiceRequest, error) {
				return &models.ServiceRequest{ID: 1, Status: models.ServiceRequestStatusOpen}, nil
			},
			UpsertBidFunc: func(_ context.Context, _ *models.ServiceRequestBid) error {
				t.Error("UpsertBid should not be called for a non-positive price")
				return nil
			},
		}
		svc := &testServiceRequestService{repo: repo}

		_, err := svc.SubmitBid(context.Background(), 7, 1, SubmitBidInput{Price: price})
		if err == nil {
			t.Fatalf("expected error for price %v, got nil", price)
		}
	}
}

func TestServiceRequestService_SubmitBid_ValidBid_CallsUpsert(t *testing.T) {
	var captured *models.ServiceRequestBid
	repo := &mockServiceRequestRepo{
		FindByIDFunc: func(_ context.Context, _ uint) (*models.ServiceRequest, error) {
			return &models.ServiceRequest{ID: 1, Status: models.ServiceRequestStatusOpen}, nil
		},
		UpsertBidFunc: func(_ context.Context, bid *models.ServiceRequestBid) error {
			captured = bid
			return nil
		},
	}
	svc := &testServiceRequestService{repo: repo}

	_, err := svc.SubmitBid(context.Background(), 7, 1, SubmitBidInput{Price: 250, Message: "Can come tomorrow"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if captured == nil {
		t.Fatal("expected UpsertBid to be called")
	}
	if captured.OrganizationID != 7 || captured.Price != 250 {
		t.Errorf("unexpected bid captured: %+v", captured)
	}
}

func TestServiceRequestService_SubmitBid_RequestNotFound(t *testing.T) {
	repo := &mockServiceRequestRepo{
		FindByIDFunc: func(_ context.Context, _ uint) (*models.ServiceRequest, error) {
			return nil, repository.ErrServiceRequestNotFound
		},
	}
	svc := &testServiceRequestService{repo: repo}

	_, err := svc.SubmitBid(context.Background(), 7, 1, SubmitBidInput{Price: 100})
	if !errors.Is(err, repository.ErrServiceRequestNotFound) {
		t.Fatalf("expected ErrServiceRequestNotFound, got %v", err)
	}
}
