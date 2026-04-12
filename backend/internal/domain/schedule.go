package domain

import (
	"fmt"
	"time"
)

// Schedule represents a single departure on a Route.
// DepartureMins and ArrivalMins are minutes since midnight (0–1439),
// independent of date and timezone.
type Schedule struct {
	ID           int        `json:"id"`
	RouteID      int        `json:"routeId"`
	DepartureMins int       `json:"departureMins"`
	ArrivalMins  int        `json:"arrivalMins"`
	DurationMin  int        `json:"durationMin"`
	DaysOfWeek   int        `json:"daysOfWeek"` // bitmask: bit0=Mon … bit6=Sun
	ValidFrom    *time.Time `json:"validFrom,omitempty"`
	ValidUntil   *time.Time `json:"validUntil,omitempty"`
}

// TimeStr returns a "HH:MM" string for the given minutes-since-midnight value.
func TimeStr(mins int) string {
	return fmt.Sprintf("%02d:%02d", mins/60, mins%60)
}

// DayBit returns the bitmask for the given weekday (time.Monday = bit0).
func DayBit(d time.Weekday) int {
	// time.Monday==1, ..., time.Sunday==0
	if d == time.Sunday {
		return 1 << 6
	}
	return 1 << (int(d) - 1)
}

// RunsOn reports whether the schedule operates on weekday d.
func (s *Schedule) RunsOn(d time.Weekday) bool {
	return s.DaysOfWeek&DayBit(d) != 0
}
