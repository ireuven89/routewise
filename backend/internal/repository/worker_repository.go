package repository

import (
	"database/sql"
	"encoding/json"
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
		INSERT INTO workers (
			organization_id, created_by, name, email, phone, is_active,
			home_address, home_latitude, home_longitude,
			home_google_place_id, home_formatted_address,
			home_address_components, home_geocoded_at,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	var homeAddressComponentsJSON []byte
	var err error
	if worker.HomeAddressComponents != nil {
		homeAddressComponentsJSON, err = json.Marshal(worker.HomeAddressComponents)
		if err != nil {
			return fmt.Errorf("failed to marshal home address components: %w", err)
		}
	}

	now := time.Now()
	err = r.db.QueryRow(
		query,
		worker.OrganizationID,
		worker.CreatedBy,
		worker.Name,
		worker.Email,
		worker.Phone,
		worker.IsActive,
		worker.HomeAddress,
		worker.HomeLatitude,
		worker.HomeLongitude,
		worker.HomeGooglePlaceID,
		worker.HomeFormattedAddress,
		homeAddressComponentsJSON,
		worker.HomeGeocodedAt,
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
		SELECT id, organization_id, created_by, name, email, phone, is_active,
		       home_address, home_latitude, home_longitude, home_google_place_id, home_formatted_address,
		       home_address_components, home_geocoded_at,
		       created_at, updated_at
		FROM workers
		WHERE id = $1 AND organization_id = $2
	`

	worker := &models.Worker{}
	var email sql.NullString
	var createdBy sql.NullInt64
	var homeAddress, homeGooglePlaceID, homeFormattedAddress sql.NullString
	var homeLatitude, homeLongitude sql.NullFloat64
	var homeAddressComponentsJSON []byte
	var homeGeocodedAt sql.NullTime

	err := r.db.QueryRow(query, id, organizationID).Scan(
		&worker.ID,
		&worker.OrganizationID,
		&createdBy,
		&worker.Name,
		&email,
		&worker.Phone,
		&worker.IsActive,
		&homeAddress,
		&homeLatitude,
		&homeLongitude,
		&homeGooglePlaceID,
		&homeFormattedAddress,
		&homeAddressComponentsJSON,
		&homeGeocodedAt,
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
	if homeAddress.Valid {
		worker.HomeAddress = homeAddress.String
	}
	if homeLatitude.Valid {
		lat := homeLatitude.Float64
		worker.HomeLatitude = &lat
	}
	if homeLongitude.Valid {
		lng := homeLongitude.Float64
		worker.HomeLongitude = &lng
	}
	if homeGooglePlaceID.Valid {
		worker.HomeGooglePlaceID = homeGooglePlaceID.String
	}
	if homeFormattedAddress.Valid {
		worker.HomeFormattedAddress = homeFormattedAddress.String
	}
	if len(homeAddressComponentsJSON) > 0 {
		if err := json.Unmarshal(homeAddressComponentsJSON, &worker.HomeAddressComponents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal home address components: %w", err)
		}
	}
	if homeGeocodedAt.Valid {
		worker.HomeGeocodedAt = &homeGeocodedAt.Time
	}

	return worker, nil
}

func (r *WorkerRepository) FindByPhoneAndCompanyCode(phone, companyCode string) (*models.Worker, error) {
	query := `
        SELECT w.id, w.organization_id, w.name, w.phone, w.email, w.is_active,  w.created_at, w.updated_at
        FROM workers w
        INNER JOIN organizations o ON w.organization_id = o.id
        WHERE w.phone = $1 AND o.company_code = $2
    `

	var worker models.Worker
	var email sql.NullString

	err := r.db.QueryRow(query, phone, companyCode).Scan(
		&worker.ID,
		&worker.OrganizationID,
		&worker.Name,
		&worker.Phone,
		&email,
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

	// Handle nullable fields
	if email.Valid {
		worker.Email = email.String
	}

	return &worker, nil
}

func (r *WorkerRepository) FindAll(organizationID uint, activeOnly bool) ([]*models.Worker, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, is_active,
		       home_address, home_latitude, home_longitude, home_google_place_id, home_formatted_address,
		       home_address_components, home_geocoded_at,
		       created_at, updated_at
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
		var homeAddress, homeGooglePlaceID, homeFormattedAddress sql.NullString
		var homeLatitude, homeLongitude sql.NullFloat64
		var homeAddressComponentsJSON []byte
		var homeGeocodedAt sql.NullTime

		err := rows.Scan(
			&worker.ID,
			&worker.OrganizationID,
			&createdBy,
			&worker.Name,
			&email,
			&worker.Phone,
			&worker.IsActive,
			&homeAddress,
			&homeLatitude,
			&homeLongitude,
			&homeGooglePlaceID,
			&homeFormattedAddress,
			&homeAddressComponentsJSON,
			&homeGeocodedAt,
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
		if homeAddress.Valid {
			worker.HomeAddress = homeAddress.String
		}
		if homeLatitude.Valid {
			lat := homeLatitude.Float64
			worker.HomeLatitude = &lat
		}
		if homeLongitude.Valid {
			lng := homeLongitude.Float64
			worker.HomeLongitude = &lng
		}
		if homeGooglePlaceID.Valid {
			worker.HomeGooglePlaceID = homeGooglePlaceID.String
		}
		if homeFormattedAddress.Valid {
			worker.HomeFormattedAddress = homeFormattedAddress.String
		}
		if len(homeAddressComponentsJSON) > 0 {
			if err := json.Unmarshal(homeAddressComponentsJSON, &worker.HomeAddressComponents); err != nil {
				fmt.Printf("failed unmarshaling home address components: %v", err)
			}
		}
		if homeGeocodedAt.Valid {
			worker.HomeGeocodedAt = &homeGeocodedAt.Time
		}

		workers = append(workers, worker)
	}

	return workers, nil
}

func (r *WorkerRepository) Update(worker *models.Worker) error {
	query := `
		UPDATE workers
		SET name = $1, email = $2, phone = $3, is_active = $4,
		    home_address = $5, home_latitude = $6, home_longitude = $7,
		    home_google_place_id = $8, home_formatted_address = $9,
		    home_address_components = $10, home_geocoded_at = $11,
		    updated_at = $12
		WHERE id = $13 AND organization_id = $14
	`

	// Marshal home address components to JSON
	var homeAddressComponentsJSON []byte
	var err error
	if worker.HomeAddressComponents != nil {
		homeAddressComponentsJSON, err = json.Marshal(worker.HomeAddressComponents)
		if err != nil {
			return fmt.Errorf("failed to marshal home address components: %w", err)
		}
	}

	result, err := r.db.Exec(
		query,
		worker.Name,
		worker.Email,
		worker.Phone,
		worker.IsActive,
		worker.HomeAddress,
		worker.HomeLatitude,
		worker.HomeLongitude,
		worker.HomeGooglePlaceID,
		worker.HomeFormattedAddress,
		homeAddressComponentsJSON,
		worker.HomeGeocodedAt,
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
