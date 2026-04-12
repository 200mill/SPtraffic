package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sptraffic/backend/internal/cache"
	"github.com/sptraffic/backend/internal/domain"
	"github.com/sptraffic/backend/internal/service"
)

// SearchHandler serves the route-search endpoint.
type SearchHandler struct {
	svc   *service.SearchService
	cache *cache.Cache
}

func NewSearchHandler(svc *service.SearchService, cache *cache.Cache) *SearchHandler {
	return &SearchHandler{svc: svc, cache: cache}
}

// Search godoc
// GET /api/search?from=CODE&to=CODE&date=YYYY-MM-DDTHH:MM&mode=all&max_legs=2
func (h *SearchHandler) Search(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02T15:04"))
	mode := c.DefaultQuery("mode", "all")
	maxLegs := 2
	switch c.Query("max_legs") {
	case "1":
		maxLegs = 1
	case "3":
		maxLegs = 3
	}

	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to are required"})
		return
	}

	date, err := time.ParseInLocation("2006-01-02T15:04", dateStr, kst())
	if err != nil {
		// Try date-only format
		date, err = time.ParseInLocation("2006-01-02", dateStr, kst())
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, use YYYY-MM-DDTHH:MM"})
			return
		}
	}

	ctx := c.Request.Context()

	// Cache lookup
	cacheKey := cache.SearchKey(from, to, fmt.Sprintf("%d%02d%02d%02d%02d", date.Year(), date.Month(), date.Day(), date.Hour(), date.Minute()), fmt.Sprintf("%s_%d", mode, maxLegs))
	var cached []domain.SearchResult
	if err := h.cache.GetSearch(ctx, cacheKey, &cached); err == nil && cached != nil {
		c.JSON(http.StatusOK, cached)
		return
	}

	results, err := h.svc.Search(ctx, domain.SearchQuery{
		FromCode: from,
		ToCode:   to,
		Date:     date,
		Mode:     mode,
		MaxLegs:  maxLegs,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = h.cache.SetSearch(ctx, cacheKey, results)
	c.JSON(http.StatusOK, results)
}

func kst() *time.Location {
	loc, _ := time.LoadLocation("Asia/Seoul")
	if loc == nil {
		return time.UTC
	}
	return loc
}
