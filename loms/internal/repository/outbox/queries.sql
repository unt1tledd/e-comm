-- name: SendOutboxMessage :exec
INSERT INTO loms.outbox (idempotency_key, data, status, kind)
VALUES (
           sqlc.arg(idempotency_key),
           sqlc.arg(data),
           'CREATED'::loms.outbox_status,
           sqlc.arg(kind)
       )
ON CONFLICT (idempotency_key) DO NOTHING;

-- name: GetOutboxMessages :many
UPDATE loms.outbox
SET status = 'IN_PROGRESS'::loms.outbox_status
WHERE idempotency_key IN (
    SELECT idempotency_key
    FROM loms.outbox
    WHERE
        status = 'CREATED'::loms.outbox_status
       OR (
        status = 'IN_PROGRESS'::loms.outbox_status
            AND updated_at < now() - sqlc.arg(in_progress_ttl)::interval
        )
    ORDER BY created_at
    LIMIT sqlc.arg(batch_size)
        FOR UPDATE SKIP LOCKED
)
RETURNING idempotency_key, data, kind;

-- name: MarkOutboxMessagesAsProcessed :execrows
UPDATE loms.outbox
SET status = 'SUCCESS'::loms.outbox_status
WHERE idempotency_key = ANY(sqlc.arg(idempotency_keys)::text[]);

-- name: MarkOutboxMessagesAsRetryable :execrows
UPDATE loms.outbox
SET status = 'CREATED'::loms.outbox_status
WHERE idempotency_key = ANY(sqlc.arg(idempotency_keys)::text[]);