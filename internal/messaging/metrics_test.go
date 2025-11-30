package messaging

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestParseMessageExecutionStats(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		want     *MessageExecutionStats
		wantErr  bool
	}{
		{
			name:     "empty metadata",
			metadata: "",
			want:     nil,
			wantErr:  false,
		},
		{
			name:     "no execution_stats field",
			metadata: `{"some_field": "value"}`,
			want:     nil,
			wantErr:  false,
		},
		{
			name: "valid execution stats",
			metadata: `{
				"execution_stats": {
					"duration_ms": 5000,
					"input_tokens": 1000,
					"output_tokens": 500,
					"cost": 0.05,
					"files_created": ["file1.txt", "file2.txt"]
				}
			}`,
			want: &MessageExecutionStats{
				DurationMS:   5000,
				InputTokens:  1000,
				OutputTokens: 500,
				CostCents:    5, // 0.05 * 100
				FilesCreated: []string{"file1.txt", "file2.txt"},
			},
			wantErr: false,
		},
		{
			name: "partial stats",
			metadata: `{
				"execution_stats": {
					"duration_ms": 2000,
					"input_tokens": 500
				}
			}`,
			want: &MessageExecutionStats{
				DurationMS:   2000,
				InputTokens:  500,
				OutputTokens: 0,
				CostCents:    0,
				FilesCreated: nil,
			},
			wantErr: false,
		},
		{
			name:     "invalid JSON",
			metadata: `not valid json`,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseMessageExecutionStats(tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMessageExecutionStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.want == nil && got == nil {
				return
			}

			if tt.want == nil || got == nil {
				t.Errorf("ParseMessageExecutionStats() = %v, want %v", got, tt.want)
				return
			}

			if got.DurationMS != tt.want.DurationMS {
				t.Errorf("DurationMS = %d, want %d", got.DurationMS, tt.want.DurationMS)
			}
			if got.InputTokens != tt.want.InputTokens {
				t.Errorf("InputTokens = %d, want %d", got.InputTokens, tt.want.InputTokens)
			}
			if got.OutputTokens != tt.want.OutputTokens {
				t.Errorf("OutputTokens = %d, want %d", got.OutputTokens, tt.want.OutputTokens)
			}
			if got.CostCents != tt.want.CostCents {
				t.Errorf("CostCents = %d, want %d", got.CostCents, tt.want.CostCents)
			}
			if len(got.FilesCreated) != len(tt.want.FilesCreated) {
				t.Errorf("FilesCreated length = %d, want %d", len(got.FilesCreated), len(tt.want.FilesCreated))
			}
		})
	}
}

func TestRecordMetrics(t *testing.T) {
	// Create temp database
	dbPath := t.TempDir() + "/test_metrics.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record some metrics
	stats := &MessageExecutionStats{
		DurationMS:   1000,
		InputTokens:  500,
		OutputTokens: 200,
		CostCents:    3,
		FilesCreated: []string{"test.txt"},
	}

	err = store.RecordMetrics("thread1", "agent1", stats)
	if err != nil {
		t.Fatalf("RecordMetrics failed: %v", err)
	}

	// Verify global metrics
	globalMetrics, err := store.GetGlobalMetrics()
	if err != nil {
		t.Fatalf("GetGlobalMetrics failed: %v", err)
	}

	if globalMetrics.TotalRuns != 1 {
		t.Errorf("TotalRuns = %d, want 1", globalMetrics.TotalRuns)
	}
	if globalMetrics.TotalTokens != 700 { // 500 + 200
		t.Errorf("TotalTokens = %d, want 700", globalMetrics.TotalTokens)
	}
	if globalMetrics.TotalCost != 0.03 { // 3 cents
		t.Errorf("TotalCost = %f, want 0.03", globalMetrics.TotalCost)
	}
	if globalMetrics.TotalDuration != 1000 {
		t.Errorf("TotalDuration = %d, want 1000", globalMetrics.TotalDuration)
	}
	if globalMetrics.TotalFiles != 1 {
		t.Errorf("TotalFiles = %d, want 1", globalMetrics.TotalFiles)
	}

	// Verify agent metrics
	agentMetrics, err := store.GetAgentMetrics("agent1")
	if err != nil {
		t.Fatalf("GetAgentMetrics failed: %v", err)
	}

	if agentMetrics.TotalRuns != 1 {
		t.Errorf("Agent TotalRuns = %d, want 1", agentMetrics.TotalRuns)
	}

	// Verify thread metrics
	threadMetrics, err := store.GetThreadMetrics("thread1")
	if err != nil {
		t.Fatalf("GetThreadMetrics failed: %v", err)
	}

	if threadMetrics.TotalRuns != 1 {
		t.Errorf("Thread TotalRuns = %d, want 1", threadMetrics.TotalRuns)
	}
}

