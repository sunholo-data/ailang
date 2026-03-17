// Package observatory provides OTLP metrics handling for the observatory.
package observatory

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	colmetricspb "go.opentelemetry.io/proto/otlp/collector/metrics/v1"
	metricspb "go.opentelemetry.io/proto/otlp/metrics/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// handleMetrics handles the OTLP metrics export endpoint.
// Claude Code sends metrics like:
// - claude_code.lines_of_code.count (type: added/removed)
// - claude_code.commit.count
// - claude_code.pull_request.count
// - claude_code.session.count
// - claude_code.active_time.total
// - claude_code.code_edit_tool.decision
func (r *OTLPReceiver) handleMetrics(w http.ResponseWriter, req *http.Request) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	var exportReq colmetricspb.ExportMetricsServiceRequest

	// Parse based on content type
	contentType := req.Header.Get("Content-Type")
	switch contentType {
	case "application/x-protobuf":
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse protobuf: %v", err), http.StatusBadRequest)
			return
		}
	case "application/json":
		if err := protojson.Unmarshal(body, &exportReq); err != nil {
			http.Error(w, fmt.Sprintf("failed to parse JSON: %v", err), http.StatusBadRequest)
			return
		}
	default:
		// Try protobuf first, fall back to JSON
		if err := proto.Unmarshal(body, &exportReq); err != nil {
			if jsonErr := protojson.Unmarshal(body, &exportReq); jsonErr != nil {
				http.Error(w, "unsupported content type", http.StatusUnsupportedMediaType)
				return
			}
		}
	}

	// Process and store metrics
	ctx := req.Context()
	var storedCount int
	for _, resourceMetrics := range exportReq.ResourceMetrics {
		count, err := r.processResourceMetrics(ctx, resourceMetrics)
		if err != nil {
			fmt.Printf("observatory: failed to process resource metrics: %v\n", err)
		}
		storedCount += count
	}

	if storedCount > 0 {
		fmt.Printf("observatory: stored %d metrics from OTLP\n", storedCount)
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"partialSuccess": map[string]any{},
	})
}

// processResourceMetrics processes a single ResourceMetrics message.
func (r *OTLPReceiver) processResourceMetrics(ctx context.Context, rm *metricspb.ResourceMetrics) (int, error) {
	// Extract resource attributes
	resourceAttrs := make(map[string]any)
	if rm.Resource != nil {
		for _, kv := range rm.Resource.Attributes {
			resourceAttrs[kv.Key] = anyValueToGo(kv.Value)
		}
	}

	sessionID := extractString(resourceAttrs, "session.id")
	workspace := extractString(resourceAttrs, "process.cwd")
	provider := extractString(resourceAttrs, "service.name")

	var storedCount int
	for _, scopeMetrics := range rm.ScopeMetrics {
		for _, metric := range scopeMetrics.Metrics {
			count, err := r.processMetric(ctx, metric, sessionID, workspace, provider, resourceAttrs)
			if err != nil {
				fmt.Printf("observatory: failed to process metric %s: %v\n", metric.Name, err)
				continue
			}
			storedCount += count
		}
	}
	return storedCount, nil
}

// processMetric processes a single metric and stores it.
func (r *OTLPReceiver) processMetric(ctx context.Context, m *metricspb.Metric, sessionID, workspace, provider string, resourceAttrs map[string]any) (int, error) {
	metricName := m.Name

	// Only capture Claude Code metrics (filter out other sources)
	if !strings.HasPrefix(metricName, "claude_code.") {
		return 0, nil
	}

	switch data := m.Data.(type) {
	case *metricspb.Metric_Sum:
		return r.processSumMetric(ctx, metricName, data.Sum, sessionID, workspace, provider, resourceAttrs)
	case *metricspb.Metric_Gauge:
		return r.processGaugeMetric(ctx, metricName, data.Gauge, sessionID, workspace, provider, resourceAttrs)
	default:
		// Skip histogram/summary for now
		return 0, nil
	}
}

// processSumMetric processes Sum (counter) metrics.
func (r *OTLPReceiver) processSumMetric(ctx context.Context, name string, sum *metricspb.Sum, sessionID, workspace, provider string, resourceAttrs map[string]any) (int, error) {
	var storedCount int
	for _, dp := range sum.DataPoints {
		metric := r.buildMetricFromDataPoint(name, "counter", dp, sessionID, workspace, provider, resourceAttrs)
		if err := r.backend.CreateMetric(ctx, metric); err != nil {
			return storedCount, fmt.Errorf("store metric: %w", err)
		}
		storedCount++
	}
	return storedCount, nil
}

// processGaugeMetric processes Gauge metrics.
func (r *OTLPReceiver) processGaugeMetric(ctx context.Context, name string, gauge *metricspb.Gauge, sessionID, workspace, provider string, resourceAttrs map[string]any) (int, error) {
	var storedCount int
	for _, dp := range gauge.DataPoints {
		metric := r.buildMetricFromDataPoint(name, "gauge", dp, sessionID, workspace, provider, resourceAttrs)
		if err := r.backend.CreateMetric(ctx, metric); err != nil {
			return storedCount, fmt.Errorf("store metric: %w", err)
		}
		storedCount++
	}
	return storedCount, nil
}

// buildMetricFromDataPoint creates a Metric from an OTLP NumberDataPoint.
func (r *OTLPReceiver) buildMetricFromDataPoint(name, metricType string, dp *metricspb.NumberDataPoint, sessionID, workspace, provider string, resourceAttrs map[string]any) *Metric {
	metric := &Metric{
		Name:               name,
		Type:               metricType,
		SessionID:          sessionID,
		Workspace:          workspace,
		Provider:           provider,
		Timestamp:          time.Unix(0, int64(dp.TimeUnixNano)),
		ResourceAttributes: resourceAttrs,
		Labels:             make(map[string]any),
	}

	// Extract labels from data point attributes
	for _, kv := range dp.Attributes {
		key := kv.Key
		val := anyValueToGo(kv.Value)
		metric.Labels[key] = val

		// Denormalize common labels for efficient queries
		if s, ok := val.(string); ok {
			switch key {
			case "type":
				metric.LabelType = s
			case "tool":
				metric.LabelTool = s
			case "decision":
				metric.LabelDecision = s
			case "language":
				metric.LabelLanguage = s
			case "model":
				metric.LabelModel = s
			}
		}
	}

	// Extract value
	switch v := dp.Value.(type) {
	case *metricspb.NumberDataPoint_AsInt:
		metric.ValueInt = v.AsInt
	case *metricspb.NumberDataPoint_AsDouble:
		metric.ValueFloat = v.AsDouble
	}

	return metric
}
