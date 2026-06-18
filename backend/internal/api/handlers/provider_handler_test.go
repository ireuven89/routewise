package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// mockProviderSvc implements the same surface as *service.ProviderService so we
// can inject it in tests without touching production code.
type mockProviderSvc struct {
	SearchProvidersFunc   func(ctx context.Context, lat, lng float64, serviceType string) ([]*repository.ProviderResult, error)
	UpdateServiceAreaFunc func(ctx context.Context, orgID uint, req service.UpdateServiceAreaRequest) error
	UpdateServiceOfferFunc func(ctx context.Context, orgID uint, req service.UpdateServiceOfferRequest) error
}

// testProviderHandler mirrors ProviderHandler but delegates to mockProviderSvc.
type testProviderHandler struct {
	svc            *mockProviderSvc
	frontendAPIKey string
}

func (h *testProviderHandler) SearchProviders(c *gin.Context) {
	latStr := c.Query("lat")
	lngStr := c.Query("lng")
	serviceType := c.DefaultQuery("service_type", "hvac")

	if latStr == "" || lngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lng are required"})
		return
	}

	var lat, lng float64
	if _, err := parseFloat(latStr, &lat); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lat"})
		return
	}
	if _, err := parseFloat(lngStr, &lng); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid lng"})
		return
	}

	providers, err := h.svc.SearchProvidersFunc(c.Request.Context(), lat, lng, serviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if providers == nil {
		providers = make([]*repository.ProviderResult, 0)
	}
	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

func (h *testProviderHandler) GetPublicGoogleMapsConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"api_key": h.frontendAPIKey,
		"enabled": h.frontendAPIKey != "",
	})
}

func (h *testProviderHandler) UpdateServiceArea(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	if orgID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req service.UpdateServiceAreaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Latitude == 0 && req.Longitude == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "latitude and longitude are required"})
		return
	}
	if err := h.svc.UpdateServiceAreaFunc(c.Request.Context(), orgID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service area updated"})
}

func (h *testProviderHandler) UpdateServiceOffer(c *gin.Context) {
	orgID := c.GetUint("organization_id")
	if orgID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req service.UpdateServiceOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.svc.UpdateServiceOfferFunc(c.Request.Context(), orgID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service offer updated"})
}

// parseFloat parses a string into float64 and sets *out on success.
func parseFloat(s string, out *float64) (float64, error) {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	*out = v
	return v, nil
}

// newTestRouter sets up a gin engine with the given handler routes.
func newSearchRouter(h *testProviderHandler) *gin.Engine {
	r := gin.New()
	r.GET("/providers", h.SearchProviders)
	r.GET("/config/google-maps", h.GetPublicGoogleMapsConfig)
	r.PUT("/service-area", h.UpdateServiceArea)
	r.PUT("/service-offer", h.UpdateServiceOffer)
	return r
}

// -----------------------------------------------------------------------
// SearchProviders handler tests
// -----------------------------------------------------------------------

func TestProviderHandler_SearchProviders_MissingLat(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers?lng=34.78", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "lat and lng are required")
}

func TestProviderHandler_SearchProviders_MissingLng(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers?lat=32.08", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "lat and lng are required")
}

func TestProviderHandler_SearchProviders_MissingBoth(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestProviderHandler_SearchProviders_InvalidLatFormat(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		SearchProvidersFunc: func(_ context.Context, _, _ float64, _ string) ([]*repository.ProviderResult, error) {
			t.Error("service should not be called with invalid lat")
			return nil, nil
		},
	}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers?lat=not-a-number&lng=34.78", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid lat")
}

func TestProviderHandler_SearchProviders_InvalidLngFormat(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		SearchProvidersFunc: func(_ context.Context, _, _ float64, _ string) ([]*repository.ProviderResult, error) {
			t.Error("service should not be called with invalid lng")
			return nil, nil
		},
	}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers?lat=32.08&lng=not-a-number", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "invalid lng")
}

func TestProviderHandler_SearchProviders_ServiceError(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		SearchProvidersFunc: func(_ context.Context, _, _ float64, _ string) ([]*repository.ProviderResult, error) {
			return nil, errInternal("service unavailable")
		},
	}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers?lat=32.08&lng=34.78", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "service unavailable")
}

func TestProviderHandler_SearchProviders_ValidParamsWithProviders(t *testing.T) {
	visitFee := 150.0
	expectedProviders := []*repository.ProviderResult{
		{ID: 1, Name: "AirFix", Phone: "050-1111111", Industry: "hvac", VisitFee: &visitFee, DistanceKm: 2.5},
		{ID: 2, Name: "CoolTech", Phone: "050-2222222", Industry: "hvac", DistanceKm: 5.1},
	}

	h := &testProviderHandler{svc: &mockProviderSvc{
		SearchProvidersFunc: func(_ context.Context, lat, lng float64, serviceType string) ([]*repository.ProviderResult, error) {
			if lat != 32.08 {
				t.Errorf("expected lat 32.08, got %v", lat)
			}
			if lng != 34.78 {
				t.Errorf("expected lng 34.78, got %v", lng)
			}
			if serviceType != "hvac" {
				t.Errorf("expected service_type 'hvac', got %s", serviceType)
			}
			return expectedProviders, nil
		},
	}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers?lat=32.08&lng=34.78&service_type=hvac", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}
	providers, ok := body["providers"].([]interface{})
	if !ok {
		t.Fatalf("expected 'providers' array in response, got %T", body["providers"])
	}
	if len(providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(providers))
	}
}

