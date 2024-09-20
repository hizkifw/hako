package hako_test

import (
	"context"
	"testing"
	"time"

	"github.com/hizkifw/hako/pkg/hako"
	"github.com/stretchr/testify/assert"
)

func TestParseExpiry(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		expiry string
		want   time.Duration
	}{
		{"1s", 1 * time.Second},
		{"1m", 1 * time.Minute},
		{"1h", 1 * time.Hour},
		{"1d", 24 * time.Hour},
		{"10s", 10 * time.Second},
		{"10m", 10 * time.Minute},
		{"10h", 10 * time.Hour},
		{"10d", 240 * time.Hour},
		{"", 0},
		{"1", 0},
		{"1ss", 0},
		{"1x", 0},
	}

	for _, tt := range tests {
		got, err := hako.ParseExpiry(tt.expiry)
		if err != nil {
			assert.Equal(tt.want, got, "ParseExpiry(%q) returned error: %v", tt.expiry, err)
		} else {
			assert.Equal(tt.want, got, "ParseExpiry(%q) = %v, want %v", tt.expiry, got, tt.want)
		}
	}
}

func TestSleepWithContext(t *testing.T) {
	assert := assert.New(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	hako.SleepWithContext(ctx, 1*time.Second)
	elapsed := time.Since(start)

	assert.True(elapsed < 200*time.Millisecond, "SleepWithContext took too long: %v", elapsed)
}
