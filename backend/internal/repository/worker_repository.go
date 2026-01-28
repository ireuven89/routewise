package repository

import (
	"database/sql"
	"fmt"
	"github.com/ireuven89/routewise/internal/models"
	"time"
)

type WorkerRepository struct {
	db *sql.DB
}

func NewWorkerRepository(db *sql.DB) *WorkerRepository {
	return &WorkerRepository{db: db}
}

func (r *WorkerRepository) Create(worker *models.Worker) error {
	query := `
		INSERT INTO workers (organization_id, created_by, name, email, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		worker.OrganizationID,
		worker.CreatedBy,
		worker.Name,
		worker.Email,
		worker.Phone,
		worker.IsActive,
		now,
		now,
	).Scan(&worker.ID)

	if err != nil {
		return err
	}

	worker.CreatedAt = now
	worker.UpdatedAt = now
	return nil
}

func (r *WorkerRepository) FindByID(id uint, organizationID uint) (*models.Worker, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, is_active, created_at, updated_at
		FROM workers
		WHERE id = $1 AND organization_id = $2
	`

	worker := &models.Worker{}
	var email sql.NullString
	var createdBy sql.NullInt64

	err := r.db.QueryRow(query, id, organizationID).Scan(
		&worker.ID,
		&worker.OrganizationID,
		&createdBy,
		&worker.Name,
		&email,
		&worker.Phone,
		&worker.IsActive,
		&worker.CreatedAt,
		&worker.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("worker not found")
	}
	if err != nil {
		return nil, err
	}

	if createdBy.Valid {
		cb := uint(createdBy.Int64)
		worker.CreatedBy = &cb
	}
	if email.Valid {
		worker.Email = email.String
	}

	return worker, nil
}

func (r *WorkerRepository) FindByPhone(phone string, organizationID uint) (*models.Worker, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, is_active, created_at, updated_at
		FROM workers
		WHERE phone = $1 AND organization_id = $2
	`

	worker := &models.Worker{}
	var email sql.NullString
	var createdBy sql.NullInt64

	err := r.db.QueryRow(query, phone, organizationID).Scan(
		&worker.ID,
		&worker.OrganizationID,
		&createdBy,
		&worker.Name,
		&email,
		&worker.Phone,
		&worker.IsActive,
		&worker.CreatedAt,
		&worker.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("worker not found")
	}
	if err != nil {
		return nil, err
	}

	if createdBy.Valid {
		cb := uint(createdBy.Int64)
		worker.CreatedBy = &cb
	}
	if email.Valid {
		worker.Email = email.String
	}

	return worker, nil
}

func (r *WorkerRepository) FindAll(organizationID uint, activeOnly bool) ([]*models.Worker, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, is_active, created_at, updated_at
		FROM workers
		WHERE organization_id = $1
	`

	args := []interface{}{organizationID}

	if activeOnly {
		query += ` AND is_active = true`
	}

	query += ` ORDER BY name ASC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workers := []*models.Worker{}

	for rows.Next() {
		worker := &models.Worker{}
		var email sql.NullString
		var createdBy sql.NullInt64

		err := rows.Scan(
			&worker.ID,
			&worker.OrganizationID,
			&createdBy,
			&worker.Name,
			&email,
			&worker.Phone,
			&worker.IsActive,
			&worker.CreatedAt,
			&worker.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if createdBy.Valid {
			cb := uint(createdBy.Int64)
			worker.CreatedBy = &cb
		}
		if email.Valid {
			worker.Email = email.String
		}

		workers = append(workers, worker)
	}

	return workers, nil
}

func (r *WorkerRepository) Update(worker *models.Worker) error {
	query := `
		UPDATE workers
		SET name = $1, email = $2, phone = $3, is_active = $4, updated_at = $5
		WHERE id = $6 AND organization_id = $7
	`

	result, err := r.db.Exec(
		query,
		worker.Name,
		worker.Email,
		worker.Phone,
		worker.IsActive,
		time.Now(),
		worker.ID,
		worker.OrganizationID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("worker not found")
	}

	return nil
}

func (r *WorkerRepository) Delete(id uint, organizationID uint) error {
	query := `DELETE FROM workers WHERE id = $1 AND organization_id = $2`

	result, err := r.db.Exec(query, id, organizationID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("worker not found")
	}

	return nil
}
