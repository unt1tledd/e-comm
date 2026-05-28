package outbox

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/config"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	xerror "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/errors"
	"go.uber.org/zap"
)

type (
	outboxRepository interface {
		GetMessages(ctx context.Context, batchSize int, inProgressTTL time.Duration) ([]entity.OutboxMessage, error)
		MarkAsProcessed(ctx context.Context, idempotencyKeys []string) error
		MarkAsRetryable(ctx context.Context, idempotencyKeys []string) error
	}

	transactor interface {
		WithTx(ctx context.Context, f func(ctx context.Context) error) (err error)
	}
)

type GlobalHandler = func(kind entity.Kind) (KindHandler, error)
type KindHandler = func(ctx context.Context, data []byte) error

type Outbox interface {
	Start(ctx context.Context, workers int, batchSize int, waitTime time.Duration, inProgressTTL time.Duration)
}

var _ Outbox = (*outboxImpl)(nil)

type outboxImpl struct {
	logger           *zap.Logger
	outboxRepository outboxRepository
	globalHandler    GlobalHandler
	cfg              *config.Config
	transactor       transactor
}

func New(
	logger *zap.Logger,
	outboxRepository outboxRepository,
	globalHandler GlobalHandler,
	cfg *config.Config,
	transactor transactor,
) *outboxImpl {
	return &outboxImpl{
		logger:           logger,
		outboxRepository: outboxRepository,
		globalHandler:    globalHandler,
		cfg:              cfg,
		transactor:       transactor,
	}
}

func (o *outboxImpl) Start(
	ctx context.Context,
	workers int,
	batchSize int,
	fetchPeriod time.Duration,
	inProgressTTL time.Duration,
) {
	wg := new(sync.WaitGroup)

	for workerID := 1; workerID <= workers; workerID++ {
		wg.Go(func() {
			o.worker(ctx, batchSize, fetchPeriod, inProgressTTL)
		})
	}
}

func (o *outboxImpl) worker(
	ctx context.Context,
	batchSize int,
	waitTime time.Duration,
	inProgressTTL time.Duration,
) {
	ticker := time.NewTicker(waitTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := o.transactor.WithTx(ctx, func(ctx context.Context) error {
				messages, err := o.outboxRepository.GetMessages(ctx, batchSize, inProgressTTL)

				if err != nil {
					o.logger.Error("can not fetch messages from outbox", zap.Error(err))
					return err
				}

				successKeys := make([]string, 0, len(messages)/2)
				failedKeys := make([]string, 0, len(messages)/2)

				for i := 0; i < len(messages); i++ {
					message := messages[i]
					key := message.IdempotencyKey

					kindHandler, handlerErr := o.globalHandler(message.Kind)

					if handlerErr != nil {
						o.logger.Error("unexpected kind", zap.Error(err))
						continue
					}

					err = kindHandler(ctx, message.Data)

					if err != nil {
						failedKeys = append(failedKeys, key)
						o.logger.Error("kind error", zap.Error(err))
						continue
					}

					successKeys = append(successKeys, key)
				}

				err = o.outboxRepository.MarkAsProcessed(ctx, successKeys)

				var updateErr xerror.OutboxStatusUpdateError

				switch {
				case errors.As(err, &updateErr):
					o.logger.Warn("partial outbox status update",
						zap.Int("expected", updateErr.Expected),
						zap.Int64("actual", updateErr.Actual),
					)
				case err != nil:
					o.logger.Error("outbox processing failed", zap.Error(err))
					return err
				}

				err = o.outboxRepository.MarkAsRetryable(ctx, failedKeys)

				switch {
				case errors.As(err, &updateErr):
					o.logger.Warn("partial outbox status update",
						zap.Int("expected", updateErr.Expected),
						zap.Int64("actual", updateErr.Actual),
					)
				case err != nil:
					o.logger.Error("outbox processing failed", zap.Error(err))
					return err
				}

				return nil
			})

			if err != nil {
				o.logger.Error("worker stage error", zap.Error(err))
			}
		}
	}
}
