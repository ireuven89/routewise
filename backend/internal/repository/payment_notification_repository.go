package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/lib/pq"
)

var (
	ErrPaymentRequestAlreadyActive = errors.New("payment request already active for this job")
	ErrPaymentNotificationNotFound = errors.New("payment notification not found")
)

type PaymentNotificationRepository struct {
	db *sql.DB
}

func NewPaymentNotificationRepository(db *sql.DB) *PaymentNotificationRepository {
	return &PaymentNotificationRepository{db: db}
}

// CreateIfNotActive inserts a new pending notification. Idempotency is enforced by the
// database (a unique partial index on job_id for pending/sent/paid rows), not app-level
// locking. Populates n.ID/SMSStatus/PaymentStatus/CreatedAt/UpdatedAt on success.
func (r *PaymentNotificationRepository) CreateIfNotActive(ctx context.Context, n *models.PaymentNotification) error {
	now := time.Now()
	err := r.db.QueryRowContext(ctx, `
		INSERT INTO payment_notifications
			(organization_id, job_id, customer_id, amount, recipient_phone, message_body,
			 sms_status, payment_status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$10)
		RETURNING id`,
		n.OrganizationID, n.JobID, n.CustomerID, n.Amount, n.RecipientPhone, n.MessageBody,
		models.SMSStatusPending, models.PaymentStatusPending, n.CreatedBy, now,
	).Scan(&n.ID)

	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrPaymentRequestAlreadyActive
		}
		return err
	}

	n.SMSStatus = models.SMSStatusPending
	n.PaymentStatus = models.PaymentStatusPending
	n.CreatedAt, n.UpdatedAt = now, now
	return nil
}

// UpdateSMSResult is called once after the SMS send attempt completes (success or failure).
func (r *PaymentNotificationRepository) UpdateSMSResult(ctx context.Context, id uint, smsStatus models.SMSStatus, paymentStatus models.PaymentStatus, sentAt *time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE payment_notifications
		SET sms_status = $1, payment_status = $2, sent_at = $3, updated_at = $4
		WHERE id = $5`,
		smsStatus, paymentStatus, sentAt, time.Now(), id,
	)
	return err
}

// MarkPaid transitions a 'sent' notification to 'paid'. Returns ErrPaymentNotificationNotFound
// if no row matches id+orgID+payment_status='sent' (covers both "doesn't exist" and "wrong state").
func (r *PaymentNotificationRepository) MarkPaid(ctx context.Context, id, orgID uint) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE payment_notifications
		SET payment_status = $1, paid_at = $2, updated_at = $2
		WHERE id = $3 AND organization_id = $4 AND payment_status = $5`,
		models.PaymentStatusPaid, time.Now(), id, orgID, models.PaymentStatusSent,
	)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPaymentNotificationNotFound
	}
	return nil
}

// ListByJobID returns all notifications for a job, most recent first.
func (r *PaymentNotificationRepository) ListByJobID(ctx context.Context, jobID, orgID uint) ([]*models.PaymentNotification, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, organization_id, job_id, customer_id, amount, recipient_phone, message_body,
		       sms_status, payment_status, sent_at, paid_at, created_by, created_at, updated_at
		FROM payment_notifications
		WHERE job_id = $1 AND organization_id = $2
		ORDER BY created_at DESC`, jobID, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.PaymentNotification
	for rows.Next() {
		n := &models.PaymentNotification{}
		var sentAt, paidAt sql.NullTime
		var createdBy sql.NullInt64
		if err := rows.Scan(&n.ID, &n.OrganizationID, &n.JobID, &n.CustomerID, &n.Amount,
			&n.RecipientPhone, &n.MessageBody, &n.SMSStatus, &n.PaymentStatus,
			&sentAt, &paidAt, &createdBy, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		if sentAt.Valid {
			n.SentAt = &sentAt.Time
		}
		if paidAt.Valid {
			n.PaidAt = &paidAt.Time
		}
		if createdBy.Valid {
			cb := uint(createdBy.Int64)
			n.CreatedBy = &cb
		}
		out = append(out, n)
	}
	return out, rows.Err()
}
