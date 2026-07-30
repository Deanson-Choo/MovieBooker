package models

// Defines the Movie struct used across both the API and worker services.
// Maps to the movies table in PostgreSQL and is also serialised to/from Redis cache.

type Movie struct {
	ID        int    `json:"id" db:"id"`
	Title     string `json:"title" db:"title"`
	PosterURL string `json:"poster_url" db:"poster_url"`
}
