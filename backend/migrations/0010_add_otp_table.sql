-- Create worker_otps table
CREATE TABLE worker_otps (
                             id SERIAL PRIMARY KEY,
                             phone VARCHAR(20) NOT NULL,
                             company_code VARCHAR(50) NOT NULL,
                             otp_code VARCHAR(6) NOT NULL,
                             expires_at TIMESTAMP NOT NULL,
                             verified BOOLEAN DEFAULT false,
                             created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for fast lookups
CREATE INDEX idx_worker_otps_lookup ON worker_otps(phone, company_code, verified, expires_at);
CREATE INDEX idx_worker_otps_expires ON worker_otps(expires_at);

-- Add comment
COMMENT ON TABLE worker_otps IS 'Stores OTP codes for worker authentication';