------------------------------------------------------------
-- Add created_by column to technicians table
-- References the organization_user who created the technician
------------------------------------------------------------

ALTER TABLE technicians
    ADD COLUMN IF NOT EXISTS created_by INTEGER REFERENCES organization_users(id);