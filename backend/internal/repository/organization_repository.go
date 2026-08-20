package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

type OrganizationRepository struct {
	db *sql.DB
}

func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// FindByID returns an organization by its ID with all fields including service area and pricing.
func (r *OrganizationRepository) FindByID(id uint) (*models.Organization, error) {
	query := `
		SELECT id, name, phone, industry, company_code,
		       latitude, longitude, address, service_radius_km,
		       google_place_id, formatted_address, address_components, geocoded_at,
		       visit_fee, repair_estimate_min, repair_estimate_max,
		       bit_payment_enabled, bit_phone_number, bit_business_name, auto_send_payment_sms,
		       created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	org := &models.Organization{}
	var lat, lng, visitFee, repairMin, repairMax sql.NullFloat64
	var addr, placeID, formattedAddress sql.NullString
	var geocodedAt sql.NullTime
	var addressComponentsJSON []byte

	err := r.db.QueryRow(query, id).Scan(
		&org.ID, &org.Name, &org.Phone, &org.Industry, &org.CompanyCode,
		&lat, &lng, &addr, &org.ServiceRadiusKm,
		&placeID, &formattedAddress, &addressComponentsJSON, &geocodedAt,
		&visitFee, &repairMin, &repairMax,
		&org.BitPaymentEnabled, &org.BitPhoneNumber, &org.BitBusinessName, &org.AutoSendPaymentSMS,
		&org.CreatedAt, &org.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("organization not found")
	}
	if err != nil {
		return nil, err
	}

	if lat.Valid {
		org.Latitude = &lat.Float64
	}
	if lng.Valid {
		org.Longitude = &lng.Float64
	}
	if addr.Valid {
		org.Address = addr.String
	}
	if placeID.Valid {
		org.GooglePlaceID = placeID.String
	}
	if formattedAddress.Valid {
		org.FormattedAddress = formattedAddress.String
	}
	if geocodedAt.Valid {
		org.GeocodedAt = &geocodedAt.Time
	}
	if visitFee.Valid {
		org.VisitFee = &visitFee.Float64
	}
	if repairMin.Valid {
		org.RepairEstimateMin = &repairMin.Float64
	}
	if repairMax.Valid {
		org.RepairEstimateMax = &repairMax.Float64
	}
	if addressComponentsJSON != nil {
		_ = json.Unmarshal(addressComponentsJSON, &org.AddressComponents)
	}

	return org, nil
}

// UpdateServiceArea persists the organization's service area (address + geocoding + radius).
func (r *OrganizationRepository) UpdateServiceArea(ctx context.Context, orgID uint, lat, lng, radiusKm float64, address, placeID, formattedAddress string, addressComponents map[string]string) error {
	componentsJSON, err := json.Marshal(addressComponents)
	if err != nil {
		return err
	}

	_, err = r.db.ExecContext(ctx, `
		UPDATE organizations
		SET latitude = $1, longitude = $2, service_radius_km = $3,
		    address = $4, google_place_id = $5, formatted_address = $6,
		    address_components = $7, geocoded_at = $8, updated_at = $9
		WHERE id = $10`,
		lat, lng, radiusKm, address, placeID, formattedAddress,
		componentsJSON, time.Now(), time.Now(), orgID,
	)
	return err
}

// UpdateServiceOffer persists pricing fields for the organization.
func (r *OrganizationRepository) UpdateServiceOffer(ctx context.Context, orgID uint, visitFee, repairMin, repairMax *float64) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		SET visit_fee = $1, repair_estimate_min = $2, repair_estimate_max = $3, updated_at = $4
		WHERE id = $5`,
		visitFee, repairMin, repairMax, time.Now(), orgID,
	)
	return err
}

// UpdatePaymentSettings persists Bit payment collection settings for the organization.
func (r *OrganizationRepository) UpdatePaymentSettings(ctx context.Context, orgID uint, enabled bool, phone, businessName string, autoSend bool) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		SET bit_payment_enabled = $1, bit_phone_number = $2, bit_business_name = $3,
		    auto_send_payment_sms = $4, updated_at = $5
		WHERE id = $6`,
		enabled, phone, businessName, autoSend, time.Now(), orgID,
	)
	return err
}

// ProviderResult is a public-safe view of an organization for the customer discovery page.
type ProviderResult struct {
	ID                uint     `json:"id"`
	Name              string   `json:"name"`
	Phone             string   `json:"phone"`
	Industry          string   `json:"industry"`
	Address           string   `json:"address"`
	VisitFee          *float64 `json:"visit_fee"`
	RepairEstimateMin *float64 `json:"repair_estimate_min"`
	RepairEstimateMax *float64 `json:"repair_estimate_max"`
	DistanceKm        float64  `json:"distance_km"`
}

// FindProvidersInArea returns organizations whose service radius covers the given location.
// Uses a subquery so the computed distance alias can be referenced in WHERE.
func (r *OrganizationRepository) FindProvidersInArea(ctx context.Context, lat, lng float64, serviceType string, limit int) ([]*ProviderResult, error) {
	q := `
		SELECT id, name, phone, industry, formatted_addr,
		       visit_fee, repair_estimate_min, repair_estimate_max, distance_km
		FROM (
		    SELECT id, name, phone, industry,
		           COALESCE(formatted_address, address, '') AS formatted_addr,
		           visit_fee, repair_estimate_min, repair_estimate_max, service_radius_km,
		           (6371 * acos(
		               LEAST(1.0,
		                   cos(radians($1)) * cos(radians(latitude)) *
		                   cos(radians(longitude) - radians($2)) +
		                   sin(radians($1)) * sin(radians(latitude))
		               )
		           )) AS distance_km
		    FROM organizations
		    WHERE latitude IS NOT NULL
		      AND longitude IS NOT NULL
		      AND industry = $3
		      AND visit_fee IS NOT NULL
		) sub
		WHERE distance_km <= service_radius_km
		ORDER BY distance_km ASC
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, q, lat, lng, serviceType, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*ProviderResult
	for rows.Next() {
		p := &ProviderResult{}
		var visitFee, repairMin, repairMax sql.NullFloat64
		err = rows.Scan(
			&p.ID, &p.Name, &p.Phone, &p.Industry, &p.Address,
			&visitFee, &repairMin, &repairMax, &p.DistanceKm,
		)
		if err != nil {
			return nil, err
		}
		if visitFee.Valid {
			p.VisitFee = &visitFee.Float64
		}
		if repairMin.Valid {
			p.RepairEstimateMin = &repairMin.Float64
		}
		if repairMax.Valid {
			p.RepairEstimateMax = &repairMax.Float64
		}
		results = append(results, p)
	}
	return results, rows.Err()
}
