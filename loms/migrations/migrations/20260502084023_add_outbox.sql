-- +goose Up
CREATE TYPE loms.outbox_status as ENUM ('CREATED', 'IN_PROGRESS', 'SUCCESS');

CREATE TABLE loms.outbox
(
    idempotency_key TEXT PRIMARY KEY,
    data JSONB NOT NULL,
    status loms.outbox_status NOT NULL DEFAULT 'CREATED',
    kind INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION loms.update_outbox_timestamp() RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE OR REPLACE TRIGGER trigger_update_outbox_timestamp
    BEFORE UPDATE
    ON loms.outbox
    FOR EACH ROW
EXECUTE FUNCTION loms.update_outbox_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_update_outbox_timestamp ON loms.outbox;
DROP FUNCTION IF EXISTS loms.update_outbox_timestamp;
DROP TABLE IF EXISTS loms.outbox;
DROP TYPE IF EXISTS loms.outbox_status;
