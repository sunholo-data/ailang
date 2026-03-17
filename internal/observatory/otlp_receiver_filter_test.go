package observatory

import (
	"os"
	"testing"
)

// TestShouldFilterSpan tests the span filtering logic with default config.
func TestShouldFilterSpan(t *testing.T) {
	receiver := &OTLPReceiver{filterConfig: DefaultSpanFilterConfig()}

	tests := []struct {
		name          string
		spanName      string
		resourceAttrs map[string]any
		shouldFilter  bool
	}{
		{
			name:         "GCP Trace internal span",
			spanName:     "google.devtools.cloudtrace.v2.TraceService.BatchWriteSpans",
			shouldFilter: true,
		},
		{
			name:         "OTEL SDK internal span",
			spanName:     "opentelemetry.sdk.trace.SpanProcessor",
			shouldFilter: true,
		},
		{
			name:         "Health check endpoint",
			spanName:     "/health",
			shouldFilter: true,
		},
		{
			name:         "Static assets",
			spanName:     "/assets/main.js",
			shouldFilter: true,
		},
		{
			name:         "Polling endpoint",
			spanName:     "/api/observatory/traces",
			shouldFilter: true,
		},
		{
			name:          "Coordinator polling",
			spanName:      "messages.list",
			resourceAttrs: map[string]any{"service.name": "ailang-coordinator"},
			shouldFilter:  true,
		},
		{
			name:         "Normal user operation",
			spanName:     "compile.typecheck",
			shouldFilter: false,
		},
		{
			name:         "AI generation span",
			spanName:     "anthropic.generate",
			shouldFilter: false,
		},
		{
			name:          "CLI messages.list is kept",
			spanName:      "messages.list",
			resourceAttrs: map[string]any{"service.name": "ailang-messages"},
			shouldFilter:  false,
		},
		{
			name:         "Control plane endpoint",
			spanName:     "/api/controlplane/exec-hierarchy",
			shouldFilter: true,
		},
		{
			name:         "Coordinator events SSE",
			spanName:     "/api/coordinator/events",
			shouldFilter: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.resourceAttrs == nil {
				tt.resourceAttrs = map[string]any{}
			}
			got := receiver.shouldFilterSpan(tt.spanName, tt.resourceAttrs)
			if got != tt.shouldFilter {
				t.Errorf("shouldFilterSpan(%q) = %v, want %v", tt.spanName, got, tt.shouldFilter)
			}
		})
	}
}

// TestSpanFilterAllow tests that allow-listed patterns bypass deny rules.
func TestSpanFilterAllow(t *testing.T) {
	config := DefaultSpanFilterConfig()
	config.AllowPatterns = []FilterPattern{
		{Type: "exact", Pattern: "coordinator.dispatch"},
		{Type: "prefix", Pattern: "/api/controlplane/exec"},
	}
	receiver := &OTLPReceiver{filterConfig: config}

	tests := []struct {
		name         string
		spanName     string
		shouldFilter bool
	}{
		{"allow-listed exact span is kept", "coordinator.dispatch", false},
		{"allow-listed prefix span is kept", "/api/controlplane/exec-hierarchy", false},
		{"non-allowed deny span still filtered", "/api/observatory/traces", true},
		{"normal span still kept", "compile.typecheck", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := receiver.shouldFilterSpan(tt.spanName, map[string]any{})
			if got != tt.shouldFilter {
				t.Errorf("shouldFilterSpan(%q) = %v, want %v", tt.spanName, got, tt.shouldFilter)
			}
		})
	}
}

// TestSpanFilterAllowOverridesDeny verifies allow takes priority over deny.
func TestSpanFilterAllowOverridesDeny(t *testing.T) {
	config := DefaultSpanFilterConfig()
	config.AllowPatterns = []FilterPattern{
		{Type: "exact", Pattern: "/health"},
	}
	receiver := &OTLPReceiver{filterConfig: config}

	got := receiver.shouldFilterSpan("/health", map[string]any{})
	if got != false {
		t.Errorf("allow should override deny: shouldFilterSpan(/health) = true, want false")
	}
}

// TestSpanFilterDeny tests custom deny patterns added via config.
func TestSpanFilterDeny(t *testing.T) {
	config := DefaultSpanFilterConfig()
	config.DenyPatterns = append(config.DenyPatterns,
		FilterPattern{Type: "exact", Pattern: "custom.noisy.op"},
		FilterPattern{Type: "prefix", Pattern: "debug."},
	)
	receiver := &OTLPReceiver{filterConfig: config}

	tests := []struct {
		name         string
		spanName     string
		shouldFilter bool
	}{
		{"custom exact deny", "custom.noisy.op", true},
		{"custom prefix deny", "debug.verbose", true},
		{"default deny still works", "/health", true},
		{"normal span still kept", "compile.typecheck", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := receiver.shouldFilterSpan(tt.spanName, map[string]any{})
			if got != tt.shouldFilter {
				t.Errorf("shouldFilterSpan(%q) = %v, want %v", tt.spanName, got, tt.shouldFilter)
			}
		})
	}
}

// TestSpanFilterDisable tests that DisableAll passes everything through.
func TestSpanFilterDisable(t *testing.T) {
	config := DefaultSpanFilterConfig()
	config.DisableAll = true
	receiver := &OTLPReceiver{filterConfig: config}

	spans := []string{"/health", "opentelemetry.sdk", "/api/observatory/traces", "messages.list"}
	for _, span := range spans {
		got := receiver.shouldFilterSpan(span, map[string]any{"service.name": "ailang-coordinator"})
		if got != false {
			t.Errorf("DisableAll: shouldFilterSpan(%q) = true, want false", span)
		}
	}
}

