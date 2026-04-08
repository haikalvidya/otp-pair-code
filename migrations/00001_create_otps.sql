-- +goose Up
CREATE TABLE IF NOT EXISTS otps (
    id BIGSERIAL PRIMARY KEY,
    user_id TEXT NOT NULL,
    otp_code VARCHAR(5) NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('created', 'validated', 'expired')),
    failed_attempts INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    validated_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_otps_user_created_at
    ON otps (user_id, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_otps_one_created_per_user
    ON otps (user_id)
    WHERE status = 'created';

-- +goose Down
DROP INDEX IF EXISTS idx_otps_one_created_per_user;
DROP INDEX IF EXISTS idx_otps_user_created_at;
DROP TABLE IF EXISTS otps;
