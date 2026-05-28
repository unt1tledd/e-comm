package app

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os/signal"
	"syscall"

	grpcruntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	lomsadapter "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/adapter/loms/grpc"
	productadapter "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/adapter/product/grpc"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/config"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/controller"
	cartapi "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/controller/cart"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/internal/repository/cart"
	cartservice "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/cart"
	itemservice "github.com/igoroutine-courses/microservices.ecommerce.cart/internal/usecase/item"
	"github.com/igoroutine-courses/microservices.ecommerce.cart/migrations"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/db"
	cartpb "github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/cart/api/cart/v1"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/loms/v1"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/product/v1"
	"github.com/igoroutine-courses/microservices.ecommerce.pkg/generated/loms/api/stocks/v1"
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

	cartRepo := cart.NewPostgresReposytory(dbPool)

	lomsConn, err := grpc.NewClient(cfg.Clients.LOMSGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("failed to connect server", zap.Error(err))
	}

	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(lomsConn)

	productClient := productadapter.NewProductClient(product.NewProductServiceClient(lomsConn))
	lomsClient := lomsadapter.NewLOMSClient(loms.NewLomsClient(lomsConn), stocks.NewStocksClient(lomsConn))

	cartService := cartservice.NewCartService(cartRepo, productClient, lomsClient)
	itemService := itemservice.NewItemService(cartRepo, productClient, lomsClient)

	controller := controller.New(cartapi.NewCartServer(itemService, cartService, logger))

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
	if err := cartpb.RegisterCartHandlerFromEndpoint(ctx, mux, grpcAddr, opts); err != nil {
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

	cartpb.RegisterCartServer(s, ctrl.GetSrv())

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
