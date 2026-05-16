package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func registerPlaceRoutes(g *gin.RouterGroup, d *Deps) {
	g.GET("/places", listPlaces(d))
	g.GET("/places/:place_id", getPlace(d))
	g.GET("/places/:place_id/reviews", listReviews(d))
}

func listPlaces(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		keyword := c.Query("keyword")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
		var kelurahanID *uuid.UUID
		if s := c.Query("kelurahan_id"); s != "" {
			if id, err := uuid.Parse(s); err == nil {
				kelurahanID = &id
			}
		}
		out, err := d.Places.ListByKeyword(c.Request.Context(), keyword, kelurahanID, limit, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func getPlace(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := d.Places.Get(c.Request.Context(), c.Param("place_id"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, p)
	}
}

func listReviews(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		out, err := d.Reviews.ListByPlace(c.Request.Context(), c.Param("place_id"), limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}
}
