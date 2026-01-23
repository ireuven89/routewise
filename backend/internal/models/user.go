package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          uint      `json:"id"`
	Email       string    `json:"email"`
	Password    string    `json:"-"` // Never send password in JSON
	CompanyName string    `json:"company_name"`
	Phone       string    `json:"phone"`
	Industry    string    `json:"industry"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (u *User) HashPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}
