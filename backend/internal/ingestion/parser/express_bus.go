// Package parser converts raw data.go.kr API responses into domain objects.
package parser

import (
	"encoding/json"
	"strings"

	"github.com/sptraffic/backend/internal/domain"
)

// --- Express Bus Terminal ---
// Endpoint: GET https://apis.data.go.kr/1613000/ExpBusInfo/GetExpBusTrminlList
// Confirmed response fields: terminalId, terminalNm
// Note: API does not return coordinates. Lat/Lon are left as 0 and must be
// enriched separately if map markers are needed.

type expressBusTerminalItem struct {
	TerminalID string `json:"terminalId"`
	TerminalNm string `json:"terminalNm"`
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
		if it.TerminalID == "" {
			continue
		}
		out = append(out, domain.Terminal{
			Code: it.TerminalID,
			Name: strings.TrimSpace(it.TerminalNm),
			Type: domain.TerminalTypeBus,
		})
	}
	return out, nil
}

// --- Express Bus City Code ---
// Endpoint: GET https://apis.data.go.kr/1613000/ExpBusInfo/GetCtyCodeList
// Response fields: cityCode, cityName

type expressBusCityItem struct {
	CityCode string `json:"cityCode"`
	CityName string `json:"cityName"`
}

// CityCode maps a cityCode to its name.
type CityCode struct {
	Code string
	Name string
}

// ParseExpressBusCityCodes parses city code list items.
func ParseExpressBusCityCodes(items json.RawMessage) ([]CityCode, error) {
	var raw struct {
		Item []expressBusCityItem `json:"item"`
	}
	if err := json.Unmarshal(items, &raw); err != nil {
		var single struct {
			Item expressBusCityItem `json:"item"`
		}
		if err2 := json.Unmarshal(items, &single); err2 != nil {
			return nil, err
		}
		raw.Item = []expressBusCityItem{single.Item}
	}

	out := make([]CityCode, 0, len(raw.Item))
	for _, it := range raw.Item {
		out = append(out, CityCode{Code: it.CityCode, Name: it.CityName})
	}
	return out, nil
}

// --- Express Bus Schedule ---
// Endpoint: GET https://apis.data.go.kr/1613000/ExpBusInfo/GetStrtpntAlocFndExpbusInfo
// Required params: depTerminalId, arrTerminalId
// Confirmed response fields: routeId, depPlaceNm, arrPlaceNm,
//   depPlandTime (YYYYMMDDHHMM), arrPlandTime (YYYYMMDDHHMM), gradeNm, charge

type expressBusScheduleItem struct {
	DepPlandTime string `json:"depPlandTime"` // "YYYYMMDDHHMM"
	ArrPlandTime string `json:"arrPlandTime"` // "YYYYMMDDHHMM"
	GradeNm      string `json:"gradeNm"`      // 우등 / 일반 / 프리미엄
}

// ExpressBusSchedule is one departure on a known dep→arr route.
// The caller already knows DepCode / ArrCode from the query params.
type ExpressBusSchedule struct {
	DepMins  int
	ArrMins  int
	Operator string // gradeNm (우등, 일반, …)
}

// ParseExpressBusSchedules parses schedule items for a single dep→arr terminal pair.
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
		dep, err := yyyymmddhhmm(it.DepPlandTime)
		if err != nil {
			continue
		}
		arr, err := yyyymmddhhmm(it.ArrPlandTime)
		if err != nil {
			continue
		}
		out = append(out, ExpressBusSchedule{
			DepMins:  dep,
			ArrMins:  arr,
			Operator: it.GradeNm,
		})
	}
	return out, nil
}
