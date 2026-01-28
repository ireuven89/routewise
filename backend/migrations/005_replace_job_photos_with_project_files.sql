-- Drop old job_photos table (no data to preserve)
DROP TABLE IF EXISTS job_photos CASCADE;

-- Create comprehensive file management table
CREATE TABLE project_files (
                               id SERIAL PRIMARY KEY,
                               project_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
                               uploaded_by INTEGER REFERENCES organization_users(id),

    -- File identification
                               file_type VARCHAR(50) NOT NULL, -- 'photo', 'document', 'report'
                               file_category VARCHAR(50), -- 'progress', 'contract', 'site_photo', 'invoice', etc.
                               file_name VARCHAR(255) NOT NULL,
                               original_file_name VARCHAR(255) NOT NULL,

    -- File metadata
                               mime_type VARCHAR(100) NOT NULL,
                               file_size INTEGER,
                               file_extension VARCHAR(10),

    -- S3 storage
                               s3_bucket VARCHAR(100) NOT NULL,
                               s3_key TEXT NOT NULL,
                               s3_url TEXT,

    -- Additional info
                               description TEXT,
                               taken_at TIMESTAMP,

    -- Timestamps
                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                               updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_project_files_project_id ON project_files(project_id);
CREATE INDEX idx_project_files_type ON project_files(file_type);
CREATE INDEX idx_project_files_category ON project_files(file_category);
CREATE INDEX idx_project_files_uploaded_by ON project_files(uploaded_by);
CREATE INDEX idx_project_files_created_at ON project_files(created_at DESC);