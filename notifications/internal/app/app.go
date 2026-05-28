package app

import (
	"context"
	"os/signal"
	"syscall"
	"time"

	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/config"
	kafkaconsumer "github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/consumer/kafka"
	"github.com/igoroutine-courses/microservices.ecommerce.notifications/internal/usecase/notifier"

	"go.uber.org/zap"
)

func Run(logger *zap.Logger, cfg *config.Config) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if len(cfg.KafkaBrokerAddrs()) == 0 {
		logger.Fatal("no kafka brokers configured (KAFKA_BROKERS)")
	}

	orderStatusNotifier, err := notifier.NewOrderStatusRestNotifier(logger, cfg)
	if err != nil {
		logger.Fatal("order status notifier init error", zap.Error(err))
	}

	consumerErrCh := make(chan error, 1)
	go func() {
		if err := kafkaconsumer.RunConsumer(ctx, logger, cfg, orderStatusNotifier); err != nil {
			consumerErrCh <- err
		}
		close(consumerErrCh)
	}()

	select {
	case <-ctx.Done():
	case err := <-consumerErrCh:
		if err != nil {
			logger.Error("kafka consumer stopped", zap.Error(err))
		}
		cancel()
	}

	logger.Info("shutting down notifications")
	time.Sleep(cfg.App.ShutdownGracefulDelay)
	logger.Info("notifications stopped")
}
