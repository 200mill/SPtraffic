package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sptraffic/backend/internal/cache"
)

// HealthHandler serves the /health endpoint.
type HealthHandler struct {
	db    *pgxpool.Pool
	cache *cache.Cache
}

func NewHealthHandler(db *pgxpool.Pool, cache *cache.Cache) *HealthHandler {
	return &HealthHandler{db: db, cache: cache}
}

// Health godoc
// GET /health
// Returns 200 if DB and Redis are reachable; 503 otherwise.
func (h *HealthHandler) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	status := gin.H{
		"db":    "ok",
		"redis": "ok",
	}
	code := http.StatusOK

	if err := h.db.Ping(ctx); err != nil {
		status["db"] = "unreachable: " + err.Error()
		code = http.StatusServiceUnavailable
	}

	if err := h.cache.Ping(ctx); err != nil {
		status["redis"] = "unreachable: " + err.Error()
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, status)
}
