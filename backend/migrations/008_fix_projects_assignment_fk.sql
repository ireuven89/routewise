-- migrations/000008_fix_project_assignments.up.sql

-- Drop old foreign key
ALTER TABLE project_assignments
    DROP CONSTRAINT IF EXISTS project_assignments_worker_id_fkey;

-- Add correct foreign key to workers table
ALTER TABLE project_assignments
    ADD CONSTRAINT project_assignments_worker_id_fkey
        FOREIGN KEY (worker_id) REFERENCES workers(id) ON DELETE CASCADE;