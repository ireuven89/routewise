package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

// ErrJobNotFound: no job with that id in the organization (the service re-exports it, so
// handlers can map it to 404 with errors.Is).
var ErrJobNotFound = errors.New("job not found")

/*type JobRepository interface {
	CreateServiceCall(ctx context.Context, organizationID uint, request *models.CreateServiceCallRequest) (*models.CreateServiceCallResponse, error)
	Create(ctx context.Context, job *models.Job) error
	CreateTx(ctx context.Context, tx *sql.Tx, job *models.Job) (*uint, error)
	FindById(ctx context.Context, id string) (*models.Job, error)
	FindAll(ctx context.Context) ([]*models.Job, error)
}*/

type JobRepository struct {
	db           *sql.DB
	customerRepo *CustomerRepository
}

func NewJobRepository(db *sql.DB, customerRepo *CustomerRepository) *JobRepository {
	return &JobRepository{db: db,
		customerRepo: customerRepo,
	}
}

func (r *JobRepository) CreateServiceCall(ctx context.Context, organizationID uint, request *models.CreateServiceCallRequest) (*models.CreateServiceCallResponse, error) {
	tx, err := r.db.Begin()
	if err != nil {
		fmt.Printf("error starting transaction: %v\n", err)
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}

	defer tx.Rollback()

	createdCustomer := false
	customer, err := r.customerRepo.FindByPhoneTx(ctx, tx, organizationID, request.Customer.Phone)

	if err != nil {
		if err == CustomerNotFoundError {
			customer = &models.Customer{
				OrganizationID:    organizationID,
				Phone:             request.Customer.Phone,
				Email:             request.Customer.Email,
				Name:              request.Customer.Name,
				Address:           request.Customer.Address,
				Latitude:          request.Customer.Latitude,
				Longitude:         request.Customer.Longitude,
				GooglePlaceID:     request.Customer.GooglePlaceID,
				FormattedAddress:  request.Customer.FormattedAddress,
				AddressComponents: request.Customer.AddressComponents,
			}
			err = r.customerRepo.CreateTx(ctx, tx, customer)
			if err != nil {
				return nil, fmt.Errorf("could not create customer: %w", err)
			}
			createdCustomer = true
		} else {
			return nil, fmt.Errorf("could not find customer: %w", err)
		}
	}

	job := &models.Job{
		OrganizationID:  organizationID,
		CustomerID:      customer.ID,
		Title:           request.Job.Title,
		Description:     request.Job.Description,
		ScheduledAt:     request.Job.ScheduledDate,
		Status:          models.JobStatus(request.Job.Status),
		DurationMinutes: DefaultJobDurationMinutes,
	}
	if request.Job.CreatedBy != 0 {
		createdBy := request.Job.CreatedBy
		job.CreatedBy = &createdBy
	}

	if request.TechnicianID != nil {
		job.TechnicianID = request.TechnicianID
	}

	jobID, err := r.CreateTx(ctx, tx, job)

	if err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.CreateServiceCallResponse{
		CustomerID:      customer.ID,
		JobID:           *jobID,
		CustomerCreated: createdCustomer,
		Message:         "Service call created successfully",
	}, nil

}

// DefaultJobDurationMinutes is used when a job is created without a duration.
const DefaultJobDurationMinutes = 60

// setInitialAssignment offers a newly created job to its technician: pending until they accept.
func setInitialAssignment(job *models.Job) {
	if job.TechnicianID != nil {
		pending := models.AssignmentPending
		job.AssignmentStatus = &pending
	} else {
		job.AssignmentStatus = nil
	}
}

