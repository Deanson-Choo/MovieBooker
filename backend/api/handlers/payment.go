package handlers

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Deanson-Choo/MovieBooker/shared/cache"
	"github.com/Deanson-Choo/MovieBooker/shared/db"
	"github.com/Deanson-Choo/MovieBooker/shared/models"
	"github.com/gin-gonic/gin"

	"github.com/google/uuid"
)

// validateSeatsAndExtendTTL atomically verifies that every seat lock in the session
// is still held by the requesting session, then extends all seat lock TTLs and the
// session cart TTL to a fixed payment window.
//
// KEYS: [seat_key_1, ..., seat_key_n, session_key]  (session_key is always last)
// ARGV: [session_id, ttl_seconds]
//
// Returns: -1 = one or more locks expired, 0 = wrong owner, 1 = all verified and extended
const validateSeatsAndExtendTTL = `
	local session_id = ARGV[1]
	local ttl = tonumber(ARGV[2])
	local session_key = KEYS[#KEYS]

	for i = 1, #KEYS - 1 do
		local val = redis.call("GET", KEYS[i])
		if val == false then
			return -1
		elseif val ~= session_id then
			return 0
		end
	end

	for i = 1, #KEYS - 1 do
		redis.call("EXPIRE", KEYS[i], ttl)
	end
	redis.call("EXPIRE", session_key, ttl)

	return 1
`

type PaymentResponse struct {
	BookingID  string `json:"booking_id"`
	ShowtimeID string `json:"showtime_id"`
	SeatIDs    []int  `json:"seat_ids"`
}

