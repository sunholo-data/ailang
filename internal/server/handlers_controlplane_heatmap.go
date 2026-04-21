package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/observatory"
)

// ============================================================================
// Heatmap API Types
// ============================================================================

// HeatmapCell represents a single day's activity data
type HeatmapCell struct {
	Date        string  `json:"date"` // YYYY-MM-DD
	TaskCount   int     `json:"taskCount"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"successRate"` // 0.0 to 1.0
}

// HeatmapResponse is the response for GET /api/controlplane/heatmap
type HeatmapResponse struct {
	Cells  []HeatmapCell `json:"cells"`
	Totals struct {
		Tasks int     `json:"tasks"`
		Cost  float64 `json:"cost"`
	} `json:"totals"`
}

// HeatmapGridCell is a cell in the grid format response
type HeatmapGridCell struct {
	Date        string  `json:"date"`
	TaskCount   int     `json:"count"`
	Cost        float64 `json:"cost"`
	SuccessRate float64 `json:"successRate"`
	Intensity   float64 `json:"intensity"` // 0.0-1.0 for coloring
	DayOfWeek   int     `json:"dayOfWeek"` // 0=Sunday, 6=Saturday
}

// HeatmapMonthLabel is a month label for the grid header
type HeatmapMonthLabel struct {
	Name      string `json:"name"`      // "Jan", "Feb", etc.
	WeekIndex int    `json:"weekIndex"` // 0-based week column index
}

