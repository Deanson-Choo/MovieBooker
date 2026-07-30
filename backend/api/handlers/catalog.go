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
	"github.com/gin-gonic/gin"
)

type MovieCatalogItem struct {
	ID        int               `json:"id"`
	Title     string            `json:"title"`
	PosterURL string            `json:"poster_url"`
	Showtimes []models.Showtime `json:"showtimes"`
}

func GetCatalog(c *gin.Context) {
	ctx := c.Request.Context()

	key := cache.CatalogCacheKey
	movies := make([]MovieCatalogItem, 0)

	// Attempt to get the cached catalog from Redis
	cachedBytes, err := cache.Client.Get(ctx, key).Bytes()

	// Cache Hit
	if err == nil {
		c.Data(http.StatusOK, "application/json", cachedBytes)
		return
	}

	// Cache Miss
	query := `
		SELECT
			m.id,
			m.title,
			m.poster_url,
			json_agg(
				json_build_object(
					'id', s.id,
					'movie_id', s.movie_id,
					'starts_at', to_char(s.starts_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
					'hall', s.hall
				)
			) AS showtimes
		FROM movies m
		JOIN showtimes s ON m.id = s.movie_id
		GROUP BY m.id, m.title, m.poster_url;
	`

	rows, err := db.Pool.Query(ctx, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: fmt.Sprintf("Error While Querying DB: %v", err),
		})
		return
	}

	defer rows.Close()

	for rows.Next() {
		var m MovieCatalogItem
		var rawShowtimes []byte // 💡

		err := rows.Scan(&m.ID, &m.Title, &m.PosterURL, &rawShowtimes)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: fmt.Sprintf("Error While Querying DB: %v", err),
			})
			return
		}

		if err := json.Unmarshal(rawShowtimes, &m.Showtimes); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: "Malformed JSON data from database",
			})
			return
		}

		movies = append(movies, m)
	}

	// Cache the catalog in Redis
	jsonData, err := json.Marshal(movies)
	if err != nil {
		log.Printf("Error marshalling catalog data: %v", err)
		c.JSON(http.StatusOK, movies) // Return the data even if caching fails
		return
	} else {
		if err := cache.Client.Set(ctx, key, jsonData, cache.CatalogCacheTTL).Err(); err != nil {
			log.Printf("Something occurred while caching catalog: %v", err)
		}
	}

	c.Data(http.StatusOK, "application/json", jsonData)
}

func AddMovie(c *gin.Context) {
	ctx := c.Request.Context()

	var req struct {
		Title     string `json:"title"`
		PosterURL string `json:"poster_url"`
		Showtimes []struct {
			StartsAt time.Time `json:"starts_at"`
			Hall     string    `json:"hall"`
		} `json:"showtimes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "Invalid request body",
		})
		return
	}

	if req.Title == "" {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "title is required",
		})
		return
	}

	if len(req.Showtimes) != 5 {
		c.JSON(http.StatusBadRequest, models.ErrorObject{
			Error:   "Bad Request",
			Details: "exactly 5 showtimes are required",
		})
		return
	}

	for i, st := range req.Showtimes {
		if st.Hall == "" {
			c.JSON(http.StatusBadRequest, models.ErrorObject{
				Error:   "Bad Request",
				Details: fmt.Sprintf("hall is required for showtime %d", i+1),
			})
			return
		}
		if st.StartsAt.IsZero() {
			c.JSON(http.StatusBadRequest, models.ErrorObject{
				Error:   "Bad Request",
				Details: fmt.Sprintf("starts_at is required for showtime %d", i+1),
			})
			return
		}
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to begin transaction",
		})
		return
	}
	defer tx.Rollback(ctx)

	// Insert movie
	var movieID int
	err = tx.QueryRow(ctx, `INSERT INTO movies (title, poster_url) VALUES ($1, $2) RETURNING id`, req.Title, req.PosterURL).Scan(&movieID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to insert movie",
		})
		return
	}

	for _, st := range req.Showtimes {
		var showtimeID int
		err = tx.QueryRow(ctx, `INSERT INTO showtimes (movie_id, starts_at, hall) VALUES ($1, $2, $3) RETURNING id`, movieID, st.StartsAt, st.Hall).Scan(&showtimeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorObject{
				Error:   "Internal Server Error",
				Details: "Failed to insert showtime",
			})
			return
		}

		// Insert 100 seats for this showtime
		for i := 1; i <= 100; i++ {
			_, err = tx.Exec(ctx, `INSERT INTO seats (showtime_id, label) VALUES ($1, $2)`, showtimeID, fmt.Sprintf("S%d", i))
			if err != nil {
				c.JSON(http.StatusInternalServerError, models.ErrorObject{
					Error:   "Internal Server Error",
					Details: "Failed to insert seat",
				})
				return
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorObject{
			Error:   "Internal Server Error",
			Details: "Failed to commit transaction",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Movie added", "movie_id": movieID})
}

// example payload:
// {
//   "title": "Inception",
//   "poster_url": "https://...",
//   "showtimes": [
//     { "starts_at": "2026-07-31T13:00:00Z", "hall": "Hall 1" },
//     { "starts_at": "2026-07-31T15:00:00Z", "hall": "Hall 2" },
//     { "starts_at": "2026-07-31T17:00:00Z", "hall": "Hall 1" },
//     { "starts_at": "2026-07-31T19:00:00Z", "hall": "Hall 3" },
//     { "starts_at": "2026-07-31T21:00:00Z", "hall": "Hall 2" }
//   ]
// }
