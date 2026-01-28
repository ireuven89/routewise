package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type JobStatus string

const (
	StatusScheduled  JobStatus = "scheduled"
	StatusInProgress JobStatus = "in_progress"
	StatusCompleted  JobStatus = "completed"
	StatusCancelled  JobStatus = "cancelled"
)

type Job struct {
	ID              uint       `json:"id" gorm:"primaryKey"`
	OrganizationID  uint       `json:"organization_id" gorm:"not null"`
	CreatedBy       *uint      `json:"created_by"`
	CustomerID      uint       `json:"customer_id" gorm:"not null"`
	TechnicianID    *uint      `json:"worker_id"`
	Title           string     `json:"title" gorm:"not null"`
	Description     string     `json:"description"`
	Status          JobStatus  `json:"status" gorm:"default:'scheduled'"`
	ScheduledAt     time.Time  `json:"scheduled_at" gorm:"not null"`
	CompletedAt     *time.Time `json:"completed_at"`
	DurationMinutes int        `json:"duration_minutes" gorm:"default:60"`
	Price           *float64   `json:"price"`
	Metadata        JSON       `json:"metadata" gorm:"type:jsonb"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	Customer        Customer   `json:"customer" gorm:"foreignKey:CustomerID"`
	Worker          *Worker    `json:"worker,omitempty" gorm:"foreignKey:workerID"`
}

// JSON type for JSONB support
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}
