package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func healthz(c *gin.Context) { c.String(http.StatusOK, "ok") }

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	InitDB()
	defer db.Close()

	authToken := os.Getenv("AUTH_TOKEN")
	if authToken == "" {
		log.Fatalf("missing required env: %s", "AUTH_TOKEN")
	}

	rounter := gin.Default()
	_ = rounter.SetTrustedProxies(nil)
	rounter.GET("/healthz", healthz)
	register_movie_api(rounter, authToken)
	register_rating_api(rounter)
	log.Printf("listening on :%s ...", port)
	if err := rounter.Run(":" + port); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
