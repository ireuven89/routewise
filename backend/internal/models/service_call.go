package models

import "time"

type CreateServiceCallRequest struct {
	Customer     ServiceCallCustomer `json:"customer" binding:"required"`
	Job          ServiceCallJob      `json:"job" binding:"required"`
	TechnicianID *uint               `json:"technician_id"`
}

type ServiceCallCustomer struct {
	Phone             string                 `json:"phone" binding:"required"`
	Name              string                 `json:"name" binding:"required"`
	Email             string                 `json:"email"`
	Address           string                 `json:"address"`
	Latitude          *float64               `json:"latitude"`
	Longitude         *float64               `json:"longitude"`
	GooglePlaceID     string                 `json:"google_place_id"`
	FormattedAddress  string                 `json:"formatted_address"`
	AddressComponents map[string]interface{} `json:"address_components"`
}

type ServiceCallJob struct {
	Title         string    `json:"title" binding:"required"`
	Description   string    `json:"description"`
	CreatedBy     uint      `json:"created_by"`
	ScheduledDate time.Time `json:"scheduled_date" binding:"required"`
	Priority      string    `json:"priority" binding:"oneof=low medium high"`
	Status        string    `json:"status" binding:"oneof=scheduled in_progress completed cancelled"`
}

type CreateServiceCallResponse struct {
	CustomerID      uint   `json:"customer_id"`
	CustomerCreated bool   `json:"customer_created"`
	JobID           uint   `json:"job_id"`
	Message         string `json:"message"`
}
