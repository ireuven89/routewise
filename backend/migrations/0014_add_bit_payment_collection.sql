-- Migration: Add Bit Payment Collection
-- Description: Adds organization-level Bit payment settings and a payment_notifications
--              audit table for SMS-based payment collection requests. There is no payment
--              link (Bit has no public merchant/link API) — the SMS is plain-text
--              instructions: amount due + the org's Bit-registered phone number.
-- Date: 2026-08-20

-- Drop leftover schema from an abandoned earlier attempt at this feature (an unmerged
-- branch's migration that was applied to some dev databases directly, outside this repo's
-- current migration history — it used a payment link design that was later scrapped, and a
-- separate organization_payment_settings table superseded by flat columns on organizations
-- below, matching this repo's existing settings convention). Safe: both are schema-only
-- drift with no real data (payment_notifications was always empty; organization_payment_settings
-- only ever held unconfigured default rows).
DROP TABLE IF EXISTS payment_notifications;
DROP TABLE IF EXISTS organization_payment_settings;

ALTER TABLE organizations ADD COLUMN IF NOT EXISTS bit_payment_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS bit_phone_number VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS bit_business_name VARCHAR(255) NOT NULL DEFAULT '';
ALTER TABLE organizations ADD COLUMN IF NOT EXISTS auto_send_payment_sms BOOLEAN NOT NULL DEFAULT false;

COMMENT ON COLUMN organizations.bit_payment_enabled IS 'Whether this org can send Bit payment-collection SMS requests';
COMMENT ON COLUMN organizations.bit_phone_number IS 'Org phone number registered with Bit, shown to customers in payment SMS';
COMMENT ON COLUMN organizations.bit_business_name IS 'Business name shown to customers in payment SMS';
COMMENT ON COLUMN organizations.auto_send_payment_sms IS 'If true, a payment SMS is auto-sent when a job is marked completed';

-- Audit log of Bit payment-collection SMS requests sent per job (1-to-many per job).
CREATE TABLE IF NOT EXISTS payment_notifications (
    id               SERIAL PRIMARY KEY,
    organization_id  INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    job_id           INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    customer_id      INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    amount           DECIMAL(10,2) NOT NULL,
    recipient_phone  VARCHAR(32) NOT NULL,
    message_body     TEXT NOT NULL,
    sms_status       VARCHAR(20) NOT NULL DEFAULT 'pending',
    payment_status   VARCHAR(20) NOT NULL DEFAULT 'pending',
    sent_at          TIMESTAMP,
    paid_at          TIMESTAMP,
    created_by       INTEGER REFERENCES organization_users(id) ON DELETE SET NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_notifications_job_id ON payment_notifications(job_id);
CREATE INDEX IF NOT EXISTS idx_payment_notifications_organization_id ON payment_notifications(organization_id);

-- Idempotency guard: only one "live" (pending/sent/paid) notification may exist per job at
-- a time. A concurrent duplicate INSERT fails atomically with Postgres unique_violation
-- (23505) rather than relying on app-level check-then-insert locking. 'send_failed' and
-- 'cancelled' rows are excluded so a failed send can be retried by creating a new row.
CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_notifications_active_per_job
    ON payment_notifications(job_id)
    WHERE payment_status IN ('pending', 'sent', 'paid');

COMMENT ON TABLE payment_notifications IS 'Audit log of Bit payment-collection SMS requests sent per job';
COMMENT ON COLUMN payment_notifications.sms_status IS 'Twilio delivery status: pending, sent, failed, delivered';
COMMENT ON COLUMN payment_notifications.payment_status IS 'Business status: pending, sent, send_failed, paid, cancelled';
COMMENT ON COLUMN payment_notifications.message_body IS 'Rendered SMS text actually sent, kept for audit';
COMMENT ON COLUMN payment_notifications.created_by IS 'organization_users.id who triggered a manual send; NULL for auto-send-on-completion';
