package service

import (
	"testing"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/stretchr/testify/assert"
)

// Tests for business logic in PaymentLinkService
// Note: Full integration tests with database mocks would require interface-based repositories
// These tests focus on the pure business logic functions

func TestPaymentLinkService_GenerateBitLink(t *testing.T) {
	service := &PaymentLinkServiceImpl{}

	tests := []struct {
		name        string
		settings    *models.OrganizationPaymentSettings
		amount      float64
		description string
		validate    func(t *testing.T, link string)
	}{
		{
			name: "standard link generation",
			settings: &models.OrganizationPaymentSettings{
				BitPhoneNumber: "+972501234567",
			},
			amount:      500.00,
			description: "HVAC Repair",
			validate: func(t *testing.T, link string) {
				assert.Contains(t, link, "https://bit.app.link/pay")
				assert.Contains(t, link, "phone=%2B972501234567")
				assert.Contains(t, link, "amount=500.00")
				assert.Contains(t, link, "description=HVAC")
			},
		},
		{
			name: "special characters in description",
			settings: &models.OrganizationPaymentSettings{
				BitPhoneNumber: "+972501234567",
			},
			amount:      1250.50,
			description: "תיקון מזגן & חימום",
			validate: func(t *testing.T, link string) {
				assert.Contains(t, link, "https://bit.app.link/pay")
				assert.Contains(t, link, "amount=1250.50")
				// Hebrew should be URL encoded
				assert.NotContains(t, link, "תיקון")
			},
		},
		{
			name: "phone with special characters",
			settings: &models.OrganizationPaymentSettings{
				BitPhoneNumber: "+972-50-123-4567",
			},
			amount:      100.00,
			description: "Test",
			validate: func(t *testing.T, link string) {
				assert.Contains(t, link, "https://bit.app.link/pay")
				// Phone should be URL encoded
				assert.Contains(t, link, "phone=")
			},
		},
		{
			name: "large amount",
			settings: &models.OrganizationPaymentSettings{
				BitPhoneNumber: "+972501234567",
			},
			amount:      15000.99,
			description: "Major Renovation",
			validate: func(t *testing.T, link string) {
				assert.Contains(t, link, "amount=15000.99")
			},
		},
		{
			name: "amount with cents",
			settings: &models.OrganizationPaymentSettings{
				BitPhoneNumber: "+972501234567",
			},
			amount:      99.95,
			description: "Service Call",
			validate: func(t *testing.T, link string) {
				assert.Contains(t, link, "amount=99.95")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := service.generateBitLink(tt.settings, tt.amount, tt.description)
			assert.NotEmpty(t, link)
			if tt.validate != nil {
				tt.validate(t, link)
			}
		})
	}
}

