package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
	"github.com/lib/pq"
)

// -----------------------------------------------------------------------
// resultDriver: a fake sql/driver that returns one configured (rows, err)
// pair for every Query call and records every Exec call's arguments.
//
// This complements the fakeDriver/capturingDriver defined in
// organization_repository_test.go (same package - reused directly below via
// fakeTx/fakeResult): those two either always return the same fixed rows or
// always succeed on Exec. PaymentNotificationRepository needs a driver that
// can be told to fail a Query with a specific error (to exercise the
// pq.Error 23505 -> ErrPaymentRequestAlreadyActive translation), which
// neither existing fake supports, hence this addition.
// -----------------------------------------------------------------------

type resultDriver struct {
	queryRows driver.Rows
	queryErr  error
	execResult driver.Result
	execErr    error
	onExec     func(query string, args []driver.NamedValue)
}

func (d *resultDriver) Open(name string) (driver.Conn, error) {
	return &resultConn{d: d}, nil
}

type resultConn struct{ d *resultDriver }

func (c *resultConn) Prepare(query string) (driver.Stmt, error) {
	return &resultStmt{conn: c, query: query}, nil
}
func (c *resultConn) Close() error              { return nil }
func (c *resultConn) Begin() (driver.Tx, error) { return &fakeTx{}, nil }

type resultStmt struct {
	conn  *resultConn
	query string
}

func (s *resultStmt) Close() error  { return nil }
func (s *resultStmt) NumInput() int { return -1 }

func (s *resultStmt) Query(_ []driver.Value) (driver.Rows, error) {
	return s.conn.d.queryRows, s.conn.d.queryErr
}

func (s *resultStmt) Exec(args []driver.Value) (driver.Result, error) {
	if s.conn.d.onExec != nil {
		named := make([]driver.NamedValue, len(args))
		for i, v := range args {
			named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
		}
		s.conn.d.onExec(s.query, named)
	}
	if s.conn.d.execErr != nil {
		return nil, s.conn.d.execErr
	}
	res := s.conn.d.execResult
	if res == nil {
		res = &fakeResult{}
	}
	return res, nil
}

// customResult lets a test control RowsAffected() independently of the
// default fakeResult (which always reports 1).
type customResult struct{ rowsAffected int64 }

