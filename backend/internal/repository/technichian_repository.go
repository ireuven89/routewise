package repository

import (
	"database/sql"
	"fmt"
	"github.com/ireuven89/routewise/internal/models"
	"time"
)

type TechnicianRepository struct {
	db *sql.DB
}

func NewTechnicianRepository(db *sql.DB) *TechnicianRepository {
	return &TechnicianRepository{db: db}
}

func (r *TechnicianRepository) Create(technician *models.Technician) error {
	query := `
		INSERT INTO technicians (organization_id, created_by, name, email, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		technician.OrganizationID,
		technician.CreatedBy,
		technician.Name,
		technician.Email,
		technician.Phone,
		technician.IsActive,
		now,
		now,
	).Scan(&technician.ID)

	if err != nil {
		return err
	}

	technician.CreatedAt = now
	technician.UpdatedAt = now
	return nil
}

func (r *TechnicianRepository) FindByID(id uint, organizationID uint) (*models.Technician, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, is_active, created_at, updated_at
		FROM technicians
		WHERE id = $1 AND organization_id = $2
	`

	technician := &models.Technician{}
	var email sql.NullString
	var createdBy sql.NullInt64

	err := r.db.QueryRow(query, id, organizationID).Scan(
		&technician.ID,
		&technician.OrganizationID,
		&createdBy,
		&technician.Name,
		&email,
		&technician.Phone,
		&technician.IsActive,
		&technician.CreatedAt,
		&technician.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("technician not found")
	}
	if err != nil {
		return nil, err
	}

	if createdBy.Valid {
		cb := uint(createdBy.Int64)
		technician.CreatedBy = &cb
	}
	if email.Valid {
		technician.Email = email.String
	}

	return technician, nil
}

func (r *TechnicianRepository) FindAll(organizationID uint, activeOnly bool) ([]*models.Technician, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, is_active, created_at, updated_at
		FROM technicians
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

	technicians := []*models.Technician{}

	for rows.Next() {
		technician := &models.Technician{}
		var email sql.NullString
		var createdBy sql.NullInt64

		err := rows.Scan(
			&technician.ID,
			&technician.OrganizationID,
			&createdBy,
			&technician.Name,
			&email,
			&technician.Phone,
			&technician.IsActive,
			&technician.CreatedAt,
			&technician.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if createdBy.Valid {
			cb := uint(createdBy.Int64)
			technician.CreatedBy = &cb
		}
		if email.Valid {
			technician.Email = email.String
		}

		technicians = append(technicians, technician)
	}

	return technicians, nil
}

func (r *TechnicianRepository) Update(technician *models.Technician) error {
	query := `
		UPDATE technicians
		SET name = $1, email = $2, phone = $3, is_active = $4, updated_at = $5
		WHERE id = $6 AND organization_id = $7
	`

	result, err := r.db.Exec(
		query,
		technician.Name,
		technician.Email,
		technician.Phone,
		technician.IsActive,
		time.Now(),
		technician.ID,
		technician.OrganizationID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("technician not found")
	}

	return nil
}

func (r *TechnicianRepository) Delete(id uint, organizationID uint) error {
	query := `DELETE FROM technicians WHERE id = $1 AND organization_id = $2`

	result, err := r.db.Exec(query, id, organizationID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("technician not found")
	}

	return nil
}
