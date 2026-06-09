package domain

// RouteType categorises transport modes.
type RouteType string

const (
	RouteTypeExpress   RouteType = "express"   // 고속버스
	RouteTypeIntercity RouteType = "intercity" // 시외버스
	RouteTypeRail      RouteType = "rail"      // 간선철도
	RouteTypeMetro     RouteType = "metro"     // 도시철도/지하철
)

type Route struct {
	ID       int       `json:"id"`
	Code     string    `json:"code"`     // empty when not provided by source API
	Type     RouteType `json:"type"`
	OriginID int       `json:"originId"`
	DestID   int       `json:"destId"`
	Operator string    `json:"operator"`
}
