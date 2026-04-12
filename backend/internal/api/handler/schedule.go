package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sptraffic/backend/internal/service"
)

// ScheduleHandler serves the per-route timetable endpoint.
type ScheduleHandler struct {
	svc *service.ScheduleService
}

func NewScheduleHandler(svc *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{svc: svc}
}

// GetSchedule godoc
// GET /api/schedule/:route_id
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	routeID, err := strconv.Atoi(c.Param("route_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid route_id"})
		return
	}

	rs, err := h.svc.GetByRoute(c.Request.Context(), routeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rs)
}
