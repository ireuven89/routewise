package repository

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

// -----------------------------------------------------------------------
// execStubDriver: like capturingDriver (organization_repository_test.go) it records
// every Exec, but it also lets a test choose RowsAffected or an error, which the
// accept/decline methods branch on. Keyed by DSN like seqDriver.
// -----------------------------------------------------------------------

var execStubOnce sync.Once

type execStubState struct {
	mu           sync.Mutex
	rowsAffected int64
	execErr      error
	rowsErr      error
	calls        []seqCall
}

var (
	execStubMu       sync.Mutex
	execStubRegistry = map[string]*execStubState{}
)

type execStubDriver struct{}

func (execStubDriver) Open(name string) (driver.Conn, error) { return &execStubConn{dsn: name}, nil }

type execStubConn struct{ dsn string }

func (c *execStubConn) Prepare(q string) (driver.Stmt, error) {
	return &execStubStmt{dsn: c.dsn, query: q}, nil
}
func (c *execStubConn) Close() error              { return nil }
func (c *execStubConn) Begin() (driver.Tx, error) { return &fakeTx{}, nil }

type execStubStmt struct {
	dsn, query string
}

func (s *execStubStmt) Close() error  { return nil }
func (s *execStubStmt) NumInput() int { return -1 }
func (s *execStubStmt) Exec(args []driver.Value) (driver.Result, error) {
	execStubMu.Lock()
	st := execStubRegistry[s.dsn]
	execStubMu.Unlock()
	st.mu.Lock()
	defer st.mu.Unlock()
	st.calls = append(st.calls, seqCall{query: s.query, args: args})
	if st.execErr != nil {
		return nil, st.execErr
	}
	return stubResult{rows: st.rowsAffected, err: st.rowsErr}, nil
}
func (s *execStubStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return nil, errors.New("execStubDriver: Query not supported")
}

type stubResult struct {
	rows int64
	err  error
}

func (r stubResult) LastInsertId() (int64, error) { return 0, nil }
func (r stubResult) RowsAffected() (int64, error) { return r.rows, r.err }

func newExecStubRepo(t *testing.T, dsn string, st *execStubState) *JobRepository {
	t.Helper()
	execStubOnce.Do(func() { sql.Register("execstubdb", execStubDriver{}) })
	execStubMu.Lock()
	execStubRegistry[dsn] = st
	execStubMu.Unlock()
	t.Cleanup(func() {
		execStubMu.Lock()
		delete(execStubRegistry, dsn)
		execStubMu.Unlock()
	})
	db, err := sql.Open("execstubdb", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewJobRepository(db, nil)
}

func normalizeSQL(q string) string { return strings.Join(strings.Fields(q), " ") }

// -----------------------------------------------------------------------
// AcceptAssignment
// -----------------------------------------------------------------------

func TestJobRepository_AcceptAssignment(t *testing.T) {
	tests := []struct {
		name    string
		state   execStubState
		wantOK  bool
		wantErr bool
	}{
		{name: "pending offer accepted", state: execStubState{rowsAffected: 1}, wantOK: true},
		{name: "nothing matched (reassigned, declined, not pending, or cancelled)", state: execStubState{rowsAffected: 0}, wantOK: false},
		{name: "exec error", state: execStubState{execErr: errors.New("conn reset")}, wantErr: true},
		{name: "rows affected error", state: execStubState{rowsAffected: 1, rowsErr: errors.New("unsupported")}, wantOK: true, wantErr: true},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := tt.state
			repo := newExecStubRepo(t, "accept-"+string(rune('a'+i)), &st)

			ok, err := repo.AcceptAssignment(42, 10, 7)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			if len(st.calls) != 1 {
				t.Fatalf("expected 1 exec, got %d", len(st.calls))
			}
			call := st.calls[0]
			q := normalizeSQL(call.query)
			for _, frag := range []string{
				"SET assignment_status = 'accepted', assignment_responded_at = $1",
				"id = $2 AND organization_id = $3 AND technician_id = $4",
				"assignment_status = 'pending'",
				"status NOT IN ('cancelled', 'completed')",
			} {
				if !strings.Contains(q, frag) {
					t.Errorf("query missing %q:\n%s", frag, q)
				}
			}
			if len(call.args) != 4 {
				t.Fatalf("expected 4 args, got %d: %v", len(call.args), call.args)
			}
			if _, isTime := call.args[0].(time.Time); !isTime {
				t.Errorf("args[0] should be the response time, got %T", call.args[0])
			}
			if call.args[1] != int64(42) || call.args[2] != int64(10) || call.args[3] != int64(7) {
				t.Errorf("args (job, org, worker) = %v, %v, %v; want 42, 10, 7", call.args[1], call.args[2], call.args[3])
			}
		})
	}
}

