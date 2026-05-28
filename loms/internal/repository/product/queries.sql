-- name: CreateProduct :one
INSERT INTO loms.products (name, price)
VALUES ($1, $2)
RETURNING sku;

-- name: GetProduct :one
SELECT sku, name, price
FROM loms.products
WHERE sku = $1;