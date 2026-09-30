//go:build integration

package integration

import (
	"database/sql"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ireuven89/routewise/pkg/utils"
)

// -----------------------------------------------------------------------
// Technician accept / decline of assigned jobs
// -----------------------------------------------------------------------

type technicianLogin struct {
	id    uint
	token string
}

// addTechnician inserts an active technician into the organization and mints the same
// kind of token the mobile OTP login issues (user_type "worker", worker ID in the
// organization_user_id claim). The workers HTTP handler isn't wired into the harness.
func addTechnician(t *testing.T, orgID uint, name string) technicianLogin {
	t.Helper()
	n := atomic.AddInt64(&uniq, 1)
	var id uint
	if err := testDB.QueryRow(`
		INSERT INTO workers (organization_id, name, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, true, NOW(), NOW()) RETURNING id`,
		orgID, name, fmt.Sprintf("+97253%07d", n)).Scan(&id); err != nil {
		t.Fatalf("insert technician: %v", err)
	}
	token, err := utils.GenerateToken(id, orgID, "", "", "worker")
	if err != nil {
		t.Fatalf("generate worker token: %v", err)
	}
	return technicianLogin{id: id, token: token}
}

func addCustomer(t *testing.T, orgID uint) uint {
	t.Helper()
	n := atomic.AddInt64(&uniq, 1)
	var id uint
	if err := testDB.QueryRow(`
		INSERT INTO customers (organization_id, name, phone, address, created_at, updated_at)
		VALUES ($1, 'Dana', $2, 'Herzl 1, Tel Aviv', NOW(), NOW()) RETURNING id`,
		orgID, fmt.Sprintf("+97254%07d", n)).Scan(&id); err != nil {
		t.Fatalf("insert customer: %v", err)
	}
	return id
}

// createAssignedJob is the owner creating a job with a technician in the web app.
func createAssignedJob(t *testing.T, owner provider, customerID uint, technicianID *uint) uint {
	t.Helper()
	var job struct {
		ID uint `json:"id"`
	}
	body := map[string]interface{}{
		"customer_id":  customerID,
		"title":        "Fix AC",
		"scheduled_at": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
	}
	if technicianID != nil {
		body["technician_id"] = *technicianID
	}
	status := doJSON(t, http.MethodPost, "/jobs", owner.token, body, &job)
	expectStatus(t, status, http.StatusCreated, "create job")
	return job.ID
}

type assignmentRow struct {
	technicianID     sql.NullInt64
	assignmentStatus sql.NullString
	respondedAt      sql.NullTime
	status           string
	completedAt      sql.NullTime
	declinedBy       sql.NullInt64
	declineReason    sql.NullString
	declinedAt       sql.NullTime
}

func loadAssignment(t *testing.T, jobID uint) assignmentRow {
	t.Helper()
	var r assignmentRow
	if err := testDB.QueryRow(`
		SELECT technician_id, assignment_status, assignment_responded_at, status, completed_at,
		       declined_by_worker_id, decline_reason, declined_at
		FROM jobs WHERE id = $1`, jobID).Scan(
		&r.technicianID, &r.assignmentStatus, &r.respondedAt, &r.status, &r.completedAt,
		&r.declinedBy, &r.declineReason, &r.declinedAt); err != nil {
		t.Fatalf("load job %d: %v", jobID, err)
	}
	return r
}

func setStatus(t *testing.T, token string, jobID uint, status string) int {
	t.Helper()
	return doJSON(t, http.MethodPatch, fmt.Sprintf("/jobs/%d/status", jobID), token,
		map[string]string{"status": status}, nil)
}

