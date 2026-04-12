package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// Intercity bus (시외버스) parsers.
// Endpoint base: https://apis.data.go.kr/1613000/SuburbsBusInfoService
//
// Terminal list:  GET /getSttnList
// Schedule:       GET /getAlocFndSuberbsBusInfo  (requires depTerminalId + arrTerminalId)
//
// NOTE: Field names follow the TAGO SuburbsBusInfoService spec.
// Verify gpslati/gpslong casing against live responses — TAGO bus-stop APIs
// use lowercase (gpslati, gpslong); the intercity terminal list may differ.

type intercityTerminalItem struct {
	TerminalID string  `json:"terminalId"`
	TerminalNm string  `json:"terminalNm"`
	CityCode   string  `json:"cityCode"`
	CityNm     string  `json:"cityNm"`
	Gpslati    float64 `json:"gpslati,string"` // WGS84 latitude
	Gpslong    float64 `json:"gpslong,string"` // WGS84 longitude
}

// ParseIntercityTerminals parses 시외버스 terminal items.
func ParseIntercityTerminals(items json.RawMessage) ([]domain.Terminal, error) {
	var raw struct {
		Item []intercityTerminalItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item intercityTerminalItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []intercityTerminalItem{single.Item}
	}

	out := make([]domain.Terminal, 0, len(raw.Item))
	for _, it := range raw.Item {
		if it.TerminalID == "" {
			continue
		}
		out = append(out, domain.Terminal{
			Code:       it.TerminalID,
			Name:       strings.TrimSpace(it.TerminalNm),
			Type:       domain.TerminalTypeBus,
			RegionCode: it.CityCode,
			Lat:        it.Gpslati,
			Lon:        it.Gpslong,
		})
	}
	return out, nil
}

// IntercitySchedule is one departure on a known dep→arr route.
// The caller already knows DepCode / ArrCode from the query params.
type IntercitySchedule struct {
	DepMins  int
	ArrMins  int
	Operator string // busGradeNm
}

type intercityScheduleItem struct {
	DepPlandTime string `json:"depPlandTime"` // "YYYYMMDDHHMM"
	ArrPlandTime string `json:"arrPlandTime"` // "YYYYMMDDHHMM"
	BusGradeNm   string `json:"busGradeNm"`   // 직행, 완행, 고속형, …
}

// ParseIntercitySchedules parses 시외버스 schedule items for a single dep→arr pair.
func ParseIntercitySchedules(items json.RawMessage) ([]IntercitySchedule, error) {
	var raw struct {
		Item []intercityScheduleItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item intercityScheduleItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []intercityScheduleItem{single.Item}
	}

	out := make([]IntercitySchedule, 0, len(raw.Item))
	for _, it := range raw.Item {
		dep, err := yyyymmddhhmm(it.DepPlandTime)
		if err != nil {
			continue
		}
		arr, err := yyyymmddhhmm(it.ArrPlandTime)
		if err != nil {
			continue
		}
		out = append(out, IntercitySchedule{
			DepMins:  dep,
			ArrMins:  arr,
			Operator: it.BusGradeNm,
		})
	}
	return out, nil
}
