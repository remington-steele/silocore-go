SET ROLE silocore_admin;

-- Create a table for orders
CREATE TABLE ordr (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenant(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES usr(id) ON DELETE CASCADE,
    order_number VARCHAR(64) NOT NULL,
    status VARCHAR(64) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'cancelled')),
    total_amount DECIMAL(10, 2) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, order_number)
);

-- Create a trigger to automatically update the updated_at column
CREATE TRIGGER update_ordr_updated_at
BEFORE UPDATE ON ordr
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Enable Row Level Security on ordr table
ALTER TABLE ordr ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for ordr table
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies 
        WHERE tablename = 'ordr' AND policyname = 'ordr_isolation_policy'
    ) THEN
        CREATE POLICY ordr_isolation_policy ON ordr
        USING (
            tenant_id = tenant_context() 
            OR 
            tenant_context() IS NULL
        );
    END IF;
END
$$;

-- Create indexes for better performance
CREATE INDEX idx_ordr_tenant_id ON ordr (tenant_id);
CREATE INDEX idx_ordr_user_id ON ordr (user_id);
CREATE INDEX idx_ordr_status ON ordr (status);
CREATE INDEX idx_ordr_created_at ON ordr (created_at); 