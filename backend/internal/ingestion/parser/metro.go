package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// Metro (도시철도/지하철) parsers.
// Dataset:  https://www.data.go.kr/data/15098598/openapi.do
// Provider: 국토교통부_(TAGO)_도시철도운행정보
//
// City codes:  GET /1613000/SubwayInfoService/getCtyCodeList
// Stations:    GET /1613000/SubwayInfoService/getCtyAcctoSubwaySttnList?cityCode=XXX
// Schedule:    GET /1613000/SubwayInfoService/getStrtpntAlocFndSubwayInfo
//              (requires depPlaceId + arrPlaceId — station IDs)
//
// Confirmed response fields:
//   Station:   sttnId, sttnNm
//   Schedule:  depPlandTime, arrPlandTime (YYYYMMDDHHMM, 12 chars)

// ─── Stations ─────────────────────────────────────────────────────────────────

type metroStationItem struct {
	SttnID   string `json:"sttnId"`   // station code
	SttnNm   string `json:"sttnNm"`   // station name
	CityCode string `json:"cityCode"` // used as regionCode
}

// ParseMetroStations parses getCtyAcctoSubwaySttnList items into Terminal domain objects.
func ParseMetroStations(items json.RawMessage) ([]domain.Terminal, error) {
	var raw struct {
		Item []metroStationItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item metroStationItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []metroStationItem{single.Item}
	}

	seen := make(map[string]bool)
	out := make([]domain.Terminal, 0, len(raw.Item))
	for _, it := range raw.Item {
		if it.SttnID == "" || seen[it.SttnID] {
			continue
		}
		seen[it.SttnID] = true
		out = append(out, domain.Terminal{
			Code:       it.SttnID,
			Name:       strings.TrimSpace(it.SttnNm),
			Type:       domain.TerminalTypeMetro,
			RegionCode: it.CityCode,
		})
	}
	return out, nil
}

// ─── Schedules ────────────────────────────────────────────────────────────────

// MetroSchedule is one departure on a known dep→arr station pair.
type MetroSchedule struct {
	DepMins  int
	ArrMins  int
	LineNm   string // subway line name
	TrainNo  string
}

type metroScheduleItem struct {
	DepPlandTime string `json:"depPlandTime"` // "YYYYMMDDHHMM"
	ArrPlandTime string `json:"arrPlandTime"` // "YYYYMMDDHHMM"
	LineNm       string `json:"lnCd"`         // line code (used as operator label)
	TrainNo      string `json:"trainNo"`
}

// ParseMetroSchedules parses getStrtpntAlocFndSubwayInfo items for a single dep→arr pair.
func ParseMetroSchedules(items json.RawMessage) ([]MetroSchedule, error) {
	var raw struct {
		Item []metroScheduleItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item metroScheduleItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []metroScheduleItem{single.Item}
	}

	out := make([]MetroSchedule, 0, len(raw.Item))
	for _, it := range raw.Item {
		dep, err := yyyymmddhhmm(it.DepPlandTime)
		if err != nil {
			continue
		}
		arr, err := yyyymmddhhmm(it.ArrPlandTime)
		if err != nil {
			continue
		}
		out = append(out, MetroSchedule{
			DepMins: dep,
			ArrMins: arr,
			LineNm:  it.LineNm,
			TrainNo: it.TrainNo,
		})
	}
	return out, nil
}
