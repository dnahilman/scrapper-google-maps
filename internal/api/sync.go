package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerSyncRoutes(g *gin.RouterGroup, d *Deps) {
	g.GET("/sync/status", syncStatus(d))
}

func syncStatus(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		stats, err := d.Sync.Stats(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, stats)
	}
}
