package main

// Entry point for the API service.
// Initialises the Gin router, connects to PostgreSQL (via shared/db) and Redis (via shared/cache),
// registers all route handlers (catalog, booking, payment), attaches middleware (idempotency),
// and starts the HTTP server on the configured port.

import (
	"context"
	"log"
	"net/http"
	"os"
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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

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
	allowedOrigins := []string{"http://localhost:4000"}
	if frontendOrigin := os.Getenv("FRONTEND_URL"); frontendOrigin != "" {
		allowedOrigins = append(allowedOrigins, frontendOrigin)
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "Idempotency-Key", "X-Session-ID"},
		MaxAge:       12 * time.Hour,
	}))

	router.GET("/api/catalog", handlers.GetCatalog)

	admin := router.Group("/api/admin")
	admin.Use(func(c *gin.Context) {
		key := os.Getenv("ADMIN_API_KEY")
		if key == "" || c.GetHeader("Authorization") != "Bearer "+key {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}
		c.Next()
	})
	admin.POST("/catalog", handlers.AddMovie)

	router.GET("/api/booking/showtimes/:showtime_id/seats", handlers.GetSeats)
	router.POST("/api/booking/showtimes/:showtime_id/seats/:seat_id/lock", handlers.LockSeat)
	router.POST("/api/booking/showtimes/:showtime_id/seats/:seat_id/unlock", handlers.UnlockSeat)

	router.POST("/api/payment/pay", handlers.Pay)

	if err := router.Run(":" + port); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
