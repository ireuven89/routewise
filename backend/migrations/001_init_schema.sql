-- Users/Authentication
CREATE TABLE IF NOT EXISTS users  (
                       id SERIAL PRIMARY KEY,
                       email VARCHAR(255) UNIQUE NOT NULL,
                       password_hash VARCHAR(255) NOT NULL,
                       company_name VARCHAR(255) NOT NULL,
                       phone VARCHAR(20),
                       industry VARCHAR(50) DEFAULT 'hvac', -- hvac, plumbing, electrical, etc.
                       created_at TIMESTAMP DEFAULT NOW(),
                       updated_at TIMESTAMP DEFAULT NOW()
);

-- Technicians
CREATE TABLE IF NOT EXISTS technicians (
                             id SERIAL PRIMARY KEY,
                             user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
                             name VARCHAR(255) NOT NULL,
                             email VARCHAR(255),
                             phone VARCHAR(20) NOT NULL,
                             is_active BOOLEAN DEFAULT true,
                             created_at TIMESTAMP DEFAULT NOW(),
                             updated_at TIMESTAMP DEFAULT NOW()
);

-- Customers
CREATE TABLE IF NOT EXISTS customers (
                           id SERIAL PRIMARY KEY,
                           user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
                           name VARCHAR(255) NOT NULL,
                           email VARCHAR(255),
                           phone VARCHAR(20) NOT NULL,
                           address TEXT NOT NULL,
                           latitude DECIMAL(10, 8),
                           longitude DECIMAL(11, 8),
                           notes TEXT,
                           created_at TIMESTAMP DEFAULT NOW(),
                           updated_at TIMESTAMP DEFAULT NOW()
);

-- Jobs
CREATE TABLE IF NOT EXISTS jobs (
                      id SERIAL PRIMARY KEY,
                      user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
                      customer_id INTEGER REFERENCES customers(id) ON DELETE CASCADE,
                      technician_id INTEGER REFERENCES technicians(id) ON DELETE SET NULL,
                      title VARCHAR(255) NOT NULL,
                      description TEXT,
                      status VARCHAR(50) DEFAULT 'scheduled', -- scheduled, in_progress, completed, cancelled
                      scheduled_at TIMESTAMP NOT NULL,
                      completed_at TIMESTAMP,
                      duration_minutes INTEGER DEFAULT 60,
                      price DECIMAL(10, 2),
                      metadata JSONB, -- For industry-specific data
                      created_at TIMESTAMP DEFAULT NOW(),
                      updated_at TIMESTAMP DEFAULT NOW()
);

-- Job notes/updates
CREATE TABLE IF NOT EXISTS job_updates (
                             id SERIAL PRIMARY KEY,
                             job_id INTEGER REFERENCES jobs(id) ON DELETE CASCADE,
                             technician_id INTEGER REFERENCES technicians(id),
                             note TEXT NOT NULL,
                             created_at TIMESTAMP DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_jobs_user_id ON jobs(user_id);
CREATE INDEX idx_jobs_scheduled_at ON jobs(scheduled_at);
CREATE INDEX idx_jobs_technician_id ON jobs(technician_id);
CREATE INDEX idx_customers_user_id ON customers(user_id);
CREATE INDEX idx_technicians_user_id ON technicians(user_id);