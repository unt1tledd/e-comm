-- name: AddCartItem :exec
INSERT INTO cart_items (user_id, sku, count)
VALUES ($1, $2, $3)
ON CONFLICT (user_id, sku)
DO UPDATE SET count = cart_items.count + EXCLUDED.count;

-- name: GetCartItems :many
SELECT sku, count 
FROM cart_items
WHERE user_id = $1;

-- name: DeleteCartItem :exec
DELETE FROM cart_items
WHERE user_id = $1 AND sku = $2;

-- name: ClearCart :exec
DELETE FROM cart_items
WHERE user_id = $1;