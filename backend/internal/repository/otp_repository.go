package repository

import (
	"database/sql"
	"time"
)

type OTPRepository struct {
	db *sql.DB
}

func NewOTPRepository(db *sql.DB) *OTPRepository {
	return &OTPRepository{db: db}
}

// Save creates new OTP
func (r *OTPRepository) Save(phone, companyCode, otpCode string, expiresAt time.Time) error {
	query := `
        INSERT INTO worker_otps (phone, company_code, otp_code, expires_at)
        VALUES ($1, $2, $3, $4)
    `
	_, err := r.db.Exec(query, phone, companyCode, otpCode, expiresAt)
	return err
}

// Verify checks if OTP is valid
func (r *OTPRepository) Verify(phone, companyCode, otpCode string) (int, error) {
	query := `
        SELECT id FROM worker_otps
        WHERE phone = $1 
        AND company_code = $2
        AND otp_code = $3
        AND expires_at > NOW()
        AND verified = false
        ORDER BY created_at DESC
        LIMIT 1
    `

	var otpID int
	err := r.db.QueryRow(query, phone, companyCode, otpCode).Scan(&otpID)
	if err == sql.ErrNoRows {
		return 0, nil // Invalid OTP
	}
	return otpID, err
}

// MarkVerified marks OTP as used
func (r *OTPRepository) MarkVerified(otpID int) error {
	query := `UPDATE worker_otps SET verified = true WHERE id = $1`
	_, err := r.db.Exec(query, otpID)
	return err
}

// Cleanup removes expired OTPs (run daily)
func (r *OTPRepository) Cleanup() error {
	query := `DELETE FROM worker_otps WHERE expires_at < NOW() - INTERVAL '1 hour'`
	_, err := r.db.Exec(query)
	return err
}
