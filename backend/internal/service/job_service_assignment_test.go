package service

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

// These tests drive the real JobSvc. Unlike the other *_service_test.go files (which mirror
// the service logic in a test wrapper), the accept/decline rules live in JobSvc itself, so
// mirroring would test a copy. JobSvc.repo is a concrete *repository.JobRepository, so it is
// backed here by a tiny fake SQL driver: SELECTs from jobs return one configured job row,
// INSERTs return an id, and every Exec is recorded with a configurable RowsAffected / error.
// The worker lookup is an interface (workerLookup) and is mocked directly.

// -----------------------------------------------------------------------
// Fake SQL driver
// -----------------------------------------------------------------------

var jobFakeDBOnce sync.Once

type jobDBCall struct {
	query string
	args  []driver.Value
}

type jobDBState struct {
	mu         sync.Mutex
	jobRow     []driver.Value // nil => the job doesn't exist (sql.ErrNoRows)
	execRows   int64
	execErr    error
	insertedID int64
	execs      []jobDBCall
	queries    []jobDBCall
}

var (
	jobDBMu       sync.Mutex
	jobDBRegistry = map[string]*jobDBState{}
)

type jobFakeDriver struct{}

func (jobFakeDriver) Open(dsn string) (driver.Conn, error) { return &jobFakeConn{dsn: dsn}, nil }

type jobFakeConn struct{ dsn string }

func (c *jobFakeConn) Prepare(q string) (driver.Stmt, error) {
	return &jobFakeStmt{dsn: c.dsn, q: q}, nil
}
func (c *jobFakeConn) Close() error              { return nil }
func (c *jobFakeConn) Begin() (driver.Tx, error) { return nil, errors.New("tx not supported") }

type jobFakeStmt struct{ dsn, q string }

func (s *jobFakeStmt) state() *jobDBState {
	jobDBMu.Lock()
	defer jobDBMu.Unlock()
	return jobDBRegistry[s.dsn]
}
func (s *jobFakeStmt) Close() error  { return nil }
func (s *jobFakeStmt) NumInput() int { return -1 }
func (s *jobFakeStmt) Exec(args []driver.Value) (driver.Result, error) {
	st := s.state()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.execs = append(st.execs, jobDBCall{s.q, args})
	if st.execErr != nil {
		return nil, st.execErr
	}
	return driver.RowsAffected(st.execRows), nil
}
func (s *jobFakeStmt) Query(args []driver.Value) (driver.Rows, error) {
	st := s.state()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.queries = append(st.queries, jobDBCall{s.q, args})
	switch {
	case strings.Contains(s.q, "INSERT INTO jobs"):
		return &jobFakeRows{cols: []string{"id"}, data: [][]driver.Value{{st.insertedID}}}, nil
	case strings.Contains(s.q, "FROM jobs j"):
		r := &jobFakeRows{cols: jobRowCols}
		if st.jobRow != nil {
			r.data = [][]driver.Value{st.jobRow}
		}
		return r, nil
	}
	return nil, fmt.Errorf("unexpected query: %s", s.q)
}

type jobFakeRows struct {
	cols []string
	data [][]driver.Value
	i    int
}

func (r *jobFakeRows) Columns() []string { return r.cols }
func (r *jobFakeRows) Close() error      { return nil }
func (r *jobFakeRows) Next(dest []driver.Value) error {
	if r.i >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.i])
	r.i++
	return nil
}

// jobRowCols matches the column count of repository.jobSelect.
var jobRowCols = make([]string, 32)

// jobState describes the stored job the fake DB returns.
type jobState struct {
	technicianID *uint
	status       models.JobStatus
	assignment   *models.AssignmentStatus
}

func (j jobState) row(id, orgID uint) []driver.Value {
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	var tech, assign, workerID driver.Value
	if j.technicianID != nil {
		tech = int64(*j.technicianID)
		workerID = tech
	}
	if j.assignment != nil {
		assign = string(*j.assignment)
	}
	status := j.status
	if status == "" {
		status = models.StatusScheduled
	}
	return []driver.Value{
		int64(id), int64(orgID), nil, int64(3), tech, "Fix AC",
		"", string(status), now, nil, int64(60),
		nil, nil, now, now,
		assign, nil, nil, "", nil,
		int64(3), "Dana", "050", "", "Herzl 1", nil, nil, "",
		workerID, "Avi", "052",
		"",
	}
}

// mockWorkerLookup implements workerLookup.
type mockWorkerLookup struct {
	FindByIDFunc func(id, orgID uint) (*models.Worker, error)
	calls        int
}

