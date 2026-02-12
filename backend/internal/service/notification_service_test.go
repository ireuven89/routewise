package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTwilioClient is a mock for the Twilio REST client
type MockTwilioClient struct {
	mock.Mock
}

// MockTwilioAPI is a mock for the Twilio API
type MockTwilioAPI struct {
	mock.Mock
}

// CreateMessage is a mock method for sending SMS
func (m *MockTwilioAPI) CreateMessage(params interface{}) (interface{}, error) {
	args := m.Called(params)
	return args.Get(0), args.Error(1)
}

// MockNotificationService implements NotificationService for testing
type MockNotificationService struct {
	mock.Mock
}

func (m *MockNotificationService) SendSMS(phone, message string) error {
	args := m.Called(phone, message)
	return args.Error(0)
}

func TestNotificationService_SendSMS(t *testing.T) {
	tests := []struct {
		name        string
		phone       string
		message     string
		setupMock   func(mock *MockNotificationService)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful SMS send",
			phone:   "+972501234567",
			message: "שלום! תשלום עבור עבודה בסך 500.00 ש\"ח",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567", "שלום! תשלום עבור עבודה בסך 500.00 ש\"ח").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "successful SMS with English message",
			phone:   "+972501234567",
			message: "Hello! Payment for job $500.00",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567", "Hello! Payment for job $500.00").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "Twilio API error",
			phone:   "+972501234567",
			message: "Test message",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567", "Test message").
					Return(errors.New("failed to send SMS: Twilio API error"))
			},
			wantErr:     true,
			errContains: "failed to send SMS",
		},
		{
			name:    "invalid phone number",
			phone:   "invalid",
			message: "Test message",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "invalid", "Test message").
					Return(errors.New("failed to send SMS: invalid phone number"))
			},
			wantErr:     true,
			errContains: "failed to send SMS",
		},
		{
			name:    "empty phone number",
			phone:   "",
			message: "Test message",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "", "Test message").
					Return(errors.New("failed to send SMS: phone number is required"))
			},
			wantErr:     true,
			errContains: "failed to send SMS",
		},
		{
			name:    "long message with special characters",
			phone:   "+972501234567",
			message: "שלום {{customer_name}}! עבור עבודה: {{job_title}}. סכום: {{amount}} ש\"ח. לתשלום: {{link}}",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567",
					"שלום {{customer_name}}! עבור עבודה: {{job_title}}. סכום: {{amount}} ש\"ח. לתשלום: {{link}}").
					Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockNotificationService)
			tt.setupMock(mockService)

			err := mockService.SendSMS(tt.phone, tt.message)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestNotificationService_SendSMS_EdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		phone     string
		message   string
		setupMock func(mock *MockNotificationService)
		wantErr   bool
	}{
		{
			name:    "empty message",
			phone:   "+972501234567",
			message: "",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567", "").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "very long message",
			phone:   "+972501234567",
			message: string(make([]byte, 1600)), // SMS limit is typically 1600 chars
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567", string(make([]byte, 1600))).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "phone with country code and dashes",
			phone:   "+972-50-123-4567",
			message: "Test message",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972-50-123-4567", "Test message").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "message with URL",
			phone:   "+972501234567",
			message: "Payment link: https://bit.app.link/pay?phone=123&amount=500",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567",
					"Payment link: https://bit.app.link/pay?phone=123&amount=500").
					Return(nil)
			},
			wantErr: false,
		},
		{
			name:    "message with newlines",
			phone:   "+972501234567",
			message: "שלום\nתשלום עבור עבודה\nסכום: 500 ש\"ח",
			setupMock: func(mock *MockNotificationService) {
				mock.On("SendSMS", "+972501234567", "שלום\nתשלום עבור עבודה\nסכום: 500 ש\"ח").
					Return(nil)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockNotificationService)
			tt.setupMock(mockService)

			err := mockService.SendSMS(tt.phone, tt.message)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockService.AssertExpectations(t)
		})
	}
}
