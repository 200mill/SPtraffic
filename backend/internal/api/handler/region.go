package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sptraffic/backend/internal/repository"
)

// RegionHandler serves the region list endpoint.
type RegionHandler struct {
	repo repository.TerminalRepository
}

func NewRegionHandler(repo repository.TerminalRepository) *RegionHandler {
	return &RegionHandler{repo: repo}
}

// ListRegions godoc
// GET /api/regions
func (h *RegionHandler) ListRegions(c *gin.Context) {
	regions, err := h.repo.FindRegions(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, regions)
}
