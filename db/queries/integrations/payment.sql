-- name: CreatePaymentOrder :one
INSERT INTO payment_orders (
    id, provider, merchant_order_id, amount_cents, currency, subject, status, created_by
) VALUES ($1, $2, $3, $4, $5, $6, 'pending', $7)
RETURNING *;

-- name: GetPaymentOrder :one
SELECT * FROM payment_orders WHERE id = $1;

-- name: GetPaymentOrderByMerchantID :one
SELECT * FROM payment_orders WHERE provider = $1 AND merchant_order_id = $2;

-- name: AttachPaymentSession :one
UPDATE payment_orders
SET provider_order_id = $2, payment_url = $3, provider_payload = $4,
    expires_at = $5, updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: MarkPaymentOrderFailed :exec
UPDATE payment_orders
SET status = 'failed', updated_at = now()
WHERE id = $1 AND status = 'pending';

-- name: InsertPaymentCallback :one
INSERT INTO payment_callbacks (
    id, provider, event_id, order_id, provider_transaction_id, amount_cents,
    currency, status, payload_hash, processed_at, error_code
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now(), $10)
ON CONFLICT (provider, event_id) DO NOTHING
RETURNING *;

-- name: GetPaymentCallback :one
SELECT * FROM payment_callbacks WHERE provider = $1 AND event_id = $2;

-- name: MarkPaymentOrderPaid :one
UPDATE payment_orders
SET status = 'paid', provider_transaction_id = $2, paid_at = $3, updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: MarkPaymentOrderClosed :one
UPDATE payment_orders
SET status = 'closed', updated_at = now()
WHERE id = $1 AND status = 'pending'
RETURNING *;
