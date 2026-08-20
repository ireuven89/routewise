package service

import (
	"fmt"
	"os"

	"github.com/twilio/twilio-go"
	twapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// NotificationService is a thin, reusable SMS-sending abstraction over Twilio, shared by
// AuthService (worker OTP) and PaymentService (Bit payment requests).
type NotificationService interface {
	SendSMS(phone, message string) error
}

type TwilioNotificationService struct {
	client     *twilio.RestClient
	fromNumber string
}

func NewTwilioNotificationService() NotificationService {
	return &TwilioNotificationService{
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: os.Getenv("TWILIO_SID"),
			Password: os.Getenv("TWILIO_AUTH_TOKEN"),
		}),
		fromNumber: os.Getenv("TWILIO_PHONE_NUMBER"),
	}
}

func (s *TwilioNotificationService) SendSMS(phone, message string) error {
	params := &twapi.CreateMessageParams{}
	params.SetTo(phone)
	params.SetFrom(s.fromNumber)
	params.SetBody(message)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio send failed: %w", err)
	}
	return nil
}