// -----------------------------------------------------------------------
// DeclineAssignment
// -----------------------------------------------------------------------

func TestJobRepository_DeclineAssignment(t *testing.T) {
	tests := []struct {
		name    string
		reason  string
		state   execStubState
		wantOK  bool
		wantErr bool
	}{
		{name: "declined with reason", reason: "Too far", state: execStubState{rowsAffected: 1}, wantOK: true},
		{name: "declined without reason", reason: "", state: execStubState{rowsAffected: 1}, wantOK: true},
		{name: "not this worker's scheduled job", reason: "x", state: execStubState{rowsAffected: 0}, wantOK: false},
		{name: "exec error", reason: "x", state: execStubState{execErr: errors.New("boom")}, wantErr: true},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := tt.state
			repo := newExecStubRepo(t, "decline-"+string(rune('a'+i)), &st)

			ok, err := repo.DeclineAssignment(42, 10, 7, tt.reason)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if ok != tt.wantOK {
				t.Errorf("ok = %v, want %v", ok, tt.wantOK)
			}
			call := st.calls[0]
			q := normalizeSQL(call.query)
			for _, frag := range []string{
				"technician_id = NULL, assignment_status = NULL, assignment_responded_at = NULL",
				"declined_by_worker_id = $1",
				"decline_reason = NULLIF($2, '')",
				"declined_at = $3",
				"WHERE id = $4 AND organization_id = $5 AND technician_id = $1 AND status = 'scheduled'",
			} {
				if !strings.Contains(q, frag) {
					t.Errorf("query missing %q:\n%s", frag, q)
				}
			}
			if len(call.args) != 5 {
				t.Fatalf("expected 5 args, got %d", len(call.args))
			}
			if call.args[0] != int64(7) || call.args[1] != tt.reason ||
				call.args[3] != int64(42) || call.args[4] != int64(10) {
				t.Errorf("args = %v; want [7 %q <time> 42 10]", call.args, tt.reason)
			}
		})
	}
}

// -----------------------------------------------------------------------
// AssignTechnician: (re)assignment resets or keeps the offer
// -----------------------------------------------------------------------

func TestJobRepository_AssignTechnician(t *testing.T) {
	tech := uint(12)
	tests := []struct {
		name     string
		tech     *uint
		state    execStubState
		wantErr  bool
		wantTech interface{}
	}{
		{name: "assign technician", tech: &tech, state: execStubState{rowsAffected: 1}, wantTech: int64(12)},
		{name: "unassign", tech: nil, state: execStubState{rowsAffected: 1}, wantTech: nil},
		{name: "job not found in organization", tech: &tech, state: execStubState{rowsAffected: 0}, wantErr: true, wantTech: int64(12)},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := tt.state
			repo := newExecStubRepo(t, "assign-"+string(rune('a'+i)), &st)

			err := repo.AssignTechnician(42, 10, tt.tech)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			call := st.calls[0]
			q := normalizeSQL(call.query)
			for _, frag := range []string{
				// unassigning clears the offer
				"WHEN $1::int IS NULL THEN NULL",
				// same technician keeps their answer
				"WHEN technician_id IS NOT DISTINCT FROM $1::int THEN assignment_status",
				// anyone else gets a fresh pending offer
				"ELSE 'pending' END",
				"WHERE id = $3 AND organization_id = $4",
			} {
				if !strings.Contains(q, frag) {
					t.Errorf("query missing %q:\n%s", frag, q)
				}
			}
			if call.args[0] != tt.wantTech {
				t.Errorf("technician arg = %#v, want %#v", call.args[0], tt.wantTech)
			}
			if call.args[2] != int64(42) || call.args[3] != int64(10) {
				t.Errorf("job/org args = %v/%v, want 42/10", call.args[2], call.args[3])
			}
		})
	}
}

