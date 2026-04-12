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

// yyyymmddhhmm parses a "YYYYMMDDHHMM" (12-char) timestamp into minutes since midnight.
// Used by the TAGO express bus API where depPlandTime / arrPlandTime are "202404121430".
func yyyymmddhhmm(s string) (int, error) {
	if len(s) < 12 {
		return 0, fmt.Errorf("invalid YYYYMMDDHHMM %q", s)
	}
	return hhmm(s[8:12])
}
