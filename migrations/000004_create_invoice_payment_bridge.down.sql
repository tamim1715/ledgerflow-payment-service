-- 000004_create_invoice_payment_bridge.down.sql (PostgreSQL)

DROP TABLE IF EXISTS webhook_events;
DROP INDEX IF EXISTS idx_payments_bill;
ALTER TABLE payments DROP COLUMN IF EXISTS bill_id;
ALTER TABLE bills DROP COLUMN IF EXISTS paid_amount;