// -----------------------------------------------------------------------
// Create: a job created with a technician starts as a pending offer
// -----------------------------------------------------------------------

func TestJobRepository_Create_SetsInitialAssignment(t *testing.T) {
	tech := uint(12)
	accepted := models.AssignmentAccepted
	tests := []struct {
		name       string
		job        *models.Job
		wantArg    interface{}
		wantStatus *models.AssignmentStatus
	}{
		{name: "with technician -> pending",
			job:     &models.Job{OrganizationID: 10, CustomerID: 1, TechnicianID: &tech, Title: "Fix"},
			wantArg: "pending", wantStatus: ptrAssignment(models.AssignmentPending)},
		{name: "without technician -> NULL",
			job:     &models.Job{OrganizationID: 10, CustomerID: 1, Title: "Fix"},
			wantArg: nil, wantStatus: nil},
		{name: "caller-supplied status is overwritten on create",
			job:     &models.Job{OrganizationID: 10, CustomerID: 1, TechnicianID: &tech, Title: "Fix", AssignmentStatus: &accepted},
			wantArg: "pending", wantStatus: ptrAssignment(models.AssignmentPending)},
		{name: "stale status without technician is cleared",
			job:     &models.Job{OrganizationID: 10, CustomerID: 1, Title: "Fix", AssignmentStatus: &accepted},
			wantArg: nil, wantStatus: nil},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, state := newSeqTestDB(t, "create-assign-"+string(rune('a'+i)), idResult(99))
			t.Cleanup(func() { db.Close() })
			repo := NewJobRepository(db, nil)

			if err := repo.Create(tt.job); err != nil {
				t.Fatalf("Create: %v", err)
			}
			if tt.job.ID != 99 {
				t.Errorf("ID = %d, want 99", tt.job.ID)
			}
			call := callContaining(t, state, "INSERT INTO jobs")
			if !strings.Contains(call.query, "assignment_status") {
				t.Fatalf("insert does not write assignment_status: %s", call.query)
			}
			if len(call.args) != 14 {
				t.Fatalf("expected 14 args, got %d", len(call.args))
			}
			if call.args[13] != tt.wantArg {
				t.Errorf("assignment_status arg = %#v, want %#v", call.args[13], tt.wantArg)
			}
			if (tt.job.AssignmentStatus == nil) != (tt.wantStatus == nil) ||
				(tt.wantStatus != nil && *tt.job.AssignmentStatus != *tt.wantStatus) {
				t.Errorf("job.AssignmentStatus = %v, want %v", tt.job.AssignmentStatus, tt.wantStatus)
			}
		})
	}
}

func ptrAssignment(s models.AssignmentStatus) *models.AssignmentStatus { return &s }

// -----------------------------------------------------------------------
// FindByID maps the new assignment / decline columns
// -----------------------------------------------------------------------

var jobSelectCols = []string{
	"id", "organization_id", "created_by", "customer_id", "technician_id", "title",
	"description", "status", "scheduled_at", "completed_at", "duration_minutes",
	"price", "metadata", "created_at", "updated_at",
	"assignment_status", "assignment_responded_at", "declined_by_worker_id", "decline_reason", "declined_at",
	"c_id", "c_name", "c_phone", "c_email", "c_address", "c_lat", "c_lng", "c_formatted",
	"w_id", "w_name", "w_phone",
	"dw_name",
}

func jobSelectRow(technicianID, assignment, respondedAt, declinedBy, reason, declinedAt, workerID, workerName, declinedName driver.Value) []driver.Value {
	now := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	return []driver.Value{
		int64(42), int64(10), nil, int64(3), technicianID, "Fix AC",
		"", "scheduled", now, nil, int64(60),
		nil, []byte(`{"source":"manual"}`), now, now,
		assignment, respondedAt, declinedBy, reason, declinedAt,
		int64(3), "Dana", "050", "", "Herzl 1", 32.1, 34.8, "Herzl 1, TLV",
		workerID, workerName, "052",
		declinedName,
	}
}

