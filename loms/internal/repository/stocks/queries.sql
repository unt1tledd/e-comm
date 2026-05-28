-- name: CreateStock :exec
INSERT INTO loms.available_stocks (sku)
VALUES ($1);

-- name: DecrementAvailableStock :execrows
UPDATE loms.available_stocks
SET count = count - $2
WHERE sku = $1
  AND count >= $2;

-- name: AddToAvailableStock :exec
INSERT INTO loms.available_stocks AS a (sku, count)
VALUES ($1, $2)
ON CONFLICT (sku) DO UPDATE
    SET count = a.count + EXCLUDED.count;

-- name: GetAvailableStockAmount :one
SELECT COALESCE(
               (SELECT count
                FROM loms.available_stocks
                WHERE sku = $1),
               0
       )::integer AS count;

-- name: UpsertAvailableStock :exec
INSERT INTO loms.available_stocks (sku, count)
VALUES ($1, $2)
ON CONFLICT (sku) DO UPDATE SET count = EXCLUDED.count;
