package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sptraffic/backend/internal/cache"
	"github.com/sptraffic/backend/internal/repository"
)

// RealtimeArrival is one upcoming train at a metro station.
type RealtimeArrival struct {
	Line        string `json:"line"`
	Direction   string `json:"direction"`
	Destination string `json:"destination"`
	ArriveSecs  int    `json:"arriveSecs"`
	Message     string `json:"message"`
}

// MetroHandler serves the real-time metro arrival endpoint.
type MetroHandler struct {
	termRepo repository.TerminalRepository
	cache    *cache.Cache
	seoulKey string // SEOUL_METRO_KEY; empty = real-time disabled
}

func NewMetroHandler(
	termRepo repository.TerminalRepository,
	cacheClient *cache.Cache,
	seoulKey string,
) *MetroHandler {
	return &MetroHandler{termRepo: termRepo, cache: cacheClient, seoulKey: seoulKey}
}

// GetRealtime godoc
// GET /api/terminals/:code/realtime
// Returns live next-train arrivals from the Seoul Open Data Plaza API.
// Returns an empty array for non-Seoul stations or when SEOUL_METRO_KEY is not set.
func (h *MetroHandler) GetRealtime(c *gin.Context) {
	ctx := c.Request.Context()
	code := c.Param("code")
	cacheKey := cache.RealtimeKey(code)

	// Cache hit
	var cached []RealtimeArrival
	if err := h.cache.GetJSON(ctx, cacheKey, &cached); err == nil && cached != nil {
		c.JSON(http.StatusOK, cached)
		return
	}

	if h.seoulKey == "" {
		c.JSON(http.StatusOK, []RealtimeArrival{})
		return
	}

	term, err := h.termRepo.FindByCode(ctx, code)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "terminal not found"})
		return
	}

	arrivals, err := fetchSeoulRealtime(ctx, h.seoulKey, term.Name)
	if err != nil {
		// Non-Seoul stations or API errors return empty gracefully
		arrivals = []RealtimeArrival{}
	}

	_ = h.cache.SetJSON(ctx, cacheKey, arrivals, cache.RealtimeTTL)
	c.JSON(http.StatusOK, arrivals)
}

// ─── Seoul Open Data Plaza API ───────────────────────────────────────────────

const seoulMetroBaseURL = "http://swopenapi.seoul.go.kr/api/subway"

type seoulRealtimeResponse struct {
	ErrorMessage struct {
		Status  int    `json:"status"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"errorMessage"`
	RealtimeArrivalList []seoulArrivalItem `json:"realtimeArrivalList"`
}

type seoulArrivalItem struct {
	SubwayNm   string `json:"subwayNm"`   // line name (e.g. "1호선")
	UpdnLine   string `json:"updnLine"`   // direction (상행/하행)
	BstatnNm   string `json:"bstatnNm"`   // destination station
	BarvlDt    string `json:"barvlDt"`    // seconds until arrival (string)
	ArvlMsg2   string `json:"arvlMsg2"`   // human-readable message
}

func fetchSeoulRealtime(ctx context.Context, key, stationName string) ([]RealtimeArrival, error) {
	endpoint := fmt.Sprintf("%s/%s/json/realtimeStationArrival/0/20/%s",
		seoulMetroBaseURL, key, url.PathEscape(stationName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw seoulRealtimeResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	// INFO-000 = success; INFO-200 = no data (valid for non-Seoul stations)
	if raw.ErrorMessage.Code != "INFO-000" && raw.ErrorMessage.Code != "" {
		return nil, fmt.Errorf("seoul API: %s", raw.ErrorMessage.Message)
	}

	out := make([]RealtimeArrival, 0, len(raw.RealtimeArrivalList))
	for _, it := range raw.RealtimeArrivalList {
		secs := 0
		fmt.Sscanf(it.BarvlDt, "%d", &secs)
		out = append(out, RealtimeArrival{
			Line:        it.SubwayNm,
			Direction:   it.UpdnLine,
			Destination: it.BstatnNm,
			ArriveSecs:  secs,
			Message:     it.ArvlMsg2,
		})
	}
	return out, nil
}

