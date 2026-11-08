-- +goose Up
CREATE TABLE payment_orders (
    id UUID PRIMARY KEY,
    provider VARCHAR(30) NOT NULL,
    merchant_order_id VARCHAR(120) NOT NULL,
    provider_order_id VARCHAR(120),
    provider_transaction_id VARCHAR(120),
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'CNY',
    subject VARCHAR(200) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    payment_url TEXT NOT NULL DEFAULT '',
    provider_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    expires_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    created_by UUID REFERENCES identity_users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT payment_orders_amount_check CHECK (amount_cents > 0),
    CONSTRAINT payment_orders_status_check CHECK (status IN ('pending', 'paid', 'failed', 'closed')),
    CONSTRAINT payment_orders_provider_merchant_unique UNIQUE (provider, merchant_order_id)
);

CREATE UNIQUE INDEX payment_orders_provider_transaction_unique
    ON payment_orders (provider, provider_transaction_id)
    WHERE provider_transaction_id IS NOT NULL;

CREATE INDEX payment_orders_created_idx
    ON payment_orders (created_at DESC);

CREATE TABLE payment_callbacks (
    id UUID PRIMARY KEY,
    provider VARCHAR(30) NOT NULL,
    event_id VARCHAR(200) NOT NULL,
    order_id UUID NOT NULL REFERENCES payment_orders(id) ON DELETE RESTRICT,
    provider_transaction_id VARCHAR(120) NOT NULL DEFAULT '',
    amount_cents BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL,
    status VARCHAR(20) NOT NULL,
    payload_hash CHAR(64) NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    error_code VARCHAR(80) NOT NULL DEFAULT '',
    CONSTRAINT payment_callbacks_event_unique UNIQUE (provider, event_id),
    CONSTRAINT payment_callbacks_status_check CHECK (status IN ('accepted', 'rejected')),
    CONSTRAINT payment_callbacks_amount_check CHECK (amount_cents > 0)
);

CREATE INDEX payment_callbacks_order_idx
    ON payment_callbacks (order_id, received_at DESC);

-- +goose Down
DROP TABLE IF EXISTS payment_callbacks;
DROP TABLE IF EXISTS payment_orders;
