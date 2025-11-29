package pipeline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"
)

// PipelineMetrics contains timing and resource metrics from compilation
type PipelineMetrics struct {
	// Timing metrics (milliseconds)
	// Single-file pipeline phases
	LexTime            int64 `json:"lex_ms,omitempty"`
	ParseTime          int64 `json:"parse_ms,omitempty"`
	ElaborateTime      int64 `json:"elaborate_ms,omitempty"`
	TypeCheckTime      int64 `json:"typecheck_ms,omitempty"`
	DictElabTime       int64 `json:"dict_elab_ms,omitempty"`
	MonomorphTime      int64 `json:"monomorph_ms,omitempty"`
	LowerTime          int64 `json:"lower_ms,omitempty"`
	LinkTime           int64 `json:"link_ms,omitempty"`
	ANFVerifyTime      int64 `json:"anf_verify_ms,omitempty"`

	// Module pipeline phases
	LoadTime           int64 `json:"load_ms,omitempty"`    // Module loading (for multi-module)
	TopoTime           int64 `json:"topo_ms,omitempty"`    // Topological sort
	CompileTime        int64 `json:"compile_ms,omitempty"` // All modules compilation (aggregate)

	// Shared phases
	EvalTime           int64 `json:"eval_ms,omitempty"`
	TotalTime          int64 `json:"total_ms"`

	// Resource metrics
	MemoryDeltaBytes   int64 `json:"memory_delta_bytes"`
	AllocsCount        int64 `json:"allocs_count"`

	// Compilation stats
	ModulesCompiled    int   `json:"modules_compiled"`
	Specializations    int   `json:"specializations"`
	OperatorsLowered   int   `json:"operators_lowered"`

	// Context
	Filename           string `json:"filename"`
	IsModule           bool   `json:"is_module"`
	Timestamp          int64  `json:"timestamp"`
}

// MetricsCollector collects pipeline metrics
type MetricsCollector struct {
	enabled        bool
	hubURL         string
	startMem       runtime.MemStats
	startTime      time.Time
	metrics        PipelineMetrics
}

// NewMetricsCollector creates a new metrics collector
// Only enabled if AILANG_METRICS=1 is set
func NewMetricsCollector(filename string, isModule bool) *MetricsCollector {
	enabled := os.Getenv("AILANG_METRICS") == "1"

	mc := &MetricsCollector{
		enabled: enabled,
		hubURL:  os.Getenv("AILANG_HUB_URL"),
		metrics: PipelineMetrics{
			Filename:  filename,
			IsModule:  isModule,
			Timestamp: time.Now().UnixMilli(),
		},
	}

	if enabled {
		// Capture initial memory stats
		runtime.ReadMemStats(&mc.startMem)
		mc.startTime = time.Now()
	}

	return mc
}

// IsEnabled returns whether metrics collection is enabled
func (mc *MetricsCollector) IsEnabled() bool {
	return mc.enabled
}

// RecordPhase records a phase timing from the Result.PhaseTimings map
func (mc *MetricsCollector) RecordPhase(name string, durationMs int64) {
	if !mc.enabled {
		return
	}

	switch name {
	case "lex":
		mc.metrics.LexTime = durationMs
	case "parse":
		mc.metrics.ParseTime = durationMs
	case "load":
		mc.metrics.LoadTime = durationMs
	case "topo":
		mc.metrics.TopoTime = durationMs
	case "elaborate":
		mc.metrics.ElaborateTime = durationMs
	case "typecheck":
		mc.metrics.TypeCheckTime = durationMs
	case "dict_elaboration", "dict_elab":
		mc.metrics.DictElabTime = durationMs
	case "monomorphization":
		mc.metrics.MonomorphTime = durationMs
	case "lower":
		mc.metrics.LowerTime = durationMs
	case "link":
		mc.metrics.LinkTime = durationMs
	case "anf_verify":
		mc.metrics.ANFVerifyTime = durationMs
	case "compile":
		mc.metrics.CompileTime = durationMs
	case "evaluate":
		mc.metrics.EvalTime = durationMs
	}
}

// RecordFromResult populates metrics from a pipeline Result
func (mc *MetricsCollector) RecordFromResult(result *Result) {
	if !mc.enabled || result == nil {
		return
	}

	// Debug: show what's in the PhaseTimings
	if os.Getenv("AILANG_METRICS_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "[METRICS DEBUG] PhaseTimings has %d entries:\n", len(result.PhaseTimings))
		for name, durationMs := range result.PhaseTimings {
			fmt.Fprintf(os.Stderr, "[METRICS DEBUG]   %s: %dms\n", name, durationMs)
		}
	}

	for name, durationMs := range result.PhaseTimings {
		mc.RecordPhase(name, durationMs)
	}
}

// SetStats sets compilation statistics
func (mc *MetricsCollector) SetStats(modulesCompiled, specializations, operatorsLowered int) {
	if !mc.enabled {
		return
	}
	mc.metrics.ModulesCompiled = modulesCompiled
	mc.metrics.Specializations = specializations
	mc.metrics.OperatorsLowered = operatorsLowered
}

