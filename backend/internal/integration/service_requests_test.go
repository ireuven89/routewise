//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var uniq int64

type provider struct {
	token string
	orgID uint
	phone string
}

// registerProvider signs up an hvac organization and gives it a 20km service area around
// (lat, lng), via the same endpoints the signup and /settings pages use.
func registerProvider(t *testing.T, label string, lat, lng float64) provider {
	t.Helper()
	n := atomic.AddInt64(&uniq, 1)
	phone := fmt.Sprintf("+97252%07d", n)

	var reg struct {
		Token        string `json:"token"`
		Organization struct {
			ID uint `json:"id"`
		} `json:"organization"`
	}
	status := doJSON(t, http.MethodPost, "/register", "", map[string]interface{}{
		"email":        fmt.Sprintf("%s-%d@integration.test", strings.ReplaceAll(strings.ToLower(label), " ", "-"), n),
		"password":     "integration-pass",
		"name":         label,
		"phone":        phone,
		"company_name": label,
		"industry":     "hvac",
	}, &reg)
	expectStatus(t, status, http.StatusCreated, "register "+label)

	status = doJSON(t, http.MethodPut, "/organization/service-area", reg.Token, map[string]interface{}{
		"latitude": lat, "longitude": lng, "address": label + " base", "service_radius_km": 20,
	}, nil)
	expectStatus(t, status, http.StatusOK, "set service area for "+label)

	return provider{token: reg.Token, orgID: reg.Organization.ID, phone: phone}
}

type postJob struct {
	name, phone, address string
	lat, lng             float64
	preferredTime        *time.Time
}

// postServiceRequest is the customer submitting the /find-service "Post Job" form.
func postServiceRequest(t *testing.T, j postJob) (id uint, accessToken string) {
	t.Helper()
	body := map[string]interface{}{
		"service_type":   "hvac",
		"description":    "AC is leaking",
		"customer_name":  j.name,
		"customer_phone": j.phone,
		"latitude":       j.lat,
		"longitude":      j.lng,
		"address":        j.address,
	}
	if j.preferredTime != nil {
		body["preferred_time"] = j.preferredTime.Format(time.RFC3339)
	}
	var resp struct {
		ID          uint   `json:"id"`
		AccessToken string `json:"access_token"`
	}
	status := doJSON(t, http.MethodPost, "/public/service-requests", "", body, &resp)
	expectStatus(t, status, http.StatusCreated, "post service request")
	return resp.ID, resp.AccessToken
}

func submitBid(t *testing.T, p provider, requestID uint, price float64, etaMinutes int) uint {
	t.Helper()
	var resp struct {
		Bid struct {
			ID uint `json:"id"`
		} `json:"bid"`
	}
	status := doJSON(t, http.MethodPut, fmt.Sprintf("/leads/%d/bid", requestID), p.token,
		map[string]interface{}{"price": price, "eta_minutes": etaMinutes, "message": "on it"}, &resp)
	expectStatus(t, status, http.StatusOK, "submit bid")
	return resp.Bid.ID
}

func award(t *testing.T, accessToken string, bidID uint) int {
	t.Helper()
	return doJSON(t, http.MethodPost, "/public/service-requests/"+accessToken+"/award",
		"", map[string]interface{}{"bid_id": bidID}, nil)
}

type jobRow struct {
	orgID, customerID uint
	status            string
	price             float64
	scheduledAt       time.Time
	source            string
	serviceRequestID  string
}

func awardedJob(t *testing.T, requestID uint) (jobID uint, j jobRow) {
	t.Helper()
	err := testDB.QueryRow(`
		SELECT j.id, j.organization_id, j.customer_id, j.status, j.price, j.scheduled_at,
		       j.metadata->>'source', j.metadata->>'service_request_id'
		FROM service_requests sr JOIN jobs j ON j.id = sr.job_id
		WHERE sr.id = $1`, requestID,
	).Scan(&jobID, &j.orgID, &j.customerID, &j.status, &j.price, &j.scheduledAt, &j.source, &j.serviceRequestID)
	if err != nil {
		t.Fatalf("load job for awarded request %d: %v", requestID, err)
	}
	return jobID, j
}

func countCustomers(t *testing.T, orgID uint, phone string) int {
	t.Helper()
	var n int
	if err := testDB.QueryRow(`SELECT count(*) FROM customers WHERE organization_id = $1 AND phone = $2`,
		orgID, phone).Scan(&n); err != nil {
		t.Fatalf("count customers: %v", err)
	}
	return n
}

