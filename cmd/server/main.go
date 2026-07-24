package main

import (
	"log"

	"github.com/KhikmatovaNozee/orderFlow/internal/router"
)

func main() {
	r := router.New()

	log.Println("Server starting on :9999")
	if err := r.Run(":9999"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
