-- Migration: Add Smart Dispatching (broadcast bidding)
-- Description: Adds service_requests (customer job posts from the public /find-service
--              page), service_request_bids (per-org bids on a request), and
--              service_request_notifications (audit log of WhatsApp/SMS sends). Replaces
--              the previously broken per-provider "Request Service" flow (the frontend
--              already called POST /api/v1/public/service-requests, but no route, handler,
--              or table for it existed on main).
-- Date: 2026-09-23

-- The customer's job post, created anonymously from the public find-service page. There is
-- no customer login system, so access_token (an unguessable random UUID) is the customer's
-- identity for this request: it lets them revisit /find-service/requests/:token to see bids
-- come in and award one.
CREATE TABLE IF NOT EXISTS service_requests (
    id              SERIAL PRIMARY KEY,
    access_token    UUID NOT NULL,
    service_type    VARCHAR(50) NOT NULL,
    description     TEXT,
    customer_name   VARCHAR(255) NOT NULL,
    customer_phone  VARCHAR(32) NOT NULL,
    latitude        FLOAT NOT NULL,
    longitude       FLOAT NOT NULL,
    address         TEXT,
    preferred_time  TIMESTAMP,
    status          VARCHAR(20) NOT NULL DEFAULT 'open',
    awarded_bid_id  INTEGER,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_requests_access_token ON service_requests(access_token);
CREATE INDEX IF NOT EXISTS idx_service_requests_status ON service_requests(status);
CREATE INDEX IF NOT EXISTS idx_service_requests_created_at ON service_requests(created_at);

-- One bid per org per request, DB-enforced: a concurrent double-submit fails atomically
-- (unique_violation) instead of relying on check-then-insert locking. A second submission
-- from the same org is an UPSERT (see service_request_repository.go UpsertBid), not a new row.
CREATE TABLE IF NOT EXISTS service_request_bids (
    id                  SERIAL PRIMARY KEY,
    service_request_id INTEGER NOT NULL REFERENCES service_requests(id) ON DELETE CASCADE,
    organization_id     INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    price               DECIMAL(10,2) NOT NULL,
    eta_minutes         INTEGER,
    message             TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'submitted',
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (service_request_id, organization_id)
);

CREATE INDEX IF NOT EXISTS idx_service_request_bids_request_id ON service_request_bids(service_request_id);
CREATE INDEX IF NOT EXISTS idx_service_request_bids_org_id ON service_request_bids(organization_id);

-- Wire the awarded-bid FK now that service_request_bids exists (can't forward-reference a
-- not-yet-created table within the same CREATE TABLE).
ALTER TABLE service_requests
    ADD CONSTRAINT fk_service_requests_awarded_bid
    FOREIGN KEY (awarded_bid_id) REFERENCES service_request_bids(id) ON DELETE SET NULL;

-- Audit log of WhatsApp (to orgs) and SMS (to the customer) sends for this feature: new-lead
-- alerts, and award/rejection notices. organization_id is nullable because customer-facing
-- rows (tracking link, award confirmation) have no organization recipient.
CREATE TABLE IF NOT EXISTS service_request_notifications (
    id                  SERIAL PRIMARY KEY,
    service_request_id INTEGER NOT NULL REFERENCES service_requests(id) ON DELETE CASCADE,
    organization_id     INTEGER REFERENCES organizations(id) ON DELETE CASCADE,
    kind                VARCHAR(30) NOT NULL,
    channel             VARCHAR(20) NOT NULL DEFAULT 'whatsapp',
    recipient_phone     VARCHAR(32) NOT NULL,
    message_body        TEXT NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    sent_at             TIMESTAMP,
    created_at          TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_service_request_notifications_request_id ON service_request_notifications(service_request_id);

-- Prevents the async fan-out from double-notifying the same org for the same request on
-- retry. Only 'new_lead' needs this guard; other kinds are naturally one-per-recipient
-- already (a request is only awarded once).
CREATE UNIQUE INDEX IF NOT EXISTS idx_service_request_notifications_unique_lead
    ON service_request_notifications(service_request_id, organization_id, kind)
    WHERE kind = 'new_lead';

COMMENT ON TABLE service_requests IS 'Customer job posts from the public find-service broadcast bidding flow';
COMMENT ON COLUMN service_requests.access_token IS 'Unguessable token letting an anonymous customer revisit /find-service/requests/:token; this IS the customer''s identity for the request, there is no login';
COMMENT ON COLUMN service_requests.status IS 'open, awarded, cancelled';
COMMENT ON TABLE service_request_bids IS 'Per-organization bids on a broadcast service request; one bid per org per request, enforced by the unique constraint';
COMMENT ON COLUMN service_request_bids.status IS 'submitted, awarded, rejected';
COMMENT ON TABLE service_request_notifications IS 'Audit log of WhatsApp (org-facing) / SMS (customer-facing) sends for smart dispatching';
COMMENT ON COLUMN service_request_notifications.kind IS 'new_lead (whatsapp, org), bid_awarded (whatsapp, org), bid_rejected (whatsapp, org), tracking_link (sms, customer), award_confirmation (sms, customer)';
