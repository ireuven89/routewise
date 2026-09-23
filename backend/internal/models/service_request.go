package models

import "time"

type ServiceRequestStatus string

const (
	ServiceRequestStatusOpen      ServiceRequestStatus = "open"
	ServiceRequestStatusAwarded   ServiceRequestStatus = "awarded"
	ServiceRequestStatusCancelled ServiceRequestStatus = "cancelled"
)

type BidStatus string

const (
	BidStatusSubmitted BidStatus = "submitted"
	BidStatusAwarded   BidStatus = "awarded"
	BidStatusRejected  BidStatus = "rejected"
)

// NotificationKind identifies which message in the smart-dispatching flow a
// service_request_notifications row records.
type NotificationKind string

const (
	NotificationKindNewLead           NotificationKind = "new_lead"           // whatsapp -> org
	NotificationKindBidAwarded        NotificationKind = "bid_awarded"        // whatsapp -> winning org
	NotificationKindBidRejected       NotificationKind = "bid_rejected"       // whatsapp -> losing orgs
	NotificationKindTrackingLink      NotificationKind = "tracking_link"      // sms -> customer
	NotificationKindAwardConfirmation NotificationKind = "award_confirmation" // sms -> customer
)

// NotificationStatus mirrors the sent/failed lifecycle of a single WhatsApp/SMS send.
type NotificationStatus string

const (
	NotificationStatusPending NotificationStatus = "pending"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

// ServiceRequest is a customer's job post from the public /find-service broadcast bidding
// flow. It is not tied to any one organization until a bid is awarded.
type ServiceRequest struct {
	ID            uint                 `json:"id"`
	AccessToken   string               `json:"-"` // never serialized except once, explicitly, at creation
	ServiceType   string               `json:"service_type"`
	Description   string               `json:"description"`
	CustomerName  string               `json:"customer_name"`
	CustomerPhone string               `json:"customer_phone"`
	Latitude      float64              `json:"latitude"`
	Longitude     float64              `json:"longitude"`
	Address       string               `json:"address"`
	PreferredTime *time.Time           `json:"preferred_time,omitempty"`
	Status        ServiceRequestStatus `json:"status"`
	AwardedBidID  *uint                `json:"awarded_bid_id,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

// ServiceRequestBid is a single organization's bid on a ServiceRequest.
type ServiceRequestBid struct {
	ID                uint      `json:"id"`
	ServiceRequestID  uint      `json:"service_request_id"`
	OrganizationID    uint      `json:"organization_id"`
	OrganizationName  string    `json:"organization_name,omitempty"` // joined in for customer-facing listing
	OrganizationPhone string    `json:"-"`                           // joined in for internal WhatsApp notification use only
	Price             float64   `json:"price"`
	EtaMinutes        *int      `json:"eta_minutes,omitempty"`
	Message           string    `json:"message"`
	Status            BidStatus `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// ServiceRequestNotification audits a single WhatsApp (org-facing) or SMS (customer-facing)
// send for smart dispatching, mirroring the payment_notifications audit pattern.
type ServiceRequestNotification struct {
	ID               uint               `json:"id"`
	ServiceRequestID uint               `json:"service_request_id"`
	OrganizationID   *uint              `json:"organization_id,omitempty"`
	Kind             NotificationKind   `json:"kind"`
	Channel          string             `json:"channel"`
	RecipientPhone   string             `json:"recipient_phone"`
	MessageBody      string             `json:"message_body"`
	Status           NotificationStatus `json:"status"`
	SentAt           *time.Time         `json:"sent_at,omitempty"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}
