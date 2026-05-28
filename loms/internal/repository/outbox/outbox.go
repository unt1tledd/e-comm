package outbox

import (
	"context"
	"fmt"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	sqlc "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox/sqlc"

	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type (
	DB interface {
		Begin(ctx context.Context) (pgx.Tx, error)
		sqlc.DBTX
	}
)

type outboxRepository struct {
	queries *sqlc.Queries
	db      DB
}

func NewOutboxRepository(db DB) *outboxRepository {
	return &outboxRepository{
		queries: sqlc.New(db),
		db:      db,
	}
}

func (o *outboxRepository) getQueries(ctx context.Context) *sqlc.Queries {
	if tx, err := transactor.ExtractTx(ctx); err == nil {
		return sqlc.New(tx)
	}

	return o.queries
}

func (o *outboxRepository) SendOutboxMessage(ctx context.Context, message *entity.OutboxMessage) error {
	err := o.getQueries(ctx).SendOutboxMessage(ctx, sqlc.SendOutboxMessageParams{
		IdempotencyKey: message.IdempotencyKey,
		Data:           message.Data,
		Kind:           int32(message.Kind),
	})

	if err != nil {
		return fmt.Errorf("send outbox message: %w", err)
	}

	return nil
}

func (o *outboxRepository) GetMessages(
	ctx context.Context,
	batchSize int,
	inProgressTTL time.Duration,
) ([]entity.OutboxMessage, error) {
	rows, err := o.getQueries(ctx).GetOutboxMessages(ctx, sqlc.GetOutboxMessagesParams{
		InProgressTtl: pgtype.Interval{
			Microseconds: inProgressTTL.Microseconds(),
			Valid:        true,
		},
		BatchSize: int32(batchSize),
	})

	if err != nil {
		return nil, fmt.Errorf("get outbox messages: %w", err)
	}

	result := make([]entity.OutboxMessage, 0, len(rows))
	for _, row := range rows {
		result = append(result, entity.OutboxMessage{
			IdempotencyKey: row.IdempotencyKey,
			Data:           row.Data,
			Kind:           entity.Kind(row.Kind),
		})
	}

	return result, nil
}

func (o *outboxRepository) MarkAsProcessed(ctx context.Context, idempotencyKeys []string) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	cntRows, err := o.getQueries(ctx).MarkOutboxMessagesAsProcessed(ctx, idempotencyKeys)

	if err != nil {
		return fmt.Errorf("mark outbox messages as processed: %w", err)
	}

	if int(cntRows) != len(idempotencyKeys) {
		return xerror.OutboxStatusUpdateError{
			Expected: len(idempotencyKeys),
			Actual:   cntRows,
		}
	}

	return nil
}

func (o *outboxRepository) MarkAsRetryable(ctx context.Context, idempotencyKeys []string) error {
	if len(idempotencyKeys) == 0 {
		return nil
	}

	cntRows, err := o.getQueries(ctx).MarkOutboxMessagesAsRetryable(ctx, idempotencyKeys)
	if err != nil {
		return fmt.Errorf("mark outbox messages as retryable: %w", err)
	}

	if int(cntRows) != len(idempotencyKeys) {
		return xerror.OutboxStatusUpdateError{
			Expected: len(idempotencyKeys),
			Actual:   cntRows,
		}
	}

	return nil
}
