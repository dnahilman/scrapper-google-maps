package api

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dnahilman/scrapper-go/internal/domain"
)

func registerInternalRoutes(g *gin.RouterGroup, d *Deps) {
	g.POST("/workers/register", workerRegister(d))
	g.POST("/workers/:id/heartbeat", workerHeartbeat(d))
	g.POST("/tasks/claim", taskClaim(d))
	g.POST("/tasks/:id/heartbeat", taskHeartbeat(d))
	g.POST("/tasks/:id/ack", taskAck(d))
	g.POST("/tasks/:id/nack", taskNack(d))
	g.POST("/places", submitPlaces(d))
}

type WorkerRegisterRequest struct {
	Name           string                 `json:"name" binding:"required"`
	Hostname       string                 `json:"hostname,omitempty"`
	MaxConcurrency int                    `json:"max_concurrency"`
	Capabilities   map[string]any         `json:"capabilities,omitempty"`
	Metadata       map[string]any         `json:"metadata,omitempty"`
}

func workerRegister(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req WorkerRegisterRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ip := c.ClientIP()
		capBytes, _ := json.Marshal(req.Capabilities)
		metaBytes, _ := json.Marshal(req.Metadata)
		conc := req.MaxConcurrency
		if conc <= 0 || conc > 16 {
			conc = 2
		}
		w := &domain.Worker{
			Name:           req.Name,
			Hostname:       req.Hostname,
			IPAddr:         ip,
			MaxConcurrency: conc,
			Capabilities:   domain.JSONB(capBytes),
			Metadata:       domain.JSONB(metaBytes),
		}
		if err := d.Workers.Register(c.Request.Context(), w); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, w)
	}
}

func workerHeartbeat(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := d.Workers.Heartbeat(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type ClaimRequest struct {
	WorkerID uuid.UUID `json:"worker_id" binding:"required"`
}

func taskClaim(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ClaimRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		ct, err := d.Queue.Claim(c.Request.Context(), req.WorkerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if ct == nil {
			c.JSON(http.StatusNoContent, nil)
			return
		}
		c.JSON(http.StatusOK, ct)
	}
}

func taskHeartbeat(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := d.Queue.Heartbeat(c.Request.Context(), id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type AckRequest struct {
	PlacesCount int    `json:"places_count"`
	ResultPath  string `json:"result_path,omitempty"`
}

func taskAck(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req AckRequest
		_ = c.ShouldBindJSON(&req)
		if err := d.Queue.Ack(c.Request.Context(), id, req.PlacesCount, req.ResultPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type NackRequest struct {
	Error string `json:"error"`
}

func taskNack(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		var req NackRequest
		_ = c.ShouldBindJSON(&req)
		if err := d.Queue.Nack(c.Request.Context(), id, req.Error); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

type SubmitPlacesRequest struct {
	TaskID      uuid.UUID              `json:"task_id" binding:"required"`
	KelurahanID uuid.UUID              `json:"kelurahan_id" binding:"required"`
	Keyword     string                 `json:"keyword" binding:"required"`
	Places      []domain.PlacePayload  `json:"places"`
}

// submitPlaces accepts a batch of scraped places + their reviews from a worker.
func submitPlaces(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req SubmitPlacesRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		count := 0
		for _, p := range req.Places {
			place := payloadToPlace(&req, &p)
			if err := d.Places.Upsert(c.Request.Context(), place); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "place_id": p.PlaceID})
				return
			}
			reviews := make([]domain.Review, 0, len(p.UserReviews))
			for _, rv := range p.UserReviews {
				rating := rv.Rating
				ageDays := rv.AgeDays
				orJSON, _ := json.Marshal(rv.OwnerResponse)
				reviews = append(reviews, domain.Review{
					PlaceID:        p.PlaceID,
					ReviewID:       rv.ReviewID,
					Name:           rv.Name,
					ProfilePicture: rv.ProfilePicture,
					Rating:         &rating,
					Description:    rv.Description,
					Images:         rv.Images,
					WhenText:       rv.When,
					AgeDays:        &ageDays,
					OwnerResponse:  domain.JSONB(orJSON),
					Extended:       rv.Extended,
				})
			}
			if len(reviews) > 0 {
				if err := d.Reviews.UpsertMany(c.Request.Context(), reviews); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
			}
			count++
		}
		c.JSON(http.StatusOK, gin.H{"upserted": count})
	}
}

func payloadToPlace(req *SubmitPlacesRequest, p *domain.PlacePayload) *domain.Place {
	addrJSON, _ := json.Marshal(p.CompleteAddress)
	hoursJSON, _ := json.Marshal(p.OpenHours)
	popJSON, _ := json.Marshal(p.PopularTimes)
	rprJSON, _ := json.Marshal(p.ReviewsPerRating)
	imagesJSON, _ := json.Marshal(p.Images)
	resJSON, _ := json.Marshal(p.Reservations)
	ooJSON, _ := json.Marshal(p.OrderOnline)
	menuJSON, _ := json.Marshal(p.Menu)
	ownerJSON, _ := json.Marshal(p.Owner)
	aboutJSON, _ := json.Marshal(p.About)

	out := &domain.Place{
		TaskID:           &req.TaskID,
		KelurahanID:      &req.KelurahanID,
		Keyword:          req.Keyword,
		PlaceID:          p.PlaceID,
		DataID:           p.DataID,
		Cid:              p.Cid,
		Title:            p.Title,
		Categories:       p.Categories,
		Category:         p.Category,
		Address:          p.Address,
		CompleteAddress:  domain.JSONB(addrJSON),
		OpenHours:        domain.JSONB(hoursJSON),
		PopularTimes:     domain.JSONB(popJSON),
		Website:          p.WebSite,
		Phone:            p.Phone,
		PlusCode:         p.PlusCode,
		ReviewCount:      p.ReviewCount,
		ReviewRating:     p.ReviewRating,
		ReviewsPerRating: domain.JSONB(rprJSON),
		Latitude:         p.Latitude,
		Longitude:        p.Longtitude,
		Status:           ifEmpty(p.Status, "active"),
		Description:      p.Description,
		ReviewsLink:      p.ReviewsLink,
		Thumbnail:        p.Thumbnail,
		Timezone:         p.Timezone,
		PriceRange:       p.PriceRange,
		Images:           domain.JSONB(imagesJSON),
		Reservations:     domain.JSONB(resJSON),
		OrderOnline:      domain.JSONB(ooJSON),
		Menu:             domain.JSONB(menuJSON),
		Owner:            domain.JSONB(ownerJSON),
		About:            domain.JSONB(aboutJSON),
		Emails:           p.Emails,
	}
	return out
}

func ifEmpty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

