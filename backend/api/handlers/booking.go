package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Deanson-Choo/MovieBooker/shared/cache"
	"github.com/Deanson-Choo/MovieBooker/shared/db"
	"github.com/Deanson-Choo/MovieBooker/shared/models"

	"github.com/redis/go-redis/v9"

	"github.com/gin-gonic/gin"
)

func LockSeat(c *gin.Context) {
	showtimeID := c.Param("showtime_id")
	seatID := c.Param("seat_id")
	sessionID := c.GetHeader("X-Session-ID")

	if showtimeID == "" || seatID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "Missing required parameters: showtime_id, seat_id path parameters and X-Session-ID header are required.",
		})
		return
	}

	ctx := c.Request.Context()
	seatKey := cache.SeatLockKey(showtimeID, seatID)
	sessionKey := cache.SessionSeatsKey(showtimeID, sessionID)

	// Check if the session already exists and get its TTL
	ttl, err := cache.Client.TTL(ctx, sessionKey).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to check session TTL",
		})
		return
	}

	var expiresAt time.Time
	if ttl > 0 {
		// Subsequent seat in an existing session
		expiresAt = time.Now().Add(ttl)
	} else {
		// No valid session (First ever seat)
		expiresAt = time.Now().Add(cache.SessionTTL)
	}

	// Attempt to acquire a soft lock on the seat using SETNX with an expiration time.
	result, err := cache.Client.SetArgs(ctx, seatKey, sessionID, redis.SetArgs{
		Mode:     "NX",
		ExpireAt: expiresAt,
	}).Result()

	// Case 1 : System Failure
	if err != nil && err != redis.Nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to acquire seat lock",
		})
		return
	}

	// CASE 2: The Conflict Path
	if err == redis.Nil || result != "OK" {
		c.JSON(http.StatusConflict, models.ErrorObject{
			Error:   "Conflict",
			Details: "Seat is already locked by another session",
		})
		return
	}

	// CASE 3: The Success Path

	// Check hardlock status in PostgreSQL to ensure the seat is still available.
	var seatStatus string
	query := `SELECT status FROM seats WHERE showtime_id = $1 AND id = $2`

	err = db.Pool.QueryRow(ctx, query, showtimeID, seatID).Scan(&seatStatus)

	if err != nil {
		cache.Client.Del(ctx, seatKey) // rollback soft lock
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to check hard lock status in database",
		})
		return
	}

	if seatStatus != "available" {
		cache.Client.Del(ctx, seatKey) // rollback soft lock
		c.JSON(http.StatusConflict, models.ErrorObject{
			Error:   "Conflict",
			Details: "Seat is already being processed or has been booked by another user",
		})
		return
	}

	// Add seat to session
	pipe := cache.Client.TxPipeline()
	pipe.SAdd(ctx, sessionKey, seatKey)
	pipe.ExpireAt(ctx, sessionKey, expiresAt)
	if _, err = pipe.Exec(ctx); err != nil {
		cache.Client.Del(ctx, seatKey) // rollback soft lock
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to update session with locked seat",
		})
		return
	}

	c.JSON(http.StatusOK, nil)
}

// unlockScript atomically verifies ownership, deletes the seat lock,
// removes the seat from the session set, and cleans up the session if it becomes empty.
//
// KEYS[1]: seat lock key
// KEYS[2]: session cart key
// ARGV[1]: expected session ID
//
// Returns: {-1, 0} = key not found, {0, 0} = wrong owner,
//
//	{1, 0} = unlocked (session still has seats), {1, 1} = unlocked (session now empty)
const unlockScript = `
	local val = redis.call("GET", KEYS[1])
	if val == false then
		return {-1, 0}
	elseif val ~= ARGV[1] then
		return {0, 0}
	end

	redis.call("DEL", KEYS[1])
	redis.call("SREM", KEYS[2], KEYS[1])

	local remaining = redis.call("SCARD", KEYS[2])
	if remaining == 0 then
		redis.call("DEL", KEYS[2])
		return {1, 1}
	end
	return {1, 0}
`

