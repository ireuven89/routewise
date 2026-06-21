-- Add service area and pricing fields to organizations for customer discovery feature
ALTER TABLE organizations ADD COLUMN latitude FLOAT;
ALTER TABLE organizations ADD COLUMN longitude FLOAT;
ALTER TABLE organizations ADD COLUMN address TEXT;
ALTER TABLE organizations ADD COLUMN service_radius_km FLOAT NOT NULL DEFAULT 20;
ALTER TABLE organizations ADD COLUMN google_place_id TEXT;
ALTER TABLE organizations ADD COLUMN formatted_address TEXT;
ALTER TABLE organizations ADD COLUMN address_components JSONB;
ALTER TABLE organizations ADD COLUMN geocoded_at TIMESTAMP;
ALTER TABLE organizations ADD COLUMN visit_fee DECIMAL(10,2);
ALTER TABLE organizations ADD COLUMN repair_estimate_min DECIMAL(10,2);
ALTER TABLE organizations ADD COLUMN repair_estimate_max DECIMAL(10,2);
