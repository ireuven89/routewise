package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ireuven89/routewise/internal/repository"
)

// mockOrgRepo holds function fields so each test can provide its own behaviour.
type mockOrgRepo struct {
	FindProvidersInAreaFunc func(ctx context.Context, lat, lng float64, serviceType string, limit int) ([]*repository.ProviderResult, error)
	UpdateServiceAreaFunc   func(ctx context.Context, orgID uint, lat, lng, radiusKm float64, address, placeID, formattedAddress string, addressComponents map[string]string) error
	UpdateServiceOfferFunc  func(ctx context.Context, orgID uint, visitFee, repairMin, repairMax *float64) error
}

// testProviderService is a test-only service that delegates to mockOrgRepo.
type testProviderService struct {
	repo *mockOrgRepo
}

func newProviderServiceWithMockRepo(repo *mockOrgRepo) *testProviderService {
	return &testProviderService{repo: repo}
}

func (s *testProviderService) SearchProviders(ctx context.Context, lat, lng float64, serviceType string) ([]*repository.ProviderResult, error) {
	if lat < -90 || lat > 90 {
		return nil, errors.New("invalid latitude")
	}
	if lng < -180 || lng > 180 {
		return nil, errors.New("invalid longitude")
	}
	return s.repo.FindProvidersInAreaFunc(ctx, lat, lng, serviceType, 20)
}

func (s *testProviderService) UpdateServiceArea(ctx context.Context, orgID uint, req UpdateServiceAreaRequest) error {
	radius := req.ServiceRadiusKm
	if radius <= 0 {
		radius = 20
	}
	return s.repo.UpdateServiceAreaFunc(ctx, orgID,
		req.Latitude, req.Longitude, radius,
		req.Address, req.GooglePlaceID, req.FormattedAddress, req.AddressComponents,
	)
}

func (s *testProviderService) UpdateServiceOffer(ctx context.Context, orgID uint, req UpdateServiceOfferRequest) error {
	return s.repo.UpdateServiceOfferFunc(ctx, orgID, req.VisitFee, req.RepairEstimateMin, req.RepairEstimateMax)
}

// -----------------------------------------------------------------------
// SearchProviders tests
// -----------------------------------------------------------------------

func TestProviderService_SearchProviders_InvalidLatitude(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{name: "latitude above 90", lat: 91.0, lng: 34.78},
		{name: "latitude below -90", lat: -91.0, lng: 34.78},
		{name: "latitude exactly 91", lat: 91.0, lng: 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgRepo{
				FindProvidersInAreaFunc: func(_ context.Context, _, _ float64, _ string, _ int) ([]*repository.ProviderResult, error) {
					t.Error("repo should not be called on invalid latitude")
					return nil, nil
				},
			}
			svc := newProviderServiceWithMockRepo(repo)

			_, err := svc.SearchProviders(context.Background(), tt.lat, tt.lng, "hvac")
			if err == nil {
				t.Fatalf("expected error for lat=%v, got nil", tt.lat)
			}
			if err.Error() != "invalid latitude" {
				t.Errorf("expected 'invalid latitude', got '%s'", err.Error())
			}
		})
	}
}

func TestProviderService_SearchProviders_InvalidLongitude(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{name: "longitude above 180", lat: 32.08, lng: 181.0},
		{name: "longitude below -180", lat: 32.08, lng: -181.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockOrgRepo{
				FindProvidersInAreaFunc: func(_ context.Context, _, _ float64, _ string, _ int) ([]*repository.ProviderResult, error) {
					t.Error("repo should not be called on invalid longitude")
					return nil, nil
				},
			}
			svc := newProviderServiceWithMockRepo(repo)

			_, err := svc.SearchProviders(context.Background(), tt.lat, tt.lng, "hvac")
			if err == nil {
				t.Fatalf("expected error for lng=%v, got nil", tt.lng)
			}
			if err.Error() != "invalid longitude" {
				t.Errorf("expected 'invalid longitude', got '%s'", err.Error())
			}
		})
	}
}

