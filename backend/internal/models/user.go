package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Organization represents a company using the system
type Organization struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	CompanyCode string `json:"company_code"`
	Phone       string `json:"phone"`
	Industry    string `json:"industry"`

	// Service area
	Latitude          *float64          `json:"latitude,omitempty"`
	Longitude         *float64          `json:"longitude,omitempty"`
	Address           string            `json:"address,omitempty"`
	ServiceRadiusKm   float64           `json:"service_radius_km"`
	GooglePlaceID     string            `json:"google_place_id,omitempty"`
	FormattedAddress  string            `json:"formatted_address,omitempty"`
	AddressComponents map[string]string `json:"address_components,omitempty"`
	GeocodedAt        *time.Time        `json:"geocoded_at,omitempty"`

	// Pricing
	VisitFee          *float64 `json:"visit_fee,omitempty"`
	RepairEstimateMin *float64 `json:"repair_estimate_min,omitempty"`
	RepairEstimateMax *float64 `json:"repair_estimate_max,omitempty"`

	// Bit payment collection
	BitPaymentEnabled  bool   `json:"bit_payment_enabled"`
	BitPhoneNumber     string `json:"bit_phone_number,omitempty"`
	BitBusinessName    string `json:"bit_business_name,omitempty"`
	AutoSendPaymentSMS bool   `json:"auto_send_payment_sms"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OrganizationUser represents admins/dispatchers who manage the organization
type OrganizationUser struct {
	ID             uint      `json:"id"`
	OrganizationID uint      `json:"organization_id"`
	Email          string    `json:"email"`
	Password       string    `json:"-"` // Never send password in JSON
	Name           string    `json:"name"`
	Role           string    `json:"role"` // admin, dispatcher, owner
	Phone          string    `json:"phone"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (u *OrganizationUser) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *OrganizationUser) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
