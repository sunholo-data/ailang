package pipeline

import (
	"fmt"
	"os"
)

// reportLoweringTelemetry prints a summary of CoreTI fallbacks during lowering (M-DX4)
func reportLoweringTelemetry(events []FallbackEvent) {
	if len(events) == 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] Lowering telemetry: No operators lowered\n")
		return
	}

	// Count by fallback type
	hits := 0
	misses := 0
	resolvedConstraints := 0
	defaults := 0

	for _, event := range events {
		switch event.Fallback {
		case "CoreTI-hit":
			hits++
		case "CoreTI-miss":
			misses++
		case "ResolvedConstraints", "ResolvedConstraints-intrinsic":
			resolvedConstraints++
		case "Default":
			defaults++
		}
	}

	total := len(events)
	fmt.Fprintf(os.Stderr, "[DEBUG] Lowering telemetry: %d operators processed\n", total)
	fmt.Fprintf(os.Stderr, "[DEBUG]   CoreTI hits: %d (%.1f%%)\n", hits, float64(hits)/float64(total)*100)
	fmt.Fprintf(os.Stderr, "[DEBUG]   CoreTI misses: %d (%.1f%%)\n", misses, float64(misses)/float64(total)*100)
	fmt.Fprintf(os.Stderr, "[DEBUG]   ResolvedConstraints fallback: %d (%.1f%%)\n", resolvedConstraints, float64(resolvedConstraints)/float64(total)*100)
	fmt.Fprintf(os.Stderr, "[DEBUG]   Default fallback: %d (%.1f%%)\n", defaults, float64(defaults)/float64(total)*100)

	// Show CoreTI misses details (these are the gaps we want to fix)
	if misses > 0 {
		fmt.Fprintf(os.Stderr, "[DEBUG] CoreTI miss details:\n")
		for _, event := range events {
			if event.Fallback == "CoreTI-miss" {
				loc := event.Location
				if loc == "" {
					loc = "unknown"
				}
				fmt.Fprintf(os.Stderr, "[DEBUG]   %v (NodeID %d) at %s\n", event.Op, event.NodeID, loc)
			}
		}
	}
}
