package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/service"
)

// mockJobServiceForDashboard implements service.JobService by embedding the
// (nil) interface and overriding only GetDashboardStats - the only method
// DashboardHandler.GetStats calls. Any other method invoked by mistake will
// panic on the nil embedded interface, which is fine: these tests never
// exercise those paths.
type mockJobServiceForDashboard struct {
	service.JobService
	GetDashboardStatsFunc func(organizationID uint) (*models.DashboardStats, error)
}

func (m *mockJobServiceForDashboard) GetDashboardStats(organizationID uint) (*models.DashboardStats, error) {
	return m.GetDashboardStatsFunc(organizationID)
}

func newDashboardRouter(h *DashboardHandler, organizationID uint, setOrgID bool) *gin.Engine {
	r := gin.New()
	r.GET("/dashboard/stats", func(c *gin.Context) {
		if setOrgID {
			c.Set("organization_id", organizationID)
		}
		h.GetStats(c)
	})
	return r
}

func TestDashboardHandler_GetStats_Success(t *testing.T) {
	expectedStats := &models.DashboardStats{
		Revenue: &models.RevenueStats{
			Total:       10000.0,
			ThisMonth:   2200.0,
			ThisWeek:    500.0,
			AvgJobValue: 333.33,
			RevenueByMonth: []models.MonthlyRevenue{
				{Month: "2026-06", Revenue: 1000.0},
				{Month: "2026-07", Revenue: 2200.0},
			},
		},
	}

	var capturedOrgID uint
	mockSvc := &mockJobServiceForDashboard{
		GetDashboardStatsFunc: func(organizationID uint) (*models.DashboardStats, error) {
			capturedOrgID = organizationID
			return expectedStats, nil
		},
	}

	h := NewDashboardHandler(mockSvc)
	r := newDashboardRouter(h, 42, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if capturedOrgID != 42 {
		t.Errorf("expected organization_id 42 passed to service, got %d", capturedOrgID)
	}

	var body models.DashboardStats
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to parse JSON response: %v", err)
	}
	if body.Revenue == nil {
		t.Fatal("expected revenue stats in response body")
	}
	if body.Revenue.Total != 10000.0 {
		t.Errorf("expected total 10000.0, got %v", body.Revenue.Total)
	}
	if body.Revenue.ThisMonth != 2200.0 {
		t.Errorf("expected this_month 2200.0, got %v", body.Revenue.ThisMonth)
	}
	if len(body.Revenue.RevenueByMonth) != 2 {
		t.Errorf("expected 2 months of revenue breakdown, got %d", len(body.Revenue.RevenueByMonth))
	}
}

func TestDashboardHandler_GetStats_ServiceError(t *testing.T) {
	mockSvc := &mockJobServiceForDashboard{
		GetDashboardStatsFunc: func(organizationID uint) (*models.DashboardStats, error) {
			return nil, errInternal("db unavailable")
		},
	}

	h := NewDashboardHandler(mockSvc)
	r := newDashboardRouter(h, 1, true)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	assertJSONError(t, w.Body.Bytes(), "db unavailable")
}

func TestDashboardHandler_GetStats_NoOrganizationIDInContext(t *testing.T) {
	// organization_id defaults to the zero value when absent from context;
	// GetStats has no guard for this, so it should still call through to
	// the service with organizationID=0.
	var capturedOrgID uint
	orgIDCaptured := false
	mockSvc := &mockJobServiceForDashboard{
		GetDashboardStatsFunc: func(organizationID uint) (*models.DashboardStats, error) {
			capturedOrgID = organizationID
			orgIDCaptured = true
			return &models.DashboardStats{Revenue: &models.RevenueStats{}}, nil
		},
	}

	h := NewDashboardHandler(mockSvc)
	r := newDashboardRouter(h, 0, false)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/dashboard/stats", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}
	if !orgIDCaptured {
		t.Fatal("expected service to be called")
	}
	if capturedOrgID != 0 {
		t.Errorf("expected organizationID 0 when not set in context, got %d", capturedOrgID)
	}
}
