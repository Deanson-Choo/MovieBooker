package models

// Defines the Seat struct and related request/response types.
// Maps to the seats table in PostgreSQL.
// Also defines LockSeatRequest (used by POST /lock-seat)
// and the Redis key schema for seat locks (seat:<movieID>:<seatID>).

type Seat struct {
	ID         int    `json:"id"          db:"id"`
	ShowtimeID int    `json:"showtime_id" db:"showtime_id"`
	Label      string `json:"label"       db:"label"`  // e.g. "A1", "B3"
	Status     string `json:"status"      db:"status"` // "available", "booked"
}
