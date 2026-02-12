package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

// PaymentNotificationRepository handles database operations for payment notifications
type PaymentNotificationRepository struct {
	db *sql.DB
}

// NewPaymentNotificationRepository creates a new repository instance
func NewPaymentNotificationRepository(db *sql.DB) *PaymentNotificationRepository {
	return &PaymentNotificationRepository{db: db}
}

// Create inserts a new payment notification record
func (r *PaymentNotificationRepository) Create(notification *models.PaymentNotification) error {
	query := `
		INSERT INTO payment_notifications (
			organization_id, job_id, customer_id, amount, payment_link_url,
			payment_method, sent_via, recipient_phone, sms_status,
			payment_status, created_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`

	err := r.db.QueryRow(
		query,
		notification.OrganizationID,
		notification.JobID,
		notification.CustomerID,
		notification.Amount,
		notification.PaymentLinkURL,
		notification.PaymentMethod,
		notification.SentVia,
		notification.RecipientPhone,
		notification.SMSStatus,
		notification.PaymentStatus,
		notification.CreatedBy,
	).Scan(&notification.ID, &notification.CreatedAt, &notification.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create payment notification: %w", err)
	}

	return nil
}

// FindByID retrieves a payment notification by ID, scoped to organization
func (r *PaymentNotificationRepository) FindByID(id, organizationID uint) (*models.PaymentNotification, error) {
	query := `
		SELECT
			id, organization_id, job_id, customer_id, amount, payment_link_url,
			payment_method, sent_via, recipient_phone, sent_at, sms_status,
			payment_status, paid_at, created_by, created_at, updated_at
		FROM payment_notifications
		WHERE id = $1 AND organization_id = $2
	`

	notification := &models.PaymentNotification{}
	err := r.db.QueryRow(query, id, organizationID).Scan(
		&notification.ID,
		&notification.OrganizationID,
		&notification.JobID,
		&notification.CustomerID,
		&notification.Amount,
		&notification.PaymentLinkURL,
		&notification.PaymentMethod,
		&notification.SentVia,
		&notification.RecipientPhone,
		&notification.SentAt,
		&notification.SMSStatus,
		&notification.PaymentStatus,
		&notification.PaidAt,
		&notification.CreatedBy,
		&notification.CreatedAt,
		&notification.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment notification not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find payment notification: %w", err)
	}

	return notification, nil
}

// FindByJobID retrieves all payment notifications for a job, scoped to organization
func (r *PaymentNotificationRepository) FindByJobID(jobID, organizationID uint) ([]*models.PaymentNotification, error) {
	query := `
		SELECT
			id, organization_id, job_id, customer_id, amount, payment_link_url,
			payment_method, sent_via, recipient_phone, sent_at, sms_status,
			payment_status, paid_at, created_by, created_at, updated_at
		FROM payment_notifications
		WHERE job_id = $1 AND organization_id = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, jobID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query payment notifications: %w", err)
	}
	defer rows.Close()

	notifications := []*models.PaymentNotification{}
	for rows.Next() {
		notification := &models.PaymentNotification{}
		err := rows.Scan(
			&notification.ID,
			&notification.OrganizationID,
			&notification.JobID,
			&notification.CustomerID,
			&notification.Amount,
			&notification.PaymentLinkURL,
			&notification.PaymentMethod,
			&notification.SentVia,
			&notification.RecipientPhone,
			&notification.SentAt,
			&notification.SMSStatus,
			&notification.PaymentStatus,
			&notification.PaidAt,
			&notification.CreatedBy,
			&notification.CreatedAt,
			&notification.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan payment notification: %w", err)
		}
		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// UpdateSMSStatus updates the SMS delivery status and timestamp
func (r *PaymentNotificationRepository) UpdateSMSStatus(id, organizationID uint, status models.SMSStatus, sentAt *time.Time) error {
	query := `
		UPDATE payment_notifications
		SET sms_status = $1, sent_at = $2, updated_at = NOW()
		WHERE id = $3 AND organization_id = $4
	`

	result, err := r.db.Exec(query, status, sentAt, id, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update SMS status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("payment notification not found or not in organization")
	}

	return nil
}

// UpdatePaymentStatus updates the payment status and timestamp
func (r *PaymentNotificationRepository) UpdatePaymentStatus(id, organizationID uint, status models.PaymentStatus, paidAt *time.Time) error {
	query := `
		UPDATE payment_notifications
		SET payment_status = $1, paid_at = $2, updated_at = NOW()
		WHERE id = $3 AND organization_id = $4
	`

	result, err := r.db.Exec(query, status, paidAt, id, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("payment notification not found or not in organization")
	}

	return nil
}