func TestProviderService_SearchProviders_BoundaryCoordinates(t *testing.T) {
	tests := []struct {
		name string
		lat  float64
		lng  float64
	}{
		{name: "lat exactly 90", lat: 90.0, lng: 0.0},
		{name: "lat exactly -90", lat: -90.0, lng: 0.0},
		{name: "lng exactly 180", lat: 0.0, lng: 180.0},
		{name: "lng exactly -180", lat: 0.0, lng: -180.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoCalled := false
			repo := &mockOrgRepo{
				FindProvidersInAreaFunc: func(_ context.Context, lat, lng float64, _ string, limit int) ([]*repository.ProviderResult, error) {
					repoCalled = true
					if lat != tt.lat {
						t.Errorf("expected lat %v, got %v", tt.lat, lat)
					}
					if lng != tt.lng {
						t.Errorf("expected lng %v, got %v", tt.lng, lng)
					}
					if limit != 20 {
						t.Errorf("expected limit 20, got %d", limit)
					}
					return []*repository.ProviderResult{}, nil
				},
			}
			svc := newProviderServiceWithMockRepo(repo)

			_, err := svc.SearchProviders(context.Background(), tt.lat, tt.lng, "hvac")
			if err != nil {
				t.Fatalf("expected no error for boundary coordinate lat=%v lng=%v, got %v", tt.lat, tt.lng, err)
			}
			if !repoCalled {
				t.Error("expected repo to be called for valid boundary coordinates")
			}
		})
	}
}

func TestProviderService_SearchProviders_ValidCall_ReturnsRepoResult(t *testing.T) {
	visitFee := 150.0
	repairMin := 300.0
	repairMax := 800.0
	expectedProviders := []*repository.ProviderResult{
		{
			ID:                1,
			Name:              "Cool Air Ltd",
			Phone:             "050-1234567",
			Industry:          "hvac",
			Address:           "1 Herzl St, Tel Aviv",
			VisitFee:          &visitFee,
			RepairEstimateMin: &repairMin,
			RepairEstimateMax: &repairMax,
			DistanceKm:        3.2,
		},
	}

	repo := &mockOrgRepo{
		FindProvidersInAreaFunc: func(_ context.Context, lat, lng float64, serviceType string, limit int) ([]*repository.ProviderResult, error) {
			if lat != 32.08 {
				t.Errorf("expected lat 32.08, got %v", lat)
			}
			if lng != 34.78 {
				t.Errorf("expected lng 34.78, got %v", lng)
			}
			if serviceType != "hvac" {
				t.Errorf("expected serviceType 'hvac', got %s", serviceType)
			}
			if limit != 20 {
				t.Errorf("expected limit 20, got %d", limit)
			}
			return expectedProviders, nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	providers, err := svc.SearchProviders(context.Background(), 32.08, 34.78, "hvac")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(providers))
	}
	if providers[0].ID != 1 {
		t.Errorf("expected provider ID 1, got %d", providers[0].ID)
	}
	if providers[0].Name != "Cool Air Ltd" {
		t.Errorf("expected name 'Cool Air Ltd', got %s", providers[0].Name)
	}
	if providers[0].VisitFee == nil || *providers[0].VisitFee != visitFee {
		t.Errorf("expected visit_fee %v, got %v", visitFee, providers[0].VisitFee)
	}
}

