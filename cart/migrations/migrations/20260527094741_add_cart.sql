-- +goose Up
CREATE TABLE cart_items
(
    user_id BIGINT NOT NULL CHECK (user_id > 0),
    sku     BIGINT NOT NULL CHECK (sku >= 0),
    count   BIGINT NOT NULL CHECK (count > 0),

    PRIMARY KEY (user_id, sku)
);

-- +goose Down
DROP TABLE IF EXISTS cart_items
