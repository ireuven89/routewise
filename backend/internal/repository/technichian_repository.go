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
		INSERT INTO technicians (user_id, name, email, phone, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		technician.UserID,
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

func (r *TechnicianRepository) FindByID(id uint, userID uint) (*models.Technician, error) {
	query := `
		SELECT id, user_id, name, email, phone, is_active, created_at, updated_at
		FROM technicians
		WHERE id = $1 AND user_id = $2
	`

	technician := &models.Technician{}
	var email sql.NullString

	err := r.db.QueryRow(query, id, userID).Scan(
		&technician.ID,
		&technician.UserID,
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

	if email.Valid {
		technician.Email = email.String
	}

	return technician, nil
}

func (r *TechnicianRepository) FindAll(userID uint, activeOnly bool) ([]*models.Technician, error) {
	query := `
		SELECT id, user_id, name, email, phone, is_active, created_at, updated_at
		FROM technicians
		WHERE user_id = $1
	`

	args := []interface{}{userID}

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

		err := rows.Scan(
			&technician.ID,
			&technician.UserID,
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
		WHERE id = $6 AND user_id = $7
	`

	result, err := r.db.Exec(
		query,
		technician.Name,
		technician.Email,
		technician.Phone,
		technician.IsActive,
		time.Now(),
		technician.ID,
		technician.UserID,
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

func (r *TechnicianRepository) Delete(id uint, userID uint) error {
	query := `DELETE FROM technicians WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(query, id, userID)
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
