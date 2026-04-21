package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	trace "cloud.google.com/go/trace/apiv1"
	"cloud.google.com/go/trace/apiv1/tracepb"
	"github.com/sunholo-data/ailang/internal/telemetry"
	"google.golang.org/api/iterator"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func traceCommand() {
	if flag.NArg() < 2 {
		fmt.Println("Usage: ailang trace <subcommand> [options]")
		fmt.Println()
		fmt.Println("Subcommands:")
		fmt.Println("  list       List recent traces (GCP)")
		fmt.Println("  view       View details of a specific trace (GCP)")
		fmt.Println("  status     Show telemetry configuration status")
		fmt.Println("  hierarchy  Show span hierarchy from local database")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  ailang trace list --limit 10")
		fmt.Println("  ailang trace view <trace-id>")
		fmt.Println("  ailang trace status")
		fmt.Println("  ailang trace hierarchy --limit 5")
		return
	}

	subcommand := flag.Arg(1)
	switch subcommand {
	case "list":
		traceListCommand()
	case "view":
		traceViewCommand()
	case "status":
		traceStatusCommand()
	case "hierarchy":
		traceHierarchyCommand()
	default:
		fmt.Fprintf(os.Stderr, "Unknown trace subcommand: %s\n", subcommand)
		os.Exit(1)
	}
}

func traceStatusCommand() {
	fmt.Println("Telemetry Configuration Status")
	fmt.Println(strings.Repeat("─", 40))

	gcpProject := telemetry.GoogleCloudProject()
	otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	fmt.Printf("Google Cloud Project: %s\n", valueOrNone(gcpProject))
	fmt.Printf("OTLP Endpoint:        %s\n", valueOrNone(otlpEndpoint))
	fmt.Println()

	if telemetry.IsDualExportEnabled() {
		fmt.Println("Mode: Dual Export (GCP + OTLP)")
	} else if telemetry.IsGoogleCloudEnabled() {
		fmt.Println("Mode: Google Cloud Trace")
		fmt.Printf("View traces: https://console.cloud.google.com/traces/explorer?project=%s\n", gcpProject)
	} else if telemetry.IsEnabled() {
		fmt.Println("Mode: Generic OTLP")
	} else {
		fmt.Println("Mode: Disabled (no telemetry environment variables set)")
		fmt.Println()
		fmt.Println("To enable telemetry:")
		fmt.Println("  export GOOGLE_CLOUD_PROJECT=your-project-id")
		fmt.Println("  # or")
		fmt.Println("  export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318")
	}
}

