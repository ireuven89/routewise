package service

import "testing"

// TwilioNotificationService wraps the twilio-go SDK's HTTP client - it has no
// fake-transport seam in this codebase, so SendSMS itself isn't unit-testable
// without hitting the network. Per this feature's test plan, we keep this
// file intentionally light: just confirm the constructor wires up a non-nil
// value that satisfies the NotificationService interface, which is what
// AuthService/PaymentService actually depend on.

func TestNewTwilioNotificationService_ReturnsNonNilNotificationService(t *testing.T) {
	svc := NewTwilioNotificationService()
	if svc == nil {
		t.Fatal("expected a non-nil NotificationService")
	}
}

func TestNewTwilioNotificationService_ReturnsTwilioNotificationServiceConcreteType(t *testing.T) {
	svc := NewTwilioNotificationService()
	if _, ok := svc.(*TwilioNotificationService); !ok {
		t.Fatalf("expected *TwilioNotificationService, got %T", svc)
	}
}
