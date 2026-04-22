// Package microrag implements the just-in-time knowledge injection engine.
// See design_docs/planned/v0_15_0/m-brain-microrag.md for the design.
package microrag

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Route maps a file glob to a knowledge-base namespace and per-route limits.
type Route struct {
	Glob                  string  `yaml:"glob"`
	KB                    string  `yaml:"kb"`
	MaxTokensPerInjection int     `yaml:"max_tokens_per_injection"`
	RelevanceFloor        float64 `yaml:"relevance_floor"`
}

// DedupConfig holds per-namespace dedup tuning.
type DedupConfig struct {
	Windows          map[string]int     `yaml:"windows"`
	RelevanceBypass  map[string]float64 `yaml:"relevance_bypass"`
	WallClockMaxSecs int                `yaml:"wall_clock_max"`
}

// Config is the full micro-rag config (~/.ailang/microrag.yaml).
type Config struct {
	Enabled       bool        `yaml:"enabled"`
	Routes        []Route     `yaml:"routes"`
	Dedup         DedupConfig `yaml:"dedup"`
	SessionBudget int         `yaml:"session_budget"`
	MarkerStyle   string      `yaml:"marker_style"`
}

// Default values applied when the config file is missing or fields are unset.
const (
	defaultMaxTokensPerInjection = 150
	defaultRelevanceFloor        = 0.30
	defaultWindow                = 30000
	defaultRelevanceBypass       = 0.70
	defaultWallClockSecs         = 240
	defaultSessionBudget         = 5000
	defaultMarkerStyle           = "unicode" // unicode | ascii
)

// LoadConfig reads ~/.ailang/microrag.yaml (or path) and applies defaults.
// A missing file is not an error — the engine runs with defaults + the
// fallback "**/*.ail" route pointing at ailang-syntax.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{Enabled: true}

	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return cfg.applyDefaults(), nil
		}
		path = filepath.Join(home, ".ailang", "microrag.yaml")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg.applyDefaults(), nil
		}
		return nil, fmt.Errorf("read microrag config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse microrag config %q: %w", path, err)
	}
	return cfg.applyDefaults(), nil
}

func (c *Config) applyDefaults() *Config {
	if c.SessionBudget <= 0 {
		c.SessionBudget = defaultSessionBudget
	}
	if c.MarkerStyle == "" {
		c.MarkerStyle = defaultMarkerStyle
	}
	if c.Dedup.WallClockMaxSecs <= 0 {
		c.Dedup.WallClockMaxSecs = defaultWallClockSecs
	}
	if c.Dedup.Windows == nil {
		c.Dedup.Windows = map[string]int{}
	}
	if c.Dedup.RelevanceBypass == nil {
		c.Dedup.RelevanceBypass = map[string]float64{}
	}
	if _, ok := c.Dedup.Windows["default"]; !ok {
		c.Dedup.Windows["default"] = defaultWindow
	}
	if _, ok := c.Dedup.RelevanceBypass["default"]; !ok {
		c.Dedup.RelevanceBypass["default"] = defaultRelevanceBypass
	}
	if len(c.Routes) == 0 {
		c.Routes = []Route{{
			Glob:                  "**/*.ail",
			KB:                    "ailang-syntax",
			MaxTokensPerInjection: defaultMaxTokensPerInjection,
			RelevanceFloor:        defaultRelevanceFloor,
		}}
	}
	for i := range c.Routes {
		r := &c.Routes[i]
		if r.MaxTokensPerInjection <= 0 {
			r.MaxTokensPerInjection = defaultMaxTokensPerInjection
		}
		if r.RelevanceFloor <= 0 {
			r.RelevanceFloor = defaultRelevanceFloor
		}
	}
	return c
}

// WindowFor returns the dedup token window for the given namespace.
func (c *Config) WindowFor(ns string) int {
	if w, ok := c.Dedup.Windows[ns]; ok && w > 0 {
		return w
	}
	return c.Dedup.Windows["default"]
}

// BypassFor returns the relevance-bypass threshold for the given namespace.
func (c *Config) BypassFor(ns string) float64 {
	if b, ok := c.Dedup.RelevanceBypass[ns]; ok && b > 0 {
		return b
	}
	return c.Dedup.RelevanceBypass["default"]
}

// MatchRoute returns the first matching route for a file path, or nil.
// Routes with kb=="skip" return as a sentinel that callers must check.
func (c *Config) MatchRoute(filePath string) *Route {
	for i := range c.Routes {
		r := &c.Routes[i]
		if matchGlob(r.Glob, filePath) {
			return r
		}
	}
	return nil
}

// matchGlob is a small **-aware glob matcher: ** matches any number of path
// segments (including zero); * matches a single segment. Falls back to
// path.Match semantics for the per-segment portions.
func matchGlob(pattern, filePath string) bool {
	if pattern == "" {
		return false
	}
	// Normalize to forward slashes for matching consistency.
	filePath = filepath.ToSlash(filePath)

	// Split on /**/ — any number of intermediate segments allowed.
	if strings.Contains(pattern, "**") {
		// Special case "**/*.ext" — match basename anywhere.
		parts := strings.Split(pattern, "**")
		if len(parts) == 2 {
			prefix := strings.TrimSuffix(parts[0], "/")
			suffix := strings.TrimPrefix(parts[1], "/")
			if prefix != "" && !strings.HasPrefix(filePath, prefix) {
				return false
			}
			return matchSuffixPattern(suffix, filePath)
		}
	}
	ok, _ := filepath.Match(pattern, filePath)
	return ok
}

func matchSuffixPattern(suffix, filePath string) bool {
	if suffix == "" {
		return true
	}
	segs := strings.Split(filePath, "/")
	for i := 0; i <= len(segs); i++ {
		candidate := strings.Join(segs[i:], "/")
		ok, _ := filepath.Match(suffix, candidate)
		if ok {
			return true
		}
		// Also try basename-only match if suffix has no slash.
		if !strings.Contains(suffix, "/") && i == len(segs)-1 {
			ok2, _ := filepath.Match(suffix, segs[i])
			if ok2 {
				return true
			}
		}
	}
	return false
}
