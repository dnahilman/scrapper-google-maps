package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dnahilman/scrapper-go/internal/logger"
)

func zerologRequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = uuid.NewString()
		}
		c.Set("request_id", rid)
		c.Header("X-Request-ID", rid)

		c.Next()

		// Skip noisy health-check logs.
		if c.Request.Method == "GET" && c.Request.URL.Path == "/api/v1/health" {
			return
		}

		latency := time.Since(start)
		logger.L().Info().
			Str("request_id", rid).
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", latency).
			Str("ip", c.ClientIP()).
			Msg("http")
	}
}

func masterTokenAuth(expected string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if expected == "" {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "MASTER_TOKEN not configured"})
			return
		}
		raw := c.GetHeader("Authorization")
		const prefix = "Bearer "
		if !strings.HasPrefix(raw, prefix) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}
		if raw[len(prefix):] != expected {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid token"})
			return
		}
		c.Next()
	}
}