func TestJobRepository_FindByID_AssignmentFields(t *testing.T) {
	responded := time.Date(2026, 9, 28, 10, 0, 0, 0, time.UTC)
	declined := time.Date(2026, 9, 27, 8, 0, 0, 0, time.UTC)

	tests := []struct {
		name  string
		row   []driver.Value
		check func(t *testing.T, j *models.Job)
	}{
		{
			name: "accepted assignment",
			row:  jobSelectRow(int64(7), "accepted", responded, nil, "", nil, int64(7), "Avi", ""),
			check: func(t *testing.T, j *models.Job) {
				if j.TechnicianID == nil || *j.TechnicianID != 7 {
					t.Errorf("TechnicianID = %v", j.TechnicianID)
				}
				if j.AssignmentStatus == nil || *j.AssignmentStatus != models.AssignmentAccepted {
					t.Errorf("AssignmentStatus = %v", j.AssignmentStatus)
				}
				if j.AssignmentRespondedAt == nil || !j.AssignmentRespondedAt.Equal(responded) {
					t.Errorf("AssignmentRespondedAt = %v", j.AssignmentRespondedAt)
				}
				if j.Worker == nil || j.Worker.ID != 7 || j.Worker.Name != "Avi" {
					t.Errorf("Worker = %+v", j.Worker)
				}
				if j.DeclinedByWorkerID != nil || j.DeclinedAt != nil || j.DeclineReason != "" {
					t.Errorf("unexpected decline info: %v %v %q", j.DeclinedByWorkerID, j.DeclinedAt, j.DeclineReason)
				}
			},
		},
		{
			name: "declined and back in the pool",
			row:  jobSelectRow(nil, nil, nil, int64(7), "Too far", declined, nil, "", "Avi"),
			check: func(t *testing.T, j *models.Job) {
				if j.TechnicianID != nil || j.AssignmentStatus != nil || j.Worker != nil {
					t.Errorf("expected unassigned job, got tech=%v status=%v worker=%v", j.TechnicianID, j.AssignmentStatus, j.Worker)
				}
				if j.DeclinedByWorkerID == nil || *j.DeclinedByWorkerID != 7 {
					t.Errorf("DeclinedByWorkerID = %v", j.DeclinedByWorkerID)
				}
				if j.DeclineReason != "Too far" || j.DeclinedByName != "Avi" {
					t.Errorf("reason=%q name=%q", j.DeclineReason, j.DeclinedByName)
				}
				if j.DeclinedAt == nil || !j.DeclinedAt.Equal(declined) {
					t.Errorf("DeclinedAt = %v", j.DeclinedAt)
				}
			},
		},
		{
			name: "pending offer, customer and metadata loaded",
			row:  jobSelectRow(int64(7), "pending", nil, nil, "", nil, int64(7), "Avi", ""),
			check: func(t *testing.T, j *models.Job) {
				if j.AssignmentStatus == nil || *j.AssignmentStatus != models.AssignmentPending {
					t.Errorf("AssignmentStatus = %v", j.AssignmentStatus)
				}
				if j.AssignmentRespondedAt != nil {
					t.Errorf("AssignmentRespondedAt should be nil, got %v", j.AssignmentRespondedAt)
				}
				if j.Customer.ID != 3 || j.Customer.Name != "Dana" || j.Customer.OrganizationID != 10 {
					t.Errorf("Customer = %+v", j.Customer)
				}
				if j.Metadata["source"] != "manual" {
					t.Errorf("Metadata = %v", j.Metadata)
				}
			},
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, state := newSeqTestDB(t, "findbyid-assign-"+string(rune('a'+i)), rowsResult(jobSelectCols, tt.row))
			t.Cleanup(func() { db.Close() })

			job, err := NewJobRepository(db, nil).FindByID(42, 10)
			if err != nil {
				t.Fatalf("FindByID: %v", err)
			}
			call := callContaining(t, state, "FROM jobs j")
			if !strings.Contains(call.query, "WHERE j.id = $1 AND j.organization_id = $2") {
				t.Errorf("query not scoped by organization: %s", call.query)
			}
			if call.args[0] != int64(42) || call.args[1] != int64(10) {
				t.Errorf("args = %v", call.args)
			}
			tt.check(t, job)
		})
	}
}

func TestJobRepository_FindByID_NotFound(t *testing.T) {
	db, _ := newSeqTestDB(t, "findbyid-assign-missing", rowsResult(jobSelectCols))
	t.Cleanup(func() { db.Close() })
	if _, err := NewJobRepository(db, nil).FindByID(1, 10); err == nil || err.Error() != "job not found" {
		t.Fatalf("err = %v, want \"job not found\"", err)
	}
}
