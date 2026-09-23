package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

// -----------------------------------------------------------------------
// AwardBid tests
//
// AwardBid's Query calls, in order (Exec calls always succeed and don't
// consume a queued result):
//   1. SELECT status ... FOR UPDATE
//   2. SELECT organization_id, price, eta_minutes (winning bid)
//   3. UPDATE ... 'rejected' RETURNING organization_id (losers)
//   4. FindByPhoneTx (winner's existing customer)
//   5. CustomerRepository.CreateTx (only when 4 found nothing)
//   6. JobRepository.CreateTx
// -----------------------------------------------------------------------

func rowsResult(cols []string, data ...[]driver.Value) seqResult {
	return seqResult{rows: &trackedRows{fakeRows: fakeRows{cols: cols, data: data}}}
}

func statusResult(status string) seqResult {
	return rowsResult([]string{"status"}, []driver.Value{status})
}

func winningBidResult(orgID int64, price float64, eta driver.Value) seqResult {
	return rowsResult([]string{"organization_id", "price", "eta_minutes"}, []driver.Value{orgID, price, eta})
}

func losersResult(orgIDs ...int64) seqResult {
	var data [][]driver.Value
	for _, id := range orgIDs {
		data = append(data, []driver.Value{id})
	}
	return rowsResult([]string{"organization_id"}, data...)
}

func noCustomerResult() seqResult {
	return rowsResult([]string{"id", "organization_id", "name", "email", "phone", "address"})
}

func existingCustomerResult(id, orgID int64) seqResult {
	return rowsResult([]string{"id", "organization_id", "name", "email", "phone", "address"},
		[]driver.Value{id, orgID, "Dana (existing)", "", "0501234567", "Old address"})
}

func idResult(id int64) seqResult {
	return rowsResult([]string{"id"}, []driver.Value{id})
}

func newAwardTestRepo(t *testing.T, dsn string, results ...seqResult) (*ServiceRequestRepository, *seqState) {
	t.Helper()
	db, state := newSeqTestDB(t, dsn, results...)
	t.Cleanup(func() { db.Close() })
	customerRepo := NewCustomerRepository(db)
	return NewServiceRequestRepository(db, customerRepo, NewJobRepository(db, customerRepo)), state
}

func awardTestRequest(preferred *time.Time) *models.ServiceRequest {
	return &models.ServiceRequest{
		ID:            42,
		ServiceType:   "hvac",
		Description:   "AC is leaking",
		CustomerName:  "Dana",
		CustomerPhone: "0501234567",
		Latitude:      32.08,
		Longitude:     34.78,
		Address:       "Rothschild 1, Tel Aviv",
		PreferredTime: preferred,
	}
}

func callContaining(t *testing.T, state *seqState, fragment string) seqCall {
	t.Helper()
	for _, c := range state.calls {
		if strings.Contains(c.query, fragment) {
			return c
		}
	}
	t.Fatalf("no query containing %q was executed", fragment)
	return seqCall{}
}

func TestAwardBid_NewCustomer_CreatesCustomerAndJob(t *testing.T) {
	preferred := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)
	repo, state := newAwardTestRepo(t, "award-new-customer",
		statusResult("open"),
		winningBidResult(7, 350.0, int64(30)),
		losersResult(8, 9),
		noCustomerResult(),
		idResult(100), // customer insert
		idResult(200), // job insert
	)

	res, err := repo.AwardBid(context.Background(), awardTestRequest(&preferred), 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.WinnerOrgID != 7 {
		t.Errorf("expected winner org 7, got %d", res.WinnerOrgID)
	}
	if len(res.LoserOrgIDs) != 2 || res.LoserOrgIDs[0] != 8 || res.LoserOrgIDs[1] != 9 {
		t.Errorf("expected losers [8 9], got %v", res.LoserOrgIDs)
	}
	if !res.CustomerCreated || res.CustomerID != 100 {
		t.Errorf("expected new customer 100, got created=%v id=%d", res.CustomerCreated, res.CustomerID)
	}
	if res.JobID != 200 {
		t.Errorf("expected job 200, got %d", res.JobID)
	}

	customer := callContaining(t, state, "INSERT INTO customers")
	if customer.args[0] != int64(7) {
		t.Errorf("customer should belong to winning org 7, got %v", customer.args[0])
	}
	if customer.args[2] != "Dana" || customer.args[4] != "0501234567" || customer.args[5] != "Rothschild 1, Tel Aviv" {
		t.Errorf("customer name/phone/address not taken from request: %v", customer.args[2:6])
	}
	if customer.args[6] != 32.08 || customer.args[7] != 34.78 {
		t.Errorf("customer coordinates not taken from request: %v, %v", customer.args[6], customer.args[7])
	}
	if customer.args[10] != nil {
		t.Errorf("expected NULL address_components (empty string is invalid jsonb), got %#v", customer.args[10])
	}

	job := callContaining(t, state, "INSERT INTO jobs")
	if job.args[0] != int64(7) || job.args[2] != int64(100) {
		t.Errorf("job should be org 7 / customer 100, got org=%v customer=%v", job.args[0], job.args[2])
	}
	if job.args[6] != string(models.StatusScheduled) {
		t.Errorf("expected scheduled status, got %v", job.args[6])
	}
	if got, ok := job.args[7].(time.Time); !ok || !got.Equal(preferred) {
		t.Errorf("expected job scheduled at preferred time %v, got %v", preferred, job.args[7])
	}
	if job.args[9] != 350.0 {
		t.Errorf("expected job price to be winning bid 350, got %v", job.args[9])
	}
}

