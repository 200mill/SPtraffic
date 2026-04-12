package parser

import (
	"fmt"
	"strconv"
)

// hhmm parses a "HHMM" string into minutes since midnight.
func hhmm(s string) (int, error) {
	if len(s) != 4 {
		return 0, fmt.Errorf("invalid HHMM %q", s)
	}
	h, err := strconv.Atoi(s[:2])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(s[2:])
	if err != nil {
		return 0, err
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("out-of-range HHMM %q", s)
	}
	return h*60 + m, nil
}