func TestRecordMetricsAggregation(t *testing.T) {
	// Create temp database
	dbPath := t.TempDir() + "/test_metrics_agg.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record multiple metrics
	stats1 := &MessageExecutionStats{
		DurationMS:   1000,
		InputTokens:  500,
		OutputTokens: 200,
		CostCents:    3,
		FilesCreated: []string{"file1.txt"},
	}

	stats2 := &MessageExecutionStats{
		DurationMS:   2000,
		InputTokens:  1000,
		OutputTokens: 400,
		CostCents:    7,
		FilesCreated: []string{"file2.txt", "file3.txt"},
	}

	err = store.RecordMetrics("thread1", "agent1", stats1)
	if err != nil {
		t.Fatalf("RecordMetrics failed: %v", err)
	}

	err = store.RecordMetrics("thread1", "agent1", stats2)
	if err != nil {
		t.Fatalf("RecordMetrics failed: %v", err)
	}

	// Verify aggregated global metrics
	globalMetrics, err := store.GetGlobalMetrics()
	if err != nil {
		t.Fatalf("GetGlobalMetrics failed: %v", err)
	}

	if globalMetrics.TotalRuns != 2 {
		t.Errorf("TotalRuns = %d, want 2", globalMetrics.TotalRuns)
	}
	if globalMetrics.TotalTokens != 2100 { // (500+200) + (1000+400)
		t.Errorf("TotalTokens = %d, want 2100", globalMetrics.TotalTokens)
	}
	if globalMetrics.TotalCost != 0.10 { // 3 + 7 cents
		t.Errorf("TotalCost = %f, want 0.10", globalMetrics.TotalCost)
	}
	if globalMetrics.TotalDuration != 3000 { // 1000 + 2000
		t.Errorf("TotalDuration = %d, want 3000", globalMetrics.TotalDuration)
	}
	if globalMetrics.TotalFiles != 3 { // 1 + 2
		t.Errorf("TotalFiles = %d, want 3", globalMetrics.TotalFiles)
	}

	// Verify averages
	if globalMetrics.AvgTokens != 1050 { // 2100 / 2
		t.Errorf("AvgTokens = %f, want 1050", globalMetrics.AvgTokens)
	}
	if globalMetrics.AvgCost != 0.05 { // 0.10 / 2
		t.Errorf("AvgCost = %f, want 0.05", globalMetrics.AvgCost)
	}
}

func TestGetMetricsTrends(t *testing.T) {
	// Create temp database
	dbPath := t.TempDir() + "/test_trends.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record some metrics
	stats := &MessageExecutionStats{
		DurationMS:   1000,
		InputTokens:  500,
		OutputTokens: 200,
		CostCents:    3,
	}

	err = store.RecordMetrics("thread1", "agent1", stats)
	if err != nil {
		t.Fatalf("RecordMetrics failed: %v", err)
	}

	// Get hourly trends for global scope
	trends, err := store.GetMetricsTrends("global", "", "hour", 24)
	if err != nil {
		t.Fatalf("GetMetricsTrends failed: %v", err)
	}

	if len(trends) == 0 {
		t.Error("Expected at least one trend data point")
	}

	// Verify trend data point
	if len(trends) > 0 {
		point := trends[len(trends)-1] // Most recent point
		if point["runs"].(int) != 1 {
			t.Errorf("Trend runs = %v, want 1", point["runs"])
		}
		if point["tokens"].(int) != 700 {
			t.Errorf("Trend tokens = %v, want 700", point["tokens"])
		}
	}
}

