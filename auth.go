package main

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RequireBearer(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authz := c.GetHeader("Authorization")
		if !strings.HasPrefix(authz, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Missing or invalid authentication information",
			})
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer"))
		if token == "" || token != expected {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    "UNAUTHORIZED",
				"message": "Missing or invalid authentication information",
			})
			return
		}
		c.Next()
	}
}