func TestPaymentLinkService_FormatSMSMessage(t *testing.T) {
	service := &PaymentLinkServiceImpl{}

	tests := []struct {
		name     string
		settings *models.OrganizationPaymentSettings
		customer *models.Customer
		job      *models.Job
		amount   float64
		link     string
		validate func(t *testing.T, message string)
	}{
		{
			name: "Hebrew template with all variables",
			settings: &models.OrganizationPaymentSettings{
				SMSTemplateHe: "שלום {{customer_name}}! תשלום עבור {{job_title}} בסך {{amount}} ש\"ח. {{link}}",
			},
			customer: &models.Customer{Name: "יוחנן דו"},
			job:      &models.Job{Title: "תיקון מזגן"},
			amount:   500.00,
			link:     "https://bit.app.link/pay?phone=123&amount=500",
			validate: func(t *testing.T, message string) {
				assert.Contains(t, message, "שלום יוחנן דו!")
				assert.Contains(t, message, "תיקון מזגן")
				assert.Contains(t, message, "500.00")
				assert.Contains(t, message, "https://bit.app.link/pay")
				assert.NotContains(t, message, "{{customer_name}}")
				assert.NotContains(t, message, "{{job_title}}")
			},
		},
		{
			name: "English template",
			settings: &models.OrganizationPaymentSettings{
				SMSTemplateHe: "Hello {{customer_name}}! Payment for {{job_title}} - {{amount}} ILS. {{link}}",
			},
			customer: &models.Customer{Name: "John Doe"},
			job:      &models.Job{Title: "HVAC Repair"},
			amount:   1250.50,
			link:     "https://bit.app.link/pay?phone=123&amount=1250.50",
			validate: func(t *testing.T, message string) {
				assert.Contains(t, message, "Hello John Doe!")
				assert.Contains(t, message, "HVAC Repair")
				assert.Contains(t, message, "1250.50")
			},
		},
		{
			name: "template with decimal formatting",
			settings: &models.OrganizationPaymentSettings{
				SMSTemplateHe: "סכום: {{amount}}",
			},
			customer: &models.Customer{Name: "Test"},
			job:      &models.Job{Title: "Test"},
			amount:   99.99,
			link:     "https://test.link",
			validate: func(t *testing.T, message string) {
				assert.Contains(t, message, "99.99")
			},
		},
		{
			name: "template with only customer name",
			settings: &models.OrganizationPaymentSettings{
				SMSTemplateHe: "שלום {{customer_name}}",
			},
			customer: &models.Customer{Name: "דוד כהן"},
			job:      &models.Job{Title: "Job"},
			amount:   100.00,
			link:     "https://link.com",
			validate: func(t *testing.T, message string) {
				assert.Contains(t, message, "דוד כהן")
				assert.NotContains(t, message, "{{customer_name}}")
			},
		},
		{
			name: "template with all variables separately",
			settings: &models.OrganizationPaymentSettings{
				SMSTemplateHe: "Name: {{customer_name}}, Job: {{job_title}}, Amount: {{amount}}, Link: {{link}}",
			},
			customer: &models.Customer{Name: "Alice"},
			job:      &models.Job{Title: "Plumbing"},
			amount:   750.00,
			link:     "https://payment.link",
			validate: func(t *testing.T, message string) {
				assert.Contains(t, message, "Name: Alice")
				assert.Contains(t, message, "Job: Plumbing")
				assert.Contains(t, message, "Amount: 750.00")
				assert.Contains(t, message, "Link: https://payment.link")
			},
		},
		{
			name: "template with special characters in customer name",
			settings: &models.OrganizationPaymentSettings{
				SMSTemplateHe: "{{customer_name}} - {{amount}}",
			},
			customer: &models.Customer{Name: "O'Brien & Sons"},
			job:      &models.Job{Title: "Job"},
			amount:   200.00,
			link:     "https://link.com",
			validate: func(t *testing.T, message string) {
				assert.Contains(t, message, "O'Brien & Sons")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := service.formatSMSMessage(tt.settings, tt.customer, tt.job, tt.amount, tt.link)
			assert.NotEmpty(t, message)
			if tt.validate != nil {
				tt.validate(t, message)
			}
		})
	}
}

func TestPaymentLinkService_ErrorConstants(t *testing.T) {
	// Test that error constants are properly defined
	tests := []struct {
		name string
		err  error
	}{
		{"job not completed", ErrJobNotCompleted},
		{"job has no price", ErrJobHasNoPrice},
		{"customer has no phone", ErrCustomerHasNoPhone},
		{"payment not enabled", ErrPaymentNotEnabled},
		{"payment already sent", ErrPaymentAlreadySent},
		{"bit phone not configured", ErrBitPhoneNotConfigured},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NotNil(t, tt.err)
			assert.NotEmpty(t, tt.err.Error())
		})
	}
}

func TestPaymentLinkService_BitLinkFormat(t *testing.T) {
	service := &PaymentLinkServiceImpl{}

	settings := &models.OrganizationPaymentSettings{
		BitPhoneNumber: "+972501234567",
	}

	link := service.generateBitLink(settings, 500.00, "Test Job")

	// Verify URL structure
	assert.True(t, len(link) > 0, "Link should not be empty")
	assert.Contains(t, link, "https://", "Should use HTTPS")
	assert.Contains(t, link, "bit.app.link", "Should use Bit domain")
	assert.Contains(t, link, "/pay", "Should have /pay path")
	assert.Contains(t, link, "?", "Should have query parameters")
	assert.Contains(t, link, "phone=", "Should have phone parameter")
	assert.Contains(t, link, "amount=", "Should have amount parameter")
	assert.Contains(t, link, "description=", "Should have description parameter")
}

func TestPaymentLinkService_SMSTemplateVariables(t *testing.T) {
	service := &PaymentLinkServiceImpl{}

	// Test that all variables are replaced
	settings := &models.OrganizationPaymentSettings{
		SMSTemplateHe: "{{customer_name}} {{job_title}} {{amount}} {{link}}",
	}

	customer := &models.Customer{Name: "John"}
	job := &models.Job{Title: "Repair"}
	amount := 100.00
	link := "https://test.link"

	message := service.formatSMSMessage(settings, customer, job, amount, link)

	// Verify no template variables remain
	assert.NotContains(t, message, "{{")
	assert.NotContains(t, message, "}}")

	// Verify all values are present
	assert.Contains(t, message, "John")
	assert.Contains(t, message, "Repair")
	assert.Contains(t, message, "100.00")
	assert.Contains(t, message, "https://test.link")
}