func (m *mockWorkerLookup) FindByID(id, orgID uint) (*models.Worker, error) {
	m.calls++
	return m.FindByIDFunc(id, orgID)
}

func activeWorkerLookup() *mockWorkerLookup {
	return &mockWorkerLookup{FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
		return &models.Worker{ID: id, OrganizationID: orgID, IsActive: true}, nil
	}}
}

var dsnSeq int64
var dsnSeqMu sync.Mutex

// newJobSvcWithFakeDB builds a real JobSvc over the fake driver.
func newJobSvcWithFakeDB(t *testing.T, st *jobDBState, workers workerLookup) *JobSvc {
	t.Helper()
	jobFakeDBOnce.Do(func() { sql.Register("jobfakedb", jobFakeDriver{}) })
	dsnSeqMu.Lock()
	dsnSeq++
	dsn := fmt.Sprintf("jobsvc-%d", dsnSeq)
	dsnSeqMu.Unlock()

	jobDBMu.Lock()
	jobDBRegistry[dsn] = st
	jobDBMu.Unlock()
	t.Cleanup(func() {
		jobDBMu.Lock()
		delete(jobDBRegistry, dsn)
		jobDBMu.Unlock()
	})
	db, err := sql.Open("jobfakedb", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if workers == nil {
		workers = activeWorkerLookup()
	}
	return NewJobService(repository.NewJobRepository(db, nil), workers)
}

func uintPtr(v uint) *uint                                             { return &v }
func assignmentPtr(s models.AssignmentStatus) *models.AssignmentStatus { return &s }

const (
	testOrgID    uint = 10
	testWorkerID uint = 7
	otherWorker  uint = 8
	testJobID    uint = 42
)

var (
	pending  = assignmentPtr(models.AssignmentPending)
	accepted = assignmentPtr(models.AssignmentAccepted)
)

// -----------------------------------------------------------------------
// AcceptJob
// -----------------------------------------------------------------------

func TestJobSvc_AcceptJob(t *testing.T) {
	tests := []struct {
		name       string
		job        *jobState // nil => job doesn't exist
		execRows   int64
		execErr    error
		wantErr    error // sentinel checked with errors.Is; nil = success
		wantAnyErr bool
		wantExec   bool
	}{
		{name: "pending offer to this worker is accepted",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, execRows: 1, wantExec: true},
		{name: "job does not exist",
			job: nil, wantErr: ErrJobNotFound},
		{name: "job assigned to another worker looks like not found",
			job: &jobState{technicianID: uintPtr(otherWorker), assignment: pending}, wantErr: ErrJobNotFound},
		{name: "unassigned job looks like not found",
			job: &jobState{}, wantErr: ErrJobNotFound},
		{name: "already accepted",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, wantErr: ErrInvalidAssignmentState},
		{name: "assigned but no assignment status",
			job: &jobState{technicianID: uintPtr(testWorkerID)}, wantErr: ErrInvalidAssignmentState},
		{name: "pending offer on a cancelled job",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending, status: models.StatusCancelled}, wantErr: ErrInvalidAssignmentState},
		{name: "pending offer on a completed job",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending, status: models.StatusCompleted}, wantErr: ErrInvalidAssignmentState},
		{name: "pending offer on an in-progress job is accepted",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending, status: models.StatusInProgress}, execRows: 1, wantExec: true},
		{name: "state changed between read and update (0 rows)",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, execRows: 0, wantExec: true, wantErr: ErrInvalidAssignmentState},
		{name: "database error propagates",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, execErr: errors.New("db down"), wantExec: true, wantAnyErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &jobDBState{execRows: tt.execRows, execErr: tt.execErr}
			if tt.job != nil {
				st.jobRow = tt.job.row(testJobID, testOrgID)
			}
			svc := newJobSvcWithFakeDB(t, st, nil)

			err := svc.AcceptJob(testJobID, testOrgID, testWorkerID)

			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			case tt.wantAnyErr:
				if err == nil || errors.Is(err, ErrInvalidAssignmentState) {
					t.Fatalf("err = %v, want raw db error", err)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
			}
			if got := len(st.execs) > 0; got != tt.wantExec {
				t.Fatalf("update executed = %v, want %v", got, tt.wantExec)
			}
			if tt.wantExec {
				e := st.execs[0]
				if !strings.Contains(e.query, "assignment_status = 'accepted'") {
					t.Errorf("unexpected update: %s", e.query)
				}
				if e.args[1] != int64(testJobID) || e.args[2] != int64(testOrgID) || e.args[3] != int64(testWorkerID) {
					t.Errorf("update args = %v", e.args)
				}
			}
			// the lookup is always scoped to the organization
			if len(st.queries) == 0 || st.queries[0].args[1] != int64(testOrgID) {
				t.Errorf("job lookup not scoped to org: %+v", st.queries)
			}
		})
	}
}

