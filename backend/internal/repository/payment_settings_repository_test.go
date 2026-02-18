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

func TestPaymentSettingsRepository_GetByOrganizationID(t *testing.T) {
	tests := []struct {
		name           string
		organizationID uint
		setupMock      func(mock sqlmock.Sqlmock)
		wantErr        bool
		errContains    string
		validate       func(t *testing.T, settings *models.OrganizationPaymentSettings)
	}{
		{
			name:           "successful retrieval with enabled payments",
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "organization_id", "bit_payment_enabled", "bit_phone_number",
					"bit_business_name", "auto_send_on_completion", "sms_template_he",
					"sms_template_en", "created_at", "updated_at",
				}).AddRow(
					1, 1, true, "+972501234567",
					"ACME HVAC", true, "שלום {{customer_name}}",
					"Hello {{customer_name}}", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`SELECT .+ FROM organization_payment_settings WHERE organization_id = \$1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, settings *models.OrganizationPaymentSettings) {
				assert.Equal(t, uint(1), settings.ID)
				assert.Equal(t, uint(1), settings.OrganizationID)
				assert.True(t, settings.BitPaymentEnabled)
				assert.Equal(t, "+972501234567", settings.BitPhoneNumber)
				assert.Equal(t, "ACME HVAC", settings.BitBusinessName)
				assert.True(t, settings.AutoSendOnCompletion)
				assert.Contains(t, settings.SMSTemplateHe, "{{customer_name}}")
			},
		},
		{
			name:           "successful retrieval with disabled payments",
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{
					"id", "organization_id", "bit_payment_enabled", "bit_phone_number",
					"bit_business_name", "auto_send_on_completion", "sms_template_he",
					"sms_template_en", "created_at", "updated_at",
				}).AddRow(
					1, 1, false, "", "", false, "", "", time.Now(), time.Now(),
				)
				mock.ExpectQuery(`SELECT .+ FROM organization_payment_settings WHERE organization_id = \$1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			wantErr: false,
			validate: func(t *testing.T, settings *models.OrganizationPaymentSettings) {
				assert.False(t, settings.BitPaymentEnabled)
				assert.Empty(t, settings.BitPhoneNumber)
			},
		},
		{
			name:           "not found",
			organizationID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM organization_payment_settings WHERE organization_id = \$1`).
					WithArgs(999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			errContains: "payment settings not found for organization",
		},
		{
			name:           "database error",
			organizationID: 1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`SELECT .+ FROM organization_payment_settings WHERE organization_id = \$1`).
					WithArgs(1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			errContains: "failed to get payment settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			repo := NewPaymentSettingsRepository(db)
			settings, err := repo.GetByOrganizationID(tt.organizationID)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, settings)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, settings)
				if tt.validate != nil {
					tt.validate(t, settings)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPaymentSettingsRepository_Update(t *testing.T) {
	tests := []struct {
		name        string
		settings    *models.OrganizationPaymentSettings
		setupMock   func(mock sqlmock.Sqlmock)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful update - enable payments",
			settings: &models.OrganizationPaymentSettings{
				OrganizationID:       1,
				BitPaymentEnabled:    true,
				BitPhoneNumber:       "+972501234567",
				BitBusinessName:      "ACME HVAC",
				AutoSendOnCompletion: true,
				SMSTemplateHe:        "שלום {{customer_name}}",
				SMSTemplateEn:        "Hello {{customer_name}}",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"updated_at"}).
					AddRow(time.Now())
				mock.ExpectQuery(`UPDATE organization_payment_settings SET`).
					WithArgs(true, "+972501234567", "ACME HVAC", true,
						"שלום {{customer_name}}", "Hello {{customer_name}}", 1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "successful update - disable payments",
			settings: &models.OrganizationPaymentSettings{
				OrganizationID:       1,
				BitPaymentEnabled:    false,
				BitPhoneNumber:       "",
				BitBusinessName:      "",
				AutoSendOnCompletion: false,
				SMSTemplateHe:        "",
				SMSTemplateEn:        "",
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"updated_at"}).
					AddRow(time.Now())
				mock.ExpectQuery(`UPDATE organization_payment_settings SET`).
					WithArgs(false, "", "", false, "", "", 1).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			settings: &models.OrganizationPaymentSettings{
				OrganizationID:    999,
				BitPaymentEnabled: true,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE organization_payment_settings SET`).
					WithArgs(true, "", "", false, "", "", 999).
					WillReturnError(sql.ErrNoRows)
			},
			wantErr:     true,
			errContains: "payment settings not found for organization",
		},
		{
			name: "database error",
			settings: &models.OrganizationPaymentSettings{
				OrganizationID:    1,
				BitPaymentEnabled: true,
			},
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(`UPDATE organization_payment_settings SET`).
					WithArgs(true, "", "", false, "", "", 1).
					WillReturnError(sql.ErrConnDone)
			},
			wantErr:     true,
			errContains: "failed to update payment settings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer db.Close()

			tt.setupMock(mock)

			repo := NewPaymentSettingsRepository(db)
			err = repo.Update(tt.settings)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
				assert.NotZero(t, tt.settings.UpdatedAt)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