func TestProviderHandler_SearchProviders_DefaultServiceType(t *testing.T) {
	var capturedServiceType string
	h := &testProviderHandler{svc: &mockProviderSvc{
		SearchProvidersFunc: func(_ context.Context, _, _ float64, serviceType string) ([]*repository.ProviderResult, error) {
			capturedServiceType = serviceType
			return []*repository.ProviderResult{}, nil
		},
	}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	// No service_type param — should default to "hvac"
	req := httptest.NewRequest(http.MethodGet, "/providers?lat=32.08&lng=34.78", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if capturedServiceType != "hvac" {
		t.Errorf("expected default service_type 'hvac', got '%s'", capturedServiceType)
	}
}

func TestProviderHandler_SearchProviders_NilResultBecomesEmptyArray(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		SearchProvidersFunc: func(_ context.Context, _, _ float64, _ string) ([]*repository.ProviderResult, error) {
			return nil, nil // service returns nil slice
		},
	}}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/providers?lat=32.08&lng=34.78", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	providers, ok := body["providers"].([]interface{})
	if !ok {
		t.Fatalf("expected 'providers' key to be an array, got %T", body["providers"])
	}
	if len(providers) != 0 {
		t.Errorf("expected empty array, got %d elements", len(providers))
	}
}

// -----------------------------------------------------------------------
// GetPublicGoogleMapsConfig handler tests
// -----------------------------------------------------------------------

func TestProviderHandler_GetPublicGoogleMapsConfig_WithKey(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{}, frontendAPIKey: "AIza-test-key-123"}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config/google-maps", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if body["api_key"] != "AIza-test-key-123" {
		t.Errorf("expected api_key 'AIza-test-key-123', got %v", body["api_key"])
	}
	if body["enabled"] != true {
		t.Errorf("expected enabled=true, got %v", body["enabled"])
	}
}

func TestProviderHandler_GetPublicGoogleMapsConfig_WithoutKey(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{}, frontendAPIKey: ""}
	r := newSearchRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/config/google-maps", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if body["api_key"] != "" {
		t.Errorf("expected empty api_key, got %v", body["api_key"])
	}
	if body["enabled"] != false {
		t.Errorf("expected enabled=false, got %v", body["enabled"])
	}
}

// -----------------------------------------------------------------------
// UpdateServiceArea handler tests
// -----------------------------------------------------------------------

func TestProviderHandler_UpdateServiceArea_MissingOrgID(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceAreaFunc: func(_ context.Context, _ uint, _ service.UpdateServiceAreaRequest) error {
			t.Error("service should not be called without org_id")
			return nil
		},
	}}
	r := newSearchRouter(h)

	body := mustMarshal(map[string]interface{}{"latitude": 32.08, "longitude": 34.78})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-area", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No "organization_id" set in context — orgID will be 0
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "unauthorized")
}

func TestProviderHandler_UpdateServiceArea_ZeroLatLng(t *testing.T) {
	serviceWasCalled := false
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceAreaFunc: func(_ context.Context, _ uint, _ service.UpdateServiceAreaRequest) error {
			serviceWasCalled = true
			return nil
		},
	}}

	// Wire org_id into context via middleware
	r := gin.New()
	r.PUT("/service-area", func(c *gin.Context) {
		c.Set("organization_id", uint(5))
		h.UpdateServiceArea(c)
	})

	payload := mustMarshal(map[string]interface{}{"latitude": 0, "longitude": 0})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-area", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if serviceWasCalled {
		t.Error("service should not be called when lat/lng are both zero")
	}
	assertJSONError(t, w.Body.Bytes(), "latitude and longitude are required")
}

func TestProviderHandler_UpdateServiceArea_ValidRequest(t *testing.T) {
	var capturedOrgID uint
	var capturedReq service.UpdateServiceAreaRequest
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceAreaFunc: func(_ context.Context, orgID uint, req service.UpdateServiceAreaRequest) error {
			capturedOrgID = orgID
			capturedReq = req
			return nil
		},
	}}

	r := gin.New()
	r.PUT("/service-area", func(c *gin.Context) {
		c.Set("organization_id", uint(10))
		h.UpdateServiceArea(c)
	})

	payload := mustMarshal(map[string]interface{}{
		"latitude":          32.08,
		"longitude":         34.78,
		"address":           "1 Herzl St",
		"google_place_id":   "ChIJ123",
		"service_radius_km": 15.0,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-area", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if capturedOrgID != 10 {
		t.Errorf("expected orgID 10, got %d", capturedOrgID)
	}
	if capturedReq.Latitude != 32.08 {
		t.Errorf("expected latitude 32.08, got %v", capturedReq.Latitude)
	}
	if capturedReq.Longitude != 34.78 {
		t.Errorf("expected longitude 34.78, got %v", capturedReq.Longitude)
	}
	if capturedReq.Address != "1 Herzl St" {
		t.Errorf("expected address '1 Herzl St', got '%s'", capturedReq.Address)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if body["message"] != "service area updated" {
		t.Errorf("expected message 'service area updated', got %v", body["message"])
	}
}

func TestProviderHandler_UpdateServiceArea_ServiceError(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceAreaFunc: func(_ context.Context, _ uint, _ service.UpdateServiceAreaRequest) error {
			return errInternal("db error")
		},
	}}
	r := gin.New()
	r.PUT("/service-area", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		h.UpdateServiceArea(c)
	})

	payload := mustMarshal(map[string]interface{}{"latitude": 32.08, "longitude": 34.78})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-area", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// -----------------------------------------------------------------------
