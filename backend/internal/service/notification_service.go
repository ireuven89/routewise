package service

import (
	"fmt"
	"os"

	"github.com/twilio/twilio-go"
	twapi "github.com/twilio/twilio-go/rest/api/v2010"
)

// NotificationService is a thin, reusable messaging abstraction over Twilio, used by
// ServiceRequestService to notify organizations of new leads over WhatsApp and customers
// over SMS. It's intentionally standalone rather than reusing AuthService's private Twilio
// client, since that client isn't exposed outside auth_service.go.
type NotificationService interface {
	SendSMS(phone, message string) error
	SendWhatsApp(phone, message string) error
}

type TwilioNotificationService struct {
	client             *twilio.RestClient
	fromNumber         string
	whatsappFromNumber string
}

func NewTwilioNotificationService() NotificationService {
	return &TwilioNotificationService{
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: os.Getenv("TWILIO_SID"),
			Password: os.Getenv("TWILIO_AUTH_TOKEN"),
		}),
		fromNumber:         os.Getenv("TWILIO_PHONE_NUMBER"),
		whatsappFromNumber: os.Getenv("TWILIO_WHATSAPP_NUMBER"),
	}
}

func (s *TwilioNotificationService) SendSMS(phone, message string) error {
	params := &twapi.CreateMessageParams{}
	params.SetTo(phone)
	params.SetFrom(s.fromNumber)
	params.SetBody(message)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio sms send failed: %w", err)
	}
	return nil
}

// SendWhatsApp sends via Twilio's WhatsApp API (Sandbox or an approved sender depending on
// TWILIO_WHATSAPP_NUMBER). Both To and From must be "whatsapp:"-prefixed E.164 numbers; for
// the Sandbox, the recipient must have already joined via Twilio's join code, otherwise this
// call fails and the caller should record it as a failed notification, not treat it as fatal.
func (s *TwilioNotificationService) SendWhatsApp(phone, message string) error {
	params := &twapi.CreateMessageParams{}
	params.SetTo("whatsapp:" + phone)
	params.SetFrom("whatsapp:" + s.whatsappFromNumber)
	params.SetBody(message)

	_, err := s.client.Api.CreateMessage(params)
	if err != nil {
		return fmt.Errorf("twilio whatsapp send failed: %w", err)
	}
	return nil
}
