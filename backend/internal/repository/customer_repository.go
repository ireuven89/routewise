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

var CustomerNotFoundError = errors.New("customer not found")

type customerDB struct {
	ID             uint      `sql:"id""`
	OrganizationID uint      `sql:"organization_id" `
	CreatedBy      *uint     `sql:"created_by"`
	Name           string    `sql:"name" `
	Email          string    `sql:"email"`
	Phone          string    `sql:"phone" `
	Address        string    `sql:"address"`
	Latitude       *float64  `sql:"latitude"`
	Longitude      *float64  `sql:"longitude"`
	Notes          string    `sql:"notes"`
	CreatedAt      time.Time `sql:"created_at"`
	UpdatedAt      time.Time `sql:"updated_at"`
}

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) FindByPhoneTx(ctx context.Context, tx *sql.Tx, organizationID uint, phone string) (*models.Customer, error) {
	q := `SELECT id, organization_id, name, email, phone, address 
		  FROM customers
		  WHERE phone = $1 AND organization_id = $2`

	res := &models.Customer{}
	err := tx.QueryRowContext(ctx, q, phone, organizationID).Scan(
		&res.ID,
		&res.OrganizationID,
		&res.Name,
		&res.Email,
		&res.Phone,
		&res.Address)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, CustomerNotFoundError
		}

		return nil, err
	}

	return res, nil
}

