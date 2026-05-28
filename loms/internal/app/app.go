package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	notificationskafka "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/adapter/notifications/kafka"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/config"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller"
	lomsapi "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/loms"
	productapi "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/product"
	stocksapi "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/controller/stocks"
	entity "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/entity"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/internal/outbox"
	orderrp "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/order"
	outboxrepo "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/outbox"
	productrp "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/product"
	stocksrp "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/repository/stocks"
	lomsservice "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usercase/loms"
	productservice "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usercase/product"
	stocksservice "github.com/igoroutine-courses/microservices.ecommerce.loms/internal/usercase/stocks"
	"github.com/igoroutine-courses/microservices.ecommerce.loms/migrations"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/db"
	lomspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	productpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	stockspb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/transactor"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func Run(logger *zap.Logger, cfg *config.Config) {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pgxcfg, err := pgxpool.ParseConfig(cfg.ConstructPostgresURL())

	if err != nil {
		logger.Error("can not create pgxpool cfg", zap.Error(err))
		return
	}

	pgxcfg.MaxConns = cfg.PGPool.MaxConns
	pgxcfg.MinConns = cfg.PGPool.MinConns
	pgxcfg.HealthCheckPeriod = cfg.PGPool.HealthCheckPeriod
	pgxcfg.MaxConnLifetime = cfg.PGPool.MaxConnLifetime
	pgxcfg.MaxConnIdleTime = cfg.PGPool.MaxConnIdleTime

	dbPool, err := pgxpool.NewWithConfig(ctx, pgxcfg)
	if err != nil {
		logger.Error("can not create pgxpool", zap.Error(err))
		return
	}

	defer dbPool.Close()
	db.SetupPostgres(dbPool, logger, migrations.Migrations)

	orderRepo := orderrp.NewPostgresReposytory(dbPool)
	productRepo := productrp.NewPostgresReposytory(dbPool)
	stocksRepo := stocksrp.NewPostgresReposytory(dbPool)
	transactor := transactor.NewTransactor(dbPool)
	outboxRepository := outboxrepo.NewOutboxRepository(dbPool)

	kafkaBrokers := cfg.KafkaBrokerAddrs()
	if len(kafkaBrokers) == 0 {
		logger.Fatal("no kafka brokers configured (KAFKA_BROKERS)")
	}

	notificationPublisher := notificationskafka.NewPublisher(kafkaBrokers, cfg.Kafka.Topic)
	defer func() {
		if err = notificationPublisher.Close(); err != nil {
			logger.Error("close kafka notifications publisher", zap.Error(err))
		}
	}()

	lomsService := lomsservice.NewLomsService(orderRepo, stocksRepo, productRepo, outboxRepository, notificationPublisher, transactor)
	productService := productservice.NewProductService(productRepo, stocksRepo, transactor)
	stocksService := stocksservice.NewStocksService(stocksRepo, transactor)

	controller := controller.New(lomsapi.NewLomsServer(lomsService, logger), productapi.NewProductServer(productService), stocksapi.NewStocksServer(stocksService))

	globalOutboxHandler := func(kind entity.Kind) (outbox.KindHandler, error) {
		switch kind {
		case entity.KindNotification:
			return lomsService.OrderStatusChangedNotificationKindHandler, nil
		default:
			return nil, errors.New("unsupported outboxCore kind")
		}
	}

	outboxCore := outbox.New(logger, outboxRepository, globalOutboxHandler, cfg, transactor)

	httpServer, httpErrCh, err := runRest(ctx, logger, cfg)
	if err != nil {
		logger.Error("failed to start gateway", zap.Error(err))
		return
	}

	grpcServer, grpcErrCh, err := runGrpc(logger, cfg, controller)
	if err != nil {
		logger.Error("failed to start grpc server", zap.Error(err))
		return
	}

	outboxCore.Start(
		ctx,
		cfg.Outbox.Workers,
		cfg.Outbox.BatchSize,
		cfg.Outbox.FetchPeriod,
		cfg.Outbox.TTL,
	)

	select {
	case <-ctx.Done():
	case err := <-httpErrCh:
		logger.Error("gateway listen error", zap.Error(err))
		cancel()
	case err := <-grpcErrCh:
		logger.Error("grpc server listen error", zap.Error(err))
		cancel()
	}

	logger.Info("shutting down servers")

	shutdownCtx, shutdownCancel := context.WithTimeout(
		context.Background(),
		cfg.App.ShutdownGracefulDelay,
	)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("failed to shutdown http server", zap.Error(err))
	}

	grpcStopped := make(chan struct{})
	go func() {
		grpcServer.GracefulStop()
		close(grpcStopped)
	}()

	select {
	case <-grpcStopped:
	case <-shutdownCtx.Done():
		grpcServer.Stop()
		logger.Error("grpc graceful shutdown timed out", zap.Error(shutdownCtx.Err()))
	}

	logger.Info("servers stopped")
}

func runRest(ctx context.Context, logger *zap.Logger, cfg *config.Config) (*http.Server, <-chan error, error) {
	mux := grpcruntime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	grpcAddr := ":" + cfg.GRPC.Port
	if err := lomspb.RegisterLomsHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, nil, err
	}

	if err := productpb.RegisterProductServiceHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, nil, err
	}

	if err := stockspb.RegisterStocksHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
		return nil, nil, err
	}

	gatewayPort := ":" + cfg.GRPC.GatewayPort
	logger.Info("gateway listening at port", zap.String("port", gatewayPort))

	httpServer := &http.Server{
		Addr:    gatewayPort,
		Handler: corsHandler(mux, cfg),
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	return httpServer, errCh, nil
}

func runGrpc(logger *zap.Logger, cfg *config.Config, ctrl *controller.API) (*grpc.Server, <-chan error, error) {
	port := ":" + cfg.GRPC.Port
	lis, err := net.Listen("tcp", port)

	if err != nil {
		return nil, nil, err
	}

	s := grpc.NewServer()
	reflection.Register(s)

	ctrl.Register(s)

	logger.Info("grpc listening at port", zap.String("port", port))

	errCh := make(chan error, 1)
	go func() {
		if err := s.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			errCh <- err
		}
		close(errCh)
	}()

	return s, errCh, nil
}

func corsHandler(next http.Handler, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = cfg.CORS.DefaultAllowedOrigin
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
		w.Header().Set("Access-Control-Max-Age", cfg.CORS.MaxAgeInSeconds)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