// -----------------------------------------------------------------------
// DeclineJob
// -----------------------------------------------------------------------

func TestJobSvc_DeclineJob(t *testing.T) {
	tests := []struct {
		name       string
		job        *jobState
		reason     string
		execRows   int64
		execErr    error
		wantErr    error
		wantAnyErr bool
		wantExec   bool
	}{
		{name: "decline pending offer with reason",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, reason: "Too far", execRows: 1, wantExec: true},
		{name: "decline pending offer without reason",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, reason: "", execRows: 1, wantExec: true},
		{name: "decline after accepting (still scheduled) is allowed",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, reason: "Sick", execRows: 1, wantExec: true},
		{name: "job does not exist",
			job: nil, wantErr: ErrJobNotFound},
		{name: "job assigned to another worker",
			job: &jobState{technicianID: uintPtr(otherWorker), assignment: pending}, wantErr: ErrJobNotFound},
		{name: "job already in progress",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusInProgress}, wantErr: ErrInvalidAssignmentState},
		{name: "job completed",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusCompleted}, wantErr: ErrInvalidAssignmentState},
		{name: "job cancelled",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending, status: models.StatusCancelled}, wantErr: ErrInvalidAssignmentState},
		{name: "reassigned between read and update (0 rows)",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, execRows: 0, wantExec: true, wantErr: ErrInvalidAssignmentState},
		{name: "database error propagates",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, execErr: errors.New("db"), wantExec: true, wantAnyErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &jobDBState{execRows: tt.execRows, execErr: tt.execErr}
			if tt.job != nil {
				st.jobRow = tt.job.row(testJobID, testOrgID)
			}
			svc := newJobSvcWithFakeDB(t, st, nil)

			err := svc.DeclineJob(testJobID, testOrgID, testWorkerID, tt.reason)

			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			case tt.wantAnyErr:
				if err == nil {
					t.Fatal("expected db error")
				}
			default:
				if err != nil {
					t.Fatalf("unexpected err: %v", err)
				}
			}
			if got := len(st.execs) > 0; got != tt.wantExec {
				t.Fatalf("update executed = %v, want %v", got, tt.wantExec)
			}
			if tt.wantExec {
				e := st.execs[0]
				if !strings.Contains(e.query, "technician_id = NULL") {
					t.Errorf("unexpected update: %s", e.query)
				}
				if e.args[0] != int64(testWorkerID) || e.args[1] != tt.reason ||
					e.args[3] != int64(testJobID) || e.args[4] != int64(testOrgID) {
					t.Errorf("update args = %v", e.args)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------
// UpdateStatusAsWorker
// -----------------------------------------------------------------------

func TestJobSvc_UpdateStatusAsWorker(t *testing.T) {
	tests := []struct {
		name      string
		job       *jobState
		status    string
		execRows  *int64 // nil => 1
		wantErr   error
		wantExec  bool
		wantInSQL string
		wantFrom  models.JobStatus
	}{
		{name: "start accepted job",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, status: "in_progress",
			wantExec: true, wantInSQL: "SET status = $1, updated_at = $2", wantFrom: models.StatusScheduled},
		{name: "complete accepted job sets completed_at",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusInProgress}, status: "completed",
			wantExec: true, wantInSQL: "completed_at = COALESCE($7, completed_at)", wantFrom: models.StatusInProgress},
		{name: "cannot complete a job that was never started",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, status: "completed", wantErr: ErrInvalidTransition},
		{name: "cannot start a job twice",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusInProgress}, status: "in_progress", wantErr: ErrInvalidTransition},
		{name: "cannot reopen a completed job",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusCompleted}, status: "in_progress", wantErr: ErrInvalidTransition},
		{name: "cannot complete a completed job again",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusCompleted}, status: "completed", wantErr: ErrInvalidTransition},
		{name: "cannot start a cancelled job",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusCancelled}, status: "in_progress", wantErr: ErrInvalidTransition},
		{name: "cannot complete a cancelled job",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted, status: models.StatusCancelled}, status: "completed", wantErr: ErrInvalidTransition},
		{name: "status changed between read and update (0 rows), e.g. owner cancelled",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, status: "in_progress", execRows: new(int64),
			wantExec: true, wantInSQL: "AND status = $6", wantFrom: models.StatusScheduled, wantErr: ErrInvalidTransition},
		{name: "pending offer must be accepted first",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}, status: "in_progress", wantErr: ErrMustAcceptFirst},
		{name: "no assignment status must be accepted first",
			job: &jobState{technicianID: uintPtr(testWorkerID)}, status: "completed", wantErr: ErrMustAcceptFirst},
		{name: "technician cannot cancel",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, status: "cancelled", wantErr: ErrInvalidStatus},
		{name: "technician cannot move back to scheduled",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, status: "scheduled", wantErr: ErrInvalidStatus},
		{name: "unknown status",
			job: &jobState{technicianID: uintPtr(testWorkerID), assignment: accepted}, status: "done", wantErr: ErrInvalidStatus},
		{name: "someone else's job",
			job: &jobState{technicianID: uintPtr(otherWorker), assignment: accepted}, status: "in_progress", wantErr: ErrJobNotFound},
		{name: "missing job",
			job: nil, status: "in_progress", wantErr: ErrJobNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &jobDBState{execRows: 1}
			if tt.execRows != nil {
				st.execRows = *tt.execRows
			}
			if tt.job != nil {
				st.jobRow = tt.job.row(testJobID, testOrgID)
			}
			svc := newJobSvcWithFakeDB(t, st, nil)

			err := svc.UpdateStatusAsWorker(testJobID, testOrgID, testWorkerID, tt.status)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got := len(st.execs) > 0; got != tt.wantExec {
				t.Fatalf("update executed = %v, want %v", got, tt.wantExec)
			}
			if tt.wantExec {
				e := st.execs[0]
				if !strings.Contains(e.query, tt.wantInSQL) {
					t.Errorf("query %q does not contain %q", e.query, tt.wantInSQL)
				}
				if e.args[0] != tt.status {
					t.Errorf("status arg = %v, want %s", e.args[0], tt.status)
				}
				// the update is guarded by job, org, worker and the expected current status
				if e.args[2] != int64(testJobID) || e.args[3] != int64(testOrgID) ||
					e.args[4] != int64(testWorkerID) || e.args[5] != string(tt.wantFrom) {
					t.Errorf("update args = %v", e.args)
				}
			}
			if errors.Is(tt.wantErr, ErrInvalidStatus) && len(st.queries) != 0 {
				t.Error("invalid status should be rejected before loading the job")
			}
		})
	}
}