func TestServiceRequestBidding_AwardHandsLeadToWinner(t *testing.T) {
	// Tel Aviv
	winner := registerProvider(t, "Winner HVAC", 32.0853, 34.7818)
	loser := registerProvider(t, "Loser HVAC", 32.0853, 34.7818)
	customerPhone := "+972500000101"

	requestID, token := postServiceRequest(t, postJob{
		name: "Dana", phone: customerPhone, address: "Rothschild 1, Tel Aviv", lat: 32.08, lng: 34.78,
	})

	// Both providers are alerted over WhatsApp, the customer gets a tracking link by SMS.
	waitForMessage(t, winner.phone, "whatsapp", "New hvac lead")
	waitForMessage(t, loser.phone, "whatsapp", "New hvac lead")
	waitForMessage(t, customerPhone, "sms", "/find-service/requests/"+token)

	// Bidders see the lead, but not the customer's contact details.
	for _, p := range []provider{winner, loser} {
		var leads struct {
			Leads []json.RawMessage `json:"leads"`
		}
		status := doJSON(t, http.MethodGet, "/leads", p.token, nil, &leads)
		expectStatus(t, status, http.StatusOK, "list leads")
		found := false
		for _, l := range leads.Leads {
			if strings.Contains(string(l), fmt.Sprintf(`"id":%d,`, requestID)) {
				found = true
			}
			if strings.Contains(string(l), customerPhone) || strings.Contains(string(l), "Dana") {
				t.Errorf("lead listing leaks customer contact details: %s", l)
			}
		}
		if !found {
			t.Fatalf("org %d does not see lead %d", p.orgID, requestID)
		}
	}

	loserBid := submitBid(t, loser, requestID, 450, 60)
	winnerBid := submitBid(t, winner, requestID, 350, 30)

	// The customer sees both bids, cheapest first.
	var view struct {
		Bids []struct {
			ID    uint    `json:"id"`
			Price float64 `json:"price"`
		} `json:"bids"`
	}
	expectStatus(t, doJSON(t, http.MethodGet, "/public/service-requests/"+token, "", nil, &view), http.StatusOK, "get request")
	if len(view.Bids) != 2 || view.Bids[0].ID != winnerBid || view.Bids[1].ID != loserBid {
		t.Fatalf("expected bids [%d %d] cheapest first, got %+v", winnerBid, loserBid, view.Bids)
	}

	awardedAt := time.Now().UTC()
	expectStatus(t, award(t, token, winnerBid), http.StatusOK, "award")

	// The winner now has the customer and a scheduled job for it.
	jobID, job := awardedJob(t, requestID)
	if job.orgID != winner.orgID {
		t.Errorf("job belongs to org %d, want winner %d", job.orgID, winner.orgID)
	}
	if job.status != "scheduled" || job.price != 350 {
		t.Errorf("expected scheduled job at the winning price 350, got %s / %v", job.status, job.price)
	}
	if want := awardedAt.Add(30 * time.Minute); job.scheduledAt.Sub(want).Abs() > time.Minute {
		t.Errorf("expected job scheduled at award time + 30min ETA (%v UTC), got %v", want, job.scheduledAt)
	}
	if job.source != "service_request" || job.serviceRequestID != fmt.Sprint(requestID) {
		t.Errorf("job metadata doesn't point back at the request: %s / %s", job.source, job.serviceRequestID)
	}

	var name, address string
	if err := testDB.QueryRow(`SELECT name, address FROM customers WHERE id = $1 AND organization_id = $2 AND phone = $3`,
		job.customerID, winner.orgID, customerPhone).Scan(&name, &address); err != nil {
		t.Fatalf("winner's customer not created from the request: %v", err)
	}
	if name != "Dana" || address != "Rothschild 1, Tel Aviv" {
		t.Errorf("customer created as %q / %q", name, address)
	}
	if n := countCustomers(t, loser.orgID, customerPhone); n != 0 {
		t.Errorf("losing org got %d customer rows for the customer", n)
	}

	// The job shows up in the winner's normal jobs API only.
	var winnerJobs, loserJobs []struct {
		ID uint `json:"id"`
	}
	expectStatus(t, doJSON(t, http.MethodGet, "/jobs", winner.token, nil, &winnerJobs), http.StatusOK, "winner jobs")
	expectStatus(t, doJSON(t, http.MethodGet, "/jobs", loser.token, nil, &loserJobs), http.StatusOK, "loser jobs")
	if !containsJob(winnerJobs, jobID) {
		t.Errorf("job %d missing from winner's /jobs", jobID)
	}
	if containsJob(loserJobs, jobID) {
		t.Errorf("job %d visible in loser's /jobs", jobID)
	}

	// Only the winner receives the customer's contact details.
	won := waitForMessage(t, winner.phone, "whatsapp", "You won the lead")
	if !strings.Contains(won.Body, customerPhone) || !strings.Contains(won.Body, "Dana") {
		t.Errorf("winner's message lacks customer contact: %q", won.Body)
	}
	lost := waitForMessage(t, loser.phone, "whatsapp", "awarded to another provider")
	if strings.Contains(lost.Body, customerPhone) {
		t.Errorf("loser's message leaks customer phone: %q", lost.Body)
	}
	waitForMessage(t, customerPhone, "sms", "You selected Winner HVAC")
}

