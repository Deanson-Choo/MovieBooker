package models

import "time"

type Showtime struct {
	ID       int       `json:"id"`
	MovieID  int       `json:"movie_id"`
	StartsAt time.Time `json:"starts_at"`
	Hall     string    `json:"hall"`
}
