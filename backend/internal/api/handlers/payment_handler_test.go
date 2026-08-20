package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
)

// -----------------------------------------------------------------------
// PaymentHandler.service is a concrete *service.PaymentService, not an
// interface, so (unlike DashboardHandler's JobService, which dashboard_test.go
// mocks by embedding the real interface) we can't inject a mock into the real
// PaymentHandler directly. Per this package's established convention for
// concrete-service handlers (see provider_handler_test.go's
// testProviderHandler/mockProviderSvc), we mirror PaymentHandler's methods
// against a mock service with function fields.
// -----------------------------------------------------------------------

type mockPaymentSvc struct {
	SendPaymentRequestFunc func(ctx context.Context, orgID, jobID uint, userID *uint) (*models.PaymentNotification, error)
	GetForJobFunc          func(ctx context.Context, orgID, jobID uint) ([]*models.PaymentNotification, error)
	MarkPaidFunc           func(ctx context.Context, orgID, notificationID uint) error
	GetSettingsFunc        func(orgID uint) (*models.Organization, error)
	UpdateSettingsFunc     func(ctx context.Context, orgID uint, req service.UpdatePaymentSettingsRequest) error
}

// testPaymentHandler mirrors PaymentHandler exactly, delegating to mockPaymentSvc.
type testPaymentHandler struct {
	svc *mockPaymentSvc
}

func (h *testPaymentHandler) SendPaymentRequest(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	userID := c.GetUint("organization_user_id")

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	notification, err := h.svc.SendPaymentRequestFunc(c.Request.Context(), orgID, uint(jobID), &userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPaymentSettingsNotConfigured),
			errors.Is(err, service.ErrJobNotCompletedForPayment),
			errors.Is(err, service.ErrCustomerPhoneMissing):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, service.ErrJobNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "Job not found"})
		case errors.Is(err, repository.ErrPaymentRequestAlreadyActive):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			if notification != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "notification": notification})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send payment request"})
		}
		return
	}

	c.JSON(http.StatusCreated, notification)
}

func (h *testPaymentHandler) GetPaymentNotifications(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	jobID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID"})
		return
	}

	notifications, err := h.svc.GetForJobFunc(c.Request.Context(), orgID, uint(jobID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payment notifications"})
		return
	}
	if notifications == nil {
		notifications = []*models.PaymentNotification{}
	}
	c.JSON(http.StatusOK, notifications)
}

func (h *testPaymentHandler) MarkPaid(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	if err := h.svc.MarkPaidFunc(c.Request.Context(), orgID, uint(id)); err != nil {
		if errors.Is(err, repository.ErrPaymentNotificationNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to mark payment as paid"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment marked as paid"})
}

func (h *testPaymentHandler) GetPaymentSettings(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	org, err := h.svc.GetSettingsFunc(orgID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "organization not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bit_payment_enabled":   org.BitPaymentEnabled,
		"bit_phone_number":      org.BitPhoneNumber,
		"bit_business_name":     org.BitBusinessName,
		"auto_send_payment_sms": org.AutoSendPaymentSMS,
	})
}

func (h *testPaymentHandler) UpdatePaymentSettings(c *gin.Context) {
	orgID := c.GetUint("organization_id")

	var req service.UpdatePaymentSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.BitPaymentEnabled && req.BitPhoneNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone number is required when Bit payment is enabled"})
		return
	}

	if err := h.svc.UpdateSettingsFunc(c.Request.Context(), orgID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment settings updated"})
}

// -----------------------------------------------------------------------
// Router helper
// -----------------------------------------------------------------------

func newPaymentRouter(h *testPaymentHandler) *gin.Engine {
	r := gin.New()
	r.POST("/jobs/:id/payment-requests", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		c.Set("organization_user_id", uint(10))
		h.SendPaymentRequest(c)
	})
	r.GET("/jobs/:id/payment-notifications", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		h.GetPaymentNotifications(c)
	})
	r.PATCH("/payment-notifications/:id/paid", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		h.MarkPaid(c)
	})
	r.GET("/organization/payment-settings", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		h.GetPaymentSettings(c)
	})
	r.PUT("/organization/payment-settings", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		h.UpdatePaymentSettings(c)
	})
	return r
}

// -----------------------------------------------------------------------
// SendPaymentRequest handler tests
// -----------------------------------------------------------------------

func TestPaymentHandler_SendPaymentRequest_InvalidJobID(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/not-a-number/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "Invalid job ID")
}

