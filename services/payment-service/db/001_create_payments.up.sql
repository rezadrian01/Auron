CREATE TABLE IF NOT EXISTS payments (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id                  UUID NOT NULL,
    user_id                   UUID NOT NULL,
    amount                    DECIMAL(12,2) NOT NULL,
    currency                  VARCHAR(10) NOT NULL DEFAULT 'usd',
    status                    VARCHAR(50) NOT NULL DEFAULT 'pending',
    stripe_payment_intent_id  VARCHAR(255),
    stripe_client_secret      TEXT,
    failure_reason            TEXT,
    created_at                TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_payments_status CHECK (
        status IN ('pending','processing','completed','failed','refunded')
    ),
    CONSTRAINT chk_payments_amount CHECK (amount > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status ON payments(status);
