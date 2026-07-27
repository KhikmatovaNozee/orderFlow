package main

import (
	"context"
	"log"
	"os"
	"time"

	authhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/auth"
	orderrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/order"
	productrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/product"
	refreshtoken "github.com/KhikmatovaNozee/orderFlow/internal/repository/token"
	userrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/user"
	"github.com/KhikmatovaNozee/orderFlow/internal/router"
	authservice "github.com/KhikmatovaNozee/orderFlow/internal/service/auth"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("create postgres pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("ping postgres: %v", err)
	}

	if err := userrepo.RunDDL(ctx, pool); err != nil {
		log.Fatalf("run user ddl: %v", err)
	}

	if err := productrepo.RunDDL(ctx, pool); err != nil {
		log.Fatalf("run product ddl: %v", err)
	}

	if err := orderrepo.RunDDL(ctx, pool); err != nil {
		log.Fatalf("run order ddl: %v", err)
	}

	if err := refreshtoken.RunDDL(ctx, pool); err != nil {
		log.Fatalf("run refresh token ddl: %v", err)
	}

	userRepository := userrepo.NewRepository(pool)
	refreshTokenRepository := refreshtoken.NewRepository(pool)
	jwtService := authservice.NewJWTService(jwtSecret)

	authService := authservice.NewService(
		userRepository,
		refreshTokenRepository,
		jwtService,
	)
	authHandler := authhandler.NewHandler(
		authService,
		jwtService,
	)
	r := router.New(authHandler, jwtService)

	log.Println("Server starting on :8080")

	if err := r.Run(":8080"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
