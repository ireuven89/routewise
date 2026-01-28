

------------------------------------------------------------
-- 1. Rename users → organizations
------------------------------------------------------------
ALTER TABLE users RENAME TO organizations;

-- Rename columns to match new meaning
ALTER TABLE organizations RENAME COLUMN company_name TO name;

-- Remove email and password from organizations (should only be in organization_users)
ALTER TABLE organizations DROP COLUMN IF EXISTS email;
ALTER TABLE organizations DROP COLUMN IF EXISTS password_hash;

------------------------------------------------------------
-- 2. Create organization_users (admins/dispatchers)
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS organization_users (
                                    id SERIAL PRIMARY KEY,
                                    organization_id INTEGER REFERENCES organizations(id) ON DELETE CASCADE,
                                    email VARCHAR(255) UNIQUE NOT NULL,
                                    password_hash VARCHAR(255) NOT NULL,
                                    name VARCHAR(255),
                                    role VARCHAR(50) DEFAULT 'admin', -- admin, dispatcher, owner
                                    phone VARCHAR(20),
                                    created_at TIMESTAMP DEFAULT NOW(),
                                    updated_at TIMESTAMP DEFAULT NOW()
);

------------------------------------------------------------
-- 3. Add organization_id to technicians
------------------------------------------------------------
ALTER TABLE technicians
    ADD COLUMN organization_id INTEGER REFERENCES organizations(id);

-- Migrate existing data: technicians.user_id → technicians.organization_id
UPDATE technicians
SET organization_id = user_id;

-- Remove old FK
ALTER TABLE technicians DROP COLUMN user_id;

-- Add index
CREATE INDEX idx_technicians_organization_id ON technicians(organization_id);

------------------------------------------------------------
-- 4. Add organization_id + created_by to customers
------------------------------------------------------------
ALTER TABLE customers
    ADD COLUMN organization_id INTEGER REFERENCES organizations(id),
    ADD COLUMN created_by INTEGER REFERENCES organization_users(id);

-- Migrate existing data
UPDATE customers
SET organization_id = user_id;

-- Remove old FK
ALTER TABLE customers DROP COLUMN user_id;

-- Add index
CREATE INDEX idx_customers_organization_id ON customers(organization_id);

------------------------------------------------------------
-- 5. Add organization_id + created_by to jobs
------------------------------------------------------------
ALTER TABLE jobs
    ADD COLUMN organization_id INTEGER REFERENCES organizations(id),
    ADD COLUMN created_by INTEGER REFERENCES organization_users(id);

-- Migrate existing data
UPDATE jobs
SET organization_id = user_id;

-- Remove old FK
ALTER TABLE jobs DROP COLUMN user_id;

-- Add index
CREATE INDEX idx_jobs_organization_id ON jobs(organization_id);

------------------------------------------------------------
-- 6. Add job_photos table
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_photos (
                            id SERIAL PRIMARY KEY,
                            job_id INTEGER REFERENCES jobs(id) ON DELETE CASCADE,
                            technician_id INTEGER REFERENCES technicians(id),
                            url TEXT NOT NULL,
                            created_at TIMESTAMP DEFAULT NOW()
);

------------------------------------------------------------
-- 7. Add job_parts table
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_parts (
                           id SERIAL PRIMARY KEY,
                           job_id INTEGER REFERENCES jobs(id) ON DELETE CASCADE,
                           technician_id INTEGER REFERENCES technicians(id),
                           part_name VARCHAR(255) NOT NULL,
                           quantity INTEGER DEFAULT 1,
                           price DECIMAL(10,2),
                           created_at TIMESTAMP DEFAULT NOW()
);

------------------------------------------------------------
-- 8. Add job_status_updates table
------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_status_updates (
                                    id SERIAL PRIMARY KEY,
                                    job_id INTEGER REFERENCES jobs(id) ON DELETE CASCADE,
                                    technician_id INTEGER REFERENCES technicians(id),
                                    old_status VARCHAR(50),
                                    new_status VARCHAR(50),
                                    created_at TIMESTAMP DEFAULT NOW()
);

------------------------------------------------------------
-- 9. Optional: add technician location fields
------------------------------------------------------------
ALTER TABLE technicians
    ADD COLUMN last_lat DECIMAL(10,8),
    ADD COLUMN last_lng DECIMAL(11,8),
    ADD COLUMN last_seen_at TIMESTAMP;
