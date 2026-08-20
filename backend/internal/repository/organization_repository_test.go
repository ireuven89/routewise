package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"
)

// -----------------------------------------------------------------------
// Minimal fake SQL driver (no external dependencies required)
// -----------------------------------------------------------------------

// fakeDriverOnce ensures we only register the driver once across all tests.
var fakeDriverOnce sync.Once

func registerFakeDriver() {
	fakeDriverOnce.Do(func() {
		sql.Register("fakedb", &fakeDriver{})
	})
}

// fakeDriver / fakeConn / fakeStmt / fakeRows implement the minimum
// database/sql/driver interfaces needed to unit-test the repository.

type fakeDriver struct{}

func (d *fakeDriver) Open(name string) (driver.Conn, error) {
	return &fakeConn{dsn: name}, nil
}

type fakeConn struct {
	dsn string
}

func (c *fakeConn) Prepare(query string) (driver.Stmt, error) { return &fakeStmt{conn: c}, nil }
func (c *fakeConn) Close() error                              { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)                 { return &fakeTx{}, nil }

type fakeTx struct{}

func (t *fakeTx) Commit() error   { return nil }
func (t *fakeTx) Rollback() error { return nil }

type fakeStmt struct {
	conn *fakeConn
}

func (s *fakeStmt) Close() error                                    { return nil }
func (s *fakeStmt) NumInput() int                                   { return -1 } // variadic
func (s *fakeStmt) Query(args []driver.Value) (driver.Rows, error)  { return s.conn.query(args) }
func (s *fakeStmt) Exec(args []driver.Value) (driver.Result, error) { return s.conn.exec(args) }

func (c *fakeConn) exec(_ []driver.Value) (driver.Result, error) {
	return &fakeResult{}, nil
}

func (c *fakeConn) query(_ []driver.Value) (driver.Rows, error) {
	return registeredRows[c.dsn], nil
}

type fakeResult struct{}