func TestPaymentHandler_SendPaymentRequest_SettingsNotConfigured_Returns400(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, _, _ uint, _ *uint) (*models.PaymentNotification, error) {
			return nil, service.ErrPaymentSettingsNotConfigured
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPaymentHandler_SendPaymentRequest_JobNotCompleted_Returns400(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, _, _ uint, _ *uint) (*models.PaymentNotification, error) {
			return nil, service.ErrJobNotCompletedForPayment
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPaymentHandler_SendPaymentRequest_CustomerPhoneMissing_Returns400(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, _, _ uint, _ *uint) (*models.PaymentNotification, error) {
			return nil, service.ErrCustomerPhoneMissing
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPaymentHandler_SendPaymentRequest_JobNotFound_Returns404(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, _, _ uint, _ *uint) (*models.PaymentNotification, error) {
			return nil, service.ErrJobNotFound
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "Job not found")
}

func TestPaymentHandler_SendPaymentRequest_DuplicateActive_Returns409(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, _, _ uint, _ *uint) (*models.PaymentNotification, error) {
			return nil, repository.ErrPaymentRequestAlreadyActive
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}

func TestPaymentHandler_SendPaymentRequest_SMSSendFailed_Returns502WithNotification(t *testing.T) {
	notification := &models.PaymentNotification{ID: 3, PaymentStatus: models.PaymentStatusSendFailed}
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, _, _ uint, _ *uint) (*models.PaymentNotification, error) {
			return notification, errors.New("failed to send payment SMS: twilio down")
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestPaymentHandler_SendPaymentRequest_UnknownErrorNoNotification_Returns500(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, _, _ uint, _ *uint) (*models.PaymentNotification, error) {
			return nil, errors.New("unexpected failure")
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestPaymentHandler_SendPaymentRequest_Success_Returns201(t *testing.T) {
	notification := &models.PaymentNotification{ID: 3, PaymentStatus: models.PaymentStatusSent}
	var capturedOrgID, capturedJobID uint
	var capturedUserID *uint
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		SendPaymentRequestFunc: func(_ context.Context, orgID, jobID uint, userID *uint) (*models.PaymentNotification, error) {
			capturedOrgID, capturedJobID, capturedUserID = orgID, jobID, userID
			return notification, nil
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/jobs/5/payment-requests", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d (body: %s)", w.Code, w.Body.String())
	}
	if capturedOrgID != 1 {
		t.Errorf("expected orgID 1, got %d", capturedOrgID)
	}
	if capturedJobID != 5 {
		t.Errorf("expected jobID 5, got %d", capturedJobID)
	}
	if capturedUserID == nil || *capturedUserID != 10 {
		t.Errorf("expected userID 10, got %v", capturedUserID)
	}
}

// -----------------------------------------------------------------------
// GetPaymentNotifications handler tests
// -----------------------------------------------------------------------

func TestPaymentHandler_GetPaymentNotifications_InvalidJobID(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jobs/abc/payment-notifications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPaymentHandler_GetPaymentNotifications_ServiceError_Returns500(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		GetForJobFunc: func(_ context.Context, _, _ uint) ([]*models.PaymentNotification, error) {
			return nil, errors.New("db error")
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jobs/5/payment-notifications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestPaymentHandler_GetPaymentNotifications_NilResultBecomesEmptyArray(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		GetForJobFunc: func(_ context.Context, _, _ uint) ([]*models.PaymentNotification, error) {
			return nil, nil
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jobs/5/payment-notifications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != "[]" {
		t.Errorf("expected empty JSON array, got %q", w.Body.String())
	}
}

func TestPaymentHandler_GetPaymentNotifications_Success(t *testing.T) {
	expected := []*models.PaymentNotification{{ID: 1}, {ID: 2}}
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		GetForJobFunc: func(_ context.Context, orgID, jobID uint) ([]*models.PaymentNotification, error) {
			if orgID != 1 || jobID != 5 {
				t.Errorf("expected orgID=1 jobID=5, got orgID=%d jobID=%d", orgID, jobID)
			}
			return expected, nil
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/jobs/5/payment-notifications", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

// -----------------------------------------------------------------------
// MarkPaid handler tests
// -----------------------------------------------------------------------

func TestPaymentHandler_MarkPaid_InvalidID(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/payment-notifications/xyz/paid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestPaymentHandler_MarkPaid_NotFound_Returns404(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		MarkPaidFunc: func(_ context.Context, _, _ uint) error {
			return repository.ErrPaymentNotificationNotFound
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/payment-notifications/9/paid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPaymentHandler_MarkPaid_OtherError_Returns500(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		MarkPaidFunc: func(_ context.Context, _, _ uint) error {
			return errors.New("db error")
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/payment-notifications/9/paid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestPaymentHandler_MarkPaid_Success(t *testing.T) {
	var capturedOrgID, capturedID uint
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		MarkPaidFunc: func(_ context.Context, orgID, id uint) error {
			capturedOrgID, capturedID = orgID, id
			return nil
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/payment-notifications/9/paid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedOrgID != 1 || capturedID != 9 {
		t.Errorf("expected orgID=1 id=9, got orgID=%d id=%d", capturedOrgID, capturedID)
	}
}

// -----------------------------------------------------------------------
// GetPaymentSettings handler tests
// -----------------------------------------------------------------------

func TestPaymentHandler_GetPaymentSettings_NotFound_Returns404(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		GetSettingsFunc: func(_ uint) (*models.Organization, error) {
			return nil, errors.New("organization not found")
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organization/payment-settings", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestPaymentHandler_GetPaymentSettings_Success(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		GetSettingsFunc: func(_ uint) (*models.Organization, error) {
			return &models.Organization{
				BitPaymentEnabled:  true,
				BitPhoneNumber:     "050-1234567",
				BitBusinessName:    "Acme",
				AutoSendPaymentSMS: true,
			}, nil
		},
	}}
	r := newPaymentRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/organization/payment-settings", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"bit_payment_enabled":true`) {
		t.Errorf("expected bit_payment_enabled true in body, got %s", w.Body.String())
	}
}

// -----------------------------------------------------------------------
// UpdatePaymentSettings handler tests
// -----------------------------------------------------------------------

func TestPaymentHandler_UpdatePaymentSettings_EnabledWithoutPhone_Returns400(t *testing.T) {
	svcCalled := false
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		UpdateSettingsFunc: func(_ context.Context, _ uint, _ service.UpdatePaymentSettingsRequest) error {
			svcCalled = true
			return nil
		},
	}}
	r := newPaymentRouter(h)

	payload := mustMarshal(map[string]interface{}{
		"bit_payment_enabled": true,
		"bit_phone_number":    "",
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/organization/payment-settings", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if svcCalled {
		t.Error("service should not be called when phone is missing but Bit is enabled")
	}
}

func TestPaymentHandler_UpdatePaymentSettings_Success(t *testing.T) {
	var capturedReq service.UpdatePaymentSettingsRequest
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		UpdateSettingsFunc: func(_ context.Context, orgID uint, req service.UpdatePaymentSettingsRequest) error {
			capturedReq = req
			return nil
		},
	}}
	r := newPaymentRouter(h)

	payload := mustMarshal(map[string]interface{}{
		"bit_payment_enabled":   true,
		"bit_phone_number":      "050-1234567",
		"bit_business_name":     "Acme",
		"auto_send_payment_sms": true,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/organization/payment-settings", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if capturedReq.BitPhoneNumber != "050-1234567" {
		t.Errorf("expected phone '050-1234567', got %q", capturedReq.BitPhoneNumber)
	}
}

func TestPaymentHandler_UpdatePaymentSettings_ServiceError_Returns500(t *testing.T) {
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		UpdateSettingsFunc: func(_ context.Context, _ uint, _ service.UpdatePaymentSettingsRequest) error {
			return errors.New("db write failed")
		},
	}}
	r := newPaymentRouter(h)

	payload := mustMarshal(map[string]interface{}{"bit_payment_enabled": false})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/organization/payment-settings", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestPaymentHandler_UpdatePaymentSettings_DisablingWithoutPhone_Allowed(t *testing.T) {
	// Disabling Bit payment should not require a phone number.
	svcCalled := false
	h := &testPaymentHandler{svc: &mockPaymentSvc{
		UpdateSettingsFunc: func(_ context.Context, _ uint, _ service.UpdatePaymentSettingsRequest) error {
			svcCalled = true
			return nil
		},
	}}
	r := newPaymentRouter(h)

	payload := mustMarshal(map[string]interface{}{"bit_payment_enabled": false, "bit_phone_number": ""})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/organization/payment-settings", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !svcCalled {
		t.Error("expected service to be called when disabling")
	}
}
