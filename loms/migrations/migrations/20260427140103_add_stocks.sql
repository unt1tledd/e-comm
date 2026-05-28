-- +goose Up
CREATE TABLE loms.available_stocks
(
    sku   BIGINT NOT NULL PRIMARY KEY REFERENCES loms.products (sku) ON DELETE CASCADE,
    count BIGINT NOT NULL DEFAULT 0 CHECK (count >= 0)
);
 
CREATE TABLE loms.reserved_stocks
(
    sku      BIGINT NOT NULL,
    user_id BIGINT  NOT NULL,
    count   BIGINT NOT NULL CHECK (count > 0),

    PRIMARY KEY (sku, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS loms.reserved_stocks;
DROP TABLE IF EXISTS loms.available_stocks;
