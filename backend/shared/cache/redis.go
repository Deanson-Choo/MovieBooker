package cache

// Initialises and exposes a shared Redis client using go-redis.
// Reads connection config from REDIS_URL first, then falls back to REDIS_ADDR.
// Used by the API service for both catalog caching (TTL-based) and seat locking (SETNX).

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	CatalogCacheKey = "movies:all"
	CatalogCacheTTL = 1 * time.Hour // Cache catalog for 1 hour

	SeatCacheTTL = 10 * time.Minute // Cache seats for 10 minutes

	SessionTTL  = 10 * time.Minute // Valid session for 10 minutes
	SeatLockTTL = 10 * time.Minute // Lock seat for 10 minutes

	IdempotencyKeyTTL = 24 * time.Hour // Idempotency key valid for 24 hours

)

func ShowtimeSeatsKey(showtimeID string) string {
	return fmt.Sprintf("showtime:%s:seats", showtimeID)
}

func SeatLockKey(showtimeID, seatID string) string {
	return fmt.Sprintf("showtime:%s:seat:%s:lock", showtimeID, seatID)
}

func ExtractSeatIDFromLockKey(lockKey string) (int, error) {
	parts := strings.Split(lockKey, ":")

	if len(parts) != 5 {
		return -1, fmt.Errorf("invalid lock key format: %s", lockKey)
	}

	return strconv.Atoi(parts[3])
}

func SessionSeatsKey(showtimeID, sessionID string) string {
	return fmt.Sprintf("showtime:%s:session:%s:seats", showtimeID, sessionID)
}

func IdempotencyKey(idempotencyKey string) string {
	return fmt.Sprintf("idempotency:%s", idempotencyKey)
}

var Client *redis.Client

func Init() error {
	redisURL := os.Getenv("REDIS_URL")

	if redisURL != "" {
		options, err := redis.ParseURL(redisURL)
		if err != nil {
			return fmt.Errorf("redis: failed to parse REDIS_URL: %w", err)
		}

		Client = redis.NewClient(options)
	} else {
		addr := os.Getenv("REDIS_ADDR")
		if addr == "" {
			addr = "localhost:6379"
		}

		Client = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       0,
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := Client.Ping(ctx).Err(); err != nil {
		if redisURL != "" {
			return fmt.Errorf("redis: failed to connect using REDIS_URL: %w", err)
		}

		return fmt.Errorf("redis: failed to connect to %s: %w", os.Getenv("REDIS_ADDR"), err)
	}

	return nil
}
