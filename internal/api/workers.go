package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

func registerWorkerRoutes(g *gin.RouterGroup, d *Deps) {
	g.GET("/workers", listWorkers(d))
	g.POST("/workers/:id/drain", drainWorker(d))
	g.DELETE("/workers/:id", deleteWorker(d))
}

func listWorkers(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		out, err := d.Workers.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func drainWorker(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := d.Workers.SetStatus(c.Request.Context(), id, domain.WorkerStatusDraining); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func deleteWorker(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := d.Workers.Delete(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
