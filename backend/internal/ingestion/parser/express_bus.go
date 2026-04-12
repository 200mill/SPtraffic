// Package parser converts raw data.go.kr API responses into domain objects.
//
// NOTE: The exact field names below are based on the documented spec of the
// "고속버스 운행정보 조회서비스" (BusSttnInfoInqireService).
// Verify and adjust after obtaining a real API key and testing live responses.
package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// --- Express Bus Terminal ---

type expressBusTerminalItem struct {
	TerminalID   string  `json:"terminalId"`
	TerminalNm   string  `json:"terminalNm"`
	CityCode     string  `json:"cityCode"`
	CityNm       string  `json:"cityNm"`
	Do           string  `json:"do"`           // 도/광역시 명
	Lat          float64 `json:"lat,string"`
	Lng          float64 `json:"lng,string"`
}

// ParseExpressBusTerminals converts the raw items JSON into Terminal domain objects.
func ParseExpressBusTerminals(items json.RawMessage) ([]domain.Terminal, error) {
	var raw struct {
		Item []expressBusTerminalItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		// single item case
		var single struct {
			Item expressBusTerminalItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []expressBusTerminalItem{single.Item}
	}

	out := make([]domain.Terminal, 0, len(raw.Item))
	for _, it := range raw.Item {
		out = append(out, domain.Terminal{
			Code:       it.TerminalID,
			Name:       strings.TrimSpace(it.TerminalNm),
			Type:       domain.TerminalTypeBus,
			RegionCode: it.CityCode,
			Lat:        it.Lat,
			Lon:        it.Lng,
		})
	}
	return out, nil
}

// --- Express Bus Schedule ---

type expressBusScheduleItem struct {
	DepTerminalID string `json:"depTerminalId"`
	ArrTerminalID string `json:"arrTerminalId"`
	DepPlandTime  string `json:"depPlandTime"`  // "HHMM"
	ArrPlandTime  string `json:"arrPlandTime"`  // "HHMM"
	Operator      string `json:"busOperatorNm"`
	GradeNm       string `json:"gradeNm"` // 우등 / 일반
}

// ParseExpressBusSchedules converts raw schedule items into (Route, Schedule) pairs.
// The caller is responsible for resolving terminal codes to IDs before saving.
type ExpressBusSchedule struct {
	DepCode   string
	ArrCode   string
	DepMins   int
	ArrMins   int
	Operator  string
}

func ParseExpressBusSchedules(items json.RawMessage) ([]ExpressBusSchedule, error) {
	var raw struct {
		Item []expressBusScheduleItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item expressBusScheduleItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []expressBusScheduleItem{single.Item}
	}

	out := make([]ExpressBusSchedule, 0, len(raw.Item))
	for _, it := range raw.Item {
		dep, err := hhmm(it.DepPlandTime)
		if err != nil {
			continue
		}
		arr, err := hhmm(it.ArrPlandTime)
		if err != nil {
			continue
		}
		out = append(out, ExpressBusSchedule{
			DepCode:  it.DepTerminalID,
			ArrCode:  it.ArrTerminalID,
			DepMins:  dep,
			ArrMins:  arr,
			Operator: it.Operator,
		})
	}
	return out, nil
}
