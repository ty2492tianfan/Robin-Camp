package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Movie_Creat struct {
	ID          int     `json:"id"`
	Title       string  `json:"title"`
	Genre       string  `json:"genre"`
	ReleaseDate string  `json:"releaseDate"`
	Distributor *string `json:"distributor,omitempty"`
	Budget      *int64  `json:"budget,omitempty"`
	MpaRating   *string `json:"mpaRating,omitempty"`
}

type BoxOffice struct {
	Revenue     Revenue `json:"revenue"`
	Currency    string  `json:"currency"`
	Source      string  `json:"source"`
	LastUpdated string  `json:"lastUpdated"`
}

type Movie struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	ReleaseDate string     `json:"releaseDate"`
	Genre       string     `json:"genre"`
	Distributor *string    `json:"distributor,omitempty"`
	Budget      *int64     `json:"budget,omitempty"`
	MpaRating   *string    `json:"mpaRating,omitempty"`
	BoxOffice   *BoxOffice `json:"boxOffice,omitempty"`
}

type Revenue struct {
	Worldwide         int64  `json:"worldwide"`
	OpeningWeekendUSA *int64 `json:"openingWeekendUSA,omitempty"`
}

type upstreamRevenue struct {
	Worldwide         int64  `json:"worldwide"`
	OpeningWeekendUSA *int64 `json:"openingWeekendUSA,omitempty"`
}

type upstreamBoxOffice struct {
	Title       string          `json:"title"`
	Distributor *string         `json:"distributor,omitempty"`
	ReleaseDate *string         `json:"releaseDate,omitempty"`
	Budget      *int64          `json:"budget,omitempty"`
	Revenue     upstreamRevenue `json:"revenue"`
	MpaRating   *string         `json:"mpaRating,omitempty"`
}

type MovieListItem struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	ReleaseDate string     `json:"releaseDate"`
	Genre       string     `json:"genre"`
	BoxOffice   *BoxOffice `json:"boxOffice"`
}

func register_movie_api(r *gin.Engine, expectedToken string) {
	r.GET("/movies", listMoviesHandler)
	auth := r.Group("/")
	auth.Use(RequireBearer(expectedToken))
	auth.POST("/movies", createMovieHandler)
}

func createMovieHandler(c *gin.Context) {
	var req Movie_Creat
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    "INVALID_REQUEST",
			"message": "Invalid JSON body",
		})
		return
	}
	if req.Title == "" || req.Genre == "" || req.ReleaseDate == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "title, genre, releaseDate are required",
		})
		return
	}
	rd, err := time.Parse("2006-01-02", req.ReleaseDate)
	if err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"code":    "VALIDATION_ERROR",
			"message": "releaseDate must be YYYY-MM-DD",
		})
		return
	}
	const q = `
        INSERT INTO movies (title, genre, release_date, distributor, budget, mpa_rating)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, title, release_date, genre, distributor, budget, mpa_rating, box_office
    `
	var (
		id           int64
		title        string
		releaseDate  time.Time
		genre        string
		distributor  *string
		budget       *int64
		mpaRating    *string
		boxOfficeRaw []byte
	)

	if err := db.QueryRowContext(
		c.Request.Context(),
		q,
		req.Title,
		req.Genre,
		rd,
		req.Distributor,
		req.Budget,
		req.MpaRating,
	).Scan(&id, &title, &releaseDate, &genre, &distributor, &budget, &mpaRating, &boxOfficeRaw); err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "unique") || strings.Contains(strings.ToLower(msg), "duplicate") {
			c.JSON(http.StatusBadRequest, gin.H{
				"code":    "BAD_REQUEST",
				"message": "title already exists",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    "INTERNAL_ERROR",
			"message": "create failed",
		})
		return
	}
	resp := Movie{
		ID:          fmt.Sprintf("m_%d", id),
		Title:       title,
		ReleaseDate: releaseDate.Format("2006-01-02"),
		Genre:       genre,
		Distributor: distributor,
		Budget:      budget,
		MpaRating:   mpaRating,
	}
	bo, status, err := fetchBoxOffice(req.Title)
	if err != nil {
		if status == http.StatusNotFound {
		} else {
		}
	} else if bo != nil {
		if resp.Distributor == nil && bo.Distributor != nil {
			resp.Distributor = bo.Distributor
		}
		if resp.Budget == nil && bo.Budget != nil {
			resp.Budget = bo.Budget
		}
		if resp.MpaRating == nil && bo.MpaRating != nil {
			resp.MpaRating = bo.MpaRating
		}
		merged := BoxOffice{
			Revenue: Revenue{
				Worldwide:         bo.Revenue.Worldwide,
				OpeningWeekendUSA: bo.Revenue.OpeningWeekendUSA,
			},
			Currency:    "USD",
			Source:      "BoxOfficeAPI",
			LastUpdated: time.Now().UTC().Format(time.RFC3339),
		}

		jb, _ := json.Marshal(merged)
		_, _ = db.ExecContext(
			c.Request.Context(),
			`
        UPDATE movies
           SET distributor = COALESCE($1, distributor),
               budget      = COALESCE($2, budget),
               mpa_rating  = COALESCE($3, mpa_rating),
               box_office  = $4::jsonb
         WHERE id = $5
        `,
			resp.Distributor,
			resp.Budget,
			resp.MpaRating,
			string(jb),
			id,
		)
		resp.BoxOffice = &merged
	}

	c.Header("Location", "/movies/"+title)
	c.JSON(http.StatusCreated, resp)
}

