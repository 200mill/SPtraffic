package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sptraffic/backend/internal/api/handler"
	"github.com/sptraffic/backend/internal/api/middleware"
	"golang.org/x/time/rate"
)

// RouterConfig bundles all options for NewRouter.
type RouterConfig struct {
	CORSOrigin string
	AdminKey   string
}

// NewRouter builds and returns the Gin engine with all routes registered.
func NewRouter(
	termH *handler.TerminalHandler,
	searchH *handler.SearchHandler,
	schedH *handler.ScheduleHandler,
	regionH *handler.RegionHandler,
	healthH *handler.HealthHandler,
	adminH *handler.AdminHandler,
	cfg RouterConfig,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(middleware.CORS(cfg.CORSOrigin))
	r.Use(middleware.RateLimit(rate.Limit(30), 60)) // 30 req/s per IP, burst 60

	api := r.Group("/api")
	{
		api.GET("/terminals", termH.ListTerminals)
		api.GET("/terminals/:code", termH.GetTerminal)
		api.GET("/terminals/:code/departures", termH.GetDepartures)
		api.GET("/search", searchH.Search)
		api.GET("/schedule/:route_id", schedH.GetSchedule)
		api.GET("/regions", regionH.ListRegions)
	}

	// Admin endpoints — require X-Admin-Key header
	admin := r.Group("/api/admin", middleware.AdminKey(cfg.AdminKey))
	{
		admin.POST("/ingest", adminH.TriggerIngest)
		admin.GET("/status", adminH.IngestStatus)
	}

	r.GET("/health", healthH.Health)

	return r
}
