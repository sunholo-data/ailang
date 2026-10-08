package main

import (
	"testing"
	"time"
)

func TestReleaseAt(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 12, 0, 0, 0, time.UTC) }
	rs := []release{{"v0.52.0", day(2)}, {"v0.52.1", day(3)}, {"v0.52.5", day(7)}}
	for _, c := range []struct {
		at   time.Time
		want string
	}{
		{day(1), ""},         // before the first release
		{day(2), "v0.52.0"},  // tagged the same instant
		{day(5), "v0.52.1"},  // between releases: the one current at the time
		{day(30), "v0.52.5"}, // after the last
	} {
		if got := releaseAt(rs, c.at); got != c.want {
			t.Errorf("releaseAt(%s) = %q, want %q", c.at.Format("01-02"), got, c.want)
		}
	}
}

func TestSkipped(t *testing.T) {
	skip := []string{"qwen", "motoko-local"}
	for model, want := range map[string]bool{
		"opencode-qwen3-8-27b":     true,
		"motoko-local-qwen3-8-27b": true,
		"claude-haiku-5-5":         false,
		"motoko-or-glm-5-3-flash":  false,
	} {
		if got := skipped(model, skip); got != want {
			t.Errorf("skipped(%q) = %v, want %v", model, got, want)
		}
	}
}

func TestReleaseTagPattern(t *testing.T) {
	for tag, want := range map[string]bool{"v0.52.5": true, "v0.26.0-rc1": false, "v1.0": false, "nightly": false} {
		if got := releaseTagRE.MatchString(tag); got != want {
			t.Errorf("releaseTagRE(%q) = %v, want %v", tag, got, want)
		}
	}
}