func TestJobAssignment_AcceptStartComplete(t *testing.T) {
	owner := registerProvider(t, "Accept Flow HVAC", 32.08, 34.78)
	tech := addTechnician(t, owner.orgID, "Avi")
	other := addTechnician(t, owner.orgID, "Moshe")
	customerID := addCustomer(t, owner.orgID)

	jobID := createAssignedJob(t, owner, customerID, &tech.id)

	r := loadAssignment(t, jobID)
	if !r.assignmentStatus.Valid || r.assignmentStatus.String != "pending" {
		t.Fatalf("new assigned job assignment_status = %v, want pending", r.assignmentStatus)
	}

	// The technician sees it as a pending offer in their own list.
	var jobs []struct {
		ID               uint    `json:"id"`
		AssignmentStatus *string `json:"assignment_status"`
	}
	expectStatus(t, doJSON(t, http.MethodGet, "/jobs", tech.token, nil, &jobs), http.StatusOK, "technician list jobs")
	if len(jobs) != 1 || jobs[0].ID != jobID || jobs[0].AssignmentStatus == nil || *jobs[0].AssignmentStatus != "pending" {
		t.Fatalf("technician jobs = %+v, want only job %d pending", jobs, jobID)
	}

	// Another technician can neither see nor accept it.
	expectStatus(t, doJSON(t, http.MethodGet, fmt.Sprintf("/jobs/%d", jobID), other.token, nil, nil),
		http.StatusNotFound, "other technician get job")
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/accept", jobID), other.token, nil, nil),
		http.StatusNotFound, "other technician accept")

	// Owners can't accept on the technician's behalf.
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/accept", jobID), owner.token, nil, nil),
		http.StatusForbidden, "owner accept")

	// Technicians can't create jobs.
	expectStatus(t, doJSON(t, http.MethodPost, "/jobs", tech.token, map[string]interface{}{
		"customer_id": customerID, "title": "x", "scheduled_at": time.Now().UTC().Format(time.RFC3339),
	}, nil), http.StatusForbidden, "technician create job")

	// Starting before accepting is refused.
	expectStatus(t, setStatus(t, tech.token, jobID, "in_progress"), http.StatusConflict, "start before accept")

	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/accept", jobID), tech.token, nil, nil),
		http.StatusOK, "accept")
	r = loadAssignment(t, jobID)
	if r.assignmentStatus.String != "accepted" || !r.respondedAt.Valid {
		t.Fatalf("after accept: status=%v responded_at=%v", r.assignmentStatus, r.respondedAt)
	}

	// Accepting twice is a conflict.
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/accept", jobID), tech.token, nil, nil),
		http.StatusConflict, "accept twice")

	// Re-assigning the same technician keeps their answer.
	expectStatus(t, doJSON(t, http.MethodPatch, fmt.Sprintf("/jobs/%d/assign", jobID), owner.token,
		map[string]interface{}{"technician_id": tech.id}, nil), http.StatusOK, "reassign same technician")
	if r = loadAssignment(t, jobID); r.assignmentStatus.String != "accepted" {
		t.Fatalf("reassigning the same technician reset the answer to %v", r.assignmentStatus)
	}

	// Technicians can't cancel.
	expectStatus(t, setStatus(t, tech.token, jobID, "cancelled"), http.StatusBadRequest, "technician cancel")

	expectStatus(t, setStatus(t, tech.token, jobID, "in_progress"), http.StatusOK, "start")
	// Can't decline once started.
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/decline", jobID), tech.token, nil, nil),
		http.StatusConflict, "decline in-progress job")
	expectStatus(t, setStatus(t, tech.token, jobID, "completed"), http.StatusOK, "complete")

	r = loadAssignment(t, jobID)
	if r.status != "completed" || !r.completedAt.Valid {
		t.Fatalf("after complete: status=%s completed_at=%v", r.status, r.completedAt)
	}
	if !r.technicianID.Valid || uint(r.technicianID.Int64) != tech.id {
		t.Fatalf("technician changed to %v", r.technicianID)
	}
}

