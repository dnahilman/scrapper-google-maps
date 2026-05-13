package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/dnahilman/scrapper-go/internal/domain"
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
	g.POST("/jobs/:id/cancel", cancelJob(d))
	g.POST("/jobs/:id/retry-failed", retryFailed(d))
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
		c.Status(http.StatusNoContent)
	}
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
		c.JSON(http.StatusOK, gin.H{"requeued": res.RowsAffected})
	}
}
