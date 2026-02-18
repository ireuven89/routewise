package models

import "time"

type Customer struct {
	ID                uint                   `json:"id"`
	OrganizationID    uint                   `json:"organization_id"`
	CreatedBy         *uint                  `json:"created_by"`
	Name              string                 `json:"name"`
	Email             string                 `json:"email"`
	Phone             string                 `json:"phone"`
	Address           string                 `json:"address"`
	Latitude          *float64               `json:"latitude"`
	Longitude         *float64               `json:"longitude"`
	GooglePlaceID     string                 `json:"google_place_id,omitempty"`
	FormattedAddress  string                 `json:"formatted_address,omitempty"`
	AddressComponents map[string]interface{} `json:"address_components,omitempty" gorm:"type:jsonb"`
	GeocodedAt        *time.Time             `json:"geocoded_at,omitempty"`
	Notes             string                 `json:"notes"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}
