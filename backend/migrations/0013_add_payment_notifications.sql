-- Migration 0013: Add payment notifications system
-- Creates tables for tracking payment links sent to customers via SMS (Bit payment app)

-- Table: payment_notifications
-- Tracks every payment link sent to customers
CREATE TABLE payment_notifications (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    job_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    customer_id INTEGER NOT NULL REFERENCES customers(id) ON DELETE CASCADE,

    -- Payment details
    amount DECIMAL(10, 2) NOT NULL,
    payment_link_url TEXT NOT NULL,
    payment_method VARCHAR(50) DEFAULT 'bit',

    -- Delivery tracking
    sent_via VARCHAR(20) NOT NULL DEFAULT 'sms',
    recipient_phone VARCHAR(50) NOT NULL,
    sent_at TIMESTAMP,
    sms_status VARCHAR(50) DEFAULT 'pending',

    -- Payment tracking
    payment_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    paid_at TIMESTAMP,

    -- Audit
    created_by INTEGER REFERENCES organization_users(id),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT fk_payment_org FOREIGN KEY (organization_id) REFERENCES organizations(id),
    CONSTRAINT fk_payment_job FOREIGN KEY (job_id) REFERENCES jobs(id),
    CONSTRAINT fk_payment_customer FOREIGN KEY (customer_id) REFERENCES customers(id)
);

CREATE INDEX idx_payment_notifications_org ON payment_notifications(organization_id);
CREATE INDEX idx_payment_notifications_job ON payment_notifications(job_id);
CREATE INDEX idx_payment_notifications_status ON payment_notifications(payment_status);

-- Table: organization_payment_settings
-- Configure payment behavior per organization
CREATE TABLE organization_payment_settings (
    id SERIAL PRIMARY KEY,
    organization_id INTEGER NOT NULL UNIQUE REFERENCES organizations(id) ON DELETE CASCADE,

    -- Bit payment configuration
    bit_payment_enabled BOOLEAN DEFAULT false,
    bit_phone_number VARCHAR(50),
    bit_business_name VARCHAR(255),

    -- Behavior settings
    auto_send_on_completion BOOLEAN DEFAULT false,

    -- SMS templates (supports {{customer_name}}, {{job_title}}, {{amount}}, {{link}})
    sms_template_he TEXT DEFAULT 'שלום {{customer_name}}, העבודה "{{job_title}}" הושלמה. סכום לתשלום: ₪{{amount}}. תשלום דרך Bit: {{link}}',
    sms_template_en TEXT DEFAULT 'Hi {{customer_name}}, job "{{job_title}}" completed. Amount: ₪{{amount}}. Pay via Bit: {{link}}',

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT fk_payment_settings_org FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

-- Seed with defaults for existing organizations
INSERT INTO organization_payment_settings (organization_id, bit_payment_enabled, auto_send_on_completion)
SELECT id, false, false FROM organizations;