func TestJobAssignment_Decline(t *testing.T) {
	owner := registerProvider(t, "Decline Flow HVAC", 32.08, 34.78)
	tech := addTechnician(t, owner.orgID, "Avi")
	replacement := addTechnician(t, owner.orgID, "Moshe")
	customerID := addCustomer(t, owner.orgID)

	jobID := createAssignedJob(t, owner, customerID, &tech.id)

	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/decline", jobID), tech.token,
		map[string]string{"reason": "Too far away"}, nil), http.StatusOK, "decline")

	r := loadAssignment(t, jobID)
	if r.technicianID.Valid || r.assignmentStatus.Valid || r.respondedAt.Valid {
		t.Fatalf("declined job still assigned: tech=%v status=%v responded=%v", r.technicianID, r.assignmentStatus, r.respondedAt)
	}
	if !r.declinedBy.Valid || uint(r.declinedBy.Int64) != tech.id {
		t.Fatalf("declined_by_worker_id = %v, want %d", r.declinedBy, tech.id)
	}
	if r.declineReason.String != "Too far away" || !r.declinedAt.Valid {
		t.Fatalf("decline_reason=%v declined_at=%v", r.declineReason, r.declinedAt)
	}
	if r.status != "scheduled" {
		t.Fatalf("status = %s, want scheduled", r.status)
	}

	// The job is gone from the technician's view.
	expectStatus(t, doJSON(t, http.MethodGet, fmt.Sprintf("/jobs/%d", jobID), tech.token, nil, nil),
		http.StatusNotFound, "declining technician get job")
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/accept", jobID), tech.token, nil, nil),
		http.StatusNotFound, "accept after decline")

	// The owner sees who declined and why.
	var ownerView struct {
		WorkerID       *uint   `json:"worker_id"`
		DeclinedByName string  `json:"declined_by_name"`
		DeclineReason  string  `json:"decline_reason"`
		Assignment     *string `json:"assignment_status"`
	}
	expectStatus(t, doJSON(t, http.MethodGet, fmt.Sprintf("/jobs/%d", jobID), owner.token, nil, &ownerView),
		http.StatusOK, "owner get declined job")
	if ownerView.WorkerID != nil || ownerView.Assignment != nil ||
		ownerView.DeclinedByName != "Avi" || ownerView.DeclineReason != "Too far away" {
		t.Fatalf("owner view = %+v", ownerView)
	}

	// Reassigning to someone else makes a fresh pending offer they can accept.
	expectStatus(t, doJSON(t, http.MethodPatch, fmt.Sprintf("/jobs/%d/assign", jobID), owner.token,
		map[string]interface{}{"technician_id": replacement.id}, nil), http.StatusOK, "reassign")
	if r = loadAssignment(t, jobID); r.assignmentStatus.String != "pending" {
		t.Fatalf("reassigned job assignment_status = %v, want pending", r.assignmentStatus)
	}
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/accept", jobID), replacement.token, nil, nil),
		http.StatusOK, "replacement accepts")

	// Unassigning clears the offer.
	expectStatus(t, doJSON(t, http.MethodPatch, fmt.Sprintf("/jobs/%d/assign", jobID), owner.token,
		map[string]interface{}{"technician_id": nil}, nil), http.StatusOK, "unassign")
	if r = loadAssignment(t, jobID); r.technicianID.Valid || r.assignmentStatus.Valid {
		t.Fatalf("unassigned job: tech=%v status=%v", r.technicianID, r.assignmentStatus)
	}
}

func TestJobAssignment_DeclineWithoutReasonAndCrossOrg(t *testing.T) {
	owner := registerProvider(t, "Decline NoReason HVAC", 32.08, 34.78)
	otherOrg := registerProvider(t, "Other Org HVAC", 32.08, 34.78)
	tech := addTechnician(t, owner.orgID, "Avi")
	foreignTech := addTechnician(t, otherOrg.orgID, "Foreign")
	customerID := addCustomer(t, owner.orgID)

	// Assigning a technician from another organization is rejected.
	status := doJSON(t, http.MethodPost, "/jobs", owner.token, map[string]interface{}{
		"customer_id": customerID, "title": "x", "technician_id": foreignTech.id,
		"scheduled_at": time.Now().UTC().Format(time.RFC3339),
	}, nil)
	expectStatus(t, status, http.StatusBadRequest, "create job with foreign technician")

	jobID := createAssignedJob(t, owner, customerID, &tech.id)

	// A technician of another organization can't touch it.
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/decline", jobID), foreignTech.token, nil, nil),
		http.StatusNotFound, "foreign technician decline")

	// No body at all: reason is optional and stored as NULL.
	expectStatus(t, doJSON(t, http.MethodPost, fmt.Sprintf("/jobs/%d/decline", jobID), tech.token, nil, nil),
		http.StatusOK, "decline without reason")
	if r := loadAssignment(t, jobID); r.declineReason.Valid {
		t.Fatalf("empty reason stored as %q, want NULL", r.declineReason.String)
	}
}