func traceListCommand() {
	// Parse flags
	fs := flag.NewFlagSet("trace list", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Maximum number of traces to list")
	hours := fs.Int("hours", 1, "Look back this many hours")
	filter := fs.String("filter", "", "Filter by span name (e.g., 'ailang run')")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	showAll := fs.Bool("all", false, "Show all traces including internal OTEL exporter traces")

	// Skip "ailang trace list" args
	if err := fs.Parse(os.Args[3:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	projectID := telemetry.GoogleCloudProject()
	if projectID == "" {
		fmt.Fprintf(os.Stderr, "Error: GOOGLE_CLOUD_PROJECT or OTLP_GOOGLE_CLOUD_PROJECT not set\n")
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := trace.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating trace client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	endTime := time.Now()
	startTime := endTime.Add(-time.Duration(*hours) * time.Hour)

	req := &tracepb.ListTracesRequest{
		ProjectId: projectID,
		View:      tracepb.ListTracesRequest_ROOTSPAN,
		StartTime: timestampProto(startTime),
		EndTime:   timestampProto(endTime),
		PageSize:  int32(*limit),
	}

	if *filter != "" {
		req.Filter = fmt.Sprintf("+root:/%s/", *filter)
	}

	it := client.ListTraces(ctx, req)

	var traces []map[string]interface{}
	count := 0

	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error listing traces: %v\n", err)
			os.Exit(1)
		}

		// Skip internal OTEL exporter traces unless --all is specified
		if !*showAll && len(resp.Spans) > 0 {
			rootName := resp.Spans[0].Name
			if isInternalTrace(rootName) {
				continue
			}
		}

		count++
		if count > *limit {
			break
		}

		traceData := map[string]interface{}{
			"trace_id":   resp.TraceId,
			"span_count": len(resp.Spans),
		}

		if len(resp.Spans) > 0 {
			rootSpan := resp.Spans[0]
			traceData["name"] = rootSpan.Name
			traceData["start_time"] = rootSpan.StartTime.AsTime().Format(time.RFC3339)
			if rootSpan.EndTime != nil {
				duration := rootSpan.EndTime.AsTime().Sub(rootSpan.StartTime.AsTime())
				traceData["duration_ms"] = duration.Milliseconds()
			}
			if rootSpan.Labels != nil {
				traceData["labels"] = rootSpan.Labels
			}
		}

		traces = append(traces, traceData)
	}

	if *jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(traces)
		return
	}

	if len(traces) == 0 {
		fmt.Printf("No traces found in the last %d hour(s)\n", *hours)
		fmt.Printf("View in console: https://console.cloud.google.com/traces/explorer?project=%s\n", projectID)
		return
	}

	fmt.Printf("Recent Traces (last %d hour(s), project: %s)\n", *hours, projectID)
	fmt.Println(strings.Repeat("─", 80))

	for _, t := range traces {
		name := t["name"].(string)
		traceID := t["trace_id"].(string)
		startTime := t["start_time"].(string)
		spanCount := t["span_count"].(int)

		durationStr := ""
		if d, ok := t["duration_ms"]; ok {
			durationStr = fmt.Sprintf(" (%dms)", d.(int64))
		}

		fmt.Printf("• %s%s\n", name, durationStr)
		// Show full trace ID so users can copy it for `trace view`
		fmt.Printf("  ID: %s | Spans: %d | Started: %s\n", traceID, spanCount, startTime)

		if labels, ok := t["labels"].(map[string]string); ok && len(labels) > 0 {
			for k, v := range labels {
				if k == "file.path" || k == "is_repl" {
					fmt.Printf("  %s: %s\n", k, v)
				}
			}
		}
		fmt.Println()
	}

	fmt.Printf("View in console: https://console.cloud.google.com/traces/explorer?project=%s\n", projectID)
}

func traceViewCommand() {
	if flag.NArg() < 3 {
		fmt.Println("Usage: ailang trace view <trace-id>")
		fmt.Println()
		fmt.Println("The trace ID must be the full 32-character hex ID from 'ailang trace list'")
		return
	}

	traceID := flag.Arg(2)

	// Validate trace ID format (should be 32 hex characters)
	if len(traceID) != 32 {
		fmt.Fprintf(os.Stderr, "Error: Invalid trace ID length (%d chars, expected 32)\n", len(traceID))
		fmt.Fprintf(os.Stderr, "Use 'ailang trace list' to get the full trace ID\n")
		os.Exit(1)
	}

	projectID := telemetry.GoogleCloudProject()
	if projectID == "" {
		fmt.Fprintf(os.Stderr, "Error: GOOGLE_CLOUD_PROJECT or OTLP_GOOGLE_CLOUD_PROJECT not set\n")
		os.Exit(1)
	}

	ctx := context.Background()
	client, err := trace.NewClient(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating trace client: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	req := &tracepb.GetTraceRequest{
		ProjectId: projectID,
		TraceId:   traceID,
	}

	resp, err := client.GetTrace(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting trace: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nTip: Make sure you're using the full 32-character trace ID.\n")
		fmt.Fprintf(os.Stderr, "View in browser: https://console.cloud.google.com/traces/explorer?project=%s&traceId=%s\n", projectID, traceID)
		os.Exit(1)
	}

	fmt.Printf("Trace: %s\n", resp.TraceId)
	fmt.Printf("Spans: %d\n", len(resp.Spans))
	fmt.Println(strings.Repeat("─", 60))

	// Build parent-child relationships
	spanByID := make(map[uint64]*tracepb.TraceSpan)
	for _, span := range resp.Spans {
		spanByID[span.SpanId] = span
	}

	// Print spans in hierarchy
	for _, span := range resp.Spans {
		indent := ""
		if span.ParentSpanId != 0 {
			indent = "  └─ "
		}

		duration := ""
		if span.EndTime != nil {
			d := span.EndTime.AsTime().Sub(span.StartTime.AsTime())
			duration = fmt.Sprintf(" (%s)", d.Round(time.Microsecond))
		}

		fmt.Printf("%s%s%s\n", indent, span.Name, duration)

		if len(span.Labels) > 0 {
			for k, v := range span.Labels {
				fmt.Printf("%s    %s: %s\n", indent, k, v)
			}
		}
	}

	fmt.Println()
	fmt.Printf("View in console: https://console.cloud.google.com/traces/explorer?project=%s&traceId=%s\n", projectID, resp.TraceId)
}

func valueOrNone(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

func timestampProto(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}

// isInternalTrace returns true for OTEL exporter internal traces that should be hidden by default
func isInternalTrace(name string) bool {
	// Google Cloud Trace exporter internal spans
	if strings.HasPrefix(name, "google.devtools.cloudtrace") {
		return true
	}
	// OTLP exporter internal spans
	if strings.HasPrefix(name, "opentelemetry.") {
		return true
	}
	// Health check endpoints (high-frequency, low-value)
	if name == "/health" || name == "health.check" {
		return true
	}
	return false
}