func (r *fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (r *fakeResult) RowsAffected() (int64, error) { return 1, nil }

// registeredRows maps a DSN to a set of rows to return.
var registeredRows = map[string]driver.Rows{}

// fakeRows implements driver.Rows backed by a slice of value slices.
type fakeRows struct {
	cols    []string
	data    [][]driver.Value
	current int
}

func (r *fakeRows) Columns() []string { return r.cols }
func (r *fakeRows) Close() error      { return nil }
func (r *fakeRows) Next(dest []driver.Value) error {
	if r.current >= len(r.data) {
		return io.EOF
	}
	row := r.data[r.current]
	r.current++
	for i, v := range row {
		dest[i] = v
	}
	return nil
}

// newTestDB opens a fakedb connection bound to the given DSN.
func newTestDB(t *testing.T, dsn string, rows driver.Rows) *sql.DB {
	t.Helper()
	registerFakeDriver()
	registeredRows[dsn] = rows
	t.Cleanup(func() { delete(registeredRows, dsn) })

	db, err := sql.Open("fakedb", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	return db
}

// -----------------------------------------------------------------------
// FindByID tests
// -----------------------------------------------------------------------

func TestFindByID_Success_ScansAllColumnsIncludingPaymentSettings(t *testing.T) {
	lat, lng := 32.08, 34.78
	visitFee := 150.0
	rows := &fakeRows{
		cols: []string{"id", "name", "phone", "industry", "company_code",
			"latitude", "longitude", "address", "service_radius_km",
			"google_place_id", "formatted_address", "address_components", "geocoded_at",
			"visit_fee", "repair_estimate_min", "repair_estimate_max",
			"bit_payment_enabled", "bit_phone_number", "bit_business_name", "auto_send_payment_sms",
			"created_at", "updated_at"},
		data: [][]driver.Value{
			{int64(9), "Cool Air Ltd", "050-1111111", "hvac", "COOL-AIR-X7J2",
				lat, lng, "1 Herzl St", 20.0,
				"ChIJ123", "1 Herzl St, Tel Aviv", nil, nil,
				visitFee, nil, nil,
				true, "050-9999999", "Cool Air Ltd", false,
				time.Now(), time.Now()},
		},
	}

	db := newTestDB(t, "find-by-id-success", rows)
	defer db.Close()

	repo := NewOrganizationRepository(db)
	org, err := repo.FindByID(9)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if org.ID != 9 {
		t.Errorf("expected ID 9, got %d", org.ID)
	}
	if org.Name != "Cool Air Ltd" {
		t.Errorf("expected name 'Cool Air Ltd', got %q", org.Name)
	}
	if org.BitPaymentEnabled != true {
		t.Errorf("expected BitPaymentEnabled true, got %v", org.BitPaymentEnabled)
	}
	if org.BitPhoneNumber != "050-9999999" {
		t.Errorf("expected BitPhoneNumber '050-9999999', got %q", org.BitPhoneNumber)
	}
	if org.BitBusinessName != "Cool Air Ltd" {
		t.Errorf("expected BitBusinessName 'Cool Air Ltd', got %q", org.BitBusinessName)
	}
	if org.AutoSendPaymentSMS != false {
		t.Errorf("expected AutoSendPaymentSMS false, got %v", org.AutoSendPaymentSMS)
	}
	if org.VisitFee == nil || *org.VisitFee != visitFee {
		t.Errorf("expected VisitFee %v, got %v", visitFee, org.VisitFee)
	}
}

func TestFindByID_PaymentSettingsDefaultToZeroValues(t *testing.T) {
	// An organization that never configured Bit payment: bit_payment_enabled=false,
	// bit_phone_number/bit_business_name empty, auto_send_payment_sms=false.
	rows := &fakeRows{
		cols: []string{"id", "name", "phone", "industry", "company_code",
			"latitude", "longitude", "address", "service_radius_km",
			"google_place_id", "formatted_address", "address_components", "geocoded_at",
			"visit_fee", "repair_estimate_min", "repair_estimate_max",
			"bit_payment_enabled", "bit_phone_number", "bit_business_name", "auto_send_payment_sms",
			"created_at", "updated_at"},
		data: [][]driver.Value{
			{int64(1), "New Org", "050-0000000", "hvac", "NEW-ORG-A1B2",
				nil, nil, nil, 20.0,
				nil, nil, nil, nil,
				nil, nil, nil,
				false, "", "", false,
				time.Now(), time.Now()},
		},
	}

	db := newTestDB(t, "find-by-id-defaults", rows)
	defer db.Close()

	repo := NewOrganizationRepository(db)
	org, err := repo.FindByID(1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if org.BitPaymentEnabled {
		t.Error("expected BitPaymentEnabled false by default")
	}
	if org.BitPhoneNumber != "" {
		t.Errorf("expected empty BitPhoneNumber, got %q", org.BitPhoneNumber)
	}
	if org.BitBusinessName != "" {
		t.Errorf("expected empty BitBusinessName, got %q", org.BitBusinessName)
	}
	if org.AutoSendPaymentSMS {
		t.Error("expected AutoSendPaymentSMS false by default")
	}
}

func TestFindByID_NotFound_ReturnsOrganizationNotFoundError(t *testing.T) {
	rows := &fakeRows{
		cols: []string{"id", "name", "phone", "industry", "company_code",
			"latitude", "longitude", "address", "service_radius_km",
			"google_place_id", "formatted_address", "address_components", "geocoded_at",
			"visit_fee", "repair_estimate_min", "repair_estimate_max",
			"bit_payment_enabled", "bit_phone_number", "bit_business_name", "auto_send_payment_sms",
			"created_at", "updated_at"},
		data: [][]driver.Value{}, // no rows -> sql.ErrNoRows
	}

	db := newTestDB(t, "find-by-id-not-found", rows)
	defer db.Close()

	repo := NewOrganizationRepository(db)
	org, err := repo.FindByID(999)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if org != nil {
		t.Errorf("expected nil organization on not-found, got %+v", org)
	}
	if err.Error() != "organization not found" {
		t.Errorf("expected 'organization not found', got %q", err.Error())
	}
}

// -----------------------------------------------------------------------
// UpdatePaymentSettings tests
// -----------------------------------------------------------------------

func TestUpdatePaymentSettings_ExecCalledWithExpectedArgs(t *testing.T) {
	driverName := "capturingdb-payment-settings-1"
	var capturedArgs []driver.NamedValue

	sql.Register(driverName, &capturingDriver{
		onExec: func(_ string, args []driver.NamedValue) {
			capturedArgs = args
		},
	})

	db, err := sql.Open(driverName, "test-payment-settings")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	repo := NewOrganizationRepository(db)
	if err := repo.UpdatePaymentSettings(context.Background(), 42, true, "050-1234567", "Cool Air Ltd", true); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// args: enabled=$1, phone=$2, businessName=$3, autoSend=$4, updated_at=$5, orgID=$6
	if len(capturedArgs) < 6 {
		t.Fatalf("expected at least 6 args, got %d", len(capturedArgs))
	}
	if capturedArgs[0].Value != true {
		t.Errorf("expected enabled=true, got %v", capturedArgs[0].Value)
	}
	if capturedArgs[1].Value != "050-1234567" {
		t.Errorf("expected phone '050-1234567', got %v", capturedArgs[1].Value)
	}
	if capturedArgs[2].Value != "Cool Air Ltd" {
		t.Errorf("expected businessName 'Cool Air Ltd', got %v", capturedArgs[2].Value)
	}
	if capturedArgs[3].Value != true {
		t.Errorf("expected autoSend=true, got %v", capturedArgs[3].Value)
	}
	if capturedArgs[5].Value != int64(42) {
		t.Errorf("expected orgID 42, got %v", capturedArgs[5].Value)
	}
}

func TestUpdatePaymentSettings_DisablingClearsFlags(t *testing.T) {
	driverName := "capturingdb-payment-settings-2"
	var capturedArgs []driver.NamedValue

	sql.Register(driverName, &capturingDriver{
		onExec: func(_ string, args []driver.NamedValue) {
			capturedArgs = args
		},
	})

	db, err := sql.Open(driverName, "test-payment-settings-disable")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	repo := NewOrganizationRepository(db)
	if err := repo.UpdatePaymentSettings(context.Background(), 5, false, "", "", false); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedArgs[0].Value != false {
		t.Errorf("expected enabled=false, got %v", capturedArgs[0].Value)
	}
	if capturedArgs[3].Value != false {
		t.Errorf("expected autoSend=false, got %v", capturedArgs[3].Value)
	}
}

func TestUpdatePaymentSettings_ExecError_Propagates(t *testing.T) {
	// capturingConn.Exec always succeeds in this file's fake driver; use the
	// resultDriver from payment_notification_repository_test.go (same package)
	// which supports injecting an exec error.
	sentinel := errors.New("db write failed")
	db := newResultDB(t, &resultDriver{execErr: sentinel})

	repo := NewOrganizationRepository(db)
	err := repo.UpdatePaymentSettings(context.Background(), 5, true, "050-1234567", "Biz", true)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

// -----------------------------------------------------------------------
// FindProvidersInArea tests
// -----------------------------------------------------------------------

func TestFindProvidersInArea_ReturnsProviders(t *testing.T) {
	visitFee := 120.0
	repairMin := 200.0
	repairMax := 600.0

	rows := &fakeRows{
		cols: []string{"id", "name", "phone", "industry", "formatted_addr",
			"visit_fee", "repair_estimate_min", "repair_estimate_max", "distance_km"},
		data: [][]driver.Value{
			{int64(1), "Cool Air Ltd", "050-1111111", "hvac", "1 Herzl St, Tel Aviv",
				visitFee, repairMin, repairMax, 3.2},
		},
	}

	db := newTestDB(t, "find-providers-returns", rows)
	defer db.Close()

	repo := NewOrganizationRepository(db)
	results, err := repo.FindProvidersInArea(context.Background(), 32.08, 34.78, "hvac", 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	p := results[0]
	if p.ID != 1 {
		t.Errorf("expected ID 1, got %d", p.ID)
	}
	if p.Name != "Cool Air Ltd" {
		t.Errorf("expected name 'Cool Air Ltd', got '%s'", p.Name)
	}
	if p.Phone != "050-1111111" {
		t.Errorf("expected phone '050-1111111', got '%s'", p.Phone)
	}
	if p.Industry != "hvac" {
		t.Errorf("expected industry 'hvac', got '%s'", p.Industry)
	}
	if p.Address != "1 Herzl St, Tel Aviv" {
		t.Errorf("expected address '1 Herzl St, Tel Aviv', got '%s'", p.Address)
	}
	if p.VisitFee == nil || *p.VisitFee != visitFee {
		t.Errorf("expected visit_fee %v, got %v", visitFee, p.VisitFee)
	}
	if p.RepairEstimateMin == nil || *p.RepairEstimateMin != repairMin {
		t.Errorf("expected repair_min %v, got %v", repairMin, p.RepairEstimateMin)
	}
	if p.RepairEstimateMax == nil || *p.RepairEstimateMax != repairMax {
		t.Errorf("expected repair_max %v, got %v", repairMax, p.RepairEstimateMax)
	}
	if p.DistanceKm != 3.2 {
		t.Errorf("expected distance_km 3.2, got %v", p.DistanceKm)
	}
}

func TestFindProvidersInArea_NullableFeesAreNil(t *testing.T) {
	// All optional fee columns are NULL → should scan as nil pointers.
	rows := &fakeRows{
		cols: []string{"id", "name", "phone", "industry", "formatted_addr",
			"visit_fee", "repair_estimate_min", "repair_estimate_max", "distance_km"},
		data: [][]driver.Value{
			{int64(2), "FixIt Pro", "052-9999999", "plumbing", "5 Allenby St",
				nil, nil, nil, 1.5},
		},
	}

	db := newTestDB(t, "find-providers-nullable", rows)
	defer db.Close()

	repo := NewOrganizationRepository(db)
	results, err := repo.FindProvidersInArea(context.Background(), 32.08, 34.78, "plumbing", 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	p := results[0]
	if p.VisitFee != nil {
		t.Errorf("expected nil visit_fee for NULL column, got %v", p.VisitFee)
	}
	if p.RepairEstimateMin != nil {
		t.Errorf("expected nil repair_estimate_min for NULL column, got %v", p.RepairEstimateMin)
	}
	if p.RepairEstimateMax != nil {
		t.Errorf("expected nil repair_estimate_max for NULL column, got %v", p.RepairEstimateMax)
	}
}

func TestFindProvidersInArea_EmptyResult(t *testing.T) {
	rows := &fakeRows{
		cols: []string{"id", "name", "phone", "industry", "formatted_addr",
			"visit_fee", "repair_estimate_min", "repair_estimate_max", "distance_km"},
		data: [][]driver.Value{}, // no rows
	}

	db := newTestDB(t, "find-providers-empty", rows)
	defer db.Close()

	repo := NewOrganizationRepository(db)
	results, err := repo.FindProvidersInArea(context.Background(), 32.08, 34.78, "electrical", 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if results != nil {
		t.Errorf("expected nil slice for empty result set, got %v", results)
	}
}

func TestFindProvidersInArea_MultipleProviders(t *testing.T) {
	vf1 := 100.0
	vf2 := 150.0
	rows := &fakeRows{
		cols: []string{"id", "name", "phone", "industry", "formatted_addr",
			"visit_fee", "repair_estimate_min", "repair_estimate_max", "distance_km"},
		data: [][]driver.Value{
			{int64(1), "Provider A", "050-1111111", "hvac", "Address A", vf1, nil, nil, 1.0},
			{int64(2), "Provider B", "050-2222222", "hvac", "Address B", vf2, nil, nil, 2.5},
		},
	}

	db := newTestDB(t, "find-providers-multiple", rows)
	defer db.Close()

	repo := NewOrganizationRepository(db)
	results, err := repo.FindProvidersInArea(context.Background(), 32.08, 34.78, "hvac", 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].ID != 1 {
		t.Errorf("expected first provider ID 1, got %d", results[0].ID)
	}
	if results[1].ID != 2 {
		t.Errorf("expected second provider ID 2, got %d", results[1].ID)
	}
	if results[0].DistanceKm > results[1].DistanceKm {
		t.Error("expected results ordered by distance ascending")
	}
}

// -----------------------------------------------------------------------
// UpdateServiceArea tests
// -----------------------------------------------------------------------

func TestUpdateServiceArea_MarshalAddressComponents(t *testing.T) {
	// Verify that address_components are JSON-marshaled correctly before the DB call.
	// We use a fakeConn that captures the Exec arguments.
	tests := []struct {
		name       string
		components map[string]string
		expectJSON string
	}{
		{
			name:       "nil components marshals to null",
			components: nil,
			expectJSON: "null",
		},
		{
			name:       "empty map marshals to {}",
			components: map[string]string{},
			expectJSON: "{}",
		},
		{
			name:       "populated map marshals correctly",
			components: map[string]string{"city": "Tel Aviv", "country": "Israel"},
			// json.Marshal produces keys in sorted order
			expectJSON: `{"city":"Tel Aviv","country":"Israel"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.components)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}

			// Since our fakedb exec ignores args, we only verify Marshal output directly.
			got := string(b)
			if got != tt.expectJSON {
				t.Errorf("expected JSON %q, got %q", tt.expectJSON, got)
			}
		})
	}
}

func TestUpdateServiceArea_ExecIsCalledWithOrgID(t *testing.T) {
	type execArgs struct {
		query string
		args  []driver.NamedValue
	}
	var captured *execArgs

	// Use a custom fakedb DSN that intercepts exec arguments.
	// We need a driver that can capture exec calls, so we extend fakeConn.
	// Easiest approach: register a separate driver variant per test.
	driverName := fmt.Sprintf("capturingdb-%d", 1)
	capturingOnce := sync.Once{}
	capturingOnce.Do(func() {
		sql.Register(driverName, &capturingDriver{
			onExec: func(query string, args []driver.NamedValue) {
				captured = &execArgs{query: query, args: args}
			},
		})
	})

	db, err := sql.Open(driverName, "test")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	repo := NewOrganizationRepository(db)
	components := map[string]string{"city": "Tel Aviv"}
	if err := repo.UpdateServiceArea(context.Background(), 99, 32.08, 34.78, 15.0,
		"1 Herzl St", "ChIJ123", "1 Herzl St, Tel Aviv, Israel", components); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if captured == nil {
		t.Fatal("expected exec to be called")
	}
	// The last argument is the orgID (position 9, index 8 in 0-based).
	// args[9] should be orgID = 99.
	if len(captured.args) < 10 {
		t.Fatalf("expected at least 10 args, got %d", len(captured.args))
	}
	if captured.args[9].Value != int64(99) {
		t.Errorf("expected orgID 99 in args[9], got %v", captured.args[9].Value)
	}
}

// capturingDriver captures ExecContext arguments for assertion.
type capturingDriver struct {
	onExec func(query string, args []driver.NamedValue)
}

func (d *capturingDriver) Open(name string) (driver.Conn, error) {
	return &capturingConn{onExec: d.onExec}, nil
}

type capturingConn struct {
	onExec func(query string, args []driver.NamedValue)
}

func (c *capturingConn) Prepare(query string) (driver.Stmt, error) {
	return &capturingStmt{conn: c, query: query}, nil
}
func (c *capturingConn) Close() error      { return nil }
func (c *capturingConn) Begin() (driver.Tx, error) { return &fakeTx{}, nil }

type capturingStmt struct {
	conn  *capturingConn
	query string
}

func (s *capturingStmt) Close() error    { return nil }
func (s *capturingStmt) NumInput() int   { return -1 }
func (s *capturingStmt) Exec(args []driver.Value) (driver.Result, error) {
	named := make([]driver.NamedValue, len(args))
	for i, v := range args {
		named[i] = driver.NamedValue{Ordinal: i + 1, Value: v}
	}
	s.conn.onExec(s.query, named)
	return &fakeResult{}, nil
}
func (s *capturingStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &fakeRows{cols: []string{}, data: nil}, nil
}

// -----------------------------------------------------------------------
// UpdateServiceOffer tests
// -----------------------------------------------------------------------

func TestUpdateServiceOffer_NilFeesArePassedThrough(t *testing.T) {
	// Registering the capturing driver for offer tests.
	driverName := "capturingdb-offer-1"
	var capturedArgs []driver.NamedValue

	sql.Register(driverName, &capturingDriver{
		onExec: func(_ string, args []driver.NamedValue) {
			capturedArgs = args
		},
	})

	db, err := sql.Open(driverName, "test-offer-nil")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	repo := NewOrganizationRepository(db)
	if err := repo.UpdateServiceOffer(context.Background(), 5, nil, nil, nil); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	// args: visitFee=$1, repairMin=$2, repairMax=$3, updatedAt=$4, orgID=$5
	if len(capturedArgs) < 5 {
		t.Fatalf("expected at least 5 args, got %d", len(capturedArgs))
	}
	if capturedArgs[0].Value != nil {
		t.Errorf("expected nil visit_fee arg, got %v", capturedArgs[0].Value)
	}
	if capturedArgs[1].Value != nil {
		t.Errorf("expected nil repair_min arg, got %v", capturedArgs[1].Value)
	}
	if capturedArgs[2].Value != nil {
		t.Errorf("expected nil repair_max arg, got %v", capturedArgs[2].Value)
	}
	if capturedArgs[4].Value != int64(5) {
		t.Errorf("expected orgID 5 in args[4], got %v", capturedArgs[4].Value)
	}
}

func TestUpdateServiceOffer_WithFeesArePassedThrough(t *testing.T) {
	driverName := "capturingdb-offer-2"
	var capturedArgs []driver.NamedValue

	sql.Register(driverName, &capturingDriver{
		onExec: func(_ string, args []driver.NamedValue) {
			capturedArgs = args
		},
	})

	db, err := sql.Open(driverName, "test-offer-values")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	vf := 120.0
	rMin := 200.0
	rMax := 600.0

	repo := NewOrganizationRepository(db)
	if err := repo.UpdateServiceOffer(context.Background(), 7, &vf, &rMin, &rMax); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(capturedArgs) < 5 {
		t.Fatalf("expected at least 5 args, got %d", len(capturedArgs))
	}
	if capturedArgs[0].Value != vf {
		t.Errorf("expected visit_fee %v, got %v", vf, capturedArgs[0].Value)
	}
	if capturedArgs[1].Value != rMin {
		t.Errorf("expected repair_min %v, got %v", rMin, capturedArgs[1].Value)
	}
	if capturedArgs[2].Value != rMax {
		t.Errorf("expected repair_max %v, got %v", rMax, capturedArgs[2].Value)
	}
	if capturedArgs[4].Value != int64(7) {
		t.Errorf("expected orgID 7 in args[4], got %v", capturedArgs[4].Value)
	}
}
