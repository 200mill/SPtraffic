package domain

// TerminalType distinguishes bus terminals from rail stations.
type TerminalType string

const (
	TerminalTypeBus   TerminalType = "bus"
	TerminalTypeRail  TerminalType = "rail"
	TerminalTypeMetro TerminalType = "metro" // 도시철도/지하철
)

type Terminal struct {
	ID         int          `json:"id"`
	Code       string       `json:"code"`
	Name       string       `json:"name"`
	Type       TerminalType `json:"type"`
	RegionCode string       `json:"regionCode"`
	Lat        float64      `json:"lat"`
	Lon        float64      `json:"lon"`
}

// Region groups terminals by administrative area.
type Region struct {
	Code string `json:"code"`
	Name string `json:"name"`
}
