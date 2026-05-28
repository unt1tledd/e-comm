-- +goose Up
CREATE SCHEMA IF NOT EXISTS loms;

CREATE TYPE loms.order_status AS ENUM ('new', 'awaiting payment', 'failed', 'paid', 'cancelled');

CREATE TABLE loms.orders
(
    id         BIGINT            GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    BIGINT            NOT NULL,
    status     loms.order_status NOT NULL DEFAULT 'new',
    created_at TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION loms.set_updated_at() RETURNS TRIGGER AS
$$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

CREATE TRIGGER set_orders_updated_at
BEFORE UPDATE ON loms.orders
FOR EACH ROW
EXECUTE FUNCTION loms.set_updated_at();

CREATE TABLE loms.order_info
(
    order_id BIGINT  NOT NULL REFERENCES loms.orders (id) ON DELETE CASCADE,
    sku      BIGINT NOT NULL CHECK (sku > 0),
    count   BIGINT NOT NULL CHECK (count > 0),
    PRIMARY KEY (order_id, sku)
);

-- +goose Down
DROP TRIGGER IF EXISTS set_orders_updated_at ON loms.orders;
DROP FUNCTION IF EXISTS loms.set_updated_at;
DROP TABLE IF EXISTS loms.order_info;
DROP TABLE IF EXISTS loms.orders;
DROP TYPE IF EXISTS loms.order_status;