func (r *JobRepository) CreateTx(ctx context.Context, tx *sql.Tx, job *models.Job) (*uint, error) {
	query := `
		INSERT INTO jobs (organization_id, created_by, customer_id, technician_id, title, description, status, scheduled_at, duration_minutes, price, metadata, created_at, updated_at, assignment_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	setInitialAssignment(job)
	now := time.Now()
	err := tx.QueryRowContext(ctx,
		query,
		job.OrganizationID,
		job.CreatedBy,
		job.CustomerID,
		job.TechnicianID,
		job.Title,
		job.Description,
		job.Status,
		job.ScheduledAt,
		job.DurationMinutes,
		job.Price,
		job.Metadata,
		now,
		now,
		job.AssignmentStatus,
	).Scan(&job.ID)

	if err != nil {
		return nil, err
	}

	job.CreatedAt = now
	job.UpdatedAt = now

	return &job.ID, nil
}

func (r *JobRepository) Create(job *models.Job) error {
	query := `
		INSERT INTO jobs (organization_id, created_by, customer_id, technician_id, title, description, status, scheduled_at, duration_minutes, price, metadata, created_at, updated_at, assignment_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id
	`

	setInitialAssignment(job)
	now := time.Now()
	err := r.db.QueryRow(
		query,
		job.OrganizationID,
		job.CreatedBy,
		job.CustomerID,
		job.TechnicianID,
		job.Title,
		job.Description,
		job.Status,
		job.ScheduledAt,
		job.DurationMinutes,
		job.Price,
		job.Metadata,
		now,
		now,
		job.AssignmentStatus,
	).Scan(&job.ID)

	if err != nil {
		return err
	}

	job.CreatedAt = now
	job.UpdatedAt = now
	return nil
}

// jobSelect loads a job together with its customer, its assigned technician and the name of
// whoever last declined it. Columns are qualified because the joined tables share names.
const jobSelect = `
	SELECT j.id, j.organization_id, j.created_by, j.customer_id, j.technician_id, j.title,
	       COALESCE(j.description, ''), j.status, j.scheduled_at, j.completed_at, j.duration_minutes,
	       j.price, j.metadata, j.created_at, j.updated_at,
	       j.assignment_status, j.assignment_responded_at, j.declined_by_worker_id,
	       COALESCE(j.decline_reason, ''), j.declined_at,
	       c.id, COALESCE(c.name, ''), COALESCE(c.phone, ''), COALESCE(c.email, ''),
	       COALESCE(c.address, ''), c.latitude, c.longitude, COALESCE(c.formatted_address, ''),
	       w.id, COALESCE(w.name, ''), COALESCE(w.phone, ''),
	       COALESCE(dw.name, '')
	FROM jobs j
	LEFT JOIN customers c ON c.id = j.customer_id AND c.organization_id = j.organization_id
	LEFT JOIN workers w ON w.id = j.technician_id
	LEFT JOIN workers dw ON dw.id = j.declined_by_worker_id
`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanJob(row rowScanner) (*models.Job, error) {
	job := &models.Job{}
	var technicianID, createdBy, declinedBy, customerID, workerID sql.NullInt64
	var completedAt, respondedAt, declinedAt sql.NullTime
	var price, custLat, custLng sql.NullFloat64
	var metadata []byte
	var assignmentStatus sql.NullString
	var worker models.Worker

	err := row.Scan(
		&job.ID, &job.OrganizationID, &createdBy, &job.CustomerID, &technicianID, &job.Title,
		&job.Description, &job.Status, &job.ScheduledAt, &completedAt, &job.DurationMinutes,
		&price, &metadata, &job.CreatedAt, &job.UpdatedAt,
		&assignmentStatus, &respondedAt, &declinedBy, &job.DeclineReason, &declinedAt,
		&customerID, &job.Customer.Name, &job.Customer.Phone, &job.Customer.Email,
		&job.Customer.Address, &custLat, &custLng, &job.Customer.FormattedAddress,
		&workerID, &worker.Name, &worker.Phone,
		&job.DeclinedByName,
	)
	if err != nil {
		return nil, err
	}

	if createdBy.Valid {
		cb := uint(createdBy.Int64)
		job.CreatedBy = &cb
	}
	if technicianID.Valid {
		tid := uint(technicianID.Int64)
		job.TechnicianID = &tid
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if price.Valid {
		job.Price = &price.Float64
	}
	if len(metadata) > 0 {
		// Best effort: a malformed blob shouldn't make the whole job unreadable.
		_ = json.Unmarshal(metadata, &job.Metadata)
	}
	if assignmentStatus.Valid {
		s := models.AssignmentStatus(assignmentStatus.String)
		job.AssignmentStatus = &s
	}
	if respondedAt.Valid {
		job.AssignmentRespondedAt = &respondedAt.Time
	}
	if declinedBy.Valid {
		d := uint(declinedBy.Int64)
		job.DeclinedByWorkerID = &d
	}
	if declinedAt.Valid {
		job.DeclinedAt = &declinedAt.Time
	}
	if customerID.Valid {
		job.Customer.ID = uint(customerID.Int64)
		job.Customer.OrganizationID = job.OrganizationID
	}
	if custLat.Valid {
		job.Customer.Latitude = &custLat.Float64
	}
	if custLng.Valid {
		job.Customer.Longitude = &custLng.Float64
	}
	if workerID.Valid {
		worker.ID = uint(workerID.Int64)
		worker.OrganizationID = job.OrganizationID
		job.Worker = &worker
	}
	return job, nil
}

func (r *JobRepository) FindByID(id uint, organizationID uint) (*models.Job, error) {
	job, err := scanJob(r.db.QueryRow(jobSelect+` WHERE j.id = $1 AND j.organization_id = $2`, id, organizationID))
	if err == sql.ErrNoRows {
		return nil, ErrJobNotFound
	}
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *JobRepository) FindAll(organizationID uint, filters map[string]interface{}, sortBy string) ([]*models.Job, error) {
	query := jobSelect + ` WHERE j.organization_id = $1`

	args := []interface{}{organizationID}
	paramCount := 1

	// Apply filters
	if status, ok := filters["status"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND j.status = $%d", paramCount)
		args = append(args, status)
	}

	if techID, ok := filters["technician_id"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND j.technician_id = $%d", paramCount)
		args = append(args, techID)
	}

	if date, ok := filters["scheduled_date"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND DATE(j.scheduled_at) = $%d", paramCount)
		args = append(args, date)
	}

	// Add sorting
	switch sortBy {
	case "scheduled_at":
		query += " ORDER BY j.scheduled_at ASC"
	case "status":
		query += " ORDER BY j.status ASC, j.scheduled_at ASC"
	default:
		query += " ORDER BY j.created_at DESC"
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []*models.Job{}
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (r *JobRepository) Update(job *models.Job) error {
	query := `
		UPDATE jobs
		SET title = $1, description = $2, scheduled_at = $3, duration_minutes = $4,
		    price = $5, status = $6, metadata = $7, updated_at = $8
		WHERE id = $9 AND organization_id = $10
	`

	result, err := r.db.Exec(
		query,
		job.Title,
		job.Description,
		job.ScheduledAt,
		job.DurationMinutes,
		job.Price,
		job.Status,
		job.Metadata,
		time.Now(),
		job.ID,
		job.OrganizationID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// AssignTechnician (re)assigns a job. A different technician gets it as a new pending offer;
// re-assigning the same one keeps their answer; unassigning clears the offer.
// (All SET expressions see the row's old values, so technician_id is compared before it changes.)
func (r *JobRepository) AssignTechnician(jobID uint, organizationID uint, technicianID *uint) error {
	query := `
		UPDATE jobs
		SET assignment_status = CASE
		        WHEN $1::int IS NULL THEN NULL
		        WHEN technician_id IS NOT DISTINCT FROM $1::int THEN assignment_status
		        ELSE 'pending' END,
		    assignment_responded_at = CASE
		        WHEN technician_id IS NOT DISTINCT FROM $1::int THEN assignment_responded_at
		        ELSE NULL END,
		    technician_id = $1, updated_at = $2
		WHERE id = $3 AND organization_id = $4
	`

	result, err := r.db.Exec(query, technicianID, time.Now(), jobID, organizationID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

// AcceptAssignment marks a pending offer as accepted. The WHERE clause re-checks ownership and
// state so a concurrent reassign/decline can't be overwritten; false means nothing matched.
func (r *JobRepository) AcceptAssignment(jobID, organizationID, workerID uint) (bool, error) {
	now := time.Now()
	result, err := r.db.Exec(`
		UPDATE jobs
		SET assignment_status = 'accepted', assignment_responded_at = $1, updated_at = $1
		WHERE id = $2 AND organization_id = $3 AND technician_id = $4
		  AND assignment_status = 'pending' AND status NOT IN ('cancelled', 'completed')
	`, now, jobID, organizationID, workerID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// UpdateStatusAsWorker moves a technician's accepted job from one status to the next, only
// if it is still in the expected status (so a concurrent cancel by the owner wins).
// false means the job wasn't this worker's accepted job in status `from`.
func (r *JobRepository) UpdateStatusAsWorker(jobID, organizationID, workerID uint, from, to models.JobStatus) (bool, error) {
	now := time.Now()
	// completed_at gets its own parameter: reusing $1 in a comparison makes Postgres deduce
	// conflicting types for it (text vs varchar).
	var completedAt *time.Time
	if to == models.StatusCompleted {
		completedAt = &now
	}
	result, err := r.db.Exec(`
		UPDATE jobs
		SET status = $1, updated_at = $2, completed_at = COALESCE($7, completed_at)
		WHERE id = $3 AND organization_id = $4 AND technician_id = $5
		  AND assignment_status = 'accepted' AND status = $6
	`, to, now, jobID, organizationID, workerID, from, completedAt)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

// DeclineAssignment hands a scheduled job back to the unassigned pool and records who
// declined and why. false means the job wasn't this worker's scheduled job.
func (r *JobRepository) DeclineAssignment(jobID, organizationID, workerID uint, reason string) (bool, error) {
	now := time.Now()
	result, err := r.db.Exec(`
		UPDATE jobs
		SET technician_id = NULL, assignment_status = NULL, assignment_responded_at = NULL,
		    declined_by_worker_id = $1, decline_reason = NULLIF($2, ''), declined_at = $3, updated_at = $3
		WHERE id = $4 AND organization_id = $5 AND technician_id = $1 AND status = 'scheduled'
	`, workerID, reason, now, jobID, organizationID)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows > 0, err
}

func (r *JobRepository) UpdateStatus(jobID uint, organizationID uint, status models.JobStatus) error {
	query := `
		UPDATE jobs
		SET status = $1, updated_at = $2
		WHERE id = $3 AND organization_id = $4
	`

	// If status is completed, also set completed_at
	if status == models.StatusCompleted {
		query = `
			UPDATE jobs
			SET status = $1, completed_at = $2, updated_at = $3
			WHERE id = $4 AND organization_id = $5
		`
		result, err := r.db.Exec(query, status, time.Now(), time.Now(), jobID, organizationID)
		if err != nil {
			return err
		}

		rows, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rows == 0 {
			return ErrJobNotFound
		}

		return nil
	}

	result, err := r.db.Exec(query, status, time.Now(), jobID, organizationID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

func (r *JobRepository) Delete(id uint, organizationID uint) error {
	query := `DELETE FROM jobs WHERE id = $1 AND organization_id = $2`

	result, err := r.db.Exec(query, id, organizationID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return ErrJobNotFound
	}

	return nil
}

func (r *JobRepository) GetRevenueStats(organizationID uint) (*models.RevenueStats, error) {
	stats := &models.RevenueStats{RevenueByMonth: []models.MonthlyRevenue{}}

	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(price), 0) FROM jobs
		WHERE organization_id = $1 AND status = 'completed' AND price IS NOT NULL
	`, organizationID).Scan(&stats.Total)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`
		SELECT COALESCE(SUM(price), 0) FROM jobs
		WHERE organization_id = $1 AND status = 'completed' AND price IS NOT NULL
		  AND DATE_TRUNC('month', completed_at) = DATE_TRUNC('month', NOW())
	`, organizationID).Scan(&stats.ThisMonth)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`
		SELECT COALESCE(SUM(price), 0) FROM jobs
		WHERE organization_id = $1 AND status = 'completed' AND price IS NOT NULL
		  AND DATE_TRUNC('week', completed_at) = DATE_TRUNC('week', NOW())
	`, organizationID).Scan(&stats.ThisWeek)
	if err != nil {
		return nil, err
	}

	err = r.db.QueryRow(`
		SELECT COALESCE(AVG(price), 0) FROM jobs
		WHERE organization_id = $1 AND status = 'completed' AND price IS NOT NULL
	`, organizationID).Scan(&stats.AvgJobValue)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(`
		SELECT TO_CHAR(DATE_TRUNC('month', completed_at), 'YYYY-MM') AS month,
		       COALESCE(SUM(price), 0) AS revenue
		FROM jobs
		WHERE organization_id = $1 AND status = 'completed' AND price IS NOT NULL
		  AND completed_at >= NOW() - INTERVAL '6 months'
		GROUP BY DATE_TRUNC('month', completed_at)
		ORDER BY DATE_TRUNC('month', completed_at) ASC
	`, organizationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var mr models.MonthlyRevenue
		if err := rows.Scan(&mr.Month, &mr.Revenue); err != nil {
			return nil, err
		}
		stats.RevenueByMonth = append(stats.RevenueByMonth, mr)
	}

	return stats, nil
}

