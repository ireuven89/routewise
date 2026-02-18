package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPaymentNotificationRepository_Create(t *testing.T) {
	tests := []struct {
		name         string
		notification *models.PaymentNotification
		setupMock    func(mock sqlmock.Sqlmock)
		wantErr      bool
		errContains  string
	}{
		{
			name: "successful creation",
			notification: &models.PaymentNotification{
				OrganizationID: 1,
				JobID:          100,
				CustomerID:     50,
				Amount:         500.00,
				PaymentLinkURL: "https://bit.app.link/pay?phone=123&amount=500",
				PaymentMethod:  "bit",
				SentVia:        "sms",
				RecipientPhone: "+972501234567",
				SMSStatus:      models.SMSStatusPending,
				PaymentStatus:  models.PaymentStatusPending,
				CreatedBy:      uintPtr(10),
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
					AddRow(1, time.Now(), time.Now())
				mock.ExpectQuery(`INSERT INTO payment_notifications`).
					WithArgs(1, 100, 50, 500.00, "https://bit.app.link/pay?phone=123&amount=500",
						"bit", "sms", "+972501234567", models.SMSStatusPending,
						models.PaymentStatusPending, uintPtr(10)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "database error",
			notification: &models.PaymentNotification{
				OrganizationID: 1,
				JobID:          100,
				CustomerID:     50,
				Amount:         500.00,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`INSERT INTO payment_notifications`).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			errContains: "failed to create payment notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			repo := NewPaymentNotificationRepository(db)
			err = repo.Create(tt.notification)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.notification.ID)
				assert.NotZero(t, tt.notification.CreatedAt)
				assert.NotZero(t, tt.notification.UpdatedAt)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentNotificationRepository_FindByID(t *testing.T) {
	tests := []struct {
		name           string
		id             uint
		organizationID uint
		setupMock      func(mock sqlmock.Sqlmock)
		wantErr        bool
		errContains    string
		validate       func(t *testing.T, notif *models.PaymentNotification)
	}{
		{
			name:           "successful retrieval",
			id:             1,
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				sentAt := time.Now()
				rows := sqlmock.NewRows([]string{
					"id", "organization_id", "job_id", "customer_id", "amount",
					"payment_link_url", "payment_method", "sent_via", "recipient_phone",
					"sent_at", "sms_status", "payment_status", "paid_at",
					"created_by", "created_at", "updated_at",
				}).AddRow(
					1, 1, 100, 50, 500.00,
					"https://bit.app.link/pay", "bit", "sms", "+972501234567",
					sentAt, models.SMSStatusSent, models.PaymentStatusSent, nil,
					uintPtr(10), time.Now(), time.Now(),
				)
				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE id = \$1 AND organization_id = \$2`).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, notif *models.PaymentNotification) {
				assert.Equal(t, uint(1), notif.ID)
				assert.Equal(t, uint(1), notif.OrganizationID)
				assert.Equal(t, uint(100), notif.JobID)
				assert.Equal(t, uint(50), notif.CustomerID)
				assert.Equal(t, 500.00, notif.Amount)
				assert.Equal(t, models.SMSStatusSent, notif.SMSStatus)
				assert.Equal(t, models.PaymentStatusSent, notif.PaymentStatus)
			},
		},
		{
			name:           "not found",
			id:             999,
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE id = \$1 AND organization_id = \$2`).
					WithArgs(999, 1).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			errContains: "payment notification not found",
		},
		{
			name:           "multi-tenant isolation - different organization",
			id:             1,
			organizationID: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE id = \$1 AND organization_id = \$2`).
					WithArgs(1, 2).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			errContains: "payment notification not found",
		},
		{
			name:           "database error",
			id:             1,
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE id = \$1 AND organization_id = \$2`).
					WithArgs(1, 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			errContains: "failed to find payment notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			repo := NewPaymentNotificationRepository(db)
			notif, err := repo.FindByID(tt.id, tt.organizationID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, notif)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, notif)
				if tt.validate != nil {
					tt.validate(t, notif)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentNotificationRepository_FindByJobID(t *testing.T) {
	tests := []struct {
		name           string
		jobID          uint
		organizationID uint
		setupMock      func(mock sqlmock.Sqlmock)
		wantErr        bool
		errContains    string
		wantCount      int
	}{
		{
			name:           "successful retrieval - multiple notifications",
			jobID:          100,
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "organization_id", "job_id", "customer_id", "amount",
					"payment_link_url", "payment_method", "sent_via", "recipient_phone",
					"sent_at", "sms_status", "payment_status", "paid_at",
					"created_by", "created_at", "updated_at",
				}).
					AddRow(1, 1, 100, 50, 500.00, "https://bit.app.link/pay1", "bit", "sms",
						"+972501234567", time.Now(), models.SMSStatusSent, models.PaymentStatusSent,
						nil, uintPtr(10), time.Now(), time.Now()).
					AddRow(2, 1, 100, 50, 500.00, "https://bit.app.link/pay2", "bit", "sms",
						"+972501234567", time.Now(), models.SMSStatusSent, models.PaymentStatusPaid,
						timePtr(time.Now()), uintPtr(10), time.Now(), time.Now())

				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE job_id = \$1 AND organization_id = \$2 ORDER BY created_at DESC`).
					WithArgs(100, 1).
					WillReturnRows(rows)
			},
			wantErr:   false,
			wantCount: 2,
		},
		{
			name:           "successful retrieval - no notifications",
			jobID:          999,
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "organization_id", "job_id", "customer_id", "amount",
					"payment_link_url", "payment_method", "sent_via", "recipient_phone",
					"sent_at", "sms_status", "payment_status", "paid_at",
					"created_by", "created_at", "updated_at",
				})
				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE job_id = \$1 AND organization_id = \$2`).
					WithArgs(999, 1).
					WillReturnRows(rows)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:           "multi-tenant isolation",
			jobID:          100,
			organizationID: 2,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "organization_id", "job_id", "customer_id", "amount",
					"payment_link_url", "payment_method", "sent_via", "recipient_phone",
					"sent_at", "sms_status", "payment_status", "paid_at",
					"created_by", "created_at", "updated_at",
				})
				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE job_id = \$1 AND organization_id = \$2`).
					WithArgs(100, 2).
					WillReturnRows(rows)
			},
			wantErr:   false,
			wantCount: 0,
		},
		{
			name:           "database error",
			jobID:          100,
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM payment_notifications WHERE job_id = \$1 AND organization_id = \$2`).
					WithArgs(100, 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			errContains: "failed to query payment notifications",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			repo := NewPaymentNotificationRepository(db)
			notifications, err := repo.FindByJobID(tt.jobID, tt.organizationID)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Len(t, notifications, tt.wantCount)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentNotificationRepository_UpdateSMSStatus(t *testing.T) {
	sentAt := time.Now()

	tests := []struct {
		name           string
		id             uint
		organizationID uint
		status         models.SMSStatus
		sentAt         *time.Time
		setupMock      func(mock sqlmock.Sqlmock)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful update to sent",
			id:             1,
			organizationID: 1,
			status:         models.SMSStatusSent,
			sentAt:         &sentAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET sms_status = \$1, sent_at = \$2`).
					WithArgs(models.SMSStatusSent, &sentAt, 1, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:           "successful update to failed",
			id:             1,
			organizationID: 1,
			status:         models.SMSStatusFailed,
			sentAt:         nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET sms_status = \$1, sent_at = \$2`).
					WithArgs(models.SMSStatusFailed, nil, 1, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:           "not found",
			id:             999,
			organizationID: 1,
			status:         models.SMSStatusSent,
			sentAt:         &sentAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET sms_status = \$1, sent_at = \$2`).
					WithArgs(models.SMSStatusSent, &sentAt, 999, 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			errContains: "payment notification not found or not in organization",
		},
		{
			name:           "multi-tenant isolation",
			id:             1,
			organizationID: 2,
			status:         models.SMSStatusSent,
			sentAt:         &sentAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET sms_status = \$1, sent_at = \$2`).
					WithArgs(models.SMSStatusSent, &sentAt, 1, 2).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			errContains: "payment notification not found or not in organization",
		},
		{
			name:           "database error",
			id:             1,
			organizationID: 1,
			status:         models.SMSStatusSent,
			sentAt:         &sentAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET sms_status = \$1, sent_at = \$2`).
					WithArgs(models.SMSStatusSent, &sentAt, 1, 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			errContains: "failed to update SMS status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			repo := NewPaymentNotificationRepository(db)
			err = repo.UpdateSMSStatus(tt.id, tt.organizationID, tt.status, tt.sentAt)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentNotificationRepository_UpdatePaymentStatus(t *testing.T) {
	paidAt := time.Now()

	tests := []struct {
		name           string
		id             uint
		organizationID uint
		status         models.PaymentStatus
		paidAt         *time.Time
		setupMock      func(mock sqlmock.Sqlmock)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful update to paid",
			id:             1,
			organizationID: 1,
			status:         models.PaymentStatusPaid,
			paidAt:         &paidAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET payment_status = \$1, paid_at = \$2`).
					WithArgs(models.PaymentStatusPaid, &paidAt, 1, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:           "successful update to sent",
			id:             1,
			organizationID: 1,
			status:         models.PaymentStatusSent,
			paidAt:         nil,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET payment_status = \$1, paid_at = \$2`).
					WithArgs(models.PaymentStatusSent, nil, 1, 1).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name:           "not found",
			id:             999,
			organizationID: 1,
			status:         models.PaymentStatusPaid,
			paidAt:         &paidAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET payment_status = \$1, paid_at = \$2`).
					WithArgs(models.PaymentStatusPaid, &paidAt, 999, 1).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			errContains: "payment notification not found or not in organization",
		},
		{
			name:           "multi-tenant isolation",
			id:             1,
			organizationID: 2,
			status:         models.PaymentStatusPaid,
			paidAt:         &paidAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET payment_status = \$1, paid_at = \$2`).
					WithArgs(models.PaymentStatusPaid, &paidAt, 1, 2).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr:     true,
			errContains: "payment notification not found or not in organization",
		},
		{
			name:           "database error",
			id:             1,
			organizationID: 1,
			status:         models.PaymentStatusPaid,
			paidAt:         &paidAt,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(`UPDATE payment_notifications SET payment_status = \$1, paid_at = \$2`).
					WithArgs(models.PaymentStatusPaid, &paidAt, 1, 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			errContains: "failed to update payment status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			repo := NewPaymentNotificationRepository(db)
			err = repo.UpdatePaymentStatus(tt.id, tt.organizationID, tt.status, tt.paidAt)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Helper functions
func uintPtr(u uint) *uint {
	return &u
}

func timePtr(t time.Time) *time.Time {
	return &t
}