func (r *CustomerRepository) CreateTx(ctx context.Context, tx *sql.Tx, customer *models.Customer) error {
	query := `
		INSERT INTO customers (organization_id, created_by, name, email, phone, address, latitude, longitude,
		                       google_place_id, formatted_address, address_components, geocoded_at,
		                       notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	now := time.Now()

	// Marshal address components to JSON
	var addressComponentsJSON []byte
	var err error
	if customer.AddressComponents != nil {
		addressComponentsJSON, err = json.Marshal(customer.AddressComponents)
		if err != nil {
			return fmt.Errorf("failed to marshal address components: %w", err)
		}
	}

	err = tx.QueryRowContext(ctx,
		query,
		customer.OrganizationID,
		customer.CreatedBy,
		customer.Name,
		customer.Email,
		customer.Phone,
		customer.Address,
		customer.Latitude,
		customer.Longitude,
		customer.GooglePlaceID,
		customer.FormattedAddress,
		addressComponentsJSON,
		customer.GeocodedAt,
		customer.Notes,
		now,
		now,
	).Scan(&customer.ID)

	if err != nil {
		return err
	}

	return nil
}

func (r *CustomerRepository) Create(customer *models.Customer) error {
	query := `
		INSERT INTO customers (organization_id, created_by, name, email, phone, address, latitude, longitude,
		                       google_place_id, formatted_address, address_components, geocoded_at,
		                       notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id
	`

	now := time.Now()

	// Marshal address components to JSON
	var addressComponentsJSON []byte
	var err error
	if customer.AddressComponents != nil {
		addressComponentsJSON, err = json.Marshal(customer.AddressComponents)
		if err != nil {
			return fmt.Errorf("failed to marshal address components: %w", err)
		}
	}

	err = r.db.QueryRow(
		query,
		customer.OrganizationID,
		customer.CreatedBy,
		customer.Name,
		customer.Email,
		customer.Phone,
		customer.Address,
		customer.Latitude,
		customer.Longitude,
		customer.GooglePlaceID,
		customer.FormattedAddress,
		addressComponentsJSON,
		customer.GeocodedAt,
		customer.Notes,
		now,
		now,
	).Scan(&customer.ID)

	if err != nil {
		return err
	}

	customer.CreatedAt = now
	customer.UpdatedAt = now
	return nil
}

func (r *CustomerRepository) FindByID(id uint, organizationID uint) (*models.Customer, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, address, latitude, longitude,
		       google_place_id, formatted_address, address_components, geocoded_at,
		       notes, created_at, updated_at
		FROM customers
		WHERE id = $1 AND organization_id = $2
	`

	customer := &models.Customer{}
	var email, latitude, longitude sql.NullString
	var createdBy sql.NullInt64
	var googlePlaceID, formattedAddress sql.NullString
	var addressComponentsJSON []byte
	var geocodedAt sql.NullTime

	err := r.db.QueryRow(query, id, organizationID).Scan(
		&customer.ID,
		&customer.OrganizationID,
		&createdBy,
		&customer.Name,
		&email,
		&customer.Phone,
		&customer.Address,
		&latitude,
		&longitude,
		&googlePlaceID,
		&formattedAddress,
		&addressComponentsJSON,
		&geocodedAt,
		&customer.Notes,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("customer not found")
	}
	if err != nil {
		return nil, err
	}

	// Handle nullable fields
	if createdBy.Valid {
		cb := uint(createdBy.Int64)
		customer.CreatedBy = &cb
	}
	if email.Valid {
		customer.Email = email.String
	}
	if latitude.Valid {
		lat := parseFloat(latitude.String)
		customer.Latitude = &lat
	}
	if longitude.Valid {
		lon := parseFloat(longitude.String)
		customer.Longitude = &lon
	}
	if googlePlaceID.Valid {
		customer.GooglePlaceID = googlePlaceID.String
	}
	if formattedAddress.Valid {
		customer.FormattedAddress = formattedAddress.String
	}
	if len(addressComponentsJSON) > 0 {
		if err := json.Unmarshal(addressComponentsJSON, &customer.AddressComponents); err != nil {
			return nil, fmt.Errorf("failed to unmarshal address components: %w", err)
		}
	}
	if geocodedAt.Valid {
		customer.GeocodedAt = &geocodedAt.Time
	}

	return customer, nil
}

func (r *CustomerRepository) FindAll(organizationID uint, search string) ([]*models.Customer, error) {
	query := `
		SELECT id, organization_id, created_by, name, email, phone, address, latitude, longitude,
		       google_place_id, formatted_address, address_components, geocoded_at,
		       notes, created_at, updated_at
		FROM customers
		WHERE organization_id = $1
	`

	args := []interface{}{organizationID}

	// Add search filter
	if search != "" {
		query += ` AND (name ILIKE $2 OR phone ILIKE $2 OR address ILIKE $2)`
		args = append(args, "%"+search+"%")
	}

	query += ` ORDER BY name ASC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	customers := []*models.Customer{}

	for rows.Next() {
		customer := &models.Customer{}
		var email, notes sql.NullString
		var latitude, longitude sql.NullFloat64
		var createdBy sql.NullInt64
		var googlePlaceID, formattedAddress sql.NullString
		var addressComponentsJSON []byte
		var geocodedAt sql.NullTime

		err := rows.Scan(
			&customer.ID,
			&customer.OrganizationID,
			&createdBy,
			&customer.Name,
			&email,
			&customer.Phone,
			&customer.Address,
			&latitude,
			&longitude,
			&googlePlaceID,
			&formattedAddress,
			&addressComponentsJSON,
			&geocodedAt,
			&notes,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)

		if err != nil {
			fmt.Printf("failed scanning customer %v", err)
			return nil, err
		}

		// Handle nullable fields
		if createdBy.Valid {
			cb := uint(createdBy.Int64)
			customer.CreatedBy = &cb
		}
		if email.Valid {
			customer.Email = email.String
		}
		if latitude.Valid {
			lat := latitude.Float64
			customer.Latitude = &lat
		}
		if longitude.Valid {
			lon := longitude.Float64
			customer.Longitude = &lon
		}
		if notes.Valid {
			customer.Notes = notes.String
		}
		if googlePlaceID.Valid {
			customer.GooglePlaceID = googlePlaceID.String
		}
		if formattedAddress.Valid {
			customer.FormattedAddress = formattedAddress.String
		}
		if len(addressComponentsJSON) > 0 {
			if err := json.Unmarshal(addressComponentsJSON, &customer.AddressComponents); err != nil {
				fmt.Printf("failed unmarshaling address components: %v", err)
			}
		}
		if geocodedAt.Valid {
			customer.GeocodedAt = &geocodedAt.Time
		}

		customers = append(customers, customer)
	}

	return customers, nil
}

func formatCustomer(db *customerDB) *models.Customer {

	return &models.Customer{
		ID:             db.ID,
		OrganizationID: db.OrganizationID,
		CreatedBy:      db.CreatedBy,
		Name:           db.Name,
		Email:          db.Email,
		Phone:          db.Phone,
		Address:        db.Address,
		Longitude:      db.Longitude,
		Latitude:       db.Latitude,
		Notes:          db.Notes,
		CreatedAt:      db.CreatedAt,
		UpdatedAt:      db.UpdatedAt,
	}
}

func (r *CustomerRepository) Update(customer *models.Customer) error {
	query := `
		UPDATE customers
		SET name = $1, email = $2, phone = $3, address = $4,
		    latitude = $5, longitude = $6,
		    google_place_id = $7, formatted_address = $8, address_components = $9, geocoded_at = $10,
		    notes = $11, updated_at = $12
		WHERE id = $13 AND organization_id = $14
	`

	// Marshal address components to JSON
	var addressComponentsJSON []byte
	var err error
	if customer.AddressComponents != nil {
		addressComponentsJSON, err = json.Marshal(customer.AddressComponents)
		if err != nil {
			return fmt.Errorf("failed to marshal address components: %w", err)
		}
	}

	result, err := r.db.Exec(
		query,
		customer.Name,
		customer.Email,
		customer.Phone,
		customer.Address,
		customer.Latitude,
		customer.Longitude,
		customer.GooglePlaceID,
		customer.FormattedAddress,
		addressComponentsJSON,
		customer.GeocodedAt,
		customer.Notes,
		time.Now(),
		customer.ID,
		customer.OrganizationID,
	)

	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}

func (r *CustomerRepository) Delete(id uint, organizationID uint) error {
	query := `DELETE FROM customers WHERE id = $1 AND organization_id = $2`

	result, err := r.db.Exec(query, id, organizationID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return fmt.Errorf("customer not found")
	}

	return nil
}

// Helper function
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
