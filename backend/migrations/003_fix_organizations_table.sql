------------------------------------------------------------
-- Fix: Remove email and password_hash from organizations
-- These fields should only exist in organization_users
------------------------------------------------------------

-- Drop email and password_hash from organizations table
