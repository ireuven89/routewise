package repository

import (
	"database/sql"
	"fmt"

	"github.com/ireuven89/routewise/internal/models"
)

// PaymentSettingsRepository handles database operations for organization payment settings
type PaymentSettingsRepository struct {
	db *sql.DB
}

// NewPaymentSettingsRepository creates a new repository instance
func NewPaymentSettingsRepository(db *sql.DB) *PaymentSettingsRepository {
	return &PaymentSettingsRepository{db: db}
}

// GetByOrganizationID retrieves payment settings for an organization
func (r *PaymentSettingsRepository) GetByOrganizationID(organizationID uint) (*models.OrganizationPaymentSettings, error) {
	query := `
		SELECT
			id, organization_id, bit_payment_enabled, bit_phone_number,
			bit_business_name, auto_send_on_completion, sms_template_he,
			sms_template_en, created_at, updated_at
		FROM organization_payment_settings
		WHERE organization_id = $1
	`

	settings := &models.OrganizationPaymentSettings{}
	err := r.db.QueryRow(query, organizationID).Scan(
		&settings.ID,
		&settings.OrganizationID,
		&settings.BitPaymentEnabled,
		&settings.BitPhoneNumber,
		&settings.BitBusinessName,
		&settings.AutoSendOnCompletion,
		&settings.SMSTemplateHe,
		&settings.SMSTemplateEn,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("payment settings not found for organization")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get payment settings: %w", err)
	}

	return settings, nil
}

// Update updates payment settings for an organization
func (r *PaymentSettingsRepository) Update(settings *models.OrganizationPaymentSettings) error {
	query := `
		UPDATE organization_payment_settings
		SET
			bit_payment_enabled = $1,
			bit_phone_number = $2,
			bit_business_name = $3,
			auto_send_on_completion = $4,
			sms_template_he = $5,
			sms_template_en = $6,
			updated_at = NOW()
		WHERE organization_id = $7
		RETURNING updated_at
	`

	err := r.db.QueryRow(
		query,
		settings.BitPaymentEnabled,
		settings.BitPhoneNumber,
		settings.BitBusinessName,
		settings.AutoSendOnCompletion,
		settings.SMSTemplateHe,
		settings.SMSTemplateEn,
		settings.OrganizationID,
	).Scan(&settings.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("payment settings not found for organization")
	}
	if err != nil {
		return fmt.Errorf("failed to update payment settings: %w", err)
	}

	return nil
}
