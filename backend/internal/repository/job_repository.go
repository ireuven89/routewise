package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

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
		return nil, fmt.Errorf("could not start transaction: %v", err)
	}
	defer tx.Rollback()

	if err != nil {
		fmt.Printf("error starting transaction: %v\n", err)
		return nil, fmt.Errorf("could not start transaction: %w", err)
	}

	createdCustomer := false
	customer, err := r.customerRepo.FindByPhoneTx(ctx, tx, organizationID, request.Customer.Phone)

	if err != nil {
		if err == CustomerNotFoundError {
			customer = &models.Customer{
				OrganizationID: organizationID,
				Phone:          request.Customer.Phone,
				Email:          request.Customer.Email,
				Name:           request.Customer.Name,
				Address:        request.Customer.Address,
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
		OrganizationID: organizationID,
		CustomerID:     customer.ID,
		Title:          request.Job.Title,
		Description:    request.Job.Description,
		ScheduledAt:    request.Job.ScheduledDate,
		Status:         models.JobStatus(request.Job.Status),
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

func (r *JobRepository) CreateTx(ctx context.Context, tx *sql.Tx, job *models.Job) (*uint, error) {
	query := `
		INSERT INTO jobs (organization_id, created_by, customer_id, technician_id, title, description, status, scheduled_at, duration_minutes, price, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

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
		INSERT INTO jobs (organization_id, created_by, customer_id, technician_id, title, description, status, scheduled_at, duration_minutes, price, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id
	`

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
	).Scan(&job.ID)

	if err != nil {
		return err
	}

	job.CreatedAt = now
	job.UpdatedAt = now
	return nil
}

func (r *JobRepository) FindByID(id uint, organizationID uint) (*models.Job, error) {
	query := `
		SELECT id, organization_id, created_by, customer_id, technician_id, title, description, status,
		       scheduled_at, completed_at, duration_minutes, price, metadata, created_at, updated_at
		FROM jobs
		WHERE id = $1 AND organization_id = $2
	`

	job := &models.Job{}
	var technicianID, createdBy sql.NullInt64
	var completedAt sql.NullTime
	var price sql.NullFloat64
	var metadata sql.NullString

	err := r.db.QueryRow(query, id, organizationID).Scan(
		&job.ID,
		&job.OrganizationID,
		&createdBy,
		&job.CustomerID,
		&technicianID,
		&job.Title,
		&job.Description,
		&job.Status,
		&job.ScheduledAt,
		&completedAt,
		&job.DurationMinutes,
		&price,
		&metadata,
		&job.CreatedAt,
		&job.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
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

	return job, nil
}

func (r *JobRepository) FindAll(organizationID uint, filters map[string]interface{}, sortBy string) ([]*models.Job, error) {
	query := `
		SELECT id, organization_id, created_by, customer_id, technician_id, title, description, status,
		       scheduled_at, completed_at, duration_minutes, price, metadata, created_at, updated_at
		FROM jobs
		WHERE organization_id = $1
	`

	args := []interface{}{organizationID}
	paramCount := 1

	// Apply filters
	if status, ok := filters["status"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND status = $%d", paramCount)
		args = append(args, status)
	}

	if techID, ok := filters["technician_id"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND technician_id = $%d", paramCount)
		args = append(args, techID)
	}

	if date, ok := filters["scheduled_date"]; ok {
		paramCount++
		query += fmt.Sprintf(" AND DATE(scheduled_at) = $%d", paramCount)
		args = append(args, date)
	}

	// Add sorting
	switch sortBy {
	case "scheduled_at":
		query += " ORDER BY scheduled_at ASC"
	case "status":
		query += " ORDER BY status ASC, scheduled_at ASC"
	default:
		query += " ORDER BY created_at DESC"
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []*models.Job{}

	for rows.Next() {
		job := &models.Job{}
		var technicianID, createdBy sql.NullInt64
		var completedAt sql.NullTime
		var price sql.NullFloat64
		var metadata sql.NullString

		err := rows.Scan(
			&job.ID,
			&job.OrganizationID,
			&createdBy,
			&job.CustomerID,
			&technicianID,
			&job.Title,
			&job.Description,
			&job.Status,
			&job.ScheduledAt,
			&completedAt,
			&job.DurationMinutes,
			&price,
			&metadata,
			&job.CreatedAt,
			&job.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Handle nullable fields
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

		jobs = append(jobs, job)
	}

	return jobs, nil
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
		return fmt.Errorf("job not found")
	}

	return nil
}

func (r *JobRepository) AssignTechnician(jobID uint, organizationID uint, technicianID *uint) error {
	query := `
		UPDATE jobs
		SET technician_id = $1, updated_at = $2
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
		return fmt.Errorf("job not found")
	}

	return nil
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
			return fmt.Errorf("job not found")
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
		return fmt.Errorf("job not found")
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
		return fmt.Errorf("job not found")
	}

	return nil
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
		return fmt.Errorf("job not found")
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
		return nil, fmt.Errorf("job not found")
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
		return fmt.Errorf("job not found")
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
		return nil, fmt.Errorf("job not found")
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
		return fmt.Errorf("job not found")
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
		return nil, fmt.Errorf("job not found")
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
