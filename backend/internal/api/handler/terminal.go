package handler

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sptraffic/backend/internal/domain"
	"github.com/sptraffic/backend/internal/repository"
)

// TerminalHandler serves terminal/station list endpoints.
type TerminalHandler struct {
	repo      repository.TerminalRepository
	routeRepo repository.RouteRepository
	schedRepo repository.ScheduleRepository
}

func NewTerminalHandler(
	repo repository.TerminalRepository,
	routeRepo repository.RouteRepository,
	schedRepo repository.ScheduleRepository,
) *TerminalHandler {
	return &TerminalHandler{repo: repo, routeRepo: routeRepo, schedRepo: schedRepo}
}

// ListTerminals godoc
// GET /api/terminals
// Query params:
//   q      — name substring search (autocomplete); returns up to 20 results
//   region — filter by region code
func (h *TerminalHandler) ListTerminals(c *gin.Context) {
	ctx := c.Request.Context()

	if q := c.Query("q"); q != "" {
		terminals, err := h.repo.SearchByName(ctx, q)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, terminals)
		return
	}

	if region := c.Query("region"); region != "" {
		terminals, err := h.repo.FindByRegion(ctx, region)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, terminals)
		return
	}

	limit := 100
	offset := 0
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v <= 500 {
		limit = v
	}
	if v, err := strconv.Atoi(c.Query("offset")); err == nil && v >= 0 {
		offset = v
	}

	terminals, total, err := h.repo.FindAllPaginated(ctx, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"items":  terminals,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// GetTerminal godoc
// GET /api/terminals/:code
func (h *TerminalHandler) GetTerminal(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("code")

	t, err := h.repo.FindByCode(ctx, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}
	c.JSON(http.StatusOK, t)
}

// GetDepartures godoc
// GET /api/terminals/:code/departures
// Query params:
//
//	after — HH:MM (defaults to current KST time)
//	limit — max results (default 20, max 60)
func (h *TerminalHandler) GetDepartures(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("code")

	term, err := h.repo.FindByCode(ctx, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	// Parse ?after=HH:MM or default to current KST time
	afterMins := currentKSTMins()
	if afterStr := c.Query("after"); afterStr != "" {
		if parsed, ok := parseHHMM(afterStr); ok {
			afterMins = parsed
		}
	}

	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 60 {
			limit = v
		}
	}

	routes, err := h.routeRepo.FindByOrigin(ctx, term.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var deps []domain.DepartureInfo
	for _, rt := range routes {
		scheds, err := h.schedRepo.FindByRouteFrom(ctx, rt.ID, afterMins)
		if err != nil {
			continue
		}
		dest, err := h.repo.FindByID(ctx, rt.DestID)
		if err != nil {
			continue
		}
		for _, sc := range scheds {
			deps = append(deps, domain.DepartureInfo{
				RouteID:       rt.ID,
				Destination:   dest.Name,
				DestCode:      dest.Code,
				RouteType:     string(rt.Type),
				Operator:      rt.Operator,
				DepartureMins: sc.DepartureMins,
				ArrivalMins:   sc.ArrivalMins,
				DurationMin:   sc.DurationMin,
			})
		}
	}

	// Sort by effective departure time (overnight wrap)
	sort.Slice(deps, func(i, j int) bool {
		di, dj := deps[i].DepartureMins, deps[j].DepartureMins
		if di < afterMins {
			di += 24 * 60
		}
		if dj < afterMins {
			dj += 24 * 60
		}
		return di < dj
	})

	if len(deps) > limit {
		deps = deps[:limit]
	}
	if deps == nil {
		deps = []domain.DepartureInfo{}
	}
	c.JSON(http.StatusOK, deps)
}

func currentKSTMins() int {
	now := time.Now().In(kst())
	return now.Hour()*60 + now.Minute()
}

func parseHHMM(s string) (int, bool) {
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	h, err1 := strconv.Atoi(s[:2])
	m, err2 := strconv.Atoi(s[3:])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}