func TestRecordMetricsNilStats(t *testing.T) {
	// Create temp database
	dbPath := t.TempDir() + "/test_nil.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Recording nil stats should not error
	err = store.RecordMetrics("thread1", "agent1", nil)
	if err != nil {
		t.Errorf("RecordMetrics with nil stats should not error: %v", err)
	}

	// Metrics should still be zero
	metrics, err := store.GetGlobalMetrics()
	if err != nil {
		t.Fatalf("GetGlobalMetrics failed: %v", err)
	}

	if metrics.TotalRuns != 0 {
		t.Errorf("TotalRuns should be 0, got %d", metrics.TotalRuns)
	}
}

func TestAggregatedMetricsJSON(t *testing.T) {
	metrics := &AggregatedMetrics{
		ScopeType:     "global",
		ScopeID:       "",
		TotalRuns:     10,
		TotalTokens:   5000,
		TotalCost:     0.50,
		TotalDuration: 30000,
		TotalFiles:    15,
		AvgTokens:     500,
		AvgCost:       0.05,
		AvgDuration:   3000,
	}

	data, err := json.Marshal(metrics)
	if err != nil {
		t.Fatalf("Failed to marshal metrics: %v", err)
	}

	var decoded AggregatedMetrics
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal metrics: %v", err)
	}

	if decoded.TotalRuns != 10 {
		t.Errorf("TotalRuns = %d, want 10", decoded.TotalRuns)
	}
	if decoded.TotalCost != 0.50 {
		t.Errorf("TotalCost = %f, want 0.50", decoded.TotalCost)
	}
}

func TestMetricsScoping(t *testing.T) {
	// Create temp database
	dbPath := t.TempDir() + "/test_scoping.db"
	store, err := OpenStore(dbPath)
	if err != nil {
		t.Fatalf("Failed to open store: %v", err)
	}
	defer func() {
		store.Close()
		os.Remove(dbPath)
	}()

	// Record metrics for different agents and threads
	stats := &MessageExecutionStats{
		DurationMS:   1000,
		InputTokens:  100,
		OutputTokens: 50,
		CostCents:    1,
	}

	// Agent1, Thread1
	store.RecordMetrics("thread1", "agent1", stats)
	// Agent1, Thread2
	store.RecordMetrics("thread2", "agent1", stats)
	// Agent2, Thread3
	store.RecordMetrics("thread3", "agent2", stats)

	// Global should have 3 runs
	global, _ := store.GetGlobalMetrics()
	if global.TotalRuns != 3 {
		t.Errorf("Global TotalRuns = %d, want 3", global.TotalRuns)
	}

	// Agent1 should have 2 runs
	agent1, _ := store.GetAgentMetrics("agent1")
	if agent1.TotalRuns != 2 {
		t.Errorf("Agent1 TotalRuns = %d, want 2", agent1.TotalRuns)
	}

	// Agent2 should have 1 run
	agent2, _ := store.GetAgentMetrics("agent2")
	if agent2.TotalRuns != 1 {
		t.Errorf("Agent2 TotalRuns = %d, want 1", agent2.TotalRuns)
	}

	// Thread1 should have 1 run
	thread1, _ := store.GetThreadMetrics("thread1")
	if thread1.TotalRuns != 1 {
		t.Errorf("Thread1 TotalRuns = %d, want 1", thread1.TotalRuns)
	}
}

// Benchmark metrics recording
func BenchmarkRecordMetrics(b *testing.B) {
	dbPath := b.TempDir() + "/bench_metrics.db"
	store, _ := OpenStore(dbPath)
	defer store.Close()

	stats := &MessageExecutionStats{
		DurationMS:   1000,
		InputTokens:  500,
		OutputTokens: 200,
		CostCents:    3,
		FilesCreated: []string{"test.txt"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.RecordMetrics("thread1", "agent1", stats)
	}
}

// Helper to get current time truncated to the day start for consistent testing
func dayStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
