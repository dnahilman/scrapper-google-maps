package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func registerCityRoutes(g *gin.RouterGroup, d *Deps) {
	g.GET("/cities", listCities(d))
	g.GET("/cities/:idOrSlug", getCity(d))
	g.POST("/cities/sync", syncAllCities(d))
	g.GET("/cities/:idOrSlug/kelurahan", listKelurahan(d))
	g.POST("/cities/:idOrSlug/kelurahan/sync", syncKelurahan(d))
}

func listCities(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		out, err := d.Cities.List(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

// resolveCity accepts either UUID or slug.
func resolveCity(d *Deps, c *gin.Context) (uuid.UUID, error) {
	param := c.Param("idOrSlug")
	if id, err := uuid.Parse(param); err == nil {
		return id, nil
	}
	city, err := d.Cities.GetBySlug(c.Request.Context(), param)
	if err != nil {
		return uuid.Nil, err
	}
	return city.ID, nil
}

func getCity(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := resolveCity(d, c)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "city not found"})
			return
		}
		city, err := d.Cities.GetByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		count, _ := d.Kelurahan.CountByCity(c.Request.Context(), id)
		c.JSON(http.StatusOK, gin.H{
			"city":            city,
			"kelurahan_count": count,
		})
	}
}

func syncAllCities(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		n, err := d.Seeder.SyncAllCities(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"synced": n})
	}
}

func listKelurahan(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := resolveCity(d, c)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "city not found"})
			return
		}
		search := c.Query("search")
		out, err := d.Kelurahan.ListByCity(c.Request.Context(), id, search)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func syncKelurahan(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := resolveCity(d, c)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "city not found"})
			return
		}
		city, err := d.Cities.GetByID(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		n, err := d.Seeder.SyncKelurahan(c.Request.Context(), city)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"city": city.Name, "kelurahan": n})
	}
}
