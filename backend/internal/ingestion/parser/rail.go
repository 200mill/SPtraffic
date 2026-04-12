package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// Rail (철도) parsers.
// Dataset:  https://www.data.go.kr/data/15098552/openapi.do
// Provider: 국토교통부_(TAGO)_열차정보
//
// City codes:  GET /1613000/TrainInfoService/GetCtyCodeList
// Stations:    GET /1613000/TrainInfoService/GetCtyAcctoTrainSttnList?cityCode=XXX
// Schedule:    GET /1613000/TrainInfoService/GetStrtpntAlocFndTrainInfo
//              (requires depPlaceId + arrPlaceId — station IDs, not names)
//
// Confirmed response fields (from official spec):
//   City code: cityname, citycode  (all lowercase)
//   Station:   nodeid, nodename    (all lowercase)
//   Schedule:  trainno, traingradename, depplandtime, arrplandtime (YYYYMMDDHHMISS, 14 chars)

// ─── City Codes ───────────────────────────────────────────────────────────────

type railCityCodeItem struct {
	CityName string `json:"cityname"`
	CityCode string `json:"citycode"`
}

// RailCityCode maps a city code to its name.
type RailCityCode struct {
	Code string
	Name string
}

// ParseRailCityCodes parses the GetCtyCodeList response.
func ParseRailCityCodes(items json.RawMessage) ([]RailCityCode, error) {
	var raw struct {
		Item []railCityCodeItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item railCityCodeItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []railCityCodeItem{single.Item}
	}

	out := make([]RailCityCode, 0, len(raw.Item))
	for _, it := range raw.Item {
		if it.CityCode == "" {
			continue
		}
		out = append(out, RailCityCode{Code: it.CityCode, Name: it.CityName})
	}
	return out, nil
}

// ─── Stations ─────────────────────────────────────────────────────────────────

type railStationItem struct {
	NodeID   string `json:"nodeid"`   // station code
	NodeName string `json:"nodename"` // station name
}

// ParseRailStations parses GetCtyAcctoTrainSttnList items into Terminal domain objects.
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
		if it.NodeID == "" || seen[it.NodeID] {
			continue
		}
		seen[it.NodeID] = true
		out = append(out, domain.Terminal{
			Code: it.NodeID,
			Name: strings.TrimSpace(it.NodeName),
			Type: domain.TerminalTypeRail,
		})
	}
	return out, nil
}

// ─── Schedules ────────────────────────────────────────────────────────────────

// RailSchedule is one departure on a known dep→arr route.
type RailSchedule struct {
	DepMins int
	ArrMins int
	TrainNo string // trainno
	TrainNm string // traingradename: KTX, ITX-새마을, 무궁화, …
}

// railScheduleItem matches the confirmed GetStrtpntAlocFndTrainInfo response.
// depplandtime / arrplandtime are "YYYYMMDDHHMISS" (14 chars).
type railScheduleItem struct {
	Depplandtime  string `json:"depplandtime"`  // "YYYYMMDDHHMISS"
	Arrplandtime  string `json:"arrplandtime"`  // "YYYYMMDDHHMISS"
	Trainno       string `json:"trainno"`
	Traingradename string `json:"traingradename"` // KTX, ITX-새마을, 무궁화호, …
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
		// depplandtime is 14 chars (YYYYMMDDHHMISS); yyyymmddhhmm reads [8:12] = HHMM
		dep, err := yyyymmddhhmm(it.Depplandtime)
		if err != nil {
			continue
		}
		arr, err := yyyymmddhhmm(it.Arrplandtime)
		if err != nil {
			continue
		}
		out = append(out, RailSchedule{
			DepMins: dep,
			ArrMins: arr,
			TrainNo: it.Trainno,
			TrainNm: it.Traingradename,
		})
	}
	return out, nil
}
