package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// runGuard sends a request through guard with the given user_type (unset when setType is
// false) and reports the response and whether the downstream handler ran.
func runGuard(t *testing.T, guard gin.HandlerFunc, userType string, setType bool) (*httptest.ResponseRecorder, bool) {
	t.Helper()
	reached := false
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		if setType {
			c.Set("user_type", userType)
		}
		c.Next()
	}, guard, func(c *gin.Context) {
		reached = true
		c.Status(http.StatusOK)
	})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	return w, reached
}

func TestRequireOrgUser(t *testing.T) {
	tests := []struct {
		name        string
		userType    string
		setType     bool
		wantStatus  int
		wantReached bool
	}{
		{name: "technician token is blocked", userType: UserTypeWorker, setType: true, wantStatus: http.StatusForbidden},
		{name: "organization user passes", userType: "organization_user", setType: true, wantStatus: http.StatusOK, wantReached: true},
		{name: "empty user type passes (legacy tokens)", userType: "", setType: true, wantStatus: http.StatusOK, wantReached: true},
		{name: "no user_type in context passes", setType: false, wantStatus: http.StatusOK, wantReached: true},
		{name: "user type match is case-sensitive", userType: "Worker", setType: true, wantStatus: http.StatusOK, wantReached: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, reached := runGuard(t, RequireOrgUser(), tt.userType, tt.setType)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if reached != tt.wantReached {
				t.Fatalf("handler reached = %v, want %v", reached, tt.wantReached)
			}
			if !tt.wantReached {
				var body map[string]string
				_ = json.Unmarshal(w.Body.Bytes(), &body)
				if body["error"] != "Not available for technicians" {
					t.Errorf("error body = %q", w.Body.String())
				}
			}
		})
	}
}

func TestRequireWorker(t *testing.T) {
	tests := []struct {
		name        string
		userType    string
		setType     bool
		wantStatus  int
		wantReached bool
	}{
		{name: "technician token passes", userType: UserTypeWorker, setType: true, wantStatus: http.StatusOK, wantReached: true},
		{name: "organization user is blocked", userType: "organization_user", setType: true, wantStatus: http.StatusForbidden},
		{name: "empty user type is blocked", userType: "", setType: true, wantStatus: http.StatusForbidden},
		{name: "no user_type in context is blocked", setType: false, wantStatus: http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, reached := runGuard(t, RequireWorker(), tt.userType, tt.setType)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if reached != tt.wantReached {
				t.Fatalf("handler reached = %v, want %v", reached, tt.wantReached)
			}
			if !tt.wantReached {
				var body map[string]string
				_ = json.Unmarshal(w.Body.Bytes(), &body)
				if body["error"] != "Only technicians can perform this action" {
					t.Errorf("error body = %q", w.Body.String())
				}
			}
		})
	}
}

func TestUserTypeWorkerConstant(t *testing.T) {
	// The mobile OTP login issues tokens with user_type "worker"; the guards depend on it.
	if UserTypeWorker != "worker" {
		t.Fatalf("UserTypeWorker = %q, want \"worker\"", UserTypeWorker)
	}
}
