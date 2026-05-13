package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var bootTime = time.Now()

func healthHandler(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		dbStatus := "ok"
		if sqlDB, err := d.DB.DB(); err != nil || sqlDB.Ping() != nil {
			dbStatus = "down"
		}
		workers, _ := d.Workers.List(c.Request.Context())
		online := 0
		for _, w := range workers {
			if w.Status == "online" {
				online++
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"status":         "ok",
			"db":             dbStatus,
			"uptime_seconds": int(time.Since(bootTime).Seconds()),
			"workers_total":  len(workers),
			"workers_online": online,
		})
	}
}
