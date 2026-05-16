package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"

	"github.com/dnahilman/scrapper-go/internal/domain"
	"github.com/dnahilman/scrapper-go/internal/logstream"
)

type CreateJobRequest struct {
	CityID         uuid.UUID `json:"city_id" binding:"required"`
	Keyword        string    `json:"keyword" binding:"required"`
	KelurahanNames []string  `json:"kelurahan_names,omitempty"`
	KelurahanIDs   []string  `json:"kelurahan_ids,omitempty"`
	Options        map[string]any `json:"options,omitempty"`
	MaxAttempts    int       `json:"max_attempts,omitempty"`
}

func registerJobRoutes(g *gin.RouterGroup, d *Deps) {
	g.POST("/jobs", createJob(d))
	g.GET("/jobs", listJobs(d))
	g.GET("/jobs/:id", getJob(d))
	g.DELETE("/jobs/:id", deleteJob(d))
	g.POST("/jobs/:id/cancel", cancelJob(d))
	g.POST("/jobs/:id/retry-failed", retryFailed(d))
	g.GET("/jobs/:id/places", listJobPlaces(d))
	g.GET("/jobs/:id/export", exportJob(d))
}

func createJob(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateJobRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Resolve kelurahan list
		var kels []domain.Kelurahan
		var err error
		switch {
		case len(req.KelurahanNames) > 0:
			kels, err = d.Kelurahan.FindByCityAndNames(c.Request.Context(), req.CityID, req.KelurahanNames)
		case len(req.KelurahanIDs) > 0:
			ids := make([]uuid.UUID, 0, len(req.KelurahanIDs))
			for _, s := range req.KelurahanIDs {
				if id, perr := uuid.Parse(s); perr == nil {
					ids = append(ids, id)
				}
			}
			err = d.DB.Where("id IN ?", ids).Find(&kels).Error
		default:
			// Select ALL kelurahan in the city
			kels, err = d.Kelurahan.ListByCity(c.Request.Context(), req.CityID, "")
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if len(kels) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no kelurahan resolved for this job"})
			return
		}

		// Build job
		optsJSON, _ := json.Marshal(req.Options)
		job := &domain.Job{
			CityID:     req.CityID,
			Keyword:    req.Keyword,
			Status:     domain.JobStatusPending,
			Options:    domain.JSONB(optsJSON),
			TotalTasks: len(kels),
		}
		if err := d.Jobs.Create(c.Request.Context(), job); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Build tasks
		maxAttempts := req.MaxAttempts
		if maxAttempts <= 0 {
			maxAttempts = 3
		}
		tasks := make([]domain.Task, 0, len(kels))
		now := time.Now()
		for _, k := range kels {
			tasks = append(tasks, domain.Task{
				JobID:        job.ID,
				KelurahanID:  k.ID,
				Status:       domain.TaskStatusQueued,
				MaxAttempts:  maxAttempts,
				VisibleAfter: now,
				EnqueuedAt:   now,
			})
		}
		if err := d.Tasks.CreateMany(c.Request.Context(), tasks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = d.Jobs.UpdateStatus(c.Request.Context(), job.ID, domain.JobStatusRunning)
		if d.Hub != nil {
			d.Hub.Broadcast(logstream.EventJobUpdated, gin.H{"id": job.ID, "status": "running", "keyword": job.Keyword, "total_tasks": job.TotalTasks})
		}
		c.JSON(http.StatusCreated, job)
	}
}

