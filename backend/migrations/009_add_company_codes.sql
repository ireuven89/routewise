-- Add company_code column
ALTER TABLE organizations
    ADD COLUMN company_code VARCHAR(50);

-- Generate codes for existing organizations
UPDATE organizations
SET company_code = UPPER(REPLACE(name, ' ', '-')) || '-' || SUBSTRING(MD5(RANDOM()::TEXT), 1, 4)
WHERE company_code IS NULL;

-- Make it NOT NULL after setting values
ALTER TABLE organizations
    ALTER COLUMN company_code SET NOT NULL;

-- Add unique constraint
ALTER TABLE organizations
    ADD CONSTRAINT organizations_company_code_unique UNIQUE (company_code);

-- Add index for fast lookups
CREATE INDEX idx_organizations_company_code ON organizations(company_code);