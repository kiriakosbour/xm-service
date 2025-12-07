-- 001_init.sql
-- Creates the companies table

CREATE TABLE IF NOT EXISTS companies (
    id UUID PRIMARY KEY,
    name VARCHAR(15) NOT NULL UNIQUE,
    description VARCHAR(3000),
    employees INT NOT NULL CHECK (employees >= 0),
    registered BOOLEAN NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('Corporations', 'NonProfit', 'Cooperative', 'Sole Proprietorship')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for name lookups (uniqueness check)
CREATE INDEX IF NOT EXISTS idx_companies_name ON companies(name);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_companies_updated_at ON companies;
CREATE TRIGGER update_companies_updated_at
    BEFORE UPDATE ON companies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
