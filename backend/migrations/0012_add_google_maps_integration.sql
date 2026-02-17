-- Migration: Add Google Maps Integration
-- Description: Add Google Places and geocoding fields to customers and technicians tables
-- Date: 2026-02-15

-- Add Google Maps fields to customers table
ALTER TABLE customers
    ADD COLUMN google_place_id VARCHAR(255),
    ADD COLUMN formatted_address TEXT,
    ADD COLUMN address_components JSONB,
    ADD COLUMN geocoded_at TIMESTAMP;

-- Add home address fields to workers table (renamed from technicians in migration 007)
ALTER TABLE workers
    ADD COLUMN home_address TEXT,
    ADD COLUMN home_latitude DECIMAL(10,8),
    ADD COLUMN home_longitude DECIMAL(11,8),
    ADD COLUMN home_google_place_id VARCHAR(255),
    ADD COLUMN home_formatted_address TEXT,
    ADD COLUMN home_address_components JSONB,
    ADD COLUMN home_geocoded_at TIMESTAMP;

-- Create indexes for efficient lookups
CREATE INDEX idx_customers_place_id ON customers(google_place_id) WHERE google_place_id IS NOT NULL;
CREATE INDEX idx_customers_lat_lng ON customers(latitude, longitude) WHERE latitude IS NOT NULL AND longitude IS NOT NULL;
CREATE INDEX idx_workers_home_lat_lng ON workers(home_latitude, home_longitude) WHERE home_latitude IS NOT NULL AND home_longitude IS NOT NULL;

-- Add comments for documentation
COMMENT ON COLUMN customers.google_place_id IS 'Unique Google Place ID for this customer address';
COMMENT ON COLUMN customers.formatted_address IS 'Google-formatted canonical address';
COMMENT ON COLUMN customers.address_components IS 'Structured address components (street, city, postal_code, country)';
COMMENT ON COLUMN customers.geocoded_at IS 'Timestamp of last geocoding operation';

COMMENT ON COLUMN workers.home_address IS 'Worker home address for route optimization';
COMMENT ON COLUMN workers.home_latitude IS 'Home address latitude coordinate';
COMMENT ON COLUMN workers.home_longitude IS 'Home address longitude coordinate';
COMMENT ON COLUMN workers.home_google_place_id IS 'Unique Google Place ID for worker home address';
COMMENT ON COLUMN workers.home_formatted_address IS 'Google-formatted canonical home address';
COMMENT ON COLUMN workers.home_address_components IS 'Structured home address components';
COMMENT ON COLUMN workers.home_geocoded_at IS 'Timestamp of last home address geocoding';