func UnlockSeat(c *gin.Context) {
	showtimeID := c.Param("showtime_id")
	seatID := c.Param("seat_id")
	sessionID := c.GetHeader("X-Session-ID")

	if showtimeID == "" || seatID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "Missing required parameters: showtime_id, seat_id path parameters and X-Session-ID header are required.",
		})
		return
	}

	ctx := c.Request.Context()
	seatKey := cache.SeatLockKey(showtimeID, seatID)
	sessionKey := cache.SessionSeatsKey(showtimeID, sessionID)

	luaResult, err := cache.Client.Eval(ctx, unlockScript, []string{seatKey, sessionKey}, sessionID).Slice()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to execute unlock operation",
		})
		return
	}

	switch luaResult[0].(int64) {
	case -1:
		c.JSON(http.StatusNotFound, models.ErrorObject{
			Error:   "Not Found",
			Details: "Seat lock does not exist or has already expired",
		})
		return
	case 0:
		c.JSON(http.StatusForbidden, models.ErrorObject{
			Error:   "Forbidden",
			Details: "You do not own the lock for this seat",
		})
		return
	}

	c.JSON(http.StatusOK, nil)
}

func GetSeats(c *gin.Context) {
	ctx := c.Request.Context()
	var seats []models.Seat

	// Parameter Extraction
	showtimeID := c.Param("showtime_id")
	sessionID := c.GetHeader("X-Session-ID")

	// Basic validation
	if showtimeID == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "Missing required parameters: showtime_id path parameter and X-Session-ID header are required.",
		})
		return
	}

	// Check Redis cache for cached seats
	key := cache.ShowtimeSeatsKey(showtimeID)
	cachedBytes, err := cache.Client.Get(ctx, key).Bytes()

	// Cache hit
	if err == nil {
		json.Unmarshal(cachedBytes, &seats)
	}

	// If cache miss, query PostgreSQL for available seats
	if seats == nil {
		query := `
			SELECT id, showtime_id, label, status
			FROM seats
			WHERE showtime_id = $1
			ORDER BY id ASC;
		`

		rows, err := db.Pool.Query(ctx, query, showtimeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: "Error While Querying DB",
			})
			return
		}
		defer rows.Close()

		for rows.Next() {
			var s models.Seat
			if err := rows.Scan(&s.ID, &s.ShowtimeID, &s.Label, &s.Status); err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorObject{
					Error:   "Internal Server Error",
					Details: "Error While Scanning DB Rows",
				})
				return
			}
			seats = append(seats, s)
		}

		// Cache the seats in Redis
		jsonData, _ := json.Marshal(seats)
		if err := cache.Client.Set(ctx, key, jsonData, cache.SeatCacheTTL).Err(); err != nil {
			log.Printf("Something occurred while caching seats for showtime %s", showtimeID)
		}
	}

	lockKeys := make([]string, 0, len(seats))
	lockIndices := make([]int, 0, len(seats))
	for i, s := range seats {
		if s.Status == "available" {
			lockKeys = append(lockKeys, cache.SeatLockKey(showtimeID, fmt.Sprintf("%d", s.ID)))
			lockIndices = append(lockIndices, i)
		}
	}
	if len(lockKeys) > 0 {
		vals, err := cache.Client.MGet(ctx, lockKeys...).Result()
		if err != nil {
			log.Printf("Failed to fetch seat locks for showtime %s: %v", showtimeID, err)
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: "Failed to fetch seat lock statuses",
			})
			return
		}
		for j, val := range vals {
			if val != nil {
				lockOwner, ok := val.(string)
				if !ok {
					continue
				}
				originalIndex := lockIndices[j]

				// Compare the Redis value to the current request's session ID
				if lockOwner == sessionID {
					seats[originalIndex].Status = "selected"
				} else {
					seats[originalIndex].Status = "locked"
				}
			}
		}
	}

	// Get TTL for the session key to determine the expiration time
	sessionKey := cache.SessionSeatsKey(showtimeID, sessionID)
	ttl, err := cache.Client.TTL(ctx, sessionKey).Result()
	if err != nil {
		log.Printf("Failed to fetch TTL for session %s: %v", sessionID, err)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to fetch session TTL",
		})
		return
	}

	var expiresAt *int64
	if ttl > 0 {
		expirationTime := time.Now().Add(ttl).Unix()
		expiresAt = &expirationTime
	}

	// Return all seats as JSON response
	c.JSON(http.StatusOK, struct {
		Seats     []models.Seat `json:"seats"`
		ExpiresAt *int64        `json:"expires_at,omitempty"`
	}{
		Seats:     seats,
		ExpiresAt: expiresAt,
	})
}
