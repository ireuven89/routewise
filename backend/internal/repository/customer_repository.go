package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/ireuven89/routewise/internal/models"
)

type customerDB struct {
	ID        uint      `sql:"id""`
	UserID    uint      `sql:"user_id" `
	Name      string    `sql:"name" `
	Email     string    `sql:"email"`
	Phone     string    `sql:"phone" `
	Address   string    `sql:"address"`
	Latitude  *float64  `sql:"latitude"`
	Longitude *float64  `sql:"longitude"`
	Notes     string    `sql:"notes"`
	CreatedAt time.Time `sql:"created_at"`
	UpdatedAt time.Time `sql:"updated_at"`
}

type CustomerRepository struct {
	db *sql.DB
}

func NewCustomerRepository(db *sql.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func (r *CustomerRepository) Create(customer *models.Customer) error {
	query := `
		INSERT INTO customers (user_id, name, email, phone, address, latitude, longitude, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	now := time.Now()
	err := r.db.QueryRow(
		query,
		customer.UserID,
		customer.Name,
		customer.Email,
		customer.Phone,
		customer.Address,
		customer.Latitude,
		customer.Longitude,
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

func (r *CustomerRepository) FindByID(id uint, userID uint) (*models.Customer, error) {
	query := `
		SELECT id, user_id, name, email, phone, address, latitude, longitude, notes, created_at, updated_at
		FROM customers
		WHERE id = $1 AND user_id = $2
	`

	customer := &models.Customer{}
	var email, latitude, longitude sql.NullString

	err := r.db.QueryRow(query, id, userID).Scan(
		&customer.ID,
		&customer.UserID,
		&customer.Name,
		&email,
		&customer.Phone,
		&customer.Address,
		&latitude,
		&longitude,
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

	return customer, nil
}

func (r *CustomerRepository) FindAll(userID uint, search string) ([]*models.Customer, error) {
	query := `
		SELECT id, user_id, name, email, phone, address, latitude, longitude, notes, created_at, updated_at
		FROM customers
		WHERE user_id = $1
	`

	args := []interface{}{userID}

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

		err := rows.Scan(
			&customer.ID,
			&customer.UserID,
			&customer.Name,
			&email,
			&customer.Phone,
			&customer.Address,
			&latitude,
			&longitude,
			&notes,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)

		if err != nil {
			fmt.Printf("failed scanning customer %v", err)
			return nil, err
		}

		// Handle nullable fields
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

		customers = append(customers, customer)
	}

	return customers, nil
}

func formatCustomer(db *customerDB) *models.Customer {

	return &models.Customer{
		ID:        db.ID,
		UserID:    db.UserID,
		Name:      db.Name,
		Email:     db.Email,
		Phone:     db.Phone,
		Address:   db.Address,
		Longitude: db.Longitude,
		Latitude:  db.Latitude,
		Notes:     db.Notes,
		CreatedAt: db.CreatedAt,
		UpdatedAt: db.UpdatedAt,
	}
}

func (r *CustomerRepository) Update(customer *models.Customer) error {
	query := `
		UPDATE customers
		SET name = $1, email = $2, phone = $3, address = $4, 
		    latitude = $5, longitude = $6, notes = $7, updated_at = $8
		WHERE id = $9 AND user_id = $10
	`

	result, err := r.db.Exec(
		query,
		customer.Name,
		customer.Email,
		customer.Phone,
		customer.Address,
		customer.Latitude,
		customer.Longitude,
		customer.Notes,
		time.Now(),
		customer.ID,
		customer.UserID,
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

func (r *CustomerRepository) Delete(id uint, userID uint) error {
	query := `DELETE FROM customers WHERE id = $1 AND user_id = $2`

	result, err := r.db.Exec(query, id, userID)
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
