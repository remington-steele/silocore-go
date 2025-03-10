SET ROLE silocore_admin;

-- Create a table for orders
CREATE TABLE "order" (
    order_id SERIAL PRIMARY KEY,
    tenant_id INTEGER NOT NULL REFERENCES tenant(tenant_id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES usr(user_id) ON DELETE CASCADE,
    order_number VARCHAR(64) NOT NULL,
    status VARCHAR(64) NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'cancelled')),
    total_amount DECIMAL(10, 2) NOT NULL,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, order_number)
);

-- Create a trigger to automatically update the updated_at column
CREATE TRIGGER update_order_updated_at
BEFORE UPDATE ON "order"
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

-- Enable Row Level Security on order table
ALTER TABLE "order" ENABLE ROW LEVEL SECURITY;

-- Create RLS policy for order table
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies 
        WHERE tablename = 'order' AND policyname = 'order_isolation_policy'
    ) THEN
        CREATE POLICY order_isolation_policy ON "order"
        USING (
            tenant_id = tenant_context() 
            OR 
            tenant_context() IS NULL
        );
    END IF;
END
$$;

-- Create indexes for better performance
CREATE INDEX idx_order_tenant_id ON "order" (tenant_id);
CREATE INDEX idx_order_user_id ON "order" (user_id);
CREATE INDEX idx_order_status ON "order" (status);
CREATE INDEX idx_order_created_at ON "order" (created_at); 