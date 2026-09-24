package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

var ErrNotificationAlreadySent = errors.New("notification already sent")

type ServiceRequestNotificationRepository struct {
	db *sql.DB
}

func NewServiceRequestNotificationRepository(db *sql.DB) *ServiceRequestNotificationRepository {
	return &ServiceRequestNotificationRepository{db: db}
}

// Create inserts a pending notification row before the actual send is attempted. For
// kind='new_lead', a duplicate (same request+org+kind) is a no-op guarded by the partial
// unique index from migration 0014, surfaced here as ErrNotificationAlreadySent so the
// caller can skip re-sending on a retried fan-out.
func (r *ServiceRequestNotificationRepository) Create(ctx context.Context, n *models.ServiceRequestNotification) error {
	query := `
		INSERT INTO service_request_notifications
			(service_request_id, organization_id, kind, channel, recipient_phone, message_body, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT (service_request_id, organization_id, kind) WHERE kind = 'new_lead' DO NOTHING
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		n.ServiceRequestID, n.OrganizationID, n.Kind, n.Channel, n.RecipientPhone, n.MessageBody, n.Status,
	).Scan(&n.ID, &n.CreatedAt, &n.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotificationAlreadySent
	}
	return err
}

func (r *ServiceRequestNotificationRepository) UpdateStatus(ctx context.Context, id uint, status models.NotificationStatus, sentAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE service_request_notifications SET status = $1, sent_at = $2, updated_at = NOW() WHERE id = $3`,
		status, sentAt, id,
	)
	return err
}
