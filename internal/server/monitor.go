package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// ProcessStats contains runtime statistics for a monitored process
type ProcessStats struct {
	InstanceID  string    `json:"instance_id"`
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"started_at"`
	DurationSec int       `json:"duration_sec"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemoryMB    float64   `json:"memory_mb"`
	Status      string    `json:"status"` // running, completed, failed

	// Source information - where the process was started from
	Source    string     `json:"source"`               // "ui", "eval", "cli", "agent"
	Command   string     `json:"command,omitempty"`    // Short command description
	FullCmd   string     `json:"full_cmd,omitempty"`   // Full command line (for debugging)
	StoppedAt *time.Time `json:"stopped_at,omitempty"` // When the process stopped

	// Telemetry from Claude sessions (populated when available)
	Turns     int     `json:"turns,omitempty"`
	TokensIn  int     `json:"tokens_in,omitempty"`
	TokensOut int     `json:"tokens_out,omitempty"`
	Cost      float64 `json:"cost,omitempty"`
}

// MonitorResponse is the API response for /api/monitor
type MonitorResponse struct {
	Timestamp time.Time      `json:"timestamp"`
	Processes []ProcessStats `json:"processes"`
	History   []ProcessStats `json:"history,omitempty"` // Recently completed/failed processes
	Summary   MonitorSummary `json:"summary"`
}

// MonitorSummary provides aggregate stats
type MonitorSummary struct {
	TotalProcesses int     `json:"total_processes"`
	TotalCPU       float64 `json:"total_cpu_percent"`
	TotalMemoryMB  float64 `json:"total_memory_mb"`
	TotalCost      float64 `json:"total_cost"`
	WarningCount   int     `json:"warning_count"` // Processes exceeding thresholds
}

// handleMonitor returns current process statistics
// GET /api/monitor
func (s *Server) handleMonitor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.collectProcessStats()
	summary := s.calculateSummary(stats)

	// Get recent history (processes that completed in last 5 minutes)
	history := s.getRecentHistory()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(MonitorResponse{
		Timestamp: time.Now(),
		Processes: stats,
		History:   history,
		Summary:   summary,
	}); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

// getRecentHistory returns processes that completed within the last 5 minutes
func (s *Server) getRecentHistory() []ProcessStats {
	s.processHistoryMu.RLock()
	defer s.processHistoryMu.RUnlock()

	cutoff := time.Now().Add(-5 * time.Minute)
	var recent []ProcessStats

	for _, proc := range s.processHistory {
		if proc.StoppedAt != nil && proc.StoppedAt.After(cutoff) {
			recent = append(recent, proc)
		}
	}

	return recent
}

// AddToHistory adds a completed/failed process to history
func (s *Server) AddToHistory(proc ProcessStats) {
	s.processHistoryMu.Lock()
	defer s.processHistoryMu.Unlock()

	now := time.Now()
	proc.StoppedAt = &now

	s.processHistory = append(s.processHistory, proc)

	// Clean up old entries (keep last 50 or last 10 minutes)
	s.cleanupHistoryLocked()
}

// cleanupHistoryLocked removes old history entries (must be called with lock held)
func (s *Server) cleanupHistoryLocked() {
	cutoff := time.Now().Add(-10 * time.Minute)
	maxEntries := 50

	// Remove entries older than cutoff
	newHistory := make([]ProcessStats, 0, len(s.processHistory))
	for _, proc := range s.processHistory {
		if proc.StoppedAt != nil && proc.StoppedAt.After(cutoff) {
			newHistory = append(newHistory, proc)
		}
	}

	// Keep only last maxEntries
	if len(newHistory) > maxEntries {
		newHistory = newHistory[len(newHistory)-maxEntries:]
	}

	s.processHistory = newHistory
}

