package repository

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/ireuven89/routewise/internal/models"
)

// -----------------------------------------------------------------------
// Sequenced fake SQL driver
//
// GetRevenueStats issues five sequential calls against *sql.DB (four
// QueryRow calls followed by one Query call). The generic fakeDriver in
// organization_repository_test.go always returns the same rows.driver.Rows
// for every call made against a DSN, which can't model five distinct
// result sets. This driver instead hands back the next queued result (rows
// or an error) on each successive call, in call order — matching exactly
// how GetRevenueStats drives the connection.
// -----------------------------------------------------------------------

var seqDriverOnce sync.Once

func registerSeqDriver() {
	seqDriverOnce.Do(func() {
		sql.Register("seqfakedb", &seqDriver{})
	})
}

// seqResult is one queued outcome for a Query/QueryRow call.
type seqResult struct {
	rows *trackedRows
	err  error
}

type seqState struct {
	mu      sync.Mutex
	results []seqResult
	idx     int
}

// seqRegistry maps a DSN to the ordered list of results to hand back.
var (
	seqRegistryMu sync.Mutex
	seqRegistry   = map[string]*seqState{}
)

type seqDriver struct{}

func (d *seqDriver) Open(name string) (driver.Conn, error) {
	return &seqConn{dsn: name}, nil
}

type seqConn struct{ dsn string }

func (c *seqConn) Prepare(query string) (driver.Stmt, error) { return &seqStmt{conn: c}, nil }
func (c *seqConn) Close() error                              { return nil }
func (c *seqConn) Begin() (driver.Tx, error)                 { return &fakeTx{}, nil }

type seqStmt struct{ conn *seqConn }

func (s *seqStmt) Close() error  { return nil }
func (s *seqStmt) NumInput() int { return -1 }
func (s *seqStmt) Exec(_ []driver.Value) (driver.Result, error) {
	return &fakeResult{}, nil
}
func (s *seqStmt) Query(_ []driver.Value) (driver.Rows, error) {
	seqRegistryMu.Lock()
	state := seqRegistry[s.conn.dsn]
	seqRegistryMu.Unlock()

	if state == nil {
		return nil, fmt.Errorf("no sequence registered for dsn %q", s.conn.dsn)
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	if state.idx >= len(state.results) {
		return nil, fmt.Errorf("unexpected extra query call (call #%d)", state.idx+1)
	}
	res := state.results[state.idx]
	state.idx++
	if res.err != nil {
		return nil, res.err
	}
	return res.rows, nil
}

// callCount reports how many queries were consumed from the sequence.
func (s *seqState) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idx
}

