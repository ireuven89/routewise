package models

import "time"

type Customer struct {
	ID        uint      `json:"id""`
	UserID    uint      `json:"user_id" `
	Name      string    `json:"name" `
	Email     string    `json:"email"`
	Phone     string    `json:"phone" `
	Address   string    `json:"address"`
	Latitude  *float64  `json:"latitude"`
	Longitude *float64  `json:"longitude"`
	Notes     string    `json:"notes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