// collectProcessStats gathers CPU/memory for all tracked and discovered processes
func (s *Server) collectProcessStats() []ProcessStats {
	s.agentsMu.RLock()
	defer s.agentsMu.RUnlock()

	trackedPIDs := make(map[int]bool)
	currentlySeenPIDs := make(map[int]bool)
	var stats []ProcessStats

	// First, add tracked agents (spawned via UI)
	for _, agent := range s.agents {
		trackedPIDs[agent.PID] = true
		currentlySeenPIDs[agent.PID] = true
		stat := ProcessStats{
			InstanceID:  agent.InstanceID,
			PID:         agent.PID,
			StartedAt:   agent.StartedAt,
			DurationSec: int(time.Since(agent.StartedAt).Seconds()),
			Status:      "running",
			Source:      "ui",                 // UI-spawned agents
			Command:     "ailang-coordinator", // Coordinator daemon
		}

		// Check if process is still running
		if agent.cmd.ProcessState != nil {
			if agent.cmd.ProcessState.Exited() {
				if agent.cmd.ProcessState.Success() {
					stat.Status = "completed"
				} else {
					stat.Status = "failed"
				}
			}
		}

		// Get CPU/memory from ps command (works on macOS and Linux)
		if stat.Status == "running" {
			cpu, mem := getTotalResourceUsage(agent.PID) // Include child processes
			stat.CPUPercent = cpu
			stat.MemoryMB = mem
		}

		stats = append(stats, stat)
	}

	// Then, scan for ALL ailang processes (eval suite, ailang run, etc.)
	// This catches processes not spawned via the UI
	orphaned := findAILangProcesses()
	for _, proc := range orphaned {
		currentlySeenPIDs[proc.PID] = true

		// Skip if already tracked
		if trackedPIDs[proc.PID] {
			continue
		}
		// Get total resources including child processes
		cpu, mem := getTotalResourceUsage(proc.PID)
		proc.CPUPercent = cpu
		proc.MemoryMB = mem

		// Check for external telemetry (from self-reporting processes like eval suite)
		if extTelem := s.GetExternalTelemetry(proc.PID); extTelem != nil {
			proc.InstanceID = extTelem.InstanceID
			proc.Turns = extTelem.Turns
			proc.TokensIn = extTelem.TokensIn
			proc.TokensOut = extTelem.TokensOut
			proc.Cost = extTelem.Cost
			if extTelem.Status != "" {
				proc.Status = extTelem.Status
			}
		}

		stats = append(stats, proc)
	}

	// Detect disappeared processes and add them to history
	s.detectDisappearedProcesses(currentlySeenPIDs)

	// Update previously seen with current state
	s.updatePreviouslySeen(stats)

	return stats
}

// detectDisappearedProcesses finds processes that were seen before but are now gone
func (s *Server) detectDisappearedProcesses(currentlySeenPIDs map[int]bool) {
	// First, collect disappeared processes while holding the lock
	var disappeared []ProcessStats
	var disappearedPIDs []int

	s.previouslySeenMu.Lock()
	for pid, proc := range s.previouslySeen {
		if !currentlySeenPIDs[pid] {
			proc.Status = "completed"
			disappeared = append(disappeared, proc)
			disappearedPIDs = append(disappearedPIDs, pid)
		}
	}
	// Remove from previouslySeen
	for _, pid := range disappearedPIDs {
		delete(s.previouslySeen, pid)
	}
	s.previouslySeenMu.Unlock()

	// Then, add to history without holding previouslySeenMu (avoids deadlock)
	for _, proc := range disappeared {
		s.AddToHistory(proc)
	}
}

// updatePreviouslySeen updates the map with currently running processes
func (s *Server) updatePreviouslySeen(stats []ProcessStats) {
	s.previouslySeenMu.Lock()
	defer s.previouslySeenMu.Unlock()

	for _, stat := range stats {
		if stat.Status == "running" {
			s.previouslySeen[stat.PID] = stat
		}
	}
}

// calculateSummary computes aggregate statistics
func (s *Server) calculateSummary(stats []ProcessStats) MonitorSummary {
	summary := MonitorSummary{}

	for _, stat := range stats {
		if stat.Status == "running" {
			summary.TotalProcesses++
			summary.TotalCPU += stat.CPUPercent
			summary.TotalMemoryMB += stat.MemoryMB
			summary.TotalCost += stat.Cost

			// Warning thresholds
			if stat.CPUPercent > 80 || stat.DurationSec > 300 {
				summary.WarningCount++
			}
		}
	}

	return summary
}

// getProcessResourceUsage uses ps command to get CPU and memory usage
// Returns (cpu_percent, memory_mb)
func getProcessResourceUsage(pid int) (float64, float64) {
	// Use ps command which works on both macOS and Linux
	// Format: %cpu, rss (resident set size in KB)
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu=,rss=")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0 // Process may have exited
	}

	// Parse output: "  4.5  12345"
	fields := strings.Fields(string(output))
	if len(fields) < 2 {
		return 0, 0
	}

	cpu, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		cpu = 0
	}

	// RSS is in KB, convert to MB
	rssKB, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		rssKB = 0
	}
	memMB := rssKB / 1024.0

	return cpu, memMB
}

// getProcessTree finds all child processes of a given PID
// Useful for tracking subprocess resource usage
func getProcessTree(pid int) []int {
	// Use pgrep to find child processes
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var children []int
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if childPID, err := strconv.Atoi(line); err == nil {
			children = append(children, childPID)
			// Recursively get grandchildren
			children = append(children, getProcessTree(childPID)...)
		}
	}
	return children
}

// getTotalResourceUsage gets combined resource usage for a process and all its children
func getTotalResourceUsage(pid int) (float64, float64) {
	cpu, mem := getProcessResourceUsage(pid)

	// Add child process resources
	for _, childPID := range getProcessTree(pid) {
		childCPU, childMem := getProcessResourceUsage(childPID)
		cpu += childCPU
		mem += childMem
	}

	return cpu, mem
}