func (r *customResult) LastInsertId() (int64, error) { return 0, nil }
func (r *customResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

var resultDriverSeq int
var resultDriverMu sync.Mutex

// newResultDB registers a fresh resultDriver under a unique driver name and
// opens a *sql.DB bound to it.
func newResultDB(t *testing.T, d *resultDriver) *sql.DB {
	t.Helper()
	resultDriverMu.Lock()
	resultDriverSeq++
	n := resultDriverSeq
	resultDriverMu.Unlock()

	driverName := fmt.Sprintf("resultdb-%d", n)
	sql.Register(driverName, d)

	db, err := sql.Open(driverName, driverName)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// -----------------------------------------------------------------------
// CreateIfNotActive tests
// -----------------------------------------------------------------------

func TestCreateIfNotActive_Success(t *testing.T) {
	rows := &fakeRows{
		cols: []string{"id"},
		data: [][]driver.Value{{int64(42)}},
	}
	db := newResultDB(t, &resultDriver{queryRows: rows})

	repo := NewPaymentNotificationRepository(db)
	n := &models.PaymentNotification{
		OrganizationID: 1,
		JobID:          2,
		CustomerID:     3,
		Amount:         150.0,
		RecipientPhone: "050-1111111",
		MessageBody:    "pay up",
	}
	before := time.Now()
	err := repo.CreateIfNotActive(context.Background(), n)
	after := time.Now()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if n.ID != 42 {
		t.Errorf("expected ID 42, got %d", n.ID)
	}
	if n.SMSStatus != models.SMSStatusPending {
		t.Errorf("expected SMSStatus pending, got %v", n.SMSStatus)
	}
	if n.PaymentStatus != models.PaymentStatusPending {
		t.Errorf("expected PaymentStatus pending, got %v", n.PaymentStatus)
	}
	if n.CreatedAt.Before(before) || n.CreatedAt.After(after) {
		t.Errorf("expected CreatedAt to be set to now, got %v", n.CreatedAt)
	}
	if n.UpdatedAt != n.CreatedAt {
		t.Errorf("expected UpdatedAt == CreatedAt on creation, got %v vs %v", n.UpdatedAt, n.CreatedAt)
	}
}

func TestCreateIfNotActive_DuplicateActiveRequest_ReturnsErrPaymentRequestAlreadyActive(t *testing.T) {
	db := newResultDB(t, &resultDriver{
		queryErr: &pq.Error{Code: "23505", Constraint: "payment_notifications_job_id_active_idx"},
	})

	repo := NewPaymentNotificationRepository(db)
	n := &models.PaymentNotification{OrganizationID: 1, JobID: 2, CustomerID: 3, Amount: 100}
	err := repo.CreateIfNotActive(context.Background(), n)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrPaymentRequestAlreadyActive) {
		t.Errorf("expected ErrPaymentRequestAlreadyActive, got %v", err)
	}
	if n.ID != 0 {
		t.Errorf("expected ID to remain unset on failure, got %d", n.ID)
	}
}

func TestCreateIfNotActive_OtherPqError_PropagesRawError(t *testing.T) {
	// A pq.Error with a different code must NOT be translated to
	// ErrPaymentRequestAlreadyActive - only 23505 (unique_violation) should be.
	pqErr := &pq.Error{Code: "23503", Message: "foreign key violation"}
	db := newResultDB(t, &resultDriver{queryErr: pqErr})

	repo := NewPaymentNotificationRepository(db)
	n := &models.PaymentNotification{OrganizationID: 1, JobID: 2, CustomerID: 3, Amount: 100}
	err := repo.CreateIfNotActive(context.Background(), n)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if errors.Is(err, ErrPaymentRequestAlreadyActive) {
		t.Error("did not expect ErrPaymentRequestAlreadyActive for a non-23505 pq error")
	}
}

func TestCreateIfNotActive_GenericDBError_Propagates(t *testing.T) {
	sentinel := errors.New("connection reset by peer")
	db := newResultDB(t, &resultDriver{queryErr: sentinel})

	repo := NewPaymentNotificationRepository(db)
	n := &models.PaymentNotification{OrganizationID: 1, JobID: 2, CustomerID: 3, Amount: 100}
	err := repo.CreateIfNotActive(context.Background(), n)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if errors.Is(err, ErrPaymentRequestAlreadyActive) {
		t.Error("did not expect ErrPaymentRequestAlreadyActive for a generic error")
	}
}

// -----------------------------------------------------------------------
// UpdateSMSResult tests
// -----------------------------------------------------------------------

func TestUpdateSMSResult_ExecCalledWithExpectedArgs(t *testing.T) {
	var capturedArgs []driver.NamedValue
	db := newResultDB(t, &resultDriver{
		onExec: func(_ string, args []driver.NamedValue) { capturedArgs = args },
	})

	repo := NewPaymentNotificationRepository(db)
	sentAt := time.Now()
	err := repo.UpdateSMSResult(context.Background(), 7, models.SMSStatusSent, models.PaymentStatusSent, &sentAt)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// args: sms_status=$1, payment_status=$2, sent_at=$3, updated_at=$4, id=$5
	if len(capturedArgs) < 5 {
		t.Fatalf("expected at least 5 args, got %d", len(capturedArgs))
	}
	if capturedArgs[0].Value != string(models.SMSStatusSent) {
		t.Errorf("expected sms_status %q, got %v", models.SMSStatusSent, capturedArgs[0].Value)
	}
	if capturedArgs[1].Value != string(models.PaymentStatusSent) {
		t.Errorf("expected payment_status %q, got %v", models.PaymentStatusSent, capturedArgs[1].Value)
	}
	if capturedArgs[4].Value != int64(7) {
		t.Errorf("expected id 7, got %v", capturedArgs[4].Value)
	}
}

func TestUpdateSMSResult_NilSentAt_PassedThrough(t *testing.T) {
	var capturedArgs []driver.NamedValue
	db := newResultDB(t, &resultDriver{
		onExec: func(_ string, args []driver.NamedValue) { capturedArgs = args },
	})

	repo := NewPaymentNotificationRepository(db)
	err := repo.UpdateSMSResult(context.Background(), 7, models.SMSStatusFailed, models.PaymentStatusSendFailed, nil)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedArgs[2].Value != nil {
		t.Errorf("expected nil sent_at, got %v", capturedArgs[2].Value)
	}
}

func TestUpdateSMSResult_ExecError_Propagates(t *testing.T) {
	sentinel := errors.New("db write failed")
	db := newResultDB(t, &resultDriver{execErr: sentinel})

	repo := NewPaymentNotificationRepository(db)
	err := repo.UpdateSMSResult(context.Background(), 1, models.SMSStatusSent, models.PaymentStatusSent, nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// -----------------------------------------------------------------------
// MarkPaid tests
// -----------------------------------------------------------------------

func TestMarkPaid_Success(t *testing.T) {
	var capturedArgs []driver.NamedValue
	db := newResultDB(t, &resultDriver{
		execResult: &customResult{rowsAffected: 1},
		onExec:     func(_ string, args []driver.NamedValue) { capturedArgs = args },
	})

	repo := NewPaymentNotificationRepository(db)
	err := repo.MarkPaid(context.Background(), 5, 9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// args: payment_status=$1 (paid), paid_at=$2, id=$3, org_id=$4, payment_status=$5 (sent, WHERE clause)
	if len(capturedArgs) < 5 {
		t.Fatalf("expected at least 5 args, got %d", len(capturedArgs))
	}
	if capturedArgs[0].Value != string(models.PaymentStatusPaid) {
		t.Errorf("expected SET payment_status=paid, got %v", capturedArgs[0].Value)
	}
	if capturedArgs[2].Value != int64(5) {
		t.Errorf("expected id 5, got %v", capturedArgs[2].Value)
	}
	if capturedArgs[3].Value != int64(9) {
		t.Errorf("expected org_id 9, got %v", capturedArgs[3].Value)
	}
	if capturedArgs[4].Value != string(models.PaymentStatusSent) {
		t.Errorf("expected WHERE payment_status=sent, got %v", capturedArgs[4].Value)
	}
}

func TestMarkPaid_ZeroRowsAffected_ReturnsErrPaymentNotificationNotFound(t *testing.T) {
	db := newResultDB(t, &resultDriver{execResult: &customResult{rowsAffected: 0}})

	repo := NewPaymentNotificationRepository(db)
	err := repo.MarkPaid(context.Background(), 5, 9)
	if !errors.Is(err, ErrPaymentNotificationNotFound) {
		t.Errorf("expected ErrPaymentNotificationNotFound, got %v", err)
	}
}

func TestMarkPaid_ExecError_Propagates(t *testing.T) {
	sentinel := errors.New("db unavailable")
	db := newResultDB(t, &resultDriver{execErr: sentinel})

	repo := NewPaymentNotificationRepository(db)
	err := repo.MarkPaid(context.Background(), 5, 9)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if errors.Is(err, ErrPaymentNotificationNotFound) {
		t.Error("did not expect ErrPaymentNotificationNotFound for an exec error")
	}
}

// -----------------------------------------------------------------------
// ListByJobID tests
// -----------------------------------------------------------------------

func TestListByJobID_ReturnsNotificationsWithNullableFields(t *testing.T) {
	sentAt := time.Now().Add(-time.Hour)
	rows := &fakeRows{
		cols: []string{"id", "organization_id", "job_id", "customer_id", "amount", "recipient_phone",
			"message_body", "sms_status", "payment_status", "sent_at", "paid_at", "created_by",
			"created_at", "updated_at"},
		data: [][]driver.Value{
			{int64(1), int64(9), int64(2), int64(3), 150.0, "050-1111111", "pay up",
				string(models.SMSStatusSent), string(models.PaymentStatusSent), sentAt, nil, int64(4),
				sentAt, sentAt},
			{int64(2), int64(9), int64(2), int64(3), 200.0, "050-1111111", "pay up 2",
				string(models.SMSStatusPending), string(models.PaymentStatusPending), nil, nil, nil,
				sentAt, sentAt},
		},
	}
	db := newResultDB(t, &resultDriver{queryRows: rows})

	repo := NewPaymentNotificationRepository(db)
	out, err := repo.ListByJobID(context.Background(), 2, 9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 notifications, got %d", len(out))
	}

	first := out[0]
	if first.ID != 1 || first.OrganizationID != 9 || first.JobID != 2 || first.CustomerID != 3 {
		t.Errorf("unexpected first notification identity fields: %+v", first)
	}
	if first.SentAt == nil {
		t.Error("expected non-nil SentAt for first row")
	}
	if first.PaidAt != nil {
		t.Error("expected nil PaidAt for first row")
	}
	if first.CreatedBy == nil || *first.CreatedBy != 4 {
		t.Errorf("expected CreatedBy 4, got %v", first.CreatedBy)
	}

	second := out[1]
	if second.SentAt != nil {
		t.Error("expected nil SentAt for second row")
	}
	if second.CreatedBy != nil {
		t.Error("expected nil CreatedBy for second row")
	}
}

func TestListByJobID_EmptyResult(t *testing.T) {
	rows := &fakeRows{
		cols: []string{"id", "organization_id", "job_id", "customer_id", "amount", "recipient_phone",
			"message_body", "sms_status", "payment_status", "sent_at", "paid_at", "created_by",
			"created_at", "updated_at"},
		data: [][]driver.Value{},
	}
	db := newResultDB(t, &resultDriver{queryRows: rows})

	repo := NewPaymentNotificationRepository(db)
	out, err := repo.ListByJobID(context.Background(), 2, 9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected 0 notifications, got %d", len(out))
	}
}

func TestListByJobID_QueryError_Propagates(t *testing.T) {
	sentinel := errors.New("query failed")
	db := newResultDB(t, &resultDriver{queryErr: sentinel})

	repo := NewPaymentNotificationRepository(db)
	out, err := repo.ListByJobID(context.Background(), 2, 9)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if out != nil {
		t.Errorf("expected nil result on error, got %v", out)
	}
}
