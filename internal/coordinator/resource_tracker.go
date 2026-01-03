// Package coordinator provides resource tracking for task execution.
package coordinator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ResourceMetrics holds current resource usage for a task
type ResourceMetrics struct {
	TaskID    string    `json:"task_id"`
	ThreadID  string    `json:"thread_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`

	// Process metrics
	CPUPercent float64 `json:"cpu_percent"`
	MemoryMB   float64 `json:"memory_mb"`
	PID        int     `json:"pid,omitempty"`

	// Token metrics (accumulated)
	TokensIn  int `json:"tokens_in"`
	TokensOut int `json:"tokens_out"`

	// Cost metrics
	Cost float64 `json:"cost"`

	// Duration
	DurationSec int `json:"duration_sec"`

	// Peak values
	PeakCPU    float64 `json:"peak_cpu"`
	PeakMemory float64 `json:"peak_memory_mb"`
}

// ResourceTracker tracks resource usage for a running task.
// It polls process metrics periodically and accumulates token usage from events.
type ResourceTracker struct {
	taskID    string
	threadID  string
	pid       int
	startTime time.Time

	mu sync.RWMutex

	// Current metrics
	cpuPercent float64
	memoryMB   float64
	tokensIn   int
	tokensOut  int
	cost       float64

	// Peak tracking
	peakCPU    float64
	peakMemory float64

	// Polling
	pollInterval time.Duration
	cancel       context.CancelFunc
	done         chan struct{}

	// Callback for metric updates
	onUpdate func(*ResourceMetrics)
}

// NewResourceTracker creates a new resource tracker for a task
func NewResourceTracker(taskID, threadID string, pid int) *ResourceTracker {
	return &ResourceTracker{
		taskID:       taskID,
		threadID:     threadID,
		pid:          pid,
		startTime:    time.Now(),
		pollInterval: 5 * time.Second,
		done:         make(chan struct{}),
	}
}

// SetPollInterval sets the polling interval for process metrics
func (rt *ResourceTracker) SetPollInterval(interval time.Duration) {
	rt.pollInterval = interval
}

// SetUpdateCallback sets the callback for metric updates
func (rt *ResourceTracker) SetUpdateCallback(callback func(*ResourceMetrics)) {
	rt.mu.Lock()
	rt.onUpdate = callback
	rt.mu.Unlock()
}

// Start begins polling for resource metrics
func (rt *ResourceTracker) Start(ctx context.Context) {
	ctx, rt.cancel = context.WithCancel(ctx)

	go func() {
		ticker := time.NewTicker(rt.pollInterval)
		defer ticker.Stop()
		defer close(rt.done)

		// Initial poll
		rt.pollMetrics()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				rt.pollMetrics()
			}
		}
	}()
}

// Stop stops polling for metrics
func (rt *ResourceTracker) Stop() {
	if rt.cancel != nil {
		rt.cancel()
		<-rt.done
	}
}

// UpdateTokens adds tokens to the accumulated count
func (rt *ResourceTracker) UpdateTokens(inputTokens, outputTokens int) {
	rt.mu.Lock()
	rt.tokensIn += inputTokens
	rt.tokensOut += outputTokens
	rt.mu.Unlock()
}

// UpdateCost updates the accumulated cost
func (rt *ResourceTracker) UpdateCost(cost float64) {
	rt.mu.Lock()
	rt.cost = cost
	rt.mu.Unlock()
}

// SetCost sets the cost (replacement, not additive)
func (rt *ResourceTracker) SetCost(cost float64) {
	rt.mu.Lock()
	rt.cost = cost
	rt.mu.Unlock()
}

// GetMetrics returns the current resource metrics
func (rt *ResourceTracker) GetMetrics() *ResourceMetrics {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	return &ResourceMetrics{
		TaskID:      rt.taskID,
		ThreadID:    rt.threadID,
		Timestamp:   time.Now(),
		CPUPercent:  rt.cpuPercent,
		MemoryMB:    rt.memoryMB,
		PID:         rt.pid,
		TokensIn:    rt.tokensIn,
		TokensOut:   rt.tokensOut,
		Cost:        rt.cost,
		DurationSec: int(time.Since(rt.startTime).Seconds()),
		PeakCPU:     rt.peakCPU,
		PeakMemory:  rt.peakMemory,
	}
}

// pollMetrics fetches current process metrics
func (rt *ResourceTracker) pollMetrics() {
	if rt.pid <= 0 {
		return
	}

	cpu, mem, err := getProcessMetrics(rt.pid)
	if err != nil {
		// Process might have exited, that's okay
		return
	}

	rt.mu.Lock()
	rt.cpuPercent = cpu
	rt.memoryMB = mem

	// Track peak values
	if cpu > rt.peakCPU {
		rt.peakCPU = cpu
	}
	if mem > rt.peakMemory {
		rt.peakMemory = mem
	}

	// Get callback before unlocking
	callback := rt.onUpdate
	rt.mu.Unlock()

	// Call update callback if set
	if callback != nil {
		callback(rt.GetMetrics())
	}
}

// getProcessMetrics fetches CPU and memory for a process using ps
func getProcessMetrics(pid int) (cpuPercent, memoryMB float64, err error) {
	// Use ps to get process stats
	// macOS and Linux compatible command
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "%cpu,%mem,rss")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("ps failed: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return 0, 0, fmt.Errorf("process not found")
	}

	// Parse the output (skip header line)
	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return 0, 0, fmt.Errorf("unexpected ps output format")
	}

	cpu, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		cpu = 0
	}

	// RSS is in KB, convert to MB
	rss, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		rss = 0
	}
	memoryMB = float64(rss) / 1024.0

	return cpu, memoryMB, nil
}

// GetCurrentPID returns the current process PID (useful for self-monitoring)
func GetCurrentPID() int {
	return os.Getpid()
}

// ResourceTrackerRegistry manages multiple resource trackers for concurrent tasks
type ResourceTrackerRegistry struct {
	mu       sync.RWMutex
	trackers map[string]*ResourceTracker
}

// NewResourceTrackerRegistry creates a new registry
func NewResourceTrackerRegistry() *ResourceTrackerRegistry {
	return &ResourceTrackerRegistry{
		trackers: make(map[string]*ResourceTracker),
	}
}

// Register adds a new tracker to the registry
func (r *ResourceTrackerRegistry) Register(taskID string, tracker *ResourceTracker) {
	r.mu.Lock()
	r.trackers[taskID] = tracker
	r.mu.Unlock()
}

// Unregister removes a tracker from the registry
func (r *ResourceTrackerRegistry) Unregister(taskID string) {
	r.mu.Lock()
	if tracker, ok := r.trackers[taskID]; ok {
		tracker.Stop()
		delete(r.trackers, taskID)
	}
	r.mu.Unlock()
}

// Get returns a tracker by task ID
func (r *ResourceTrackerRegistry) Get(taskID string) *ResourceTracker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.trackers[taskID]
}

// GetAllMetrics returns metrics for all active trackers
func (r *ResourceTrackerRegistry) GetAllMetrics() []*ResourceMetrics {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := make([]*ResourceMetrics, 0, len(r.trackers))
	for _, tracker := range r.trackers {
		metrics = append(metrics, tracker.GetMetrics())
	}
	return metrics
}

// StopAll stops all trackers
func (r *ResourceTrackerRegistry) StopAll() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, tracker := range r.trackers {
		tracker.Stop()
	}
	r.trackers = make(map[string]*ResourceTracker)
}