// Finalize calculates final metrics and optionally sends to hub
func (mc *MetricsCollector) Finalize() *PipelineMetrics {
	if !mc.enabled {
		return nil
	}

	// Calculate total time
	mc.metrics.TotalTime = time.Since(mc.startTime).Milliseconds()

	// Calculate memory delta
	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)
	mc.metrics.MemoryDeltaBytes = int64(endMem.TotalAlloc - mc.startMem.TotalAlloc)
	mc.metrics.AllocsCount = int64(endMem.Mallocs - mc.startMem.Mallocs)

	// Print summary to stderr if AILANG_METRICS_VERBOSE=1
	if os.Getenv("AILANG_METRICS_VERBOSE") == "1" {
		mc.printSummary()
	}

	// Send to hub if configured
	if mc.hubURL != "" {
		mc.sendToHub()
	}

	return &mc.metrics
}

// printSummary prints a human-readable metrics summary
func (mc *MetricsCollector) printSummary() {
	fmt.Fprintf(os.Stderr, "\n[METRICS] Pipeline Metrics for %s\n", mc.metrics.Filename)
	fmt.Fprintf(os.Stderr, "[METRICS] ----------------------------------------\n")
	fmt.Fprintf(os.Stderr, "[METRICS] Phase Timings:\n")

	// Module pipeline phases
	if mc.metrics.IsModule {
		mc.printPhase("Load", mc.metrics.LoadTime)
		mc.printPhase("TopoSort", mc.metrics.TopoTime)
		mc.printPhase("Compile", mc.metrics.CompileTime)
	} else {
		// Single-file pipeline phases
		mc.printPhase("Parse", mc.metrics.ParseTime)
		mc.printPhase("Elaborate", mc.metrics.ElaborateTime)
		mc.printPhase("TypeCheck", mc.metrics.TypeCheckTime)
		mc.printPhase("DictElab", mc.metrics.DictElabTime)
		mc.printPhase("Monomorph", mc.metrics.MonomorphTime)
		mc.printPhase("Lower", mc.metrics.LowerTime)
		mc.printPhase("Link", mc.metrics.LinkTime)
	}

	// Shared phases
	mc.printPhase("Evaluate", mc.metrics.EvalTime)
	fmt.Fprintf(os.Stderr, "[METRICS]   ----------------------\n")
	fmt.Fprintf(os.Stderr, "[METRICS]   Total:        %4dms\n", mc.metrics.TotalTime)

	// Print resources
	mc.printResources()
}

// printPhase prints a phase timing in ms (shows <1ms for sub-millisecond timings)
func (mc *MetricsCollector) printPhase(name string, durationMs int64) {
	if durationMs > 0 {
		fmt.Fprintf(os.Stderr, "[METRICS]   %-12s %4dms\n", name+":", durationMs)
	} else {
		// Still show the phase but indicate it was fast
		fmt.Fprintf(os.Stderr, "[METRICS]   %-12s <1ms\n", name+":")
	}
}

// printResources prints memory and compilation stats
func (mc *MetricsCollector) printResources() {
	fmt.Fprintf(os.Stderr, "[METRICS] Resources:\n")
	fmt.Fprintf(os.Stderr, "[METRICS]   Memory:       %s\n", formatBytes(mc.metrics.MemoryDeltaBytes))
	fmt.Fprintf(os.Stderr, "[METRICS]   Allocs:       %d\n", mc.metrics.AllocsCount)

	if mc.metrics.ModulesCompiled > 0 || mc.metrics.Specializations > 0 {
		fmt.Fprintf(os.Stderr, "[METRICS] Stats:\n")
		if mc.metrics.ModulesCompiled > 0 {
			fmt.Fprintf(os.Stderr, "[METRICS]   Modules:      %d\n", mc.metrics.ModulesCompiled)
		}
		if mc.metrics.Specializations > 0 {
			fmt.Fprintf(os.Stderr, "[METRICS]   Specializations: %d\n", mc.metrics.Specializations)
		}
		if mc.metrics.OperatorsLowered > 0 {
			fmt.Fprintf(os.Stderr, "[METRICS]   Operators:    %d\n", mc.metrics.OperatorsLowered)
		}
	}
	fmt.Fprintf(os.Stderr, "[METRICS] ----------------------------------------\n\n")
}

// sendToHub sends metrics to the collaboration hub
func (mc *MetricsCollector) sendToHub() {
	// Build telemetry request
	req := struct {
		PID        int              `json:"pid"`
		InstanceID string           `json:"instance_id"`
		Status     string           `json:"status"`
		Metrics    *PipelineMetrics `json:"metrics,omitempty"`
	}{
		PID:        os.Getpid(),
		InstanceID: fmt.Sprintf("compile_%d", os.Getpid()),
		Status:     "completed",
		Metrics:    &mc.metrics,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return // Silent fail - metrics are best-effort
	}

	// Send in background
	go func() {
		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Post(
			fmt.Sprintf("%s/api/telemetry", mc.hubURL),
			"application/json",
			bytes.NewReader(body),
		)
		if err != nil {
			return // Silent fail
		}
		resp.Body.Close()
	}()
}

// formatBytes formats bytes in human-readable form
func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
