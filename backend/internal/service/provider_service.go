package service

import (
	"context"
	"fmt"

	"github.com/ireuven89/routewise/internal/repository"
)

type UpdateServiceAreaRequest struct {
	Latitude          float64           `json:"latitude"`
	Longitude         float64           `json:"longitude"`
	Address           string            `json:"address"`
	GooglePlaceID     string            `json:"google_place_id"`
	FormattedAddress  string            `json:"formatted_address"`
	AddressComponents map[string]string `json:"address_components"`
	ServiceRadiusKm   float64           `json:"service_radius_km"`
}

type UpdateServiceOfferRequest struct {
	VisitFee          *float64 `json:"visit_fee"`
	RepairEstimateMin *float64 `json:"repair_estimate_min"`
	RepairEstimateMax *float64 `json:"repair_estimate_max"`
}

type ProviderService struct {
	orgRepo *repository.OrganizationRepository
}

func NewProviderService(orgRepo *repository.OrganizationRepository) *ProviderService {
	return &ProviderService{orgRepo: orgRepo}
}

func (s *ProviderService) SearchProviders(ctx context.Context, lat, lng float64, serviceType string) ([]*repository.ProviderResult, error) {
	if lat < -90 || lat > 90 {
		return nil, fmt.Errorf("invalid latitude")
	}
	if lng < -180 || lng > 180 {
		return nil, fmt.Errorf("invalid longitude")
	}
	return s.orgRepo.FindProvidersInArea(ctx, lat, lng, serviceType, 20)
}

func (s *ProviderService) UpdateServiceArea(ctx context.Context, orgID uint, req UpdateServiceAreaRequest) error {
	radius := req.ServiceRadiusKm
	if radius <= 0 {
		radius = 20
	}
	return s.orgRepo.UpdateServiceArea(ctx, orgID,
		req.Latitude, req.Longitude, radius,
		req.Address, req.GooglePlaceID, req.FormattedAddress, req.AddressComponents,
	)
}

func (s *ProviderService) UpdateServiceOffer(ctx context.Context, orgID uint, req UpdateServiceOfferRequest) error {
	return s.orgRepo.UpdateServiceOffer(ctx, orgID, req.VisitFee, req.RepairEstimateMin, req.RepairEstimateMax)
}