// CountOverlappingInProgressJobs returns the number of in_progress jobs for a technician
// whose time window overlaps [windowStart, windowEnd].
func (r *JobRepository) CountOverlappingInProgressJobs(
	technicianID, organizationID uint,
	windowStart, windowEnd time.Time,
) (int, error) {
	query := `
		SELECT COUNT(*) FROM jobs
		WHERE technician_id = $1 AND organization_id = $2
		  AND status = 'in_progress'
		  AND scheduled_at < $4
		  AND scheduled_at + (duration_minutes * interval '1 minute') > $3
	`
	var count int
	err := r.db.QueryRow(query, technicianID, organizationID, windowStart, windowEnd).Scan(&count)
	return count, err
}

// CountTodayJobsForTechnician returns the number of non-cancelled/completed jobs
// scheduled for a technician on the same calendar day as refTime.
func (r *JobRepository) CountTodayJobsForTechnician(
	technicianID, organizationID uint,
	refTime time.Time,
) (int, error) {
	query := `
		SELECT COUNT(*) FROM jobs
		WHERE technician_id = $1 AND organization_id = $2
		  AND status NOT IN ('cancelled', 'completed')
		  AND DATE(scheduled_at) = DATE($3)
	`
	var count int
	err := r.db.QueryRow(query, technicianID, organizationID, refTime).Scan(&count)
	return count, err
}

