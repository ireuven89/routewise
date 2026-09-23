package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/google/uuid"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
)

// maxMatchedOrgs caps how many organizations a single request fans out to, as a safety
// bound rather than a UX limit (unlike ProviderService.SearchProviders' LIMIT 20).
const maxMatchedOrgs = 50

var ErrServiceRequestValidation = errors.New("invalid service request")

type CreateServiceRequestInput struct {
	ServiceType   string
	Description   string
	CustomerName  string
	CustomerPhone string
	Latitude      float64
	Longitude     float64
	Address       string
	PreferredTime *time.Time
}

type SubmitBidInput struct {
	Price      float64
	EtaMinutes *int
	Message    string
}

type ServiceRequestService struct {
	requestRepo      *repository.ServiceRequestRepository
	notificationRepo *repository.ServiceRequestNotificationRepository
	notifier         NotificationService
	frontendBaseURL  string
}

func NewServiceRequestService(
	requestRepo *repository.ServiceRequestRepository,
	notificationRepo *repository.ServiceRequestNotificationRepository,
	notifier NotificationService,
	frontendBaseURL string,
) *ServiceRequestService {
	return &ServiceRequestService{
		requestRepo:      requestRepo,
		notificationRepo: notificationRepo,
		notifier:         notifier,
		frontendBaseURL:  frontendBaseURL,
	}
}

// CreateRequest validates and inserts the request (fast, synchronous), then resolves
// matching organizations and fires WhatsApp/SMS notifications in the background so a
// fan-out to many orgs can't make the customer's HTTP request hang on Twilio latency.
func (s *ServiceRequestService) CreateRequest(ctx context.Context, in CreateServiceRequestInput) (*models.ServiceRequest, error) {
	if in.Latitude < -90 || in.Latitude > 90 {
		return nil, fmt.Errorf("%w: invalid latitude", ErrServiceRequestValidation)
	}
	if in.Longitude < -180 || in.Longitude > 180 {
		return nil, fmt.Errorf("%w: invalid longitude", ErrServiceRequestValidation)
	}
	if in.ServiceType == "" {
		return nil, fmt.Errorf("%w: service_type is required", ErrServiceRequestValidation)
	}
	if in.CustomerName == "" || in.CustomerPhone == "" {
		return nil, fmt.Errorf("%w: customer name and phone are required", ErrServiceRequestValidation)
	}

	sr := &models.ServiceRequest{
		AccessToken:   uuid.NewString(),
		ServiceType:   in.ServiceType,
		Description:   in.Description,
		CustomerName:  in.CustomerName,
		CustomerPhone: in.CustomerPhone,
		Latitude:      in.Latitude,
		Longitude:     in.Longitude,
		Address:       in.Address,
		PreferredTime: in.PreferredTime,
		Status:        models.ServiceRequestStatusOpen,
	}
	if err := s.requestRepo.Create(ctx, sr); err != nil {
		return nil, err
	}

	go s.sendTrackingLinkSMS(sr)

	matches, err := s.requestRepo.FindMatchingOrganizations(ctx, sr.Latitude, sr.Longitude, sr.ServiceType, maxMatchedOrgs)
	if err != nil {
		sentry.CaptureException(err)
		return sr, nil
	}
	go s.dispatchNewLeadNotifications(sr, matches)

	return sr, nil
}

func (s *ServiceRequestService) GetByToken(ctx context.Context, token string) (*models.ServiceRequest, []*models.ServiceRequestBid, error) {
	sr, err := s.requestRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}
	bids, err := s.requestRepo.ListBidsForRequest(ctx, sr.ID)
	if err != nil {
		return nil, nil, err
	}
	return sr, bids, nil
}

// ListLeadsForOrg returns open leads with the customer's name and phone stripped: every
// bidder sees these, and only the awarded org gets the contact details (see AwardBid).
func (s *ServiceRequestService) ListLeadsForOrg(ctx context.Context, orgID uint) ([]*models.ServiceRequest, []*models.ServiceRequestBid, error) {
	requests, bids, err := s.requestRepo.FindOpenRequestsForOrg(ctx, orgID)
	if err != nil {
		return nil, nil, err
	}
	for _, sr := range requests {
		sr.CustomerName = ""
		sr.CustomerPhone = ""
	}
	return requests, bids, nil
}