func Pay(c *gin.Context) {
	ctx := c.Request.Context()

	idempotencyKey := c.GetHeader("Idempotency-Key")
	sessionID := c.GetHeader("X-Session-ID")
	if idempotencyKey == "" || sessionID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "Missing required headers: Idempotency-Key and X-Session-ID are required.",
		})
		return
	}

	var req struct {
		Email      string `json:"email" binding:"required,email"`
		ShowtimeID string `json:"showtime_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "Invalid request body",
		})
		return
	}

	idempotencyKeyRedis := cache.IdempotencyKey(idempotencyKey)

	locked, err := cache.Client.SetNX(ctx, idempotencyKeyRedis, "processing", cache.IdempotencyKeyTTL).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to verify idempotency key",
		})
		return
	}

	if !locked {
		// Now we fetch it to see if it's still processing or finished
		existingResponse, err := cache.Client.Get(ctx, idempotencyKeyRedis).Result()
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: "Failed to retrieve existing payment state",
			})
			return
		}

		if existingResponse == "processing" {
			c.JSON(http.StatusConflict, models.ErrorObject{
				Error:   "Conflict",
				Details: "Payment is already being processed for this idempotency key.",
			})
			return
		}

		// It's a finished payment. Return the cached JSON.
		var response PaymentResponse
		if err := json.Unmarshal([]byte(existingResponse), &response); err != nil {
			log.Printf("Failed to unmarshal cached response: %v", err)
			c.JSON(http.StatusOK, PaymentResponse{
				BookingID:  "unknown",
				ShowtimeID: "unknown",
				SeatIDs:    []int{},
			})
			return
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// -------------------------------------------------------
	// It is a new payment request. Proceed with processing.

	sessionKey := cache.SessionSeatsKey(req.ShowtimeID, sessionID)

	seatKeys, err := cache.Client.SMembers(ctx, sessionKey).Result()
	if err != nil {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to retrieve session cart",
		})
		return
	}

	if len(seatKeys) == 0 {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "No seats locked for this session",
		})
		return
	}

	// Reject if the session is about to expire — not enough runway to complete payment safely.
	ttl, err := cache.Client.TTL(ctx, sessionKey).Result()
	if err != nil {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to check session TTL",
		})
		return
	}
	if ttl < 30*time.Second {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusConflict, models.ErrorObject{
			Error:   "Conflict",
			Details: "Session is expiring soon, please re-select your seats",
		})
		return
	}

	// Verify all seat locks are owned by this session and extend TTLs to 3-minute payment window.
	luaKeys := append(seatKeys, sessionKey)
	luaResult, err := cache.Client.Eval(ctx, validateSeatsAndExtendTTL, luaKeys, sessionID, 180).Int()
	if err != nil {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to validate seat locks",
		})
		return
	}

	switch luaResult {
	case -1:
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusConflict, models.ErrorObject{
			Error:   "Conflict",
			Details: "One or more seat locks have expired",
		})
		return
	case 0:
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusConflict, models.ErrorObject{
			Error:   "Conflict",
			Details: "One or more seats are not locked by this session",
		})
		return
	}

	// To be used later for db transaction and response
	seatIDs := make([]int, len(seatKeys))
	for i, seatKey := range seatKeys {
		var seatID int
		seatID, err = cache.ExtractSeatIDFromLockKey(seatKey)
		if err != nil {
			cache.Client.Del(ctx, idempotencyKeyRedis)
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: "Failed to extract seat ID from lock key",
			})
			return
		}
		seatIDs[i] = seatID
	}

	// -------------------------------------------------------
	// Simulate payment processing delay, 90% success rate
	time.Sleep(2 * time.Second)

	success := rand.Intn(100) < 90 // 90% success rate
	if !success {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusPaymentRequired, models.ErrorObject{
			Error:   "Payment Failed",
			Details: "Payment could not be processed",
		})
		return
	}

	// -------------------------------------------------------
	// DB transaction to finalize booking and mark seats as sold
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		// Note: We simulate a refund by deleting the idempotency key, allowing the user to retry payment.
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "You have been refunded. Failed to begin database transaction",
		})
		return
	}

	defer tx.Rollback(ctx)

	query := `
		SELECT id, status FROM seats
		WHERE showtime_id = $1 AND id = ANY($2)
		FOR UPDATE;
	`
	rows, err := tx.Query(ctx, query, req.ShowtimeID, seatIDs)
	if err != nil {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "You have been refunded. Failed to lock seats in database",
		})
		return
	}
	defer rows.Close()

	lockedSeatsCount := 0
	for rows.Next() {
		var seatID int
		var status string
		if err := rows.Scan(&seatID, &status); err != nil {
			cache.Client.Del(ctx, idempotencyKeyRedis)
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: "You have been refunded. Failed to scan seat row",
			})
			return
		}
		if status != "available" {
			cache.Client.Del(ctx, idempotencyKeyRedis)
			c.JSON(http.StatusConflict, models.ErrorObject{
				Error:   "Conflict",
				Details: "You have been refunded. One or more seats are no longer available",
			})
			return
		}
		lockedSeatsCount++
	}

	if lockedSeatsCount != len(seatIDs) {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusConflict, models.ErrorObject{
			Error:   "Conflict",
			Details: "You have been refunded. Seat availability mismatch",
		})
		return
	}

	// Insert booking record
	bookingID := uuid.New().String()
	insertBookingsQuery := `
		INSERT INTO bookings (id, showtime_id, seat_ids, user_email)
		VALUES ($1, $2, $3, $4)
	`
	_, err = tx.Exec(ctx, insertBookingsQuery, bookingID, req.ShowtimeID, seatIDs, req.Email)
	if err != nil {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "You have been refunded. Failed to insert booking record",
		})
		return
	}

	// Mark seats as sold
	updateSeatsQuery := `
		UPDATE seats
		SET status = 'booked'
		WHERE showtime_id = $1 AND id = ANY($2)
	`
	commandTag, err := tx.Exec(ctx, updateSeatsQuery, req.ShowtimeID, seatIDs)
	if err != nil {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "You have been refunded. Failed to mark seats as sold",
		})
		return
	}

	if commandTag.RowsAffected() != int64(len(seatIDs)) {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusConflict, models.ErrorObject{
			Error:   "Conflict",
			Details: "You have been refunded. Seat availability changed during transaction",
		})
		return
	}

	if err := tx.Commit(ctx); err != nil {
		cache.Client.Del(ctx, idempotencyKeyRedis)
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "You have been refunded. Failed to commit transaction",
		})
		return
	}

	// Delete seat locks and session cart from Redis
	pipe := cache.Client.TxPipeline()
	for _, seatKey := range seatKeys {
		pipe.Del(ctx, seatKey)
	}
	pipe.Del(ctx, sessionKey)
	if _, err := pipe.Exec(ctx); err != nil {
		log.Printf("Warning: Failed to clean up Redis locks and session cart: %v", err)
	}

	// Invalidate the seat availability cache for the showtime
	cacheKey := cache.ShowtimeSeatsKey(req.ShowtimeID)
	if err := cache.Client.Del(ctx, cacheKey).Err(); err != nil {
		log.Printf("Warning: Failed to invalidate seat availability cache for showtime %s: %v", req.ShowtimeID, err)
	}

	// Cache the successful payment response for idempotency
	response := PaymentResponse{
		BookingID:  bookingID,
		ShowtimeID: req.ShowtimeID,
		SeatIDs:    seatIDs,
	}
	responseJSON, err := json.Marshal(response)
	if err != nil {
		log.Printf("Warning: Failed to marshal payment response for caching: %v", err)
		c.JSON(http.StatusOK, response)
		return
	}

	// Only cache if marshalling was successful
	if err := cache.Client.Set(ctx, idempotencyKeyRedis, responseJSON, cache.IdempotencyKeyTTL).Err(); err != nil {
		log.Printf("Warning: Failed to cache payment response for idempotency: %v", err)
	}

	c.JSON(http.StatusOK, response)
}