// Photo methods
func (r *JobRepository) AddPhoto(jobID uint, organizationID uint, url string, description string) error {
	// First verify the job belongs to the organization
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1 AND organization_id = $2)`
	err := r.db.QueryRow(checkQuery, jobID, organizationID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrJobNotFound
	}

	query := `
		INSERT INTO job_photos (job_id, url, description, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err = r.db.Exec(query, jobID, url, description, time.Now())
	return err
}

func (r *JobRepository) GetPhotos(jobID uint, organizationID uint) ([]map[string]interface{}, error) {
	// First verify the job belongs to the organization
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1 AND organization_id = $2)`
	err := r.db.QueryRow(checkQuery, jobID, organizationID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrJobNotFound
	}

	query := `
		SELECT id, job_id, url, description, created_at
		FROM job_photos
		WHERE job_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	photos := []map[string]interface{}{}

	for rows.Next() {
		var id, jobID uint
		var url, description string
		var createdAt time.Time

		err := rows.Scan(&id, &jobID, &url, &description, &createdAt)
		if err != nil {
			return nil, err
		}

		photo := map[string]interface{}{
			"id":          id,
			"job_id":      jobID,
			"url":         url,
			"description": description,
			"created_at":  createdAt,
		}
		photos = append(photos, photo)
	}

	return photos, nil
}

