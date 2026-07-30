package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	authhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/auth"
	healthhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/health"
	orderhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/order"
	producthandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/product"
	statshandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/stats"
	"github.com/KhikmatovaNozee/orderFlow/internal/logger"
	orderrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/order"
	productrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/product"
	statsrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/stats"
	refreshtoken "github.com/KhikmatovaNozee/orderFlow/internal/repository/token"
	userrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/user"
	"github.com/KhikmatovaNozee/orderFlow/internal/router"
	authservice "github.com/KhikmatovaNozee/orderFlow/internal/service/auth"
	orderservice "github.com/KhikmatovaNozee/orderFlow/internal/service/order"
	productservice "github.com/KhikmatovaNozee/orderFlow/internal/service/product"
	statsservice "github.com/KhikmatovaNozee/orderFlow/internal/service/stats"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	addr = ":8080"

	startupTimeout = 10 * time.Second

	shutdownTimeout = 10 * time.Second

	readHeaderTimeout = 5 * time.Second
)

func main() {
	log := logger.New(os.Getenv("LOG_LEVEL"))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("server stopped with error", slog.Any("error", err))
		os.Exit(1)
	}

	log.Info("server stopped cleanly")
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return errors.New("JWT_SECRET is not set")
	}

	startupCtx, cancelStartup := context.WithTimeout(ctx, startupTimeout)
	defer cancelStartup()

	pool, err := pgxpool.New(startupCtx, databaseURL)
	if err != nil {
		return fmt.Errorf("create postgres pool: %w", err)
	}

	defer func() {
		log.Info("closing database pool")
		pool.Close()
	}()

	if err := pool.Ping(startupCtx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	if err := runDDL(startupCtx, pool); err != nil {
		return err
	}

	userRepository := userrepo.NewRepository(pool)
	refreshTokenRepository := refreshtoken.NewRepository(pool)
	jwtService := authservice.NewJWTService(jwtSecret)

	authSvc := authservice.NewService(
		userRepository,
		refreshTokenRepository,
		jwtService,
	)
	authHandler := authhandler.NewHandler(authSvc, jwtService)
	healthHandler := healthhandler.NewHandler(pool)

	productRepository := productrepo.New(pool)
	productSvc := productservice.NewService(productRepository)
	productHandler := producthandler.NewHandler(productSvc)

	orderRepository := orderrepo.New(pool)
	orderSvc := orderservice.NewService(orderRepository)
	orderHandler := orderhandler.NewHandler(orderSvc)
	statsRepository := statsrepo.New(pool)
	statsSvc := statsservice.NewService(statsRepository)
	statsHandler := statshandler.NewHandler(statsSvc)

	engine := router.New(log, authHandler, jwtService, healthHandler, productHandler, orderHandler, statsHandler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           engine,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		log.Info("server starting", slog.String("addr", addr))

		err := srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("listen and serve: %w", err)
		}
		return nil

	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	log.Info("http server stopped accepting connections")

	return nil
}

func runDDL(ctx context.Context, pool *pgxpool.Pool) error {
	migrations := []struct {
		name string
		run  func(context.Context, *pgxpool.Pool) error
	}{
		{"users", userrepo.RunDDL},
		{"products", productrepo.RunDDL},
		{"orders", orderrepo.RunDDL},
		{"refresh_tokens", refreshtoken.RunDDL},
	}

	for _, m := range migrations {
		if err := m.run(ctx, pool); err != nil {
			return fmt.Errorf("run %s ddl: %w", m.name, err)
		}
	}

	return nil
}
