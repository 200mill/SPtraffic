package domain

import "time"

// DepartureInfo represents one upcoming departure from a terminal.
type DepartureInfo struct {
	RouteID       int    `json:"routeId"`
	Destination   string `json:"destination"`   // destination terminal name
	DestCode      string `json:"destCode"`      // destination terminal code
	RouteType     string `json:"routeType"`
	Operator      string `json:"operator"`
	DepartureMins int    `json:"departureMins"`
	ArrivalMins   int    `json:"arrivalMins"`
	DurationMin   int    `json:"durationMin"`
}

// SearchQuery holds the user's route-search parameters.
type SearchQuery struct {
	FromCode string    // departure terminal code
	ToCode   string    // arrival terminal code
	Date     time.Time // departure date + earliest time
	Mode     string    // "bus" | "rail" | "all"
	MaxLegs  int       // 1 = direct only, 2 = allow one transfer
}

// Leg is one segment of a journey (terminal-to-terminal on a single service).
type Leg struct {
	From          Terminal `json:"from"`
	To            Terminal `json:"to"`
	DepartureMins int      `json:"departureMins"` // minutes since midnight
	ArrivalMins   int      `json:"arrivalMins"`
	RouteType     string   `json:"routeType"`
	Operator      string   `json:"operator"`
}

// SearchResult is a complete journey composed of one or more Legs.
type SearchResult struct {
	Legs         []Leg `json:"legs"`
	TotalMinutes int   `json:"totalMinutes"`
	Transfers    int   `json:"transfers"` // len(Legs) - 1
}
