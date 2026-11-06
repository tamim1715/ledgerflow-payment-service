-- 000004_create_invoice_payment_bridge.up.sql (PostgreSQL)

-- 1. Add paid_amount tracking to bills
ALTER TABLE bills ADD COLUMN IF NOT EXISTS paid_amount NUMERIC(12,4) NOT NULL DEFAULT 0.0000;

-- 2. Add bill_id foreign key linkage to payments
ALTER TABLE payments ADD COLUMN IF NOT EXISTS bill_id VARCHAR(36) NULL REFERENCES bills(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_payments_bill ON payments(bill_id);

-- 3. Webhook idempotency and event audit ledger
CREATE TABLE IF NOT EXISTS webhook_events (
    id VARCHAR(36) PRIMARY KEY,
    provider VARCHAR(50) NOT NULL,
    event_id VARCHAR(255) NOT NULL,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'PROCESSED',
    processed_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_provider_event_id UNIQUE (provider, event_id)
);

CREATE INDEX IF NOT EXISTS idx_webhook_events_provider ON webhook_events(provider, event_id);