func listMoviesHandler(c *gin.Context) {
	q := c.Query("q")
	yearStr := c.Query("year")
	genre := c.Query("genre")
	distributor := c.Query("distributor")
	budgetStr := c.Query("budget")
	mpaRating := c.Query("mpaRating")
	cursorStr := c.Query("cursor")
	limit := 20
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 && n <= 100 {
			limit = n
		}
	}
	sqlStr := `
		SELECT id, title, release_date, genre, box_office::text
		FROM movies
		WHERE 1=1
	`
	args := []any{}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	if cursorStr != "" {
		if cid, err := strconv.ParseInt(cursorStr, 10, 64); err == nil {
			sqlStr += " AND id > " + arg(cid)
		}
	}
	if q != "" {
		sqlStr += " AND title ILIKE " + arg("%"+q+"%")
	}
	if yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil {
			sqlStr += " AND EXTRACT(YEAR FROM release_date) = " + arg(y)
		}
	}
	if genre != "" {
		sqlStr += " AND LOWER(genre) = LOWER(" + arg(genre) + ")"
	}
	if distributor != "" {
		sqlStr += " AND LOWER(distributor) = LOWER(" + arg(distributor) + ")"
	}
	if mpaRating != "" {
		sqlStr += " AND mpa_rating = " + arg(mpaRating)
	}
	if budgetStr != "" {
		if b, err := strconv.ParseInt(budgetStr, 10, 64); err == nil {
			sqlStr += " AND budget <= " + arg(b)
		}
	}
	sqlStr += " ORDER BY id ASC LIMIT " + arg(limit+1)

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "query failed"})
		return
	}
	defer rows.Close()

	type rowBrief struct {
		id          int64
		title       string
		releaseDate time.Time
		genre       string
		boxOffice   sql.NullString
	}

	items := make([]MovieListItem, 0, limit)
	var lastIncludedID int64
	hasMore := false

	for rows.Next() {
		var r rowBrief
		if err := rows.Scan(&r.id, &r.title, &r.releaseDate, &r.genre, &r.boxOffice); err != nil {
			continue
		}
		if len(items) == limit {
			hasMore = true
			break
		}

		item := MovieListItem{
			ID:          fmt.Sprintf("m_%d", r.id),
			Title:       r.title,
			ReleaseDate: r.releaseDate.Format("2006-01-02"),
			Genre:       r.genre,
			BoxOffice:   nil,
		}
		if r.boxOffice.Valid && r.boxOffice.String != "" {
			var bo BoxOffice
			if err := json.Unmarshal([]byte(r.boxOffice.String), &bo); err == nil {
				item.BoxOffice = &bo
			}
		}
		items = append(items, item)
		lastIncludedID = r.id
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": "scan failed"})
		return
	}
	var nextCursor *string
	if hasMore && lastIncludedID > 0 {
		idStr := strconv.FormatInt(lastIncludedID, 10)
		nextCursor = &idStr
	}
	c.JSON(http.StatusOK, gin.H{
		"items":      items,
		"nextCursor": nextCursor,
	})
}