func listJobs(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
		out, err := d.Jobs.List(c.Request.Context(), status, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func getJob(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		job, err := d.Jobs.Get(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		counts, _ := d.Tasks.CountByStatus(c.Request.Context(), id)
		c.JSON(http.StatusOK, gin.H{
			"job":     job,
			"task_counts": counts,
		})
	}
}

func cancelJob(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		if err := d.DB.Exec(
			`UPDATE tasks SET status='cancelled', completed_at=NOW() WHERE job_id=$1 AND status IN ('queued','in_progress')`,
			id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		_ = d.Jobs.UpdateStatus(c.Request.Context(), id, domain.JobStatusCancelled)
		if d.Hub != nil {
			d.Hub.Broadcast(logstream.EventJobUpdated, gin.H{"id": id, "status": "cancelled"})
		}
		c.Status(http.StatusNoContent)
	}
}

func deleteJob(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		force := c.Query("force") == "true"
		if !force {
			job, err := d.Jobs.Get(c.Request.Context(), id)
			if err == nil && job.Status == domain.JobStatusRunning {
				c.JSON(http.StatusConflict, gin.H{"error": "job is running; pass ?force=true to delete anyway"})
				return
			}
		}
		// Delete all places that belong to this job's tasks.
		if err := d.DB.Exec(`DELETE FROM places WHERE task_id IN (SELECT id FROM tasks WHERE job_id = ?)`, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := d.DB.Exec(`DELETE FROM jobs WHERE id = ?`, id).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if d.Hub != nil {
			d.Hub.Broadcast(logstream.EventJobUpdated, gin.H{"id": id, "deleted": true})
		}
		c.Status(http.StatusNoContent)
	}
}

func listJobPlaces(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
		if page < 1 {
			page = 1
		}
		if perPage < 1 || perPage > 200 {
			perPage = 20
		}
		offset := (page - 1) * perPage
		places, err := d.Places.ListByJob(c.Request.Context(), id, perPage, offset)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		total, err := d.Places.CountByJob(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"items":    places,
			"total":    total,
			"page":     page,
			"per_page": perPage,
		})
	}
}

func exportJob(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		format := c.DefaultQuery("format", "json")
		places, err := d.Places.ListAllByJob(c.Request.Context(), id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		filename := "job-" + id.String()[:8] + "-places"
		switch format {
		case "csv":
			c.Header("Content-Disposition", `attachment; filename="`+filename+`.csv"`)
			c.Header("Content-Type", "text/csv; charset=utf-8")
			w := csv.NewWriter(c.Writer)
			_ = w.Write([]string{"place_id", "title", "category", "address", "phone", "website",
				"review_rating", "review_count", "latitude", "longitude", "price", "status",
				"emails", "description", "scraped_at"})
			for _, p := range places {
				_ = w.Write([]string{
					p.PlaceID, p.Title, p.Category, p.Address, p.Phone, p.Website,
					strconv.FormatFloat(p.ReviewRating, 'f', 2, 64),
					strconv.Itoa(p.ReviewCount),
					strconv.FormatFloat(p.Latitude, 'f', 6, 64),
					strconv.FormatFloat(p.Longitude, 'f', 6, 64),
					p.Price, p.Status,
					string(mustJSONBytes(p.Emails)),
					p.Description,
					p.ScrapedAt.Format(time.RFC3339),
				})
			}
			w.Flush()
		case "xlsx":
			f := excelize.NewFile()
			sheet := "Places"
			_ = f.SetSheetName("Sheet1", sheet)
			headers := []string{"place_id", "title", "category", "address", "phone", "website",
				"review_rating", "review_count", "latitude", "longitude", "price", "status",
				"emails", "description", "scraped_at"}
			for i, h := range headers {
				cell, _ := excelize.CoordinatesToCellName(i+1, 1)
				_ = f.SetCellValue(sheet, cell, h)
			}
			for row, p := range places {
				vals := []any{
					p.PlaceID, p.Title, p.Category, p.Address, p.Phone, p.Website,
					p.ReviewRating, p.ReviewCount, p.Latitude, p.Longitude,
					p.Price, p.Status,
					string(mustJSONBytes(p.Emails)),
					p.Description,
					p.ScrapedAt.Format(time.RFC3339),
				}
				for col, v := range vals {
					cell, _ := excelize.CoordinatesToCellName(col+1, row+2)
					_ = f.SetCellValue(sheet, cell, v)
				}
			}
			var buf bytes.Buffer
			if err := f.Write(&buf); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.Header("Content-Disposition", `attachment; filename="`+filename+`.xlsx"`)
			c.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
		default: // json
			c.Header("Content-Disposition", `attachment; filename="`+filename+`.json"`)
			c.JSON(http.StatusOK, places)
		}
	}
}

func mustJSONBytes(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func retryFailed(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := uuid.Parse(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
			return
		}
		res := d.DB.Exec(
			`UPDATE tasks SET status='queued', attempt=0, worker_id=NULL,
			    visible_after=NOW(), last_error='', completed_at=NULL
			WHERE job_id=$1 AND status='failed'`, id)
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}
		_ = d.Jobs.UpdateStatus(c.Request.Context(), id, domain.JobStatusRunning)
		_ = d.Jobs.RefreshCounters(c.Request.Context(), id)
		if d.Hub != nil {
			d.Hub.Broadcast(logstream.EventJobUpdated, gin.H{"id": id, "status": "running"})
		}
		c.JSON(http.StatusOK, gin.H{"requeued": res.RowsAffected})
	}
}
