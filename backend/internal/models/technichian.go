package models

import "time"

type Technician struct {
	ID                     uint                   `json:"id" gorm:"primaryKey"`
	OrganizationID         uint                   `json:"organization_id" gorm:"not null"`
	CreatedBy              *uint                  `json:"created_by"`
	Name                   string                 `json:"name" gorm:"not null"`
	Email                  string                 `json:"email"`
	Phone                  string                 `json:"phone" gorm:"not null"`
	IsActive               bool                   `json:"is_active" gorm:"default:true"`
	LastLat                *float64               `json:"last_lat"`
	LastLng                *float64               `json:"last_lng"`
	LastSeenAt             *time.Time             `json:"last_seen_at"`
	HomeAddress            string                 `json:"home_address,omitempty"`
	HomeLatitude           *float64               `json:"home_latitude,omitempty"`
	HomeLongitude          *float64               `json:"home_longitude,omitempty"`
	HomeGooglePlaceID      string                 `json:"home_google_place_id,omitempty"`
	HomeFormattedAddress   string                 `json:"home_formatted_address,omitempty"`
	HomeAddressComponents  map[string]interface{} `json:"home_address_components,omitempty" gorm:"type:jsonb"`
	HomeGeocodedAt         *time.Time             `json:"home_geocoded_at,omitempty"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
}

type Worker struct {
	ID                     uint                   `json:"id"`
	OrganizationID         uint                   `json:"organization_id"`
	Name                   string                 `json:"name"`
	Phone                  string                 `json:"phone"`
	Email                  string                 `json:"email,omitempty"`
	Role                   string                 `json:"role,omitempty"` // 'foreman', 'electrician', etc.
	IsActive               bool                   `json:"is_active"`
	CreatedBy              *uint                  `json:"created_by,omitempty"`
	HomeAddress            string                 `json:"home_address,omitempty"`
	HomeLatitude           *float64               `json:"home_latitude,omitempty"`
	HomeLongitude          *float64               `json:"home_longitude,omitempty"`
	HomeGooglePlaceID      string                 `json:"home_google_place_id,omitempty"`
	HomeFormattedAddress   string                 `json:"home_formatted_address,omitempty"`
	HomeAddressComponents  map[string]interface{} `json:"home_address_components,omitempty"`
	HomeGeocodedAt         *time.Time             `json:"home_geocoded_at,omitempty"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
}
