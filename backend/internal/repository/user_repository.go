package repository

import (
	"database/sql"
	"errors"
	"github.com/ireuven89/routewise/internal/models"
	"time"
)

type OrganizationUserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *OrganizationUserRepository {
	return &OrganizationUserRepository{db: db}
}

// CreateOrganizationWithUser creates both an organization and its first admin user in a transaction
func (r *OrganizationUserRepository) CreateOrganizationWithUser(org *models.Organization, user *models.OrganizationUser) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create organization
	orgQuery := `
		INSERT INTO organizations (name, phone, industry, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	now := time.Now()
	err = tx.QueryRow(orgQuery, org.Name, org.Phone, org.Industry, now, now).Scan(&org.ID)
	if err != nil {
		return err
	}
	org.CreatedAt = now
	org.UpdatedAt = now

	// Create organization user
	userQuery := `
		INSERT INTO organization_users (organization_id, email, password_hash, name, role, phone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	user.OrganizationID = org.ID
	err = tx.QueryRow(
		userQuery,
		user.OrganizationID,
		user.Email,
		user.Password,
		user.Name,
		user.Role,
		user.Phone,
		now,
		now,
	).Scan(&user.ID)
	if err != nil {
		return err
	}
	user.CreatedAt = now
	user.UpdatedAt = now

	return tx.Commit()
}

func (r *OrganizationUserRepository) FindByEmail(email string) (*models.OrganizationUser, error) {
	query := `
		SELECT id, organization_id, email, password_hash, name, role, phone, created_at, updated_at
		FROM organization_users
		WHERE email = $1
	`

	user := &models.OrganizationUser{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Role,
		&user.Phone,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *OrganizationUserRepository) FindByID(id uint) (*models.OrganizationUser, error) {
	query := `
		SELECT id, organization_id, email, password_hash, name, role, phone, created_at, updated_at
		FROM organization_users
		WHERE id = $1
	`

	user := &models.OrganizationUser{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.OrganizationID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Role,
		&user.Phone,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("user not found")
	}

	if err != nil {
		return nil, err
	}

	return user, nil
}

func (r *OrganizationUserRepository) FindOrganizationByID(id uint) (*models.Organization, error) {
	query := `
		SELECT id, name, phone, industry, created_at, updated_at
		FROM organizations
		WHERE id = $1
	`

	org := &models.Organization{}
	err := r.db.QueryRow(query, id).Scan(
		&org.ID,
		&org.Name,
		&org.Phone,
		&org.Industry,
		&org.CreatedAt,
		&org.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.New("organization not found")
	}

	if err != nil {
		return nil, err
	}

	return org, nil
}
