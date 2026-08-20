package models

import "time"

type SMSStatus string

const (
	SMSStatusPending   SMSStatus = "pending"
	SMSStatusSent      SMSStatus = "sent"
	SMSStatusFailed    SMSStatus = "failed"
	SMSStatusDelivered SMSStatus = "delivered"
)

type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusSent       PaymentStatus = "sent"
	PaymentStatusSendFailed PaymentStatus = "send_failed"
	PaymentStatusPaid       PaymentStatus = "paid"
	PaymentStatusCancelled  PaymentStatus = "cancelled"
)

// PaymentNotification is an audit record of a Bit payment-collection SMS sent for a job.
type PaymentNotification struct {
	ID             uint          `json:"id"`
	OrganizationID uint          `json:"organization_id"`
	JobID          uint          `json:"job_id"`
	CustomerID     uint          `json:"customer_id"`
	Amount         float64       `json:"amount"`
	RecipientPhone string        `json:"recipient_phone"`
	MessageBody    string        `json:"message_body"`
	SMSStatus      SMSStatus     `json:"sms_status"`
	PaymentStatus  PaymentStatus `json:"payment_status"`
	SentAt         *time.Time    `json:"sent_at,omitempty"`
	PaidAt         *time.Time    `json:"paid_at,omitempty"`
	CreatedBy      *uint         `json:"created_by,omitempty"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}