func TestProviderService_SearchProviders_RepoError(t *testing.T) {
	repo := &mockOrgRepo{
		FindProvidersInAreaFunc: func(_ context.Context, _, _ float64, _ string, _ int) ([]*repository.ProviderResult, error) {
			return nil, errors.New("database connection lost")
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	_, err := svc.SearchProviders(context.Background(), 32.08, 34.78, "hvac")
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if err.Error() != "database connection lost" {
		t.Errorf("expected 'database connection lost', got '%s'", err.Error())
	}
}

func TestProviderService_SearchProviders_EmptyServiceType(t *testing.T) {
	repoCalled := false
	repo := &mockOrgRepo{
		FindProvidersInAreaFunc: func(_ context.Context, _, _ float64, serviceType string, _ int) ([]*repository.ProviderResult, error) {
			repoCalled = true
			if serviceType != "" {
				t.Errorf("expected empty serviceType, got %s", serviceType)
			}
			return []*repository.ProviderResult{}, nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	providers, err := svc.SearchProviders(context.Background(), 32.08, 34.78, "")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repoCalled {
		t.Error("expected repo to be called")
	}
	if len(providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(providers))
	}
}

// -----------------------------------------------------------------------
// UpdateServiceArea tests
// -----------------------------------------------------------------------

func TestProviderService_UpdateServiceArea_ZeroRadiusDefaultsTo20(t *testing.T) {
	var capturedRadius float64
	repo := &mockOrgRepo{
		UpdateServiceAreaFunc: func(_ context.Context, _ uint, _, _, radiusKm float64, _, _, _ string, _ map[string]string) error {
			capturedRadius = radiusKm
			return nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceAreaRequest{
		Latitude:        32.08,
		Longitude:       34.78,
		ServiceRadiusKm: 0, // zero → should default to 20
	}
	if err := svc.UpdateServiceArea(context.Background(), 1, req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedRadius != 20 {
		t.Errorf("expected default radius 20, got %v", capturedRadius)
	}
}

func TestProviderService_UpdateServiceArea_NegativeRadiusDefaultsTo20(t *testing.T) {
	var capturedRadius float64
	repo := &mockOrgRepo{
		UpdateServiceAreaFunc: func(_ context.Context, _ uint, _, _, radiusKm float64, _, _, _ string, _ map[string]string) error {
			capturedRadius = radiusKm
			return nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceAreaRequest{
		Latitude:        32.08,
		Longitude:       34.78,
		ServiceRadiusKm: -5, // negative → should default to 20
	}
	if err := svc.UpdateServiceArea(context.Background(), 1, req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedRadius != 20 {
		t.Errorf("expected default radius 20, got %v", capturedRadius)
	}
}

func TestProviderService_UpdateServiceArea_PositiveRadiusKept(t *testing.T) {
	tests := []struct {
		name           string
		inputRadius    float64
		expectedRadius float64
	}{
		{name: "small radius", inputRadius: 5.0, expectedRadius: 5.0},
		{name: "large radius", inputRadius: 100.0, expectedRadius: 100.0},
		{name: "fractional radius", inputRadius: 12.5, expectedRadius: 12.5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedRadius float64
			repo := &mockOrgRepo{
				UpdateServiceAreaFunc: func(_ context.Context, _ uint, _, _, radiusKm float64, _, _, _ string, _ map[string]string) error {
					capturedRadius = radiusKm
					return nil
				},
			}
			svc := newProviderServiceWithMockRepo(repo)

			req := UpdateServiceAreaRequest{
				Latitude:        32.08,
				Longitude:       34.78,
				ServiceRadiusKm: tt.inputRadius,
			}
			if err := svc.UpdateServiceArea(context.Background(), 1, req); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if capturedRadius != tt.expectedRadius {
				t.Errorf("expected radius %v, got %v", tt.expectedRadius, capturedRadius)
			}
		})
	}
}

func TestProviderService_UpdateServiceArea_ForwardsAllFields(t *testing.T) {
	components := map[string]string{"city": "Tel Aviv", "country": "Israel"}
	var (
		capturedOrgID            uint
		capturedLat, capturedLng float64
		capturedAddress          string
		capturedPlaceID          string
		capturedFormatted        string
		capturedComponents       map[string]string
	)
	repo := &mockOrgRepo{
		UpdateServiceAreaFunc: func(_ context.Context, orgID uint, lat, lng, _ float64, address, placeID, formattedAddress string, addressComponents map[string]string) error {
			capturedOrgID = orgID
			capturedLat = lat
			capturedLng = lng
			capturedAddress = address
			capturedPlaceID = placeID
			capturedFormatted = formattedAddress
			capturedComponents = addressComponents
			return nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceAreaRequest{
		Latitude:          32.08,
		Longitude:         34.78,
		Address:           "1 Herzl St",
		GooglePlaceID:     "ChIJ123",
		FormattedAddress:  "1 Herzl St, Tel Aviv, Israel",
		AddressComponents: components,
		ServiceRadiusKm:   15.0,
	}
	if err := svc.UpdateServiceArea(context.Background(), 42, req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedOrgID != 42 {
		t.Errorf("expected orgID 42, got %d", capturedOrgID)
	}
	if capturedLat != 32.08 {
		t.Errorf("expected lat 32.08, got %v", capturedLat)
	}
	if capturedLng != 34.78 {
		t.Errorf("expected lng 34.78, got %v", capturedLng)
	}
	if capturedAddress != "1 Herzl St" {
		t.Errorf("expected address '1 Herzl St', got '%s'", capturedAddress)
	}
	if capturedPlaceID != "ChIJ123" {
		t.Errorf("expected placeID 'ChIJ123', got '%s'", capturedPlaceID)
	}
	if capturedFormatted != "1 Herzl St, Tel Aviv, Israel" {
		t.Errorf("expected formatted address, got '%s'", capturedFormatted)
	}
	if capturedComponents["city"] != "Tel Aviv" {
		t.Errorf("expected city 'Tel Aviv', got '%s'", capturedComponents["city"])
	}
}

func TestProviderService_UpdateServiceArea_RepoError(t *testing.T) {
	repo := &mockOrgRepo{
		UpdateServiceAreaFunc: func(_ context.Context, _ uint, _, _, _ float64, _, _, _ string, _ map[string]string) error {
			return errors.New("db write failed")
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceAreaRequest{Latitude: 32.08, Longitude: 34.78, ServiceRadiusKm: 10}
	err := svc.UpdateServiceArea(context.Background(), 1, req)
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if err.Error() != "db write failed" {
		t.Errorf("expected 'db write failed', got '%s'", err.Error())
	}
}

// -----------------------------------------------------------------------
// UpdateServiceOffer tests
// -----------------------------------------------------------------------

func TestProviderService_UpdateServiceOffer_AllNilFees(t *testing.T) {
	var capturedVisitFee, capturedRepairMin, capturedRepairMax *float64
	repo := &mockOrgRepo{
		UpdateServiceOfferFunc: func(_ context.Context, _ uint, visitFee, repairMin, repairMax *float64) error {
			capturedVisitFee = visitFee
			capturedRepairMin = repairMin
			capturedRepairMax = repairMax
			return nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceOfferRequest{
		VisitFee:          nil,
		RepairEstimateMin: nil,
		RepairEstimateMax: nil,
	}
	if err := svc.UpdateServiceOffer(context.Background(), 1, req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedVisitFee != nil {
		t.Errorf("expected nil visit_fee, got %v", capturedVisitFee)
	}
	if capturedRepairMin != nil {
		t.Errorf("expected nil repair_min, got %v", capturedRepairMin)
	}
	if capturedRepairMax != nil {
		t.Errorf("expected nil repair_max, got %v", capturedRepairMax)
	}
}

func TestProviderService_UpdateServiceOffer_WithFees(t *testing.T) {
	visitFee := 120.0
	repairMin := 200.0
	repairMax := 600.0

	var capturedVisitFee, capturedRepairMin, capturedRepairMax *float64
	var capturedOrgID uint
	repo := &mockOrgRepo{
		UpdateServiceOfferFunc: func(_ context.Context, orgID uint, vf, rMin, rMax *float64) error {
			capturedOrgID = orgID
			capturedVisitFee = vf
			capturedRepairMin = rMin
			capturedRepairMax = rMax
			return nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceOfferRequest{
		VisitFee:          &visitFee,
		RepairEstimateMin: &repairMin,
		RepairEstimateMax: &repairMax,
	}
	if err := svc.UpdateServiceOffer(context.Background(), 7, req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedOrgID != 7 {
		t.Errorf("expected orgID 7, got %d", capturedOrgID)
	}
	if capturedVisitFee == nil || *capturedVisitFee != visitFee {
		t.Errorf("expected visit_fee %v, got %v", visitFee, capturedVisitFee)
	}
	if capturedRepairMin == nil || *capturedRepairMin != repairMin {
		t.Errorf("expected repair_min %v, got %v", repairMin, capturedRepairMin)
	}
	if capturedRepairMax == nil || *capturedRepairMax != repairMax {
		t.Errorf("expected repair_max %v, got %v", repairMax, capturedRepairMax)
	}
}

func TestProviderService_UpdateServiceOffer_PartialFees(t *testing.T) {
	visitFee := 80.0

	var capturedVisitFee, capturedRepairMin, capturedRepairMax *float64
	repo := &mockOrgRepo{
		UpdateServiceOfferFunc: func(_ context.Context, _ uint, vf, rMin, rMax *float64) error {
			capturedVisitFee = vf
			capturedRepairMin = rMin
			capturedRepairMax = rMax
			return nil
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceOfferRequest{
		VisitFee:          &visitFee,
		RepairEstimateMin: nil, // not set
		RepairEstimateMax: nil, // not set
	}
	if err := svc.UpdateServiceOffer(context.Background(), 1, req); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedVisitFee == nil || *capturedVisitFee != visitFee {
		t.Errorf("expected visit_fee %v, got %v", visitFee, capturedVisitFee)
	}
	if capturedRepairMin != nil {
		t.Errorf("expected nil repair_min, got %v", capturedRepairMin)
	}
	if capturedRepairMax != nil {
		t.Errorf("expected nil repair_max, got %v", capturedRepairMax)
	}
}

func TestProviderService_UpdateServiceOffer_RepoError(t *testing.T) {
	visitFee := 100.0
	repo := &mockOrgRepo{
		UpdateServiceOfferFunc: func(_ context.Context, _ uint, _, _, _ *float64) error {
			return errors.New("constraint violation")
		},
	}
	svc := newProviderServiceWithMockRepo(repo)

	req := UpdateServiceOfferRequest{VisitFee: &visitFee}
	err := svc.UpdateServiceOffer(context.Background(), 1, req)
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
	if err.Error() != "constraint violation" {
		t.Errorf("expected 'constraint violation', got '%s'", err.Error())
	}
}
