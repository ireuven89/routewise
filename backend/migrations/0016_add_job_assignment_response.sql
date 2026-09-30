-- Migration: Technician accept/decline for job assignments
-- Description: Assigning a job now *offers* it to the technician (assignment_status =
--              'pending'); they accept it in the mobile app ('accepted') before they can
--              start it, or decline it, which unassigns the job and records who declined
--              and why so the owner can pick someone else.
--              Numbered 0016 because 0015 is reserved for the Bit payment branch (PR #21);
--              the migration runner sorts numerically, so the gap is fine.
-- Date: 2026-09-28

-- NULL = no technician assigned. Only meaningful while technician_id is set.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS assignment_status VARCHAR(20)
    CHECK (assignment_status IN ('pending', 'accepted'));
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS assignment_responded_at TIMESTAMP;

-- Last decline, kept after the job is unassigned so the owner sees who declined and why.
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS declined_by_worker_id INTEGER REFERENCES workers(id) ON DELETE SET NULL;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS decline_reason TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS declined_at TIMESTAMP;

-- Jobs assigned before this feature existed are treated as already accepted, so work in
-- flight isn't suddenly blocked behind an accept step.
UPDATE jobs SET assignment_status = 'accepted'
WHERE technician_id IS NOT NULL AND assignment_status IS NULL;
