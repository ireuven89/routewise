package models

import "time"

// SMSStatus represents the delivery status of an SMS
type SMSStatus string

const (
	SMSStatusPending   SMSStatus = "pending"
	SMSStatusSent      SMSStatus = "sent"
	SMSStatusFailed    SMSStatus = "failed"
	SMSStatusDelivered SMSStatus = "delivered"
)

// PaymentStatus represents the status of a payment
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusSent      PaymentStatus = "sent"
	PaymentStatusPaid      PaymentStatus = "paid"
	PaymentStatusCancelled PaymentStatus = "cancelled"
)

// PaymentNotification tracks a payment link sent to a customer
type PaymentNotification struct {
	ID             uint          `json:"id"`
	OrganizationID uint          `json:"organization_id"`
	JobID          uint          `json:"job_id"`
	CustomerID     uint          `json:"customer_id"`
	Amount         float64       `json:"amount"`
	PaymentLinkURL string        `json:"payment_link_url"`
	PaymentMethod  string        `json:"payment_method"`
	SentVia        string        `json:"sent_via"`
	RecipientPhone string        `json:"recipient_phone"`
	SentAt         *time.Time    `json:"sent_at"`
	SMSStatus      SMSStatus     `json:"sms_status"`
	PaymentStatus  PaymentStatus `json:"payment_status"`
	PaidAt         *time.Time    `json:"paid_at"`
	CreatedBy      *uint         `json:"created_by"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`

	// Relationships (optional, populated on demand)
	Job      *Job      `json:"job,omitempty"`
	Customer *Customer `json:"customer,omitempty"`
}

// OrganizationPaymentSettings configures payment behavior for an organization
type OrganizationPaymentSettings struct {
	ID                   uint      `json:"id"`
	OrganizationID       uint      `json:"organization_id"`
	BitPaymentEnabled    bool      `json:"bit_payment_enabled"`
	BitPhoneNumber       string    `json:"bit_phone_number"`
	BitBusinessName      string    `json:"bit_business_name"`
	AutoSendOnCompletion bool      `json:"auto_send_on_completion"`
	SMSTemplateHe        string    `json:"sms_template_he"`
	SMSTemplateEn        string    `json:"sms_template_en"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}