// Owner UpdateStatus: a missing job surfaces as ErrJobNotFound (-> 404), not a raw error.
func TestJobSvc_UpdateStatus_MissingJobIsNotFound(t *testing.T) {
	tests := []struct {
		name     string
		status   string
		execRows int64
		wantErr  error
	}{
		{name: "existing job", status: "cancelled", execRows: 1},
		{name: "missing job", status: "cancelled", execRows: 0, wantErr: ErrJobNotFound},
		{name: "missing job, completing", status: "completed", execRows: 0, wantErr: ErrJobNotFound},
		{name: "invalid status", status: "done", wantErr: ErrInvalidStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newJobSvcWithFakeDB(t, &jobDBState{execRows: tt.execRows}, nil)
			err := svc.UpdateStatus(testJobID, testOrgID, tt.status)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// -----------------------------------------------------------------------
// GetByIDForWorker
// -----------------------------------------------------------------------

func TestJobSvc_GetByIDForWorker(t *testing.T) {
	tests := []struct {
		name    string
		job     *jobState
		wantErr bool
	}{
		{name: "own job", job: &jobState{technicianID: uintPtr(testWorkerID), assignment: pending}},
		{name: "other worker's job", job: &jobState{technicianID: uintPtr(otherWorker)}, wantErr: true},
		{name: "unassigned job", job: &jobState{}, wantErr: true},
		{name: "missing job", job: nil, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &jobDBState{}
			if tt.job != nil {
				st.jobRow = tt.job.row(testJobID, testOrgID)
			}
			job, err := newJobSvcWithFakeDB(t, st, nil).GetByIDForWorker(testJobID, testOrgID, testWorkerID)
			if tt.wantErr {
				if !errors.Is(err, ErrJobNotFound) || job != nil {
					t.Fatalf("got (%v, %v), want (nil, ErrJobNotFound)", job, err)
				}
				return
			}
			if err != nil || job == nil || job.ID != testJobID {
				t.Fatalf("got (%v, %v)", job, err)
			}
			if job.AssignmentStatus == nil || *job.AssignmentStatus != models.AssignmentPending {
				t.Errorf("AssignmentStatus = %v", job.AssignmentStatus)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Technician validation on assign / create
// -----------------------------------------------------------------------

func TestJobSvc_AssignTechnician_ValidatesTechnician(t *testing.T) {
	tests := []struct {
		name        string
		tech        *uint
		lookup      func(id, orgID uint) (*models.Worker, error)
		wantErr     error
		wantLookups int
		wantExec    bool
	}{
		{name: "active technician in org", tech: uintPtr(12),
			lookup: func(id, orgID uint) (*models.Worker, error) {
				return &models.Worker{ID: id, IsActive: true}, nil
			}, wantLookups: 1, wantExec: true},
		{name: "unassign skips lookup", tech: nil, wantLookups: 0, wantExec: true},
		{name: "technician not in org", tech: uintPtr(12),
			lookup:  func(id, orgID uint) (*models.Worker, error) { return nil, errors.New("worker not found") },
			wantErr: ErrInvalidTechnician, wantLookups: 1},
		{name: "lookup returns nil worker", tech: uintPtr(12),
			lookup:  func(id, orgID uint) (*models.Worker, error) { return nil, nil },
			wantErr: ErrInvalidTechnician, wantLookups: 1},
		{name: "inactive technician", tech: uintPtr(12),
			lookup: func(id, orgID uint) (*models.Worker, error) {
				return &models.Worker{ID: id, IsActive: false}, nil
			}, wantErr: ErrInvalidTechnician, wantLookups: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotOrg uint
			lookup := &mockWorkerLookup{FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
				gotOrg = orgID
				return tt.lookup(id, orgID)
			}}
			st := &jobDBState{execRows: 1}
			svc := newJobSvcWithFakeDB(t, st, lookup)

			err := svc.AssignTechnician(testJobID, testOrgID, tt.tech)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if lookup.calls != tt.wantLookups {
				t.Errorf("worker lookups = %d, want %d", lookup.calls, tt.wantLookups)
			}
			if tt.wantLookups > 0 && gotOrg != testOrgID {
				t.Errorf("worker lookup org = %d, want %d", gotOrg, testOrgID)
			}
			if got := len(st.execs) > 0; got != tt.wantExec {
				t.Fatalf("assign executed = %v, want %v", got, tt.wantExec)
			}
			if tt.wantExec && !strings.Contains(st.execs[0].query, "ELSE 'pending' END") {
				t.Errorf("assign query does not reset the offer: %s", st.execs[0].query)
			}
		})
	}
}

func TestJobSvc_Create_AssignmentAndValidation(t *testing.T) {
	inactive := &mockWorkerLookup{FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
		return &models.Worker{ID: id, IsActive: false}, nil
	}}
	tests := []struct {
		name           string
		tech           *uint
		workers        workerLookup
		wantErr        error
		wantInsert     bool
		wantAssignment interface{}
	}{
		{name: "with technician starts as pending offer", tech: uintPtr(12), wantInsert: true, wantAssignment: "pending"},
		{name: "without technician has no assignment", tech: nil, wantInsert: true, wantAssignment: nil},
		{name: "inactive technician rejected before insert", tech: uintPtr(12), workers: inactive, wantErr: ErrInvalidTechnician},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &jobDBState{insertedID: 77}
			svc := newJobSvcWithFakeDB(t, st, tt.workers)

			job, err := svc.Create(CreateJobInput{
				OrganizationID: testOrgID, CustomerID: 3, TechnicianID: tt.tech,
				Title: "Fix AC", ScheduledAt: time.Now(),
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if got := len(st.queries) > 0; got != tt.wantInsert {
				t.Fatalf("insert executed = %v, want %v", got, tt.wantInsert)
			}
			if !tt.wantInsert {
				return
			}
			if job.ID != 77 || job.DurationMinutes != repository.DefaultJobDurationMinutes {
				t.Errorf("job = %+v", job)
			}
			args := st.queries[0].args
			if args[13] != tt.wantAssignment {
				t.Errorf("assignment_status arg = %#v, want %#v", args[13], tt.wantAssignment)
			}
		})
	}
}

func TestJobSvc_CreateServiceCall_RejectsInvalidTechnician(t *testing.T) {
	lookup := &mockWorkerLookup{FindByIDFunc: func(id, orgID uint) (*models.Worker, error) {
		return nil, errors.New("not found")
	}}
	st := &jobDBState{}
	svc := newJobSvcWithFakeDB(t, st, lookup)

	_, err := svc.CreateServiceCall(context.Background(), testOrgID, &models.CreateServiceCallRequest{TechnicianID: uintPtr(99)})
	if !errors.Is(err, ErrInvalidTechnician) {
		t.Fatalf("err = %v, want ErrInvalidTechnician", err)
	}
	if len(st.queries)+len(st.execs) != 0 {
		t.Error("no SQL should run when the technician is invalid")
	}
}
