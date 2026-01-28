-- Enhance jobs table for construction projects
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS project_type VARCHAR(100);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS project_value DECIMAL(12,2);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS estimated_duration INTEGER;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS actual_start_date DATE;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS actual_end_date DATE;

-- Add comments for clarity
COMMENT ON COLUMN jobs.project_type IS 'Type: residential, commercial, renovation, service_call';
COMMENT ON COLUMN jobs.project_value IS 'Total project cost in dollars';
COMMENT ON COLUMN jobs.estimated_duration IS 'Duration in days (construction) or hours (HVAC)';

-- Create project assignments table for crew/team management
CREATE TABLE project_assignments (
                                     id SERIAL PRIMARY KEY,
                                     project_id INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
                                     worker_id INTEGER NOT NULL REFERENCES organization_users(id) ON DELETE CASCADE,
                                     role VARCHAR(50), -- 'foreman', 'electrician', 'laborer', 'technician', etc.
                                     assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                     removed_at TIMESTAMP,

    -- Prevent duplicate active assignments
                                     UNIQUE(project_id, worker_id, assigned_at)
);

-- Indexes for performance
CREATE INDEX idx_project_assignments_project ON project_assignments(project_id);
CREATE INDEX idx_project_assignments_worker ON project_assignments(worker_id);
CREATE INDEX idx_project_assignments_active ON project_assignments(project_id, worker_id) WHERE removed_at IS NULL;

-- Add comments
COMMENT ON TABLE project_assignments IS 'Tracks which workers are assigned to which projects';
COMMENT ON COLUMN project_assignments.role IS 'Worker role on this specific project';
COMMENT ON COLUMN project_assignments.removed_at IS 'NULL = still assigned, timestamp = removed from project';