// newSeqTestDB registers a sequence of results under a fresh DSN and opens a
// *sql.DB bound to it.
func newSeqTestDB(t *testing.T, dsn string, results ...seqResult) (*sql.DB, *seqState) {
	t.Helper()
	registerSeqDriver()

	state := &seqState{results: results}
	seqRegistryMu.Lock()
	seqRegistry[dsn] = state
	seqRegistryMu.Unlock()
	t.Cleanup(func() {
		seqRegistryMu.Lock()
		delete(seqRegistry, dsn)
		seqRegistryMu.Unlock()
	})

	db, err := sql.Open("seqfakedb", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db, state
}

// trackedRows wraps fakeRows and records whether Close() was invoked, so
// tests can assert the monthly-breakdown rows are always closed - including
// when a Scan mid-iteration fails.
type trackedRows struct {
	fakeRows
	closed bool
}

func (r *trackedRows) Close() error {
	r.closed = true
	return r.fakeRows.Close()
}

// scalarResult builds a single-row/single-column queued result, used for the
// four QueryRow calls (total, this-month, this-week, avg job value).
func scalarResult(v float64) seqResult {
	return seqResult{rows: &trackedRows{fakeRows: fakeRows{
		cols: []string{"value"},
		data: [][]driver.Value{{v}},
	}}}
}

// monthlyResult builds the queued result for the final monthly-breakdown
// Query call.
func monthlyResult(rows [][]driver.Value) *trackedRows {
	return &trackedRows{fakeRows: fakeRows{
		cols: []string{"month", "revenue"},
		data: rows,
	}}
}

func errResult(err error) seqResult {
	return seqResult{err: err}
}

// -----------------------------------------------------------------------
// GetRevenueStats tests
// -----------------------------------------------------------------------

func TestGetRevenueStats_HappyPath_MultipleMonths(t *testing.T) {
	monthly := monthlyResult([][]driver.Value{
		{"2026-02", 1000.0},
		{"2026-03", 1500.0},
		{"2026-04", 2200.0},
	})

	db, state := newSeqTestDB(t, "revenue-happy-path",
		scalarResult(10000.0),    // total
		scalarResult(2200.0),     // this month
		scalarResult(500.0),      // this week
		scalarResult(333.33),     // avg job value
		seqResult{rows: monthly}, // monthly breakdown
	)
	defer db.Close()

	repo := NewJobRepository(db, nil)
	stats, err := repo.GetRevenueStats(7)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Total != 10000.0 {
		t.Errorf("expected Total 10000.0, got %v", stats.Total)
	}
	if stats.ThisMonth != 2200.0 {
		t.Errorf("expected ThisMonth 2200.0, got %v", stats.ThisMonth)
	}
	if stats.ThisWeek != 500.0 {
		t.Errorf("expected ThisWeek 500.0, got %v", stats.ThisWeek)
	}
	if stats.AvgJobValue != 333.33 {
		t.Errorf("expected AvgJobValue 333.33, got %v", stats.AvgJobValue)
	}
	if len(stats.RevenueByMonth) != 3 {
		t.Fatalf("expected 3 months of revenue data, got %d", len(stats.RevenueByMonth))
	}
	expected := []models.MonthlyRevenue{
		{Month: "2026-02", Revenue: 1000.0},
		{Month: "2026-03", Revenue: 1500.0},
		{Month: "2026-04", Revenue: 2200.0},
	}
	for i, want := range expected {
		got := stats.RevenueByMonth[i]
		if got.Month != want.Month || got.Revenue != want.Revenue {
			t.Errorf("month %d: expected %+v, got %+v", i, want, got)
		}
	}
	// Ascending order check.
	if stats.RevenueByMonth[0].Month > stats.RevenueByMonth[1].Month ||
		stats.RevenueByMonth[1].Month > stats.RevenueByMonth[2].Month {
		t.Error("expected RevenueByMonth ordered ascending by month")
	}
	if !monthly.closed {
		t.Error("expected monthly breakdown rows to be closed")
	}
	if got := state.callCount(); got != 5 {
		t.Errorf("expected exactly 5 queries issued, got %d", got)
	}
}

func TestGetRevenueStats_EmptyNoRevenue(t *testing.T) {
	monthly := monthlyResult([][]driver.Value{}) // no rows

	db, _ := newSeqTestDB(t, "revenue-empty",
		scalarResult(0),
		scalarResult(0),
		scalarResult(0),
		scalarResult(0),
		seqResult{rows: monthly},
	)
	defer db.Close()

	repo := NewJobRepository(db, nil)
	stats, err := repo.GetRevenueStats(9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stats.Total != 0 || stats.ThisMonth != 0 || stats.ThisWeek != 0 || stats.AvgJobValue != 0 {
		t.Errorf("expected all-zero stats, got %+v", stats)
	}
	if len(stats.RevenueByMonth) != 0 {
		t.Errorf("expected empty RevenueByMonth, got %v", stats.RevenueByMonth)
	}
	if stats.RevenueByMonth == nil {
		t.Error("expected RevenueByMonth to be initialized to an empty (non-nil) slice")
	}
	if !monthly.closed {
		t.Error("expected monthly breakdown rows to be closed even when empty")
	}
}

func TestGetRevenueStats_ErrorOnScalarQuery_StopsAndPropagates(t *testing.T) {
	sentinel := errors.New("boom: this-month query failed")

	// Only two results queued: the first (total) succeeds, the second
	// (this-month) errors. If the implementation still proceeds to issue
	// the remaining 3 queries, the fake driver will return the "unexpected
	// extra query call" error instead, which would fail the assertions
	// below and expose the bug.
	db, state := newSeqTestDB(t, "revenue-scalar-error",
		scalarResult(10000.0),
		errResult(sentinel),
	)
	defer db.Close()

	repo := NewJobRepository(db, nil)
	stats, err := repo.GetRevenueStats(3)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected error to wrap/equal %v, got %v", sentinel, err)
	}
	if stats != nil {
		t.Errorf("expected nil stats on error, got %+v", stats)
	}
	if got := state.callCount(); got != 2 {
		t.Errorf("expected exactly 2 queries issued before bailing out, got %d", got)
	}
}

func TestGetRevenueStats_ErrorOnMonthlyQuery_Propagates(t *testing.T) {
	sentinel := errors.New("boom: monthly breakdown query failed")

	db, state := newSeqTestDB(t, "revenue-monthly-query-error",
		scalarResult(1000.0),
		scalarResult(100.0),
		scalarResult(50.0),
		scalarResult(250.0),
		errResult(sentinel),
	)
	defer db.Close()

	repo := NewJobRepository(db, nil)
	stats, err := repo.GetRevenueStats(4)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("expected error to wrap/equal %v, got %v", sentinel, err)
	}
	if stats != nil {
		t.Errorf("expected nil stats on error, got %+v", stats)
	}
	if got := state.callCount(); got != 5 {
		t.Errorf("expected all 5 queries issued, got %d", got)
	}
}

func TestGetRevenueStats_ErrorOnMonthlyScan_ClosesRowsAndPropagates(t *testing.T) {
	// The revenue column can't be parsed as a float64 - this simulates a
	// malformed row causing rows.Scan to fail mid-iteration.
	monthly := monthlyResult([][]driver.Value{
		{"2026-01", 500.0},
		{"2026-02", []byte("not-a-number")},
	})

	db, _ := newSeqTestDB(t, "revenue-monthly-scan-error",
		scalarResult(1000.0),
		scalarResult(100.0),
		scalarResult(50.0),
		scalarResult(250.0),
		seqResult{rows: monthly},
	)
	defer db.Close()

	repo := NewJobRepository(db, nil)
	stats, err := repo.GetRevenueStats(5)
	if err == nil {
		t.Fatal("expected a scan error, got nil")
	}
	if stats != nil {
		t.Errorf("expected nil stats on error, got %+v", stats)
	}
	if !monthly.closed {
		t.Error("expected monthly rows to be closed (via defer) even when Scan fails mid-iteration")
	}
}
