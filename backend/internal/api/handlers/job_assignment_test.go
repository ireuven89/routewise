package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/api/middleware"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/service"
)

// mockJobServiceForAssignment implements service.JobService by embedding the (nil)
// interface, the same approach as mockJobServiceForDashboard: only the methods the
// accept/decline/status/list/get handlers call are overridden. Anything else panics.
type mockJobServiceForAssignment struct {
	service.JobService

	AcceptJobFunc            func(id, orgID, workerID uint) error
	DeclineJobFunc           func(id, orgID, workerID uint, reason string) error
	UpdateStatusFunc         func(id, orgID uint, status string) error
	UpdateStatusAsWorkerFunc func(id, orgID, workerID uint, status string) error
	GetAllFunc               func(orgID uint, filters map[string]interface{}, sortBy string) ([]*models.Job, error)
	GetByIDFunc              func(id, orgID uint) (*models.Job, error)
	GetByIDForWorkerFunc     func(id, orgID, workerID uint) (*models.Job, error)
	AssignTechnicianFunc     func(id, orgID uint, technicianID *uint) error
	CreateFunc               func(input service.CreateJobInput) (*models.Job, error)
}

func (m *mockJobServiceForAssignment) AcceptJob(id, orgID, workerID uint) error {
	return m.AcceptJobFunc(id, orgID, workerID)
}
func (m *mockJobServiceForAssignment) DeclineJob(id, orgID, workerID uint, reason string) error {
	return m.DeclineJobFunc(id, orgID, workerID, reason)
}
func (m *mockJobServiceForAssignment) UpdateStatus(id, orgID uint, status string) error {
	return m.UpdateStatusFunc(id, orgID, status)
}
func (m *mockJobServiceForAssignment) UpdateStatusAsWorker(id, orgID, workerID uint, status string) error {
	return m.UpdateStatusAsWorkerFunc(id, orgID, workerID, status)
}
func (m *mockJobServiceForAssignment) GetAll(orgID uint, filters map[string]interface{}, sortBy string) ([]*models.Job, error) {
	return m.GetAllFunc(orgID, filters, sortBy)
}
func (m *mockJobServiceForAssignment) GetByID(id, orgID uint) (*models.Job, error) {
	return m.GetByIDFunc(id, orgID)
}
func (m *mockJobServiceForAssignment) GetByIDForWorker(id, orgID, workerID uint) (*models.Job, error) {
	return m.GetByIDForWorkerFunc(id, orgID, workerID)
}
func (m *mockJobServiceForAssignment) AssignTechnician(id, orgID uint, technicianID *uint) error {
	return m.AssignTechnicianFunc(id, orgID, technicianID)
}
func (m *mockJobServiceForAssignment) Create(input service.CreateJobInput) (*models.Job, error) {
	return m.CreateFunc(input)
}

// caller describes the token the auth middleware would have decoded.
type caller struct {
	userType string // middleware.UserTypeWorker for technicians, "" / "organization_user" otherwise
	orgID    uint
	userID   uint // organization_user_id claim (the worker ID for technician tokens)
}

var (
	technician = caller{userType: middleware.UserTypeWorker, orgID: 10, userID: 7}
	owner      = caller{userType: "organization_user", orgID: 10, userID: 1}
)

// newJobRouter mounts the job handler behind a stub that sets the same context keys
// AuthMiddleware sets.
func newJobRouter(h *JobHandler, who caller) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("organization_id", who.orgID)
		c.Set("organization_user_id", who.userID)
		c.Set("user_type", who.userType)
		c.Next()
	})
	r.GET("/jobs", h.GetAll)
	r.GET("/jobs/:id", h.GetByID)
	r.POST("/jobs", h.Create)
	r.PATCH("/jobs/:id/assign", h.AssignTechnician)
	r.PATCH("/jobs/:id/status", h.UpdateStatus)
	r.POST("/jobs/:id/accept", h.Accept)
	r.POST("/jobs/:id/decline", h.Decline)
	return r
}

