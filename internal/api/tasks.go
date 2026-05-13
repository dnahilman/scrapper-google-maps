package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func registerTaskRoutes(g *gin.RouterGroup, d *Deps) {
	g.GET("/tasks", listTasks(d))
	g.GET("/tasks/:id", getTask(d))
	g.POST("/tasks/:id/reset", resetTask(d))
}

func listTasks(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var jobID *uuid.UUID
		if s := c.Query("job_id"); s != "" {
			if id, err := uuid.Parse(s); err == nil {
				jobID = &id
			}
		}
		status := c.Query("status")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
		out, err := d.Tasks.List(c.Request.Context(), jobID, status, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func getTask(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		t, err := d.Tasks.Get(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, t)
	}
}

func resetTask(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := d.Tasks.Reset(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}