// findAILangProcesses scans for any ailang-related processes not tracked by the server
// This catches orphaned processes from crashed sessions AND eval suite processes
func findAILangProcesses() []ProcessStats {
	// Find processes matching "ailang" pattern (includes ailang, coordinator, etc.)
	cmd := exec.Command("pgrep", "-f", "ailang")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	var stats []ProcessStats
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}

		// Get full command line for this PID (helps identify what it's doing)
		cmdOutput, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
		if err != nil {
			continue
		}
		cmdLine := strings.TrimSpace(string(cmdOutput))

		// Skip the server process itself and this pgrep command
		if strings.Contains(cmdLine, "serve") || strings.Contains(cmdLine, "pgrep") {
			continue
		}

		// Skip non-ailang processes (pgrep -f can match too broadly)
		if !strings.Contains(cmdLine, "ailang") {
			continue
		}

		// Determine process type and source from command line
		instanceID := fmt.Sprintf("process_%d", pid)
		source := "cli"
		command := "ailang"
		status := "running"

		if strings.Contains(cmdLine, "eval-suite") {
			instanceID = fmt.Sprintf("eval_%d", pid)
			source = "eval"
			command = "eval-suite"
		} else if strings.Contains(cmdLine, "eval-benchmark") {
			instanceID = fmt.Sprintf("eval_%d", pid)
			source = "eval"
			command = "eval-benchmark"
		} else if strings.Contains(cmdLine, "ailang run") {
			instanceID = fmt.Sprintf("run_%d", pid)
			source = "cli"
			// Extract filename if possible
			command = extractRunCommand(cmdLine)
		} else if strings.Contains(cmdLine, "coordinator") {
			instanceID = fmt.Sprintf("coordinator_%d", pid)
			source = "coordinator"
			command = "coordinator"
		} else if strings.Contains(cmdLine, "ailang repl") {
			instanceID = fmt.Sprintf("repl_%d", pid)
			source = "cli"
			command = "repl"
		} else if strings.Contains(cmdLine, "ailang check") {
			instanceID = fmt.Sprintf("check_%d", pid)
			source = "cli"
			command = "check"
		}

		// Get process start time using ps
		startOutput, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "etime=").Output()
		durationSec := -1 // Unknown
		if err == nil {
			durationSec = parseElapsedTime(strings.TrimSpace(string(startOutput)))
		}

		cpu, mem := getProcessResourceUsage(pid)
		stats = append(stats, ProcessStats{
			InstanceID:  instanceID,
			PID:         pid,
			CPUPercent:  cpu,
			MemoryMB:    mem,
			Status:      status,
			DurationSec: durationSec,
			Source:      source,
			Command:     command,
			FullCmd:     cmdLine,
		})
	}

	return stats
}

// extractRunCommand extracts a short description from an "ailang run" command
func extractRunCommand(cmdLine string) string {
	parts := strings.Split(cmdLine, " ")
	for i, p := range parts {
		if p == "run" && i+1 < len(parts) {
			// Skip flags and get filename
			for j := i + 1; j < len(parts); j++ {
				if !strings.HasPrefix(parts[j], "-") && !strings.HasPrefix(parts[j], "--") {
					// Get just the filename, not full path
					filename := parts[j]
					if idx := strings.LastIndex(filename, "/"); idx >= 0 {
						filename = filename[idx+1:]
					}
					return "run " + filename
				}
			}
			break
		}
	}
	return "run"
}

// parseElapsedTime parses ps etime format (e.g., "12:34" for mm:ss, "1-12:34:56" for days)
func parseElapsedTime(etime string) int {
	// Format can be: [[DD-]HH:]MM:SS
	etime = strings.TrimSpace(etime)
	if etime == "" {
		return -1
	}

	var days, hours, minutes, seconds int

	// Check for days
	if strings.Contains(etime, "-") {
		parts := strings.SplitN(etime, "-", 2)
		days, _ = strconv.Atoi(parts[0])
		etime = parts[1]
	}

	// Split remaining by colons
	parts := strings.Split(etime, ":")
	switch len(parts) {
	case 3: // HH:MM:SS
		hours, _ = strconv.Atoi(parts[0])
		minutes, _ = strconv.Atoi(parts[1])
		seconds, _ = strconv.Atoi(parts[2])
	case 2: // MM:SS
		minutes, _ = strconv.Atoi(parts[0])
		seconds, _ = strconv.Atoi(parts[1])
	case 1: // SS
		seconds, _ = strconv.Atoi(parts[0])
	}

	return days*86400 + hours*3600 + minutes*60 + seconds
}
