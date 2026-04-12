package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// NOTE: Field names are based on the KORAIL "철도역 정보 조회서비스".
// Adjust after live API testing.

type railStationItem struct {
	StationCode string  `json:"stationCode"`
	StationNm   string  `json:"stationNm"`
	CityCode    string  `json:"cityCode"`
	Latitude    float64 `json:"latitude,string"`
	Longitude   float64 `json:"longitude,string"`
}

// ParseRailStations parses rail station items into Terminal domain objects.
func ParseRailStations(items json.RawMessage) ([]domain.Terminal, error) {
	var raw struct {
		Item []railStationItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item railStationItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []railStationItem{single.Item}
	}

	out := make([]domain.Terminal, 0, len(raw.Item))
	for _, it := range raw.Item {
		out = append(out, domain.Terminal{
			Code:       it.StationCode,
			Name:       strings.TrimSpace(it.StationNm),
			Type:       domain.TerminalTypeRail,
			RegionCode: it.CityCode,
			Lat:        it.Latitude,
			Lon:        it.Longitude,
		})
	}
	return out, nil
}

// RailSchedule is the parsed form of one train departure.
type RailSchedule struct {
	DepCode  string
	ArrCode  string
	DepMins  int
	ArrMins  int
	TrainNo  string
	TrainNm  string // e.g. "KTX", "무궁화"
}

type railScheduleItem struct {
	DepplandTime  string `json:"depplandTime"`  // "HHMM"
	ArrplandTime  string `json:"arrplandTime"`
	DepPlaceName  string `json:"depplacename"`
	ArrPlaceName  string `json:"arrplacename"`
	TrainNo       string `json:"trainno"`
	TrainNm       string `json:"traingradename"`
}

// ParseRailSchedules parses train schedule items.
func ParseRailSchedules(items json.RawMessage) ([]RailSchedule, error) {
	var raw struct {
		Item []railScheduleItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item railScheduleItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []railScheduleItem{single.Item}
	}

	out := make([]RailSchedule, 0, len(raw.Item))
	for _, it := range raw.Item {
		dep, err := hhmm(it.DepplandTime)
		if err != nil {
			continue
		}
		arr, err := hhmm(it.ArrplandTime)
		if err != nil {
			continue
		}
		out = append(out, RailSchedule{
			DepCode: it.DepPlaceName, // station name used as fallback code
			ArrCode: it.ArrPlaceName,
			DepMins: dep,
			ArrMins: arr,
			TrainNo: it.TrainNo,
			TrainNm: it.TrainNm,
		})
	}
	return out, nil
}