// HeatmapGridResponse is the grid format response for heatmap
type HeatmapGridResponse struct {
	Weeks       [][]HeatmapGridCell `json:"weeks"`       // weeks[weekIndex][dayIndex]
	MonthLabels []HeatmapMonthLabel `json:"monthLabels"` // month markers
	Totals      struct {
		Tasks int     `json:"tasks"`
		Cost  float64 `json:"cost"`
	} `json:"totals"`
	DateRange struct {
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"dateRange"`
}

// GET /api/controlplane/heatmap - Get daily activity data for heatmap visualization
// Uses Observatory spans for canonical telemetry data with filter support.
// Query params:
//   - days: Number of days to include (default: 90, max: 365)
//   - source_type: Filter by source (eval, coordinator, direct_api, local, other)
//   - provider: Filter by provider (claude, gemini, openai)
//   - model: Filter by model name
//   - start_date: Filter start date (YYYY-MM-DD)
//   - end_date: Filter end date (YYYY-MM-DD)
func (s *Server) handleControlPlaneHeatmap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Parse days parameter (default: 90)
	days := 90
	if daysParam := r.URL.Query().Get("days"); daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	// Parse filters
	filter := parseControlPlaneFilter(r)

	// Initialize response
	var cells []HeatmapCell
	var totalTasks int
	var totalCost float64

	// Load workspace config for reverse mapping (workspace ID -> path patterns)
	wsConfig := coordinator.LoadWorkspacesConfig()

	// Try Observatory backend first (canonical source)
	if s.obsBackend != nil {
		if sqliteBackend, ok := s.obsBackend.(*observatory.SQLiteBackend); ok {
			points, err := sqliteBackend.GetFilteredHeatmapData(ctx, filter, days, wsConfig)
			if err != nil {
				log.Printf("Failed to get observatory heatmap data: %v", err)
			} else {
				// Convert observatory data points to heatmap cells
				for _, point := range points {
					cells = append(cells, HeatmapCell{
						Date:        point.Date,
						TaskCount:   point.SpanCount, // Use span count as activity indicator
						Cost:        point.Cost,
						SuccessRate: point.SuccessRate,
					})
					totalTasks += point.SpanCount
					totalCost += point.Cost
				}
			}
		}
	}

	// If no observatory data, fall back to coordinator store (unfiltered, coordinator only)
	if len(cells) == 0 {
		now := time.Now()
		startDate := now.AddDate(0, 0, -days)

		coordStore := s.getCoordStoreForControlPlane()
		if coordStore != nil {
			coordFilter := &coordinator.TaskFilter{
				Since:     &startDate,
				OrderBy:   "created_at",
				OrderDesc: false,
			}

			tasks, err := coordStore.ListTasks(ctx, coordFilter)
			if err != nil {
				log.Printf("Failed to get tasks for heatmap: %v", err)
			} else {
				// Group tasks by date
				dateMap := make(map[string]struct {
					count     int
					cost      float64
					completed int
				})

				for _, task := range tasks {
					dateStr := task.CreatedAt.Format("2006-01-02")
					entry := dateMap[dateStr]
					entry.count++
					entry.cost += task.Cost
					if task.Status == coordinator.TaskStatusCompleted {
						entry.completed++
					}
					dateMap[dateStr] = entry
					totalTasks++
					totalCost += task.Cost
				}

				// Generate cells for all days in range
				for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
					dateStr := d.Format("2006-01-02")
					entry := dateMap[dateStr]
					successRate := 0.0
					if entry.count > 0 {
						successRate = float64(entry.completed) / float64(entry.count)
					}
					cells = append(cells, HeatmapCell{
						Date:        dateStr,
						TaskCount:   entry.count,
						Cost:        entry.cost,
						SuccessRate: successRate,
					})
				}
			}
		} else {
			// No data source - return empty data for all days
			for d := startDate; !d.After(now); d = d.AddDate(0, 0, 1) {
				cells = append(cells, HeatmapCell{
					Date:        d.Format("2006-01-02"),
					TaskCount:   0,
					Cost:        0,
					SuccessRate: 0,
				})
			}
		}
	}

	// Check format parameter - grid is now the default
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "grid" // Changed default per plan
	}

	if format == "grid" {
		// Use AILANG bridge if enabled, falls back to Go
		gridResponse := GetAILANGBridge().BuildHeatmapGrid(cells, totalTasks, totalCost, days)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(gridResponse); err != nil {
			log.Printf("Failed to encode heatmap grid response: %v", err)
		}
		return
	}

	// Legacy flat format
	response := HeatmapResponse{
		Cells: cells,
	}
	response.Totals.Tasks = totalTasks
	response.Totals.Cost = totalCost

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Failed to encode heatmap response: %v", err)
	}
}

// buildHeatmapGrid builds a week-by-week grid structure from flat cells
func buildHeatmapGrid(cells []HeatmapCell, totalTasks int, totalCost float64, days int) HeatmapGridResponse {
	now := time.Now()
	endDate := now
	startDate := now.AddDate(0, 0, -days)

	// Build a map for O(1) lookup
	cellMap := make(map[string]HeatmapCell)
	maxCount := 0
	for _, cell := range cells {
		cellMap[cell.Date] = cell
		if cell.TaskCount > maxCount {
			maxCount = cell.TaskCount
		}
	}

	// Align to Monday start
	for startDate.Weekday() != time.Monday {
		startDate = startDate.AddDate(0, 0, -1)
	}

	// Build weeks array
	var weeks [][]HeatmapGridCell
	var monthLabels []HeatmapMonthLabel
	lastMonth := -1

	for d := startDate; !d.After(endDate); {
		week := make([]HeatmapGridCell, 7)
		weekIndex := len(weeks)

		for i := 0; i < 7; i++ {
			dateStr := d.Format("2006-01-02")
			cell := cellMap[dateStr]

			// Calculate intensity (0-1) for coloring
			intensity := 0.0
			if maxCount > 0 && cell.TaskCount > 0 {
				intensity = float64(cell.TaskCount) / float64(maxCount)
			}

			week[i] = HeatmapGridCell{
				Date:        dateStr,
				TaskCount:   cell.TaskCount,
				Cost:        cell.Cost,
				SuccessRate: cell.SuccessRate,
				Intensity:   intensity,
				DayOfWeek:   int(d.Weekday()),
			}

			// Track month labels
			month := int(d.Month())
			if month != lastMonth && d.Day() <= 7 {
				monthLabels = append(monthLabels, HeatmapMonthLabel{
					Name:      d.Format("Jan"),
					WeekIndex: weekIndex,
				})
				lastMonth = month
			}

			d = d.AddDate(0, 0, 1)
		}
		weeks = append(weeks, week)
	}

	response := HeatmapGridResponse{
		Weeks:       weeks,
		MonthLabels: monthLabels,
	}
	response.Totals.Tasks = totalTasks
	response.Totals.Cost = totalCost
	response.DateRange.Start = startDate.Format("2006-01-02")
	response.DateRange.End = endDate.Format("2006-01-02")

	return response
}
