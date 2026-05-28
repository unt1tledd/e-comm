-- name: CreateOrder :one
INSERT INTO loms.orders (user_id, status)
VALUES (
            sqlc.arg(user_id),
            'awaiting payment'::loms.order_status
        )
RETURNING id;

-- name: CreateOrderItem :exec
INSERT INTO loms.order_info (order_id, sku, count)
VALUES ($1, $2, $3);

-- name: GetOrder :one
SELECT o.id, o.user_id, o.status, o.created_at, o.updated_at
FROM loms.orders o
WHERE o.id = $1;

-- name: GetOrderForUpdateByID :one
SELECT id, user_id, status, created_at, updated_at
FROM loms.orders
WHERE id = $1
FOR UPDATE;

-- name: GetOrderItems :many
SELECT sku, count
FROM loms.order_info
WHERE order_id = $1;

-- name: UpdateOrderStatus :execrows
UPDATE loms.orders
SET status = $2
WHERE id = $1;
