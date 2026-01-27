package models

import "time"

type Technician struct {
	ID             uint       `json:"id" gorm:"primaryKey"`
	OrganizationID uint       `json:"organization_id" gorm:"not null"`
	CreatedBy      *uint      `json:"created_by"`
	Name           string     `json:"name" gorm:"not null"`
	Email          string     `json:"email"`
	Phone          string     `json:"phone" gorm:"not null"`
	IsActive       bool       `json:"is_active" gorm:"default:true"`
	LastLat        *float64   `json:"last_lat"`
	LastLng        *float64   `json:"last_lng"`
	LastSeenAt     *time.Time `json:"last_seen_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
