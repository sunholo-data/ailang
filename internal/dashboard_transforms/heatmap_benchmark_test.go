package dashboard_transforms

import (
	"testing"
	"time"

	"github.com/sunholo/ailang/internal/embed"
)

// HeatmapCell mirrors the AILANG type
type HeatmapCell struct {
	Date        string  `json:"date"`
	TaskCount   int     `json:"taskCount"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"successRate"`
}

// Go implementations (baseline) - simplified versions

func goCalcIntensity(count, maxCount int) float64 {
	if maxCount == 0 || count == 0 {
		return 0.0
	}
	return float64(count) / float64(maxCount)
}

func goMaxTaskCount(cells []HeatmapCell) int {
	maxCount := 0
	for _, c := range cells {
		if c.TaskCount > maxCount {
			maxCount = c.TaskCount
		}
	}
	return maxCount
}

func goLookupCell(cells []HeatmapCell, targetDate string) HeatmapCell {
	for _, c := range cells {
		if c.Date == targetDate {
			return c
		}
	}
	return HeatmapCell{Date: targetDate}
}

// Test data generators

func generateHeatmapCells(days int) []HeatmapCell {
	cells := make([]HeatmapCell, days)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < days; i++ {
		date := baseTime.AddDate(0, 0, i)
		cells[i] = HeatmapCell{
			Date:        date.Format("2006-01-02"),
			TaskCount:   (i % 10) + 1,
			Cost:        float64(i) * 0.05,
			SuccessRate: 0.8 + float64(i%3)*0.1,
		}
	}
	return cells
}

// Benchmarks - Go baseline

func BenchmarkCalcIntensity_Go(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goCalcIntensity(5, 10)
	}
}

func BenchmarkMaxTaskCount_Go_7(b *testing.B) {
	cells := generateHeatmapCells(7)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goMaxTaskCount(cells)
	}
}

func BenchmarkMaxTaskCount_Go_30(b *testing.B) {
	cells := generateHeatmapCells(30)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goMaxTaskCount(cells)
	}
}

func BenchmarkLookupCell_Go_7(b *testing.B) {
	cells := generateHeatmapCells(7)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = goLookupCell(cells, "2024-01-04")
	}
}

// Benchmarks - AILANG

func BenchmarkCalcIntensity_AILANG(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/heatmap", "calcIntensity", 5, 10)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/heatmap", "calcIntensity", 5, 10)
	}
}

func BenchmarkMaxTaskCount_AILANG_7(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	cells := generateHeatmapCells(7)

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/heatmap", "maxTaskCount", cells)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/heatmap", "maxTaskCount", cells)
	}
}

func BenchmarkMaxTaskCount_AILANG_30(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	cells := generateHeatmapCells(30)

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/heatmap", "maxTaskCount", cells)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/heatmap", "maxTaskCount", cells)
	}
}

func BenchmarkBuildHeatmapGridAt_AILANG_7(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	cells := generateHeatmapCells(7)
	// Jan 7, 2024 00:00:00 UTC as Unix milliseconds
	endTs := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC).UnixMilli()

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/heatmap", "buildHeatmapGridAt",
		cells, 28, 1.40, 7, endTs)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/heatmap", "buildHeatmapGridAt",
			cells, 28, 1.40, 7, endTs)
	}
}

func BenchmarkBuildHeatmapGridAt_AILANG_30(b *testing.B) {
	engine := newBenchEngine(b)
	defer engine.Close()

	cells := generateHeatmapCells(30)
	// Jan 30, 2024 00:00:00 UTC as Unix milliseconds
	endTs := time.Date(2024, 1, 30, 0, 0, 0, 0, time.UTC).UnixMilli()

	// Warm up
	_, err := engine.Call("internal/dashboard_transforms/heatmap", "buildHeatmapGridAt",
		cells, 150, 7.50, 30, endTs)
	if err != nil {
		b.Fatalf("AILANG warm-up failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Call("internal/dashboard_transforms/heatmap", "buildHeatmapGridAt",
			cells, 150, 7.50, 30, endTs)
	}
}

// Functional correctness tests

func TestCalcIntensity_Correctness(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	testCases := []struct {
		count, maxCount int
		expected        float64
	}{
		{0, 10, 0.0},
		{10, 0, 0.0},
		{0, 0, 0.0},
		{5, 10, 0.5},
		{10, 10, 1.0},
	}

	for _, tc := range testCases {
		goResult := goCalcIntensity(tc.count, tc.maxCount)

		ailangResult, err := engine.Call("internal/dashboard_transforms/heatmap", "calcIntensity", tc.count, tc.maxCount)
		if err != nil {
			t.Fatalf("AILANG calcIntensity(%d, %d) failed: %v", tc.count, tc.maxCount, err)
		}
		ailangFloat, _ := embed.ToFloat(ailangResult)

		if goResult != ailangFloat {
			t.Errorf("calcIntensity(%d, %d): Go=%f, AILANG=%f", tc.count, tc.maxCount, goResult, ailangFloat)
		}
	}
}

func TestMaxTaskCount_Correctness(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	cells := generateHeatmapCells(14)
	goResult := goMaxTaskCount(cells)

	ailangResult, err := engine.Call("internal/dashboard_transforms/heatmap", "maxTaskCount", cells)
	if err != nil {
		t.Fatalf("AILANG maxTaskCount failed: %v", err)
	}
	ailangInt, _ := embed.ToInt(ailangResult)

	if goResult != ailangInt {
		t.Errorf("maxTaskCount mismatch: Go=%d, AILANG=%d", goResult, ailangInt)
	}

	t.Logf("maxTaskCount: Go=%d, AILANG=%d", goResult, ailangInt)
}

func TestBuildHeatmapGridAt_Correctness(t *testing.T) {
	engine := newTestEngine(t)
	defer engine.Close()

	cells := []HeatmapCell{
		{Date: "2024-01-15", TaskCount: 5, Cost: 0.25, SuccessRate: 1.0},
		{Date: "2024-01-16", TaskCount: 3, Cost: 0.15, SuccessRate: 0.67},
		{Date: "2024-01-17", TaskCount: 10, Cost: 0.50, SuccessRate: 0.90},
	}

	// Jan 21, 2024 00:00:00 UTC as Unix milliseconds
	endTs := time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC).UnixMilli()

	result, err := engine.Call("internal/dashboard_transforms/heatmap", "buildHeatmapGridAt",
		cells, 28, 1.40, 14, endTs)
	if err != nil {
		t.Fatalf("AILANG buildHeatmapGridAt failed: %v", err)
	}

	// Convert to Go map for inspection
	goResult, err := embed.ToGo(result)
	if err != nil {
		t.Fatalf("Failed to convert result: %v", err)
	}

	// Basic sanity checks on the result structure
	resultMap, ok := goResult.(map[string]interface{})
	if !ok {
		t.Fatalf("Expected map result, got %T", goResult)
	}

	// Check required fields exist
	requiredFields := []string{"weeks", "monthLabels", "totalTasks", "totalCost", "startDate", "endDate"}
	for _, field := range requiredFields {
		if _, ok := resultMap[field]; !ok {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Check totalTasks matches input
	if totalTasks, ok := resultMap["totalTasks"].(int64); ok {
		if totalTasks != 28 {
			t.Errorf("totalTasks mismatch: expected 28, got %d", totalTasks)
		}
	}

	t.Logf("Result: totalTasks=%v, startDate=%v, endDate=%v",
		resultMap["totalTasks"], resultMap["startDate"], resultMap["endDate"])
}
