package main

import (
	"context"
	"log"
	"os"
	"time"

	authhandler "github.com/KhikmatovaNozee/orderFlow/internal/handler/auth"
	orderrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/order"
	productrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/product"
	userrepo "github.com/KhikmatovaNozee/orderFlow/internal/repository/user"
	authservice "github.com/KhikmatovaNozee/orderFlow/internal/service/auth"

	"github.com/KhikmatovaNozee/orderFlow/internal/router"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is not set")
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

	userRepository := userrepo.NewRepository(pool)

	authService := authservice.NewService(userRepository)

	authHandler := authhandler.NewHandler(authService)

	r := router.New(authHandler)

	log.Println("Server starting on :9999")

	if err := r.Run(":9999"); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
