package service

import (
	"fmt"
	"os"

	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

// NotificationService handles sending notifications via various channels
type NotificationService interface {
	SendSMS(phone, message string) error
}

// NotificationServiceImpl implements NotificationService using Twilio
type NotificationServiceImpl struct {
	twilioClient *twilio.RestClient
}

// NewNotificationService creates a new notification service instance
func NewNotificationService() NotificationService {
	return &NotificationServiceImpl{
		twilioClient: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: os.Getenv("TWILIO_SID"),
			Password: os.Getenv("TWILIO_AUTH_TOKEN"),
		}),
	}
}

// SendSMS sends an SMS message to the specified phone number
func (s *NotificationServiceImpl) SendSMS(phone, message string) error {
	params := &api.CreateMessageParams{}
	params.SetTo(phone)
	params.SetFrom(os.Getenv("TWILIO_PHONE_NUMBER"))
	params.SetBody(message)

	_, err := s.twilioClient.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("failed to send SMS: %w", err)
	}

	return nil
}