func TestAwardBid_ExistingCustomer_ReusesCustomer(t *testing.T) {
	repo, state := newAwardTestRepo(t, "award-existing-customer",
		statusResult("open"),
		winningBidResult(7, 350.0, nil),
		losersResult(),
		existingCustomerResult(55, 7),
		idResult(200), // job insert
	)

	res, err := repo.AwardBid(context.Background(), awardTestRequest(nil), 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.CustomerCreated || res.CustomerID != 55 {
		t.Errorf("expected existing customer 55 reused, got created=%v id=%d", res.CustomerCreated, res.CustomerID)
	}
	if state.callCount() != 5 {
		t.Errorf("expected 5 queries (no customer insert), got %d", state.callCount())
	}
	for _, c := range state.calls {
		if strings.Contains(c.query, "INSERT INTO customers") {
			t.Error("existing customer must not be re-inserted")
		}
	}
	job := callContaining(t, state, "INSERT INTO jobs")
	if job.args[2] != int64(55) {
		t.Errorf("job should use existing customer 55, got %v", job.args[2])
	}
}

func TestAwardBid_Errors(t *testing.T) {
	jobErr := errors.New("insert failed")

	tests := []struct {
		name    string
		dsn     string
		results []seqResult
		wantErr error
	}{
		{
			name:    "request not found",
			dsn:     "award-err-not-found",
			results: []seqResult{rowsResult([]string{"status"})},
			wantErr: ErrServiceRequestNotFound,
		},
		{
			name:    "request already awarded",
			dsn:     "award-err-not-open",
			results: []seqResult{statusResult("awarded")},
			wantErr: ErrServiceRequestNotOpen,
		},
		{
			name: "bid not on this request",
			dsn:  "award-err-bid-not-found",
			results: []seqResult{
				statusResult("open"),
				rowsResult([]string{"organization_id", "price", "eta_minutes"}),
			},
			wantErr: ErrBidNotFound,
		},
		{
			name: "job insert fails",
			dsn:  "award-err-job-insert",
			results: []seqResult{
				statusResult("open"),
				winningBidResult(7, 350.0, nil),
				losersResult(),
				existingCustomerResult(55, 7),
				errResult(jobErr),
			},
			wantErr: jobErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, _ := newAwardTestRepo(t, tt.dsn, tt.results...)
			res, err := repo.AwardBid(context.Background(), awardTestRequest(nil), 5)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("expected error %v, got %v", tt.wantErr, err)
			}
			if res != nil {
				t.Errorf("expected nil result on error, got %+v", res)
			}
		})
	}
}

func TestAwardedJobScheduledAt(t *testing.T) {
	now := time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)
	preferred := time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		preferred *time.Time
		eta       sql.NullInt64
		want      time.Time
	}{
		{"preferred time wins over eta", &preferred, sql.NullInt64{Int64: 30, Valid: true}, preferred},
		{"eta from now when no preferred time", nil, sql.NullInt64{Int64: 45, Valid: true}, now.Add(45 * time.Minute)},
		{"now when neither given", nil, sql.NullInt64{}, now},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := awardedJobScheduledAt(&models.ServiceRequest{PreferredTime: tt.preferred}, tt.eta, now)
			if !got.Equal(tt.want) {
				t.Errorf("expected %v, got %v", tt.want, got)
			}
		})
	}
}

func TestJsonbParam(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want interface{}
	}{
		{"nil becomes NULL", nil, nil},
		{"empty becomes NULL", []byte{}, nil},
		{"json passes through", []byte(`{"a":1}`), []byte(`{"a":1}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := jsonbParam(tt.in)
			if tt.want == nil {
				if got != nil {
					t.Errorf("expected untyped nil, got %#v", got)
				}
				return
			}
			if string(got.([]byte)) != string(tt.want.([]byte)) {
				t.Errorf("expected %s, got %s", tt.want, got)
			}
		})
	}
}
