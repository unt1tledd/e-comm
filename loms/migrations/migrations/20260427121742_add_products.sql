-- +goose Up
CREATE TABLE loms.products
(
    sku    BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY ,
    name   VARCHAR(255) NOT NULL,
    price  BIGINT NOT NULL CHECK (price > 0)
);

-- +goose Down
DROP TABLE IF EXISTS loms.products;