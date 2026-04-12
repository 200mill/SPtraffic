package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// Rail (철도) parsers.
// Endpoint base: https://apis.data.go.kr/1613000/TrainInfoService
//
// Station list:  GET /getStationList
// Schedule:      GET /getSttnToDirctTrnList  (requires depPlaceNm + arrPlaceNm)
//
// NOTE: KORAIL/TAGO TrainInfoService field names below are based on the
// documented spec. Verify stationCode / stinCode casing and the schedule
// endpoint name against live responses before running a real ingest.

type railStationItem struct {
	StinCode  string  `json:"stinCode"`  // station code (5-digit)
	StinNm    string  `json:"stinNm"`    // station name
	LineNm    string  `json:"lineNm"`    // line name (KTX, 무궁화, …)
	Latitude  float64 `json:"latitude,string"`
	Longitude float64 `json:"longitude,string"`
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

	seen := make(map[string]bool)
	out := make([]domain.Terminal, 0, len(raw.Item))
	for _, it := range raw.Item {
		if it.StinCode == "" || seen[it.StinCode] {
			continue
		}
		seen[it.StinCode] = true
		out = append(out, domain.Terminal{
			Code:      it.StinCode,
			Name:      strings.TrimSpace(it.StinNm),
			Type:      domain.TerminalTypeRail,
			Lat:       it.Latitude,
			Lon:       it.Longitude,
		})
	}
	return out, nil
}

// RailSchedule is one departure on a known dep→arr route.
// The caller already knows DepCode / ArrCode from the query params.
type RailSchedule struct {
	DepMins int
	ArrMins int
	TrainNo string // train number
	TrainNm string // train grade name: KTX, ITX-새마을, 무궁화, …
}

type railScheduleItem struct {
	DepplandTime  string `json:"depplandTime"`  // "YYYYMMDDHHMM"
	ArpplandTime  string `json:"arpplandTime"`  // "YYYYMMDDHHMM"
	TrainNo       string `json:"trainNo"`
	TrainGradeNm  string `json:"trainGradeNm"` // KTX, ITX-새마을, 무궁화호, …
}

// ParseRailSchedules parses train schedule items for a single dep→arr station pair.
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
		dep, err := yyyymmddhhmm(it.DepplandTime)
		if err != nil {
			continue
		}
		arr, err := yyyymmddhhmm(it.ArpplandTime)
		if err != nil {
			continue
		}
		out = append(out, RailSchedule{
			DepMins: dep,
			ArrMins: arr,
			TrainNo: it.TrainNo,
			TrainNm: it.TrainGradeNm,
		})
	}
	return out, nil
}
