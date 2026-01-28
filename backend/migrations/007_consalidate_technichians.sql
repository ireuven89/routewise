-- Rename technicians to workers
ALTER TABLE technicians RENAME TO workers;

-- Ensure phone is required
ALTER TABLE workers ALTER COLUMN phone SET NOT NULL;

-- Add is_active for soft delete
ALTER TABLE workers ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Fix project_files to support BOTH uploaders
-- Rename existing column
ALTER TABLE project_files RENAME COLUMN uploaded_by TO uploaded_by_user;

-- Add worker uploader column
ALTER TABLE project_files ADD COLUMN uploaded_by_worker INTEGER REFERENCES workers(id);

-- At least one must be set
ALTER TABLE project_files ADD CONSTRAINT check_one_uploader
    CHECK (
        (uploaded_by_user IS NOT NULL AND uploaded_by_worker IS NULL) OR
        (uploaded_by_user IS NULL AND uploaded_by_worker IS NOT NULL)
        );

-- Indexes
CREATE INDEX idx_project_files_uploaded_by_worker ON project_files(uploaded_by_worker);