func (s *ServiceRequestService) SubmitBid(ctx context.Context, orgID, requestID uint, in SubmitBidInput) (*models.ServiceRequestBid, error) {
	sr, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil {
		return nil, err
	}
	if sr.Status != models.ServiceRequestStatusOpen {
		return nil, repository.ErrServiceRequestNotOpen
	}
	if in.Price <= 0 {
		return nil, fmt.Errorf("%w: price must be greater than zero", ErrServiceRequestValidation)
	}

	bid := &models.ServiceRequestBid{
		ServiceRequestID: requestID,
		OrganizationID:   orgID,
		Price:            in.Price,
		EtaMinutes:       in.EtaMinutes,
		Message:          in.Message,
	}
	if err := s.requestRepo.UpsertBid(ctx, bid); err != nil {
		return nil, err
	}
	return bid, nil
}

// AwardBid resolves the customer's token to a request, awards the chosen bid (which also
// creates the customer and a scheduled job in the winner's account), and notifies the
// winner, the losers, and the customer asynchronously.
func (s *ServiceRequestService) AwardBid(ctx context.Context, token string, bidID uint) (*models.ServiceRequest, error) {
	sr, err := s.requestRepo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	award, err := s.requestRepo.AwardBid(ctx, sr, bidID)
	if err != nil {
		return nil, err
	}

	sr, err = s.requestRepo.FindByID(ctx, sr.ID)
	if err != nil {
		return nil, err
	}

	bids, err := s.requestRepo.ListBidsForRequest(ctx, sr.ID)
	if err != nil {
		sentry.CaptureException(err)
		return sr, nil
	}

	go s.notifyAwardOutcome(sr, bids, award.WinnerOrgID, award.LoserOrgIDs)

	return sr, nil
}

// dispatchNewLeadNotifications sends a WhatsApp new-lead alert to each matched org. It uses
// its own timeout context (not the request's, which is cancelled once the HTTP response is
// written) so the fan-out completes even after CreateRequest has already returned.
func (s *ServiceRequestService) dispatchNewLeadNotifications(sr *models.ServiceRequest, orgs []*repository.MatchedOrg) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	for _, org := range orgs {
		orgID := org.ID
		message := buildNewLeadMessage(sr, org)
		n := &models.ServiceRequestNotification{
			ServiceRequestID: sr.ID,
			OrganizationID:   &orgID,
			Kind:             models.NotificationKindNewLead,
			Channel:          "whatsapp",
			RecipientPhone:   org.Phone,
			MessageBody:      message,
			Status:           models.NotificationStatusPending,
		}
		if err := s.notificationRepo.Create(ctx, n); err != nil {
			if !errors.Is(err, repository.ErrNotificationAlreadySent) {
				sentry.CaptureException(err)
			}
			continue
		}

		s.sendAndRecord(ctx, n.ID, func() error { return s.notifier.SendWhatsApp(org.Phone, message) })
	}
}

func (s *ServiceRequestService) sendTrackingLinkSMS(sr *models.ServiceRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	message := buildTrackingLinkSMS(sr, s.trackingURL(sr))
	n := &models.ServiceRequestNotification{
		ServiceRequestID: sr.ID,
		Kind:             models.NotificationKindTrackingLink,
		Channel:          "sms",
		RecipientPhone:   sr.CustomerPhone,
		MessageBody:      message,
		Status:           models.NotificationStatusPending,
	}
	if err := s.notificationRepo.Create(ctx, n); err != nil {
		sentry.CaptureException(err)
		return
	}
	s.sendAndRecord(ctx, n.ID, func() error { return s.notifier.SendSMS(sr.CustomerPhone, message) })
}

