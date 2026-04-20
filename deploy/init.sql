-- deploy/init.sql

-- Enable UUID extension (built-in for Postgres 13+)
-- CREATE EXTENSION IF NOT EXISTS "pgcrypto"; 

-- 1. Products Table
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    currency VARCHAR(3) NOT NULL DEFAULT 'IDR',
    price_units BIGINT NOT NULL,
    price_nanos INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Orders Table
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    status VARCHAR(50) NOT NULL, -- Match domain.OrderStatus string
    currency VARCHAR(3) NOT NULL,
    total_units BIGINT NOT NULL,
    total_nanos INT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 3. Order Items Table
CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID NOT NULL, -- References products(id) eventually
    quantity INT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    unit_units BIGINT NOT NULL,
    unit_nanos INT NOT NULL
);

-- Indexing foreign keys for join performance and cascading deletes
CREATE INDEX idx_order_items_order_id ON order_items(order_id);

-- 4. Outbox Events Table
CREATE TABLE outbox_events (
    id SERIAL PRIMARY KEY,
    aggregate_id UUID NOT NULL,
    aggregate_type VARCHAR(50) NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for the Relay Worker to quickly find pending events
CREATE INDEX idx_outbox_status_created ON outbox_events(status, created_at);
