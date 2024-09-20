package hako

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// ParseExpiry parses the expiry string into a time.Duration.
func ParseExpiry(expiry string) (time.Duration, error) {
	if expiry == "" {
		return 0, fmt.Errorf("expiry is not specified")
	}

	if len(expiry) < 2 {
		return 0, fmt.Errorf("invalid expiry format")
	}

	unit := expiry[len(expiry)-1]
	value, err := strconv.Atoi(expiry[:len(expiry)-1])
	if err != nil {
		return 0, fmt.Errorf("invalid expiry format")
	}

	dur := time.Duration(value)
	switch unit {
	case 's':
		return dur * time.Second, nil
	case 'm':
		return dur * time.Minute, nil
	case 'h':
		return dur * time.Hour, nil
	case 'd':
		return dur * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("invalid expiry unit")
	}
}

// SleepWithContext sleeps for the specified duration, but can be interrupted by
// the context.
func SleepWithContext(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