func (s *ServiceRequestService) notifyAwardOutcome(sr *models.ServiceRequest, bids []*models.ServiceRequestBid, winnerOrgID uint, loserOrgIDs []uint) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	var winnerName string
	for _, b := range bids {
		if b.OrganizationID == winnerOrgID {
			winnerName = b.OrganizationName
		}
		switch b.Status {
		case models.BidStatusAwarded:
			s.notifyOrgByPhone(ctx, sr, b.OrganizationID, b.OrganizationPhone, models.NotificationKindBidAwarded, buildBidAwardedMessage(sr))
		case models.BidStatusRejected:
			s.notifyOrgByPhone(ctx, sr, b.OrganizationID, b.OrganizationPhone, models.NotificationKindBidRejected, buildBidRejectedMessage(sr))
		}
	}
	_ = loserOrgIDs // loser identities come from the bid list above (status == rejected), which also carries each org's phone

	message := buildAwardConfirmationSMS(sr, winnerName, s.trackingURL(sr))
	n := &models.ServiceRequestNotification{
		ServiceRequestID: sr.ID,
		Kind:             models.NotificationKindAwardConfirmation,
		Channel:          "sms",
		RecipientPhone:   sr.CustomerPhone,
		MessageBody:      message,
		Status:           models.NotificationStatusPending,
	}
	if err := s.notificationRepo.Create(ctx, n); err != nil {
		sentry.CaptureException(err)
		return
	}
	s.sendAndRecord(ctx, n.ID, func() error { return s.notifier.SendSMS(sr.CustomerPhone, message) })
}

// notifyOrgByPhone sends a WhatsApp notice to a single org whose phone is already known
// (from ListBidsForRequest's join onto organizations), avoiding an extra lookup query.
func (s *ServiceRequestService) notifyOrgByPhone(ctx context.Context, sr *models.ServiceRequest, orgID uint, phone string, kind models.NotificationKind, message string) {
	if phone == "" {
		return
	}

	oid := orgID
	n := &models.ServiceRequestNotification{
		ServiceRequestID: sr.ID,
		OrganizationID:   &oid,
		Kind:             kind,
		Channel:          "whatsapp",
		RecipientPhone:   phone,
		MessageBody:      message,
		Status:           models.NotificationStatusPending,
	}
	if err := s.notificationRepo.Create(ctx, n); err != nil {
		if !errors.Is(err, repository.ErrNotificationAlreadySent) {
			sentry.CaptureException(err)
		}
		return
	}
	s.sendAndRecord(ctx, n.ID, func() error { return s.notifier.SendWhatsApp(phone, message) })
}

func (s *ServiceRequestService) sendAndRecord(ctx context.Context, notificationID uint, send func() error) {
	sendErr := send()
	status := models.NotificationStatusSent
	var sentAt *time.Time
	if sendErr != nil {
		status = models.NotificationStatusFailed
		sentry.CaptureException(sendErr)
	} else {
		now := time.Now()
		sentAt = &now
	}
	if err := s.notificationRepo.UpdateStatus(ctx, notificationID, status, sentAt); err != nil {
		sentry.CaptureException(err)
	}
}

func (s *ServiceRequestService) trackingURL(sr *models.ServiceRequest) string {
	return fmt.Sprintf("%s/find-service/requests/%s", s.frontendBaseURL, sr.AccessToken)
}

func buildNewLeadMessage(sr *models.ServiceRequest, org *repository.MatchedOrg) string {
	return fmt.Sprintf(
		"New %s lead near you. %s Address: %s. Reply in RouteWise to send your bid.",
		sr.ServiceType, sr.Description, sr.Address,
	)
}

func buildBidAwardedMessage(sr *models.ServiceRequest) string {
	return fmt.Sprintf(
		"You won the lead for %s at %s. Customer: %s, %s. The job was added to your RouteWise schedule.",
		sr.ServiceType, sr.Address, sr.CustomerName, sr.CustomerPhone,
	)
}

func buildBidRejectedMessage(sr *models.ServiceRequest) string {
	return fmt.Sprintf("The %s lead near %s was awarded to another provider.", sr.ServiceType, sr.Address)
}

func buildTrackingLinkSMS(sr *models.ServiceRequest, url string) string {
	return fmt.Sprintf(
		"Hi %s, your %s request was sent to nearby providers. Track offers and pick one: %s",
		sr.CustomerName, sr.ServiceType, url,
	)
}

func buildAwardConfirmationSMS(sr *models.ServiceRequest, orgName, url string) string {
	return fmt.Sprintf("You selected %s for your %s job. They'll be in touch. Details: %s", orgName, sr.ServiceType, url)
}
