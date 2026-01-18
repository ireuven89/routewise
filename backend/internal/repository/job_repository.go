package repository

import (
	"database/sql"
	"fmt"
	"github.com/ireuven89/routewise/internal/models"
	"time"
)

type JobRepository struct {
	db *sql.DB
}

func NewJobRepository(db *sql.DB) *JobRepository {
	return &JobRepository{db: db}
}

func (r *JobRepository) Create(job *models.Job) error {
	query := `
		INSERT INTO jobs (
			user_id, customer_id, technician_id, title, description, 
			status, scheduled_at, duration_minutes, price, metadata, 
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		job.UserID,
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

func (r *JobRepository) FindByID(id uint, userID uint) (*models.Job, error) {
	query := `
		SELECT 
			j.id, j.user_id, j.customer_id, j.technician_id, j.title, 
			j.description, j.status, j.scheduled_at, j.completed_at,
			j.duration_minutes, j.price, j.metadata, j.created_at, j.updated_at,
			c.id, c.name, c.email, c.phone, c.address, c.latitude, c.longitude
		FROM jobs j
		JOIN customers c ON j.customer_id = c.id
		WHERE j.id = $1 AND j.user_id = $2
	`

	job := &models.Job{
		Customer: models.Customer{},
	}

	var technicianID sql.NullInt64
	var completedAt sql.NullTime
	var price sql.NullFloat64
	var customerEmail, customerLat, customerLon sql.NullString

	err := r.db.QueryRow(query, id, userID).Scan(
		&job.ID,
		&job.UserID,
		&job.CustomerID,
		&technicianID,
		&job.Title,
		&job.Description,
		&job.Status,
		&job.ScheduledAt,
		&completedAt,
		&job.DurationMinutes,
		&price,
		&job.Metadata,
		&job.CreatedAt,
		&job.UpdatedAt,
		// Customer fields
		&job.Customer.ID,
		&job.Customer.Name,
		&customerEmail,
		&job.Customer.Phone,
		&job.Customer.Address,
		&customerLat,
		&customerLon,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("job not found")
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
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
	if customerEmail.Valid {
		job.Customer.Email = customerEmail.String
	}

	return job, nil
}

func (r *JobRepository) FindAll(userID uint, filters map[string]interface{}) ([]*models.Job, error) {
	query := `
		SELECT 
			j.id, j.user_id, j.customer_id, j.technician_id, j.title, 
			j.description, j.status, j.scheduled_at, j.completed_at,
			j.duration_minutes, j.price, j.created_at, j.updated_at,
			c.id, c.name, c.phone, c.address
		FROM jobs j
		JOIN customers c ON j.customer_id = c.id
		WHERE j.user_id = $1
	`

	args := []interface{}{userID}
	argCount := 1

	// Add filters
	if status, ok := filters["status"].(string); ok && status != "" {
		argCount++
		query += fmt.Sprintf(" AND j.status = $%d", argCount)
		args = append(args, status)
	}

	if techID, ok := filters["technician_id"].(uint); ok && techID > 0 {
		argCount++
		query += fmt.Sprintf(" AND j.technician_id = $%d", argCount)
		args = append(args, techID)
	}

	query += " ORDER BY j.scheduled_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := []*models.Job{}

	for rows.Next() {
		job := &models.Job{
			Customer: models.Customer{},
		}

		var technicianID sql.NullInt64
		var completedAt sql.NullTime
		var price sql.NullFloat64

		err := rows.Scan(
			&job.ID,
			&job.UserID,
			&job.CustomerID,
			&technicianID,
			&job.Title,
			&job.Description,
			&job.Status,
			&job.ScheduledAt,
			&completedAt,
			&job.DurationMinutes,
			&price,
			&job.CreatedAt,
			&job.UpdatedAt,
			// Customer fields
			&job.Customer.ID,
			&job.Customer.Name,
			&job.Customer.Phone,
			&job.Customer.Address,
		)

		if err != nil {
			return nil, err
		}

		// Handle nullable fields
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
		SET title = $1, description = $2, status = $3, scheduled_at = $4,
		    duration_minutes = $5, price = $6, metadata = $7, updated_at = $8
		WHERE id = $9 AND user_id = $10
	`

	result, err := r.db.Exec(
		query,
		job.Title,
		job.Description,
		job.Status,
		job.ScheduledAt,
		job.DurationMinutes,
		job.Price,
		job.Metadata,
		time.Now(),
		job.ID,
		job.UserID,
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

func (r *JobRepository) AssignTechnician(jobID uint, userID uint, technicianID *uint) error {
	query := `
		UPDATE jobs
		SET technician_id = $1, updated_at = $2
		WHERE id = $3 AND user_id = $4
	`

	result, err := r.db.Exec(query, technicianID, time.Now(), jobID, userID)
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

func (r *JobRepository) UpdateStatus(jobID uint, userID uint, status models.JobStatus) error {
	query := `
		UPDATE jobs
		SET status = $1, updated_at = $2
	`

	args := []interface{}{status, time.Now()}
	argCount := 2

	// If completing, set completed_at
	if status == models.StatusCompleted {
		argCount++
		query += fmt.Sprintf(", completed_at = $%d", argCount)
		args = append(args, time.Now())
	}

	argCount++
	query += fmt.Sprintf(" WHERE id = $%d", argCount)
	args = append(args, jobID)

	argCount++
	query += fmt.Sprintf(" AND user_id = $%d", argCount)
	args = append(args, userID)

	result, err := r.db.Exec(query, args...)
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

func (r *JobRepository) Delete(id uint, userID uint) error {
	query := `DELETE FROM jobs WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(query, id, userID)
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
