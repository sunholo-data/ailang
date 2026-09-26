package main

import (
	"testing"
	"time"
)

func TestResolveIdleTimeout(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want time.Duration
	}{
		{"declared 5m, what sprint-evaluator asks for", "5m", 5 * time.Minute},
		{"declared 10m", "10m", 10 * time.Minute},
		{"declared 6m", "6m", 6 * time.Minute},
		{"unset falls back to the executor default", "", 0},
		{"malformed falls back rather than guessing", "not-a-duration", 0},
		{"zero is not a real budget", "0s", 0},
		{"negative is not a real budget", "-1m", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveIdleTimeout(tt.raw); got != tt.want {
				t.Errorf("resolveIdleTimeout(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}
