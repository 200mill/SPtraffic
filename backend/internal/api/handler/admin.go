package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IngestFunc is the function signature for triggering a full data refresh.
type IngestFunc func() error

// AdminHandler exposes management endpoints protected by a static API key.
type AdminHandler struct {
	ingest   IngestFunc
	mu       sync.Mutex
	lastRun  time.Time
	lastErr  string
	running  bool
}

func NewAdminHandler(ingest IngestFunc) *AdminHandler {
	return &AdminHandler{ingest: ingest}
}

// TriggerIngest godoc
// POST /api/admin/ingest
// Header: X-Admin-Key: <key>
// Starts a background ingestion run. Returns 409 if one is already in progress.
func (h *AdminHandler) TriggerIngest(c *gin.Context) {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		c.JSON(http.StatusConflict, gin.H{"error": "ingestion already in progress"})
		return
	}
	h.running = true
	h.mu.Unlock()

	go func() {
		err := h.ingest()
		h.mu.Lock()
		defer h.mu.Unlock()
		h.running = false
		h.lastRun = time.Now()
		if err != nil {
			h.lastErr = err.Error()
		} else {
			h.lastErr = ""
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{"message": "ingestion started"})
}

// IngestStatus godoc
// GET /api/admin/status
func (h *AdminHandler) IngestStatus(c *gin.Context) {
	h.mu.Lock()
	defer h.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"running": h.running,
		"lastRun": h.lastRun,
		"lastErr": h.lastErr,
	})
}
