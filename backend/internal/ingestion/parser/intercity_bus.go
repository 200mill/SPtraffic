package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// Intercity bus (시외버스) parsers.
// Dataset:  https://www.data.go.kr/data/15098541/openapi.do
// Provider: 국토교통부_(TAGO)_시외버스정보
//
// Terminal list:  GET /1613000/SuburbsBusInfoService/GetSuberbsBusTrminlList
// Schedule:       GET /1613000/SuburbsBusInfoService/GetStrtpntAlocFndSuberbsBusInfo
//                 (requires depTerminalId + arrTerminalId)
//
// Confirmed response fields (from official spec):
//   Terminal: terminalId, terminalNm, cityName  (no coordinates provided)
//   Schedule: depPlandTime (YYYYMMDDHHMI), arrPlandTime (YYYYMMDDHHMI), gradeNm, routeId

type intercityTerminalItem struct {
	TerminalID string `json:"terminalId"`
	TerminalNm string `json:"terminalNm"`
	CityName   string `json:"cityName"`
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
			RegionCode: it.CityName, // city name used as region identifier
		})
	}
	return out, nil
}

// IntercitySchedule is one departure on a known dep→arr route.
type IntercitySchedule struct {
	DepMins  int
	ArrMins  int
	Operator string // gradeNm (직행, 완행, 고속형, …)
}

// intercityScheduleItem matches the confirmed GetStrtpntAlocFndSuberbsBusInfo response.
// depPlandTime / arrPlandTime are "YYYYMMDDHHMI" (12 chars; MI = minutes).
type intercityScheduleItem struct {
	DepPlandTime string `json:"depPlandTime"`
	ArrPlandTime string `json:"arrPlandTime"`
	GradeNm      string `json:"gradeNm"`
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
			Operator: it.GradeNm,
		})
	}
	return out, nil
}
