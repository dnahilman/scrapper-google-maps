package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/dnahilman/scrapper-go/internal/config"
	"github.com/dnahilman/scrapper-go/internal/emsifa"
	"github.com/dnahilman/scrapper-go/internal/logstream"
	"github.com/dnahilman/scrapper-go/internal/queue"
	"github.com/dnahilman/scrapper-go/internal/storage"
)

// Deps holds everything the API handlers need.
type Deps struct {
	Cfg       *config.MasterConfig
	DB        *gorm.DB
	Cities    *storage.CitiesRepo
	Kelurahan *storage.KelurahanRepo
	Workers   *storage.WorkersRepo
	Jobs      *storage.JobsRepo
	Tasks     *storage.TasksRepo
	Places    *storage.PlacesRepo
	Reviews   *storage.ReviewsRepo
	Sync      *storage.SyncRepo
	Queue     *queue.PostgresQueue
	Emsifa    *emsifa.Client
	Seeder    *emsifa.Seeder
	Hub       *logstream.Hub
}

// NewRouter wires up all routes.
func NewRouter(d *Deps) *gin.Engine {
	if d.Cfg.LogFormat != "console" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(zerologRequestLogger())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     d.Cfg.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: false,
	}))

	r.GET("/api/v1/health", healthHandler(d))

	v1 := r.Group("/api/v1")
	{
		registerCityRoutes(v1, d)
		registerJobRoutes(v1, d)
		registerTaskRoutes(v1, d)
		registerWorkerRoutes(v1, d)
		registerPlaceRoutes(v1, d)
		registerSyncRoutes(v1, d)
	}

	internal := r.Group("/api/v1/internal", masterTokenAuth(d.Cfg.MasterToken))
	registerInternalRoutes(internal, d)

	// WebSocket — same-origin only via CORS / no auth (clients are first-party).
	if d.Hub != nil {
		registerWSRoute(r, d.Hub)
	}

	// Static frontend (Svelte build)
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	return r
}