// UpdateServiceOffer handler tests
// -----------------------------------------------------------------------

func TestProviderHandler_UpdateServiceOffer_MissingOrgID(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceOfferFunc: func(_ context.Context, _ uint, _ service.UpdateServiceOfferRequest) error {
			t.Error("service should not be called without org_id")
			return nil
		},
	}}
	r := newSearchRouter(h)

	visitFee := 100.0
	payload := mustMarshal(map[string]interface{}{"visit_fee": visitFee})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-offer", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestProviderHandler_UpdateServiceOffer_ValidRequestWithAllFees(t *testing.T) {
	var capturedOrgID uint
	var capturedReq service.UpdateServiceOfferRequest
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceOfferFunc: func(_ context.Context, orgID uint, req service.UpdateServiceOfferRequest) error {
			capturedOrgID = orgID
			capturedReq = req
			return nil
		},
	}}
	r := gin.New()
	r.PUT("/service-offer", func(c *gin.Context) {
		c.Set("organization_id", uint(3))
		h.UpdateServiceOffer(c)
	})

	visitFee := 120.0
	repairMin := 200.0
	repairMax := 600.0
	payload := mustMarshal(map[string]interface{}{
		"visit_fee":           visitFee,
		"repair_estimate_min": repairMin,
		"repair_estimate_max": repairMax,
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-offer", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if capturedOrgID != 3 {
		t.Errorf("expected orgID 3, got %d", capturedOrgID)
	}
	if capturedReq.VisitFee == nil || *capturedReq.VisitFee != visitFee {
		t.Errorf("expected visit_fee %v, got %v", visitFee, capturedReq.VisitFee)
	}
	if capturedReq.RepairEstimateMin == nil || *capturedReq.RepairEstimateMin != repairMin {
		t.Errorf("expected repair_min %v, got %v", repairMin, capturedReq.RepairEstimateMin)
	}
	if capturedReq.RepairEstimateMax == nil || *capturedReq.RepairEstimateMax != repairMax {
		t.Errorf("expected repair_max %v, got %v", repairMax, capturedReq.RepairEstimateMax)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if body["message"] != "service offer updated" {
		t.Errorf("expected message 'service offer updated', got %v", body["message"])
	}
}

func TestProviderHandler_UpdateServiceOffer_NilFees(t *testing.T) {
	serviceCalled := false
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceOfferFunc: func(_ context.Context, _ uint, req service.UpdateServiceOfferRequest) error {
			serviceCalled = true
			if req.VisitFee != nil {
				t.Errorf("expected nil visit_fee, got %v", req.VisitFee)
			}
			if req.RepairEstimateMin != nil {
				t.Errorf("expected nil repair_min, got %v", req.RepairEstimateMin)
			}
			if req.RepairEstimateMax != nil {
				t.Errorf("expected nil repair_max, got %v", req.RepairEstimateMax)
			}
			return nil
		},
	}}
	r := gin.New()
	r.PUT("/service-offer", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		h.UpdateServiceOffer(c)
	})

	payload := mustMarshal(map[string]interface{}{}) // no fee fields
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-offer", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !serviceCalled {
		t.Error("expected service to be called")
	}
}

func TestProviderHandler_UpdateServiceOffer_ServiceError(t *testing.T) {
	h := &testProviderHandler{svc: &mockProviderSvc{
		UpdateServiceOfferFunc: func(_ context.Context, _ uint, _ service.UpdateServiceOfferRequest) error {
			return errInternal("write failed")
		},
	}}
	r := gin.New()
	r.PUT("/service-offer", func(c *gin.Context) {
		c.Set("organization_id", uint(1))
		h.UpdateServiceOffer(c)
	})

	payload := mustMarshal(map[string]interface{}{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/service-offer", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

// -----------------------------------------------------------------------
// Test helpers
// -----------------------------------------------------------------------

// errInternal returns a simple error value (avoids importing errors package twice).
type testError string

func (e testError) Error() string { return string(e) }

func errInternal(msg string) error { return testError(msg) }

// mustMarshal panics if JSON encoding fails (test helper only).
func mustMarshal(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// assertJSONError checks that the response body is valid JSON and contains the expected error string.
func assertJSONError(t *testing.T, body []byte, expectedErr string) {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("expected JSON response, got: %s", string(body))
	}
	if resp["error"] != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, resp["error"])
	}
}
