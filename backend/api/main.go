package main

// Entry point for the API service.
// Initialises the Gin router, connects to PostgreSQL (via shared/db) and Redis (via shared/cache),
// registers all route handlers (catalog, booking, payment), attaches middleware (idempotency),
// and starts the HTTP server on the configured port.

import (
	"context"
	"log"
	"time"

	"github.com/Deanson-Choo/MovieBooker/api/handlers"
	"github.com/Deanson-Choo/MovieBooker/shared/cache"
	"github.com/Deanson-Choo/MovieBooker/shared/db"

	"github.com/gin-contrib/cors"
	"github.com/joho/godotenv"

	"github.com/gin-gonic/gin"
)

func main() {
	// Load environment variables from .env file (if present)
	_ = godotenv.Load()

	// Initialise PostgreSQL connection pool
	if err := db.Init(context.Background()); err != nil {
		log.Fatalf("Failed to initialize PostgreSQL connection pool: %v", err)
	}
	defer db.Pool.Close()

	// Initialise Redis client
	if err := cache.Init(); err != nil {
		log.Fatalf("Failed to initialize Redis client: %v", err)
	}

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:4000"},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "Idempotency-Key", "X-Session-ID"},
		MaxAge:       12 * time.Hour,
	}))

	router.GET("/api/catalog", handlers.GetCatalog)
	// router.POST("/api/catalog", handlers.AddMovie) // Disable on production

	router.GET("/api/booking/showtimes/:showtime_id/seats", handlers.GetSeats)
	router.POST("/api/booking/showtimes/:showtime_id/seats/:seat_id/lock", handlers.LockSeat)
	router.POST("/api/booking/showtimes/:showtime_id/seats/:seat_id/unlock", handlers.UnlockSeat)

	router.POST("/api/payment/pay", handlers.Pay)

	router.Run()
}