func doRequest(r *gin.Engine, method, path string, body []byte) *httptest.ResponseRecorder {
	var req *http.Request
	if body == nil {
		req = httptest.NewRequest(method, path, nil)
	} else {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body %q: %v", w.Body.String(), err)
	}
	return out
}

// -----------------------------------------------------------------------
// POST /jobs/:id/accept
// -----------------------------------------------------------------------

func TestJobHandler_Accept(t *testing.T) {
	tests := []struct {
		name       string
		who        caller
		path       string
		svcErr     error
		wantStatus int
		wantCalled bool
		wantMsg    string // "message" on success
		wantErr    string // "error" on failure
	}{
		{name: "technician accepts pending job", who: technician, path: "/jobs/42/accept",
			wantStatus: http.StatusOK, wantCalled: true, wantMsg: "Job accepted"},
		{name: "org user is forbidden", who: owner, path: "/jobs/42/accept",
			wantStatus: http.StatusForbidden, wantErr: "Only technicians can perform this action"},
		{name: "empty user type is forbidden", who: caller{orgID: 10, userID: 7}, path: "/jobs/42/accept",
			wantStatus: http.StatusForbidden},
		{name: "non-numeric job id", who: technician, path: "/jobs/abc/accept",
			wantStatus: http.StatusBadRequest, wantErr: "Invalid job ID"},
		{name: "job not found / not assigned to this technician", who: technician, path: "/jobs/42/accept",
			svcErr: service.ErrJobNotFound, wantStatus: http.StatusNotFound, wantCalled: true, wantErr: "Job not found"},
		{name: "job is not a pending offer", who: technician, path: "/jobs/42/accept",
			svcErr: service.ErrInvalidAssignmentState, wantStatus: http.StatusConflict, wantCalled: true,
			wantErr: service.ErrInvalidAssignmentState.Error()},
		{name: "wrapped invalid state still maps to 409", who: technician, path: "/jobs/42/accept",
			svcErr: errors.Join(errors.New("ctx"), service.ErrInvalidAssignmentState), wantStatus: http.StatusConflict, wantCalled: true},
		{name: "unexpected error", who: technician, path: "/jobs/42/accept",
			svcErr: errors.New("db down"), wantStatus: http.StatusInternalServerError, wantCalled: true,
			wantErr: "Failed to update job"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			var gotID, gotOrg, gotWorker uint
			svc := &mockJobServiceForAssignment{
				AcceptJobFunc: func(id, orgID, workerID uint) error {
					called = true
					gotID, gotOrg, gotWorker = id, orgID, workerID
					return tt.svcErr
				},
			}
			w := doRequest(newJobRouter(NewJobHandler(svc), tt.who), http.MethodPost, tt.path, nil)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("service called = %v, want %v", called, tt.wantCalled)
			}
			if called && (gotID != 42 || gotOrg != tt.who.orgID || gotWorker != tt.who.userID) {
				t.Errorf("AcceptJob(%d, %d, %d), want (42, %d, %d)", gotID, gotOrg, gotWorker, tt.who.orgID, tt.who.userID)
			}
			body := decodeBody(t, w)
			if tt.wantMsg != "" && body["message"] != tt.wantMsg {
				t.Errorf("message = %v, want %q", body["message"], tt.wantMsg)
			}
			if tt.wantErr != "" && body["error"] != tt.wantErr {
				t.Errorf("error = %v, want %q", body["error"], tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------
// POST /jobs/:id/decline
// -----------------------------------------------------------------------

func TestJobHandler_Decline(t *testing.T) {
	tests := []struct {
		name       string
		who        caller
		path       string
		body       []byte
		svcErr     error
		wantStatus int
		wantCalled bool
		wantReason string
	}{
		{name: "decline with reason", who: technician, path: "/jobs/5/decline",
			body: []byte(`{"reason":"Too far away"}`), wantStatus: http.StatusOK, wantCalled: true, wantReason: "Too far away"},
		{name: "decline without body", who: technician, path: "/jobs/5/decline",
			wantStatus: http.StatusOK, wantCalled: true, wantReason: ""},
		{name: "decline with empty object", who: technician, path: "/jobs/5/decline",
			body: []byte(`{}`), wantStatus: http.StatusOK, wantCalled: true, wantReason: ""},
		{name: "malformed body is ignored (reason optional)", who: technician, path: "/jobs/5/decline",
			body: []byte(`{not json`), wantStatus: http.StatusOK, wantCalled: true, wantReason: ""},
		{name: "org user is forbidden", who: owner, path: "/jobs/5/decline",
			body: []byte(`{"reason":"x"}`), wantStatus: http.StatusForbidden},
		{name: "invalid job id", who: technician, path: "/jobs/-1/decline",
			wantStatus: http.StatusBadRequest},
		{name: "not found", who: technician, path: "/jobs/5/decline",
			svcErr: service.ErrJobNotFound, wantStatus: http.StatusNotFound, wantCalled: true},
		{name: "job already started", who: technician, path: "/jobs/5/decline",
			svcErr: service.ErrInvalidAssignmentState, wantStatus: http.StatusConflict, wantCalled: true},
		{name: "unexpected error", who: technician, path: "/jobs/5/decline",
			svcErr: errors.New("boom"), wantStatus: http.StatusInternalServerError, wantCalled: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var called bool
			var gotID, gotOrg, gotWorker uint
			var gotReason string
			svc := &mockJobServiceForAssignment{
				DeclineJobFunc: func(id, orgID, workerID uint, reason string) error {
					called = true
					gotID, gotOrg, gotWorker, gotReason = id, orgID, workerID, reason
					return tt.svcErr
				},
			}
			w := doRequest(newJobRouter(NewJobHandler(svc), tt.who), http.MethodPost, tt.path, tt.body)

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if called != tt.wantCalled {
				t.Fatalf("service called = %v, want %v", called, tt.wantCalled)
			}
			if !called {
				return
			}
			if gotID != 5 || gotOrg != tt.who.orgID || gotWorker != tt.who.userID {
				t.Errorf("DeclineJob(%d, %d, %d, ...), want (5, %d, %d, ...)", gotID, gotOrg, gotWorker, tt.who.orgID, tt.who.userID)
			}
			if gotReason != tt.wantReason {
				t.Errorf("reason = %q, want %q", gotReason, tt.wantReason)
			}
			if tt.wantStatus == http.StatusOK && decodeBody(t, w)["message"] != "Job declined" {
				t.Errorf("unexpected success body %s", w.Body.String())
			}
		})
	}
}

// -----------------------------------------------------------------------
// PATCH /jobs/:id/status
// -----------------------------------------------------------------------

func TestJobHandler_UpdateStatus(t *testing.T) {
	tests := []struct {
		name          string
		who           caller
		path          string
		body          string
		svcErr        error
		wantStatus    int
		wantWorkerSvc bool // UpdateStatusAsWorker called
		wantOwnerSvc  bool // UpdateStatus called
		wantErr       string
	}{
		{name: "technician starts accepted job", who: technician, path: "/jobs/3/status",
			body: `{"status":"in_progress"}`, wantStatus: http.StatusOK, wantWorkerSvc: true},
		{name: "technician must accept first", who: technician, path: "/jobs/3/status",
			body: `{"status":"in_progress"}`, svcErr: service.ErrMustAcceptFirst,
			wantStatus: http.StatusConflict, wantWorkerSvc: true, wantErr: "Accept the job first"},
		{name: "technician reopens a completed job", who: technician, path: "/jobs/3/status",
			body: `{"status":"in_progress"}`, svcErr: service.ErrInvalidTransition,
			wantStatus: http.StatusConflict, wantWorkerSvc: true, wantErr: "Job status does not allow this change"},
		{name: "technician sets a status they may not set", who: technician, path: "/jobs/3/status",
			body: `{"status":"cancelled"}`, svcErr: service.ErrInvalidStatus,
			wantStatus: http.StatusBadRequest, wantWorkerSvc: true, wantErr: "Invalid status"},
		{name: "technician touches someone else's job", who: technician, path: "/jobs/3/status",
			body: `{"status":"completed"}`, svcErr: service.ErrJobNotFound,
			wantStatus: http.StatusNotFound, wantWorkerSvc: true, wantErr: "Job not found"},
		{name: "technician unexpected error", who: technician, path: "/jobs/3/status",
			body: `{"status":"completed"}`, svcErr: errors.New("db"),
			wantStatus: http.StatusInternalServerError, wantWorkerSvc: true, wantErr: "Failed to update status"},
		{name: "owner overrides any status", who: owner, path: "/jobs/3/status",
			body: `{"status":"cancelled"}`, wantStatus: http.StatusOK, wantOwnerSvc: true},
		{name: "owner invalid status", who: owner, path: "/jobs/3/status",
			body: `{"status":"bogus"}`, svcErr: service.ErrInvalidStatus,
			wantStatus: http.StatusBadRequest, wantOwnerSvc: true},
		{name: "missing status field", who: technician, path: "/jobs/3/status",
			body: `{}`, wantStatus: http.StatusBadRequest},
		{name: "invalid job id", who: owner, path: "/jobs/x/status",
			body: `{"status":"completed"}`, wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var workerCalled, ownerCalled bool
			var gotWorker uint
			var gotStatus string
			svc := &mockJobServiceForAssignment{
				UpdateStatusAsWorkerFunc: func(id, orgID, workerID uint, status string) error {
					workerCalled = true
					gotWorker, gotStatus = workerID, status
					if id != 3 || orgID != tt.who.orgID {
						t.Errorf("UpdateStatusAsWorker(id=%d, org=%d)", id, orgID)
					}
					return tt.svcErr
				},
				UpdateStatusFunc: func(id, orgID uint, status string) error {
					ownerCalled = true
					gotStatus = status
					if id != 3 || orgID != tt.who.orgID {
						t.Errorf("UpdateStatus(id=%d, org=%d)", id, orgID)
					}
					return tt.svcErr
				},
			}
			w := doRequest(newJobRouter(NewJobHandler(svc), tt.who), http.MethodPatch, tt.path, []byte(tt.body))

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if workerCalled != tt.wantWorkerSvc || ownerCalled != tt.wantOwnerSvc {
				t.Fatalf("worker svc called=%v (want %v), owner svc called=%v (want %v)",
					workerCalled, tt.wantWorkerSvc, ownerCalled, tt.wantOwnerSvc)
			}
			if workerCalled && gotWorker != tt.who.userID {
				t.Errorf("workerID = %d, want %d", gotWorker, tt.who.userID)
			}
			if (workerCalled || ownerCalled) && gotStatus == "" {
				t.Error("status not forwarded to service")
			}
			if tt.wantErr != "" {
				if got := decodeBody(t, w)["error"]; got != tt.wantErr {
					t.Errorf("error = %v, want %q", got, tt.wantErr)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------
// GET /jobs and GET /jobs/:id are scoped to the technician's own jobs
// -----------------------------------------------------------------------

func TestJobHandler_GetAll_TechnicianFilterOverride(t *testing.T) {
	tests := []struct {
		name         string
		who          caller
		query        string
		wantTechID   interface{} // nil = no technician_id filter
		wantHasTechF bool
	}{
		{name: "technician sees only own jobs", who: technician, query: "",
			wantTechID: uint(7), wantHasTechF: true},
		{name: "technician cannot ask for another technician's jobs", who: technician, query: "?technician_id=99",
			wantTechID: uint(7), wantHasTechF: true},
		{name: "owner gets no technician filter", who: owner, query: "",
			wantHasTechF: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotFilters map[string]interface{}
			svc := &mockJobServiceForAssignment{
				GetAllFunc: func(orgID uint, filters map[string]interface{}, sortBy string) ([]*models.Job, error) {
					gotFilters = filters
					return []*models.Job{}, nil
				},
			}
			w := doRequest(newJobRouter(NewJobHandler(svc), tt.who), http.MethodGet, "/jobs"+tt.query, nil)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d (body %s)", w.Code, w.Body.String())
			}
			v, ok := gotFilters["technician_id"]
			if ok != tt.wantHasTechF {
				t.Fatalf("technician_id filter present = %v, want %v (filters %v)", ok, tt.wantHasTechF, gotFilters)
			}
			if ok && v != tt.wantTechID {
				t.Errorf("technician_id filter = %#v, want %#v", v, tt.wantTechID)
			}
		})
	}
}

func TestJobHandler_GetByID_RoutesByCaller(t *testing.T) {
	tests := []struct {
		name          string
		who           caller
		svcErr        error
		wantStatus    int
		wantForWorker bool
	}{
		{name: "technician uses worker-scoped lookup", who: technician, wantStatus: http.StatusOK, wantForWorker: true},
		{name: "technician gets 404 for job not assigned to them", who: technician,
			svcErr: service.ErrJobNotFound, wantStatus: http.StatusNotFound, wantForWorker: true},
		{name: "owner uses org-wide lookup", who: owner, wantStatus: http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var usedWorker, usedOwner bool
			svc := &mockJobServiceForAssignment{
				GetByIDForWorkerFunc: func(id, orgID, workerID uint) (*models.Job, error) {
					usedWorker = true
					if workerID != tt.who.userID {
						t.Errorf("workerID = %d, want %d", workerID, tt.who.userID)
					}
					if tt.svcErr != nil {
						return nil, tt.svcErr
					}
					return &models.Job{ID: id}, nil
				},
				GetByIDFunc: func(id, orgID uint) (*models.Job, error) {
					usedOwner = true
					return &models.Job{ID: id}, tt.svcErr
				},
			}
			w := doRequest(newJobRouter(NewJobHandler(svc), tt.who), http.MethodGet, "/jobs/9", nil)
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			if usedWorker != tt.wantForWorker || usedOwner == tt.wantForWorker {
				t.Errorf("worker lookup=%v owner lookup=%v, want worker=%v", usedWorker, usedOwner, tt.wantForWorker)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Assign / Create map ErrInvalidTechnician to 400
// -----------------------------------------------------------------------

func TestJobHandler_AssignTechnician_ErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		svcErr     error
		wantStatus int
		wantErr    string
	}{
		{name: "assigned", wantStatus: http.StatusOK},
		{name: "technician from another org or inactive", svcErr: service.ErrInvalidTechnician,
			wantStatus: http.StatusBadRequest, wantErr: "Technician not found or inactive"},
		{name: "unexpected error", svcErr: errors.New("db"),
			wantStatus: http.StatusInternalServerError, wantErr: "Failed to assign technician"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotTech *uint
			svc := &mockJobServiceForAssignment{
				AssignTechnicianFunc: func(id, orgID uint, technicianID *uint) error {
					gotTech = technicianID
					return tt.svcErr
				},
			}
			w := doRequest(newJobRouter(NewJobHandler(svc), owner), http.MethodPatch, "/jobs/4/assign",
				[]byte(`{"technician_id":12}`))
			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if gotTech == nil || *gotTech != 12 {
				t.Errorf("technician_id forwarded = %v, want 12", gotTech)
			}
			if tt.wantErr != "" && decodeBody(t, w)["error"] != tt.wantErr {
				t.Errorf("error = %s, want %q", w.Body.String(), tt.wantErr)
			}
		})
	}
}

func TestJobHandler_Create_InvalidTechnicianIs400(t *testing.T) {
	svc := &mockJobServiceForAssignment{
		CreateFunc: func(input service.CreateJobInput) (*models.Job, error) {
			return nil, service.ErrInvalidTechnician
		},
	}
	body := []byte(`{"customer_id":1,"technician_id":99,"title":"Fix AC","scheduled_at":"2026-10-01T09:00:00Z"}`)
	w := doRequest(newJobRouter(NewJobHandler(svc), owner), http.MethodPost, "/jobs", body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (body %s)", w.Code, w.Body.String())
	}
	if got := decodeBody(t, w)["error"]; got != "Technician not found or inactive" {
		t.Errorf("error = %v", got)
	}
}