// Part methods
func (r *JobRepository) AddPart(jobID uint, organizationID uint, name string, quantity int, price float64) error {
	// First verify the job belongs to the organization
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1 AND organization_id = $2)`
	err := r.db.QueryRow(checkQuery, jobID, organizationID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrJobNotFound
	}

	query := `
		INSERT INTO job_parts (job_id, name, quantity, price, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = r.db.Exec(query, jobID, name, quantity, price, time.Now())
	return err
}

func (r *JobRepository) GetParts(jobID uint, organizationID uint) ([]map[string]interface{}, error) {
	// First verify the job belongs to the organization
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1 AND organization_id = $2)`
	err := r.db.QueryRow(checkQuery, jobID, organizationID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrJobNotFound
	}

	query := `
		SELECT id, job_id, name, quantity, price, created_at
		FROM job_parts
		WHERE job_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	parts := []map[string]interface{}{}

	for rows.Next() {
		var id, jobID uint
		var name string
		var quantity int
		var price float64
		var createdAt time.Time

		err := rows.Scan(&id, &jobID, &name, &quantity, &price, &createdAt)
		if err != nil {
			return nil, err
		}

		part := map[string]interface{}{
			"id":         id,
			"job_id":     jobID,
			"name":       name,
			"quantity":   quantity,
			"price":      price,
			"created_at": createdAt,
		}
		parts = append(parts, part)
	}

	return parts, nil
}

// Note methods
func (r *JobRepository) AddNote(jobID uint, organizationID uint, createdBy uint, note string) error {
	// First verify the job belongs to the organization
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1 AND organization_id = $2)`
	err := r.db.QueryRow(checkQuery, jobID, organizationID).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrJobNotFound
	}

	query := `
		INSERT INTO job_notes (job_id, created_by, note, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err = r.db.Exec(query, jobID, createdBy, note, time.Now())
	return err
}

func (r *JobRepository) GetNotes(jobID uint, organizationID uint) ([]map[string]interface{}, error) {
	// First verify the job belongs to the organization
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM jobs WHERE id = $1 AND organization_id = $2)`
	err := r.db.QueryRow(checkQuery, jobID, organizationID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrJobNotFound
	}

	query := `
		SELECT id, job_id, created_by, note, created_at
		FROM job_notes
		WHERE job_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []map[string]interface{}{}

	for rows.Next() {
		var id, jobID, createdBy uint
		var note string
		var createdAt time.Time

		err := rows.Scan(&id, &jobID, &createdBy, &note, &createdAt)
		if err != nil {
			return nil, err
		}

		noteItem := map[string]interface{}{
			"id":         id,
			"job_id":     jobID,
			"created_by": createdBy,
			"note":       note,
			"created_at": createdAt,
		}
		notes = append(notes, noteItem)
	}

	return notes, nil
}