// TestSpanFilterServiceScoped tests service-scoped allow/deny patterns.
func TestSpanFilterServiceScoped(t *testing.T) {
	config := &SpanFilterConfig{
		DenyPatterns: []FilterPattern{
			{Type: "exact", Pattern: "noisy.op", Service: "service-a"},
		},
	}
	receiver := &OTLPReceiver{filterConfig: config}

	tests := []struct {
		name          string
		spanName      string
		resourceAttrs map[string]any
		shouldFilter  bool
	}{
		{"scoped deny matches correct service", "noisy.op", map[string]any{"service.name": "service-a"}, true},
		{"scoped deny skips wrong service", "noisy.op", map[string]any{"service.name": "service-b"}, false},
		{"scoped deny skips no service", "noisy.op", map[string]any{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := receiver.shouldFilterSpan(tt.spanName, tt.resourceAttrs)
			if got != tt.shouldFilter {
				t.Errorf("shouldFilterSpan(%q, service=%v) = %v, want %v",
					tt.spanName, tt.resourceAttrs["service.name"], got, tt.shouldFilter)
			}
		})
	}
}

// TestParseFilterPattern tests the pattern parser.
func TestParseFilterPattern(t *testing.T) {
	tests := []struct {
		input    string
		expected FilterPattern
	}{
		{"coordinator.dispatch", FilterPattern{Type: "exact", Pattern: "coordinator.dispatch"}},
		{"coordinator.*", FilterPattern{Type: "prefix", Pattern: "coordinator."}},
		{"*.js", FilterPattern{Type: "suffix", Pattern: ".js"}},
		{"ailang-coordinator:messages.list", FilterPattern{Type: "exact", Pattern: "messages.list", Service: "ailang-coordinator"}},
		{"svc:prefix*", FilterPattern{Type: "prefix", Pattern: "prefix", Service: "svc"}},
		{"/api/health", FilterPattern{Type: "exact", Pattern: "/api/health"}},
		{"  spaced  ", FilterPattern{Type: "exact", Pattern: "spaced"}},
		{"", FilterPattern{}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseFilterPattern(tt.input)
			if got != tt.expected {
				t.Errorf("parseFilterPattern(%q) = %+v, want %+v", tt.input, got, tt.expected)
			}
		})
	}
}

// TestLoadSpanFilterConfig tests env var parsing.
func TestLoadSpanFilterConfig(t *testing.T) {
	origAllow := os.Getenv("AILANG_SPAN_FILTER_ALLOW")
	origDeny := os.Getenv("AILANG_SPAN_FILTER_DENY")
	origDisable := os.Getenv("AILANG_SPAN_FILTER_DISABLE")
	defer func() {
		os.Setenv("AILANG_SPAN_FILTER_ALLOW", origAllow)
		os.Setenv("AILANG_SPAN_FILTER_DENY", origDeny)
		os.Setenv("AILANG_SPAN_FILTER_DISABLE", origDisable)
	}()

	t.Run("no env vars uses defaults", func(t *testing.T) {
		os.Setenv("AILANG_SPAN_FILTER_ALLOW", "")
		os.Setenv("AILANG_SPAN_FILTER_DENY", "")
		os.Setenv("AILANG_SPAN_FILTER_DISABLE", "")
		config := LoadSpanFilterConfig()
		if len(config.AllowPatterns) != 0 {
			t.Errorf("expected 0 allow patterns, got %d", len(config.AllowPatterns))
		}
		if len(config.DenyPatterns) == 0 {
			t.Error("expected default deny patterns, got 0")
		}
		if config.DisableAll {
			t.Error("expected DisableAll=false")
		}
	})

	t.Run("allow env adds patterns", func(t *testing.T) {
		os.Setenv("AILANG_SPAN_FILTER_ALLOW", "coordinator.dispatch,executor.run")
		os.Setenv("AILANG_SPAN_FILTER_DENY", "")
		os.Setenv("AILANG_SPAN_FILTER_DISABLE", "")
		config := LoadSpanFilterConfig()
		if len(config.AllowPatterns) != 2 {
			t.Errorf("expected 2 allow patterns, got %d", len(config.AllowPatterns))
		}
	})

	t.Run("deny env appends to defaults", func(t *testing.T) {
		os.Setenv("AILANG_SPAN_FILTER_ALLOW", "")
		os.Setenv("AILANG_SPAN_FILTER_DENY", "custom.op")
		os.Setenv("AILANG_SPAN_FILTER_DISABLE", "")
		defaultCount := len(DefaultSpanFilterConfig().DenyPatterns)
		config := LoadSpanFilterConfig()
		if len(config.DenyPatterns) != defaultCount+1 {
			t.Errorf("expected %d deny patterns, got %d", defaultCount+1, len(config.DenyPatterns))
		}
	})

	t.Run("disable env", func(t *testing.T) {
		os.Setenv("AILANG_SPAN_FILTER_ALLOW", "")
		os.Setenv("AILANG_SPAN_FILTER_DENY", "")
		os.Setenv("AILANG_SPAN_FILTER_DISABLE", "true")
		config := LoadSpanFilterConfig()
		if !config.DisableAll {
			t.Error("expected DisableAll=true")
		}
	})
}
