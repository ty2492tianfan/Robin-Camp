// Rate_handle.go
package main

import (
	"database/sql"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func register_rating_api(r *gin.Engine) {
	r.POST("/movies/:title/ratings", upsertRatingHandler)
	r.GET("/movies/:title/rating", getRatingAggregateHandler)
}

type ratingReq struct {
	Rating float64 `json:"rating"`
}

func upsertRatingHandler(c *gin.Context) {
	title := c.Param("title")
	var movieID int64
	if err := db.QueryRow(`SELECT id FROM movies WHERE title=$1`, title).Scan(&movieID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "movie not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "db error"})
		return
	}

	raterID := strings.TrimSpace(c.GetHeader("X-Rater-Id"))
	if raterID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    "UNAUTHORIZED",
			"message": "Missing or invalid authentication information",
		})
		return
	}
	var req ratingReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Invalid JSON body",
		})
		return
	}
	if req.Rating < 0.5 || req.Rating > 5.0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "rating must be in [0.5, 5.0]",
		})
		return
	}
	if int(req.Rating*10)%5 != 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "rating must be a multiple of 0.5",
		})
		return
	}
	const q = `
        INSERT INTO ratings (movie_id, rater_id, rating)
        VALUES ($1, $2, $3)
        ON CONFLICT (movie_id, rater_id)
        DO UPDATE SET rating = EXCLUDED.rating
        RETURNING (xmax = 0) AS inserted
    `
	var inserted bool
	if err := db.QueryRow(q, movieID, raterID, req.Rating).Scan(&inserted); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "upsert failed"})
		return
	}
	status := http.StatusOK
	if inserted {
		status = http.StatusCreated
		c.Header("Location", "/movies/"+title+"/ratings")
	}

	c.JSON(status, gin.H{
		"movieTitle": title,
		"raterId":    raterID,
		"rating":     req.Rating,
	})
}

func getRatingAggregateHandler(c *gin.Context) {
	title := c.Param("title")
	var movieID int64
	if err := db.QueryRow(`SELECT id FROM movies WHERE title=$1`, title).Scan(&movieID); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"code":    "NOT_FOUND",
				"message": "movie not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "db error"})
		return
	}
	var avg sql.NullFloat64
	var cnt int64
	if err := db.QueryRow(
		`SELECT AVG(rating)::float8, COUNT(*) FROM ratings WHERE movie_id=$1`,
		movieID,
	).Scan(&avg, &cnt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "aggregate failed"})
		return
	}
	rounded := 0.0
	if avg.Valid {
		rounded = math.Round(avg.Float64*10) / 10.0
	}
	c.JSON(http.StatusOK, gin.H{
		"average": rounded,
		"count":   cnt,
	})
}
