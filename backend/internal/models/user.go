package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Organization represents a company using the system
type Organization struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Industry  string    `json:"industry"`
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
