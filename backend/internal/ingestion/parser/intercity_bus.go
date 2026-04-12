package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// NOTE: Field names are based on the "시외버스 운행정보 조회서비스"
// (SuburbsBusInfoService). Adjust after live API testing.

type intercityTerminalItem struct {
	TerminalID string  `json:"terminalId"`
	TerminalNm string  `json:"terminalNm"`
	CityCode   string  `json:"cityCode"`
	Do         string  `json:"do"`
	GpslntX    float64 `json:"gpslntX,string"` // longitude
	GpslntY    float64 `json:"gpslntY,string"` // latitude
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
		out = append(out, domain.Terminal{
			Code:       it.TerminalID,
			Name:       strings.TrimSpace(it.TerminalNm),
			Type:       domain.TerminalTypeBus,
			RegionCode: it.CityCode,
			Lat:        it.GpslntY,
			Lon:        it.GpslntX,
		})
	}
	return out, nil
}

// IntercitySchedule is the parsed form of one intercity bus departure.
type IntercitySchedule struct {
	DepCode  string
	ArrCode  string
	DepMins  int
	ArrMins  int
	Operator string
}

type intercityScheduleItem struct {
	DepTerminalID string `json:"depTerminalId"`
	ArrTerminalID string `json:"arrTerminalId"`
	DepPlandTime  string `json:"depPlandTime"`
	ArrPlandTime  string `json:"arrPlandTime"`
	BusGradeNm    string `json:"busGradeNm"`
	BusOperatorNm string `json:"busOperatorNm"`
}

// ParseIntercitySchedules parses 시외버스 schedule items.
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
		dep, err := hhmm(it.DepPlandTime)
		if err != nil {
			continue
		}
		arr, err := hhmm(it.ArrPlandTime)
		if err != nil {
			continue
		}
		out = append(out, IntercitySchedule{
			DepCode:  it.DepTerminalID,
			ArrCode:  it.ArrTerminalID,
			DepMins:  dep,
			ArrMins:  arr,
			Operator: it.BusOperatorNm,
		})
	}
	return out, nil
}