func TestServiceRequestBidding_ReturningCustomerIsReused(t *testing.T) {
	// Haifa, so providers from other tests don't match.
	p := registerProvider(t, "Haifa HVAC", 32.7940, 34.9896)
	customerPhone := "+972500000202"
	preferred := time.Now().UTC().Add(48 * time.Hour).Truncate(time.Second)

	firstID, firstToken := postServiceRequest(t, postJob{
		name: "Noa", phone: customerPhone, address: "Herzl 5, Haifa", lat: 32.79, lng: 34.99,
	})
	expectStatus(t, award(t, firstToken, submitBid(t, p, firstID, 200, 45)), http.StatusOK, "first award")

	secondID, secondToken := postServiceRequest(t, postJob{
		name: "Noa", phone: customerPhone, address: "Herzl 5, Haifa", lat: 32.79, lng: 34.99,
		preferredTime: &preferred,
	})
	expectStatus(t, award(t, secondToken, submitBid(t, p, secondID, 300, 15)), http.StatusOK, "second award")

	_, first := awardedJob(t, firstID)
	_, second := awardedJob(t, secondID)
	if first.customerID != second.customerID {
		t.Errorf("returning customer duplicated: customer %d then %d", first.customerID, second.customerID)
	}
	if n := countCustomers(t, p.orgID, customerPhone); n != 1 {
		t.Errorf("expected exactly 1 customer row, got %d", n)
	}
	if !second.scheduledAt.Equal(preferred) {
		t.Errorf("expected second job at the customer's preferred time %v, got %v", preferred, second.scheduledAt)
	}
}

func TestServiceRequestBidding_AwardErrors(t *testing.T) {
	// Beersheba
	p := registerProvider(t, "Negev HVAC", 31.2520, 34.7915)
	requestID, token := postServiceRequest(t, postJob{
		name: "Avi", phone: "+972500000303", address: "Rager 1, Beersheba", lat: 31.25, lng: 34.79,
	})
	otherID, _ := postServiceRequest(t, postJob{
		name: "Avi", phone: "+972500000303", address: "Rager 1, Beersheba", lat: 31.25, lng: 34.79,
	})
	bid := submitBid(t, p, requestID, 250, 20)
	otherBid := submitBid(t, p, otherID, 260, 20)

	if status := award(t, token, otherBid); status != http.StatusNotFound {
		t.Errorf("awarding a bid from another request: expected 404, got %d", status)
	}
	if status := award(t, "00000000-0000-0000-0000-000000000000", bid); status != http.StatusNotFound {
		t.Errorf("awarding with an unknown token: expected 404, got %d", status)
	}

	expectStatus(t, award(t, token, bid), http.StatusOK, "award")
	if status := award(t, token, bid); status != http.StatusConflict {
		t.Errorf("awarding twice: expected 409, got %d", status)
	}

	var jobs int
	if err := testDB.QueryRow(`SELECT count(*) FROM jobs WHERE organization_id = $1`, p.orgID).Scan(&jobs); err != nil {
		t.Fatal(err)
	}
	if jobs != 1 {
		t.Errorf("expected exactly 1 job after the failed and repeated awards, got %d", jobs)
	}

	// Bids can't be changed once the request is awarded.
	status := doJSON(t, http.MethodPut, fmt.Sprintf("/leads/%d/bid", requestID), p.token,
		map[string]interface{}{"price": 999}, nil)
	if status != http.StatusConflict {
		t.Errorf("re-bidding on an awarded request: expected 409, got %d", status)
	}
}

func containsJob(jobs []struct {
	ID uint `json:"id"`
}, id uint) bool {
	for _, j := range jobs {
		if j.ID == id {
			return true
		}
	}
	return false
}
