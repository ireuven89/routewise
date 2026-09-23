package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/ireuven89/routewise/internal/models"
)

var (
	ErrServiceRequestNotFound = errors.New("service request not found")
	ErrServiceRequestNotOpen  = errors.New("service request is not open")
	ErrBidLocked              = errors.New("bid can no longer be modified")
	ErrBidNotFound            = errors.New("bid not found")
)

type ServiceRequestRepository struct {
	db *sql.DB
}

func NewServiceRequestRepository(db *sql.DB) *ServiceRequestRepository {
	return &ServiceRequestRepository{db: db}
}

// MatchedOrg is the fan-out target view of an organization matched to a new service request.
type MatchedOrg struct {
	ID    uint
	Name  string
	Phone string
}

func (r *ServiceRequestRepository) Create(ctx context.Context, sr *models.ServiceRequest) error {
	query := `
		INSERT INTO service_requests
			(access_token, service_type, description, customer_name, customer_phone,
			 latitude, longitude, address, preferred_time, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRowContext(ctx, query,
		sr.AccessToken, sr.ServiceType, sr.Description, sr.CustomerName, sr.CustomerPhone,
		sr.Latitude, sr.Longitude, sr.Address, sr.PreferredTime, sr.Status,
	).Scan(&sr.ID, &sr.CreatedAt, &sr.UpdatedAt)
}

func (r *ServiceRequestRepository) FindByToken(ctx context.Context, token string) (*models.ServiceRequest, error) {
	query := `
		SELECT id, access_token, service_type, description, customer_name, customer_phone,
		       latitude, longitude, address, preferred_time, status, awarded_bid_id,
		       created_at, updated_at
		FROM service_requests
		WHERE access_token = $1
	`
	return r.scanServiceRequest(r.db.QueryRowContext(ctx, query, token))
}

func (r *ServiceRequestRepository) FindByID(ctx context.Context, id uint) (*models.ServiceRequest, error) {
	query := `
		SELECT id, access_token, service_type, description, customer_name, customer_phone,
		       latitude, longitude, address, preferred_time, status, awarded_bid_id,
		       created_at, updated_at
		FROM service_requests
		WHERE id = $1
	`
	return r.scanServiceRequest(r.db.QueryRowContext(ctx, query, id))
}

func (r *ServiceRequestRepository) scanServiceRequest(row *sql.Row) (*models.ServiceRequest, error) {
	sr := &models.ServiceRequest{}
	var preferredTime sql.NullTime
	var awardedBidID sql.NullInt64

	err := row.Scan(
		&sr.ID, &sr.AccessToken, &sr.ServiceType, &sr.Description, &sr.CustomerName, &sr.CustomerPhone,
		&sr.Latitude, &sr.Longitude, &sr.Address, &preferredTime, &sr.Status, &awardedBidID,
		&sr.CreatedAt, &sr.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrServiceRequestNotFound
	}
	if err != nil {
		return nil, err
	}

	if preferredTime.Valid {
		sr.PreferredTime = &preferredTime.Time
	}
	if awardedBidID.Valid {
		id := uint(awardedBidID.Int64)
		sr.AwardedBidID = &id
	}
	return sr, nil
}

// FindMatchingOrganizations returns every organization (up to cap) whose service area
// covers (lat, lng) and whose industry matches serviceType, adapting the Haversine pattern
// from OrganizationRepository.FindProvidersInArea. Unlike that customer-facing search
// (which caps results for UX), a broadcast should reach every eligible org in range.
func (r *ServiceRequestRepository) FindMatchingOrganizations(ctx context.Context, lat, lng float64, serviceType string, cap int) ([]*MatchedOrg, error) {
	query := `
		SELECT id, name, phone
		FROM (
		    SELECT id, name, phone, service_radius_km,
		           (6371 * acos(
		               LEAST(1.0,
		                   cos(radians($1)) * cos(radians(latitude)) *
		                   cos(radians(longitude) - radians($2)) +
		                   sin(radians($1)) * sin(radians(latitude))
		               )
		           )) AS distance_km
		    FROM organizations
		    WHERE latitude IS NOT NULL
		      AND longitude IS NOT NULL
		      AND industry = $3
		      AND phone <> ''
		) sub
		WHERE distance_km <= service_radius_km
		ORDER BY distance_km ASC
		LIMIT $4
	`
	rows, err := r.db.QueryContext(ctx, query, lat, lng, serviceType, cap)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var matches []*MatchedOrg
	for rows.Next() {
		m := &MatchedOrg{}
		if err := rows.Scan(&m.ID, &m.Name, &m.Phone); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}

// FindOpenRequestsForOrg returns open requests within orgID's own service area/industry,
// paired index-aligned with orgID's own bid on each (nil where it hasn't bid yet), avoiding
// an N+1 query from the service layer.
func (r *ServiceRequestRepository) FindOpenRequestsForOrg(ctx context.Context, orgID uint) ([]*models.ServiceRequest, []*models.ServiceRequestBid, error) {
	query := `
		SELECT sr.id, sr.access_token, sr.service_type, sr.description, sr.customer_name, sr.customer_phone,
		       sr.latitude, sr.longitude, sr.address, sr.preferred_time, sr.status, sr.awarded_bid_id,
		       sr.created_at, sr.updated_at,
		       b.id, b.price, b.eta_minutes, b.message, b.status, b.created_at, b.updated_at
		FROM service_requests sr
		JOIN organizations o ON o.id = $1
		LEFT JOIN service_request_bids b ON b.service_request_id = sr.id AND b.organization_id = $1
		WHERE sr.status = 'open'
		  AND sr.service_type = o.industry
		  AND o.latitude IS NOT NULL AND o.longitude IS NOT NULL
		  AND (6371 * acos(
		          LEAST(1.0,
		              cos(radians(o.latitude)) * cos(radians(sr.latitude)) *
		              cos(radians(sr.longitude) - radians(o.longitude)) +
		              sin(radians(o.latitude)) * sin(radians(sr.latitude))
		          )
		      )) <= o.service_radius_km
		ORDER BY sr.created_at DESC
	`
	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var requests []*models.ServiceRequest
	var bids []*models.ServiceRequestBid
	for rows.Next() {
		sr := &models.ServiceRequest{}
		var preferredTime sql.NullTime
		var awardedBidID sql.NullInt64
		var bidID sql.NullInt64
		var bidPrice sql.NullFloat64
		var bidEta sql.NullInt64
		var bidMessage sql.NullString
		var bidStatus sql.NullString
		var bidCreatedAt, bidUpdatedAt sql.NullTime

		if err := rows.Scan(
			&sr.ID, &sr.AccessToken, &sr.ServiceType, &sr.Description, &sr.CustomerName, &sr.CustomerPhone,
			&sr.Latitude, &sr.Longitude, &sr.Address, &preferredTime, &sr.Status, &awardedBidID,
			&sr.CreatedAt, &sr.UpdatedAt,
			&bidID, &bidPrice, &bidEta, &bidMessage, &bidStatus, &bidCreatedAt, &bidUpdatedAt,
		); err != nil {
			return nil, nil, err
		}

		if preferredTime.Valid {
			sr.PreferredTime = &preferredTime.Time
		}
		if awardedBidID.Valid {
			id := uint(awardedBidID.Int64)
			sr.AwardedBidID = &id
		}
		requests = append(requests, sr)

		if bidID.Valid {
			bid := &models.ServiceRequestBid{
				ID:               uint(bidID.Int64),
				ServiceRequestID: sr.ID,
				OrganizationID:   orgID,
				Price:            bidPrice.Float64,
				Message:          bidMessage.String,
				Status:           models.BidStatus(bidStatus.String),
				CreatedAt:        bidCreatedAt.Time,
				UpdatedAt:        bidUpdatedAt.Time,
			}
			if bidEta.Valid {
				eta := int(bidEta.Int64)
				bid.EtaMinutes = &eta
			}
			bids = append(bids, bid)
		} else {
			bids = append(bids, nil)
		}
	}
	return requests, bids, rows.Err()
}

// UpsertBid inserts a new bid, or updates the org's existing bid if it's still 'submitted'.
// The ON CONFLICT ... WHERE clause means an attempt to update an already-awarded/rejected
// bid returns zero rows from RETURNING, surfaced here as ErrBidLocked.
func (r *ServiceRequestRepository) UpsertBid(ctx context.Context, bid *models.ServiceRequestBid) error {
	query := `
		INSERT INTO service_request_bids (service_request_id, organization_id, price, eta_minutes, message, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'submitted', NOW(), NOW())
		ON CONFLICT (service_request_id, organization_id) DO UPDATE
		SET price = EXCLUDED.price, eta_minutes = EXCLUDED.eta_minutes, message = EXCLUDED.message, updated_at = NOW()
		WHERE service_request_bids.status = 'submitted'
		RETURNING id, status, created_at, updated_at
	`
	err := r.db.QueryRowContext(ctx, query,
		bid.ServiceRequestID, bid.OrganizationID, bid.Price, bid.EtaMinutes, bid.Message,
	).Scan(&bid.ID, &bid.Status, &bid.CreatedAt, &bid.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrBidLocked
	}
	return err
}

func (r *ServiceRequestRepository) ListBidsForRequest(ctx context.Context, requestID uint) ([]*models.ServiceRequestBid, error) {
	query := `
		SELECT b.id, b.service_request_id, b.organization_id, o.name, o.phone,
		       b.price, b.eta_minutes, b.message, b.status, b.created_at, b.updated_at
		FROM service_request_bids b
		JOIN organizations o ON o.id = b.organization_id
		WHERE b.service_request_id = $1
		ORDER BY b.price ASC
	`
	rows, err := r.db.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bids []*models.ServiceRequestBid
	for rows.Next() {
		b := &models.ServiceRequestBid{}
		var eta sql.NullInt64
		if err := rows.Scan(
			&b.ID, &b.ServiceRequestID, &b.OrganizationID, &b.OrganizationName, &b.OrganizationPhone,
			&b.Price, &eta, &b.Message, &b.Status, &b.CreatedAt, &b.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if eta.Valid {
			v := int(eta.Int64)
			b.EtaMinutes = &v
		}
		bids = append(bids, b)
	}
	return bids, rows.Err()
}

// AwardBid atomically marks requestID awarded with bidID as the winner, marks that bid
// awarded, and rejects every other still-submitted bid on the request, returning the winner
// and loser organization IDs so the service layer can fire notifications without extra
// queries.
func (r *ServiceRequestRepository) AwardBid(ctx context.Context, requestID, bidID uint) (winnerOrgID uint, loserOrgIDs []uint, err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, nil, err
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM service_requests WHERE id = $1 FOR UPDATE`, requestID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, ErrServiceRequestNotFound
	}
	if err != nil {
		return 0, nil, err
	}
	if status != string(models.ServiceRequestStatusOpen) {
		return 0, nil, ErrServiceRequestNotOpen
	}

	err = tx.QueryRowContext(ctx,
		`SELECT organization_id FROM service_request_bids WHERE id = $1 AND service_request_id = $2`,
		bidID, requestID,
	).Scan(&winnerOrgID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil, ErrBidNotFound
	}
	if err != nil {
		return 0, nil, err
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE service_requests SET status = 'awarded', awarded_bid_id = $1, updated_at = NOW() WHERE id = $2`,
		bidID, requestID,
	); err != nil {
		return 0, nil, err
	}

	if _, err = tx.ExecContext(ctx,
		`UPDATE service_request_bids SET status = 'awarded', updated_at = NOW() WHERE id = $1`, bidID,
	); err != nil {
		return 0, nil, err
	}

	rows, err := tx.QueryContext(ctx,
		`UPDATE service_request_bids SET status = 'rejected', updated_at = NOW()
		 WHERE service_request_id = $1 AND id <> $2 AND status = 'submitted'
		 RETURNING organization_id`,
		requestID, bidID,
	)
	if err != nil {
		return 0, nil, err
	}
	for rows.Next() {
		var loserID uint
		if err = rows.Scan(&loserID); err != nil {
			rows.Close()
			return 0, nil, err
		}
		loserOrgIDs = append(loserOrgIDs, loserID)
	}
	if cerr := rows.Close(); cerr != nil {
		return 0, nil, cerr
	}
	if err = rows.Err(); err != nil {
		return 0, nil, err
	}

	if err = tx.Commit(); err != nil {
		return 0, nil, err
	}
	return winnerOrgID, loserOrgIDs, nil
}
