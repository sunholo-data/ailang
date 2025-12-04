package eval_harness

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Watchdog monitors for orphaned eval processes and kills them
// This is a safety net for cases where process group cleanup fails
type Watchdog struct {
	MaxAge      time.Duration // Kill processes older than this
	CheckPeriod time.Duration // How often to check
	Pattern     string        // Process pattern to match
	KilledCount int           // Number of orphans killed
	Enabled     bool          // Whether watchdog is active
}

// NewWatchdog creates a new Watchdog with the specified settings
func NewWatchdog(maxAge, checkPeriod time.Duration) *Watchdog {
	return &Watchdog{
		MaxAge:      maxAge,
		CheckPeriod: checkPeriod,
		Pattern:     "ailang run.*benchmark/solution.ail",
		Enabled:     true,
	}
}

// Start begins the watchdog monitoring loop
// It runs until the done channel is closed
func (w *Watchdog) Start(done <-chan struct{}) {
	if !w.Enabled {
		return
	}

	ticker := time.NewTicker(w.CheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			w.checkAndKill()
		}
	}
}

// checkAndKill finds and kills orphaned eval processes
func (w *Watchdog) checkAndKill() {
	// Find ailang processes matching pattern with their elapsed time
	// ps -eo pid,etimes,command | grep 'ailang run.*benchmark'
	// etimes = elapsed time in seconds
	cmd := exec.Command("bash", "-c",
		`ps -eo pid,etimes,command | grep -E 'ailang run.*benchmark' | grep -v grep`)
	output, _ := cmd.Output()

	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		elapsedSec, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		elapsed := time.Duration(elapsedSec) * time.Second

		if elapsed > w.MaxAge {
			log.Printf("WATCHDOG: Killing orphan PID %d (running %v, max %v)",
				pid, elapsed, w.MaxAge)
			// Try to kill process group first, fall back to single process
			if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
			w.KilledCount++
		}
	}
}

// Report returns a summary of watchdog activity
func (w *Watchdog) Report() string {
	if !w.Enabled {
		return "Watchdog disabled"
	}
	if w.KilledCount == 0 {
		return "No orphaned processes detected"
	}
	return fmt.Sprintf("Killed %d orphaned process(es)", w.KilledCount)
}

// KillOrphans performs an immediate check and kill of orphaned processes
// This can be called during shutdown for extra cleanup
func (w *Watchdog) KillOrphans() int {
	if !w.Enabled {
		return 0
	}
	beforeCount := w.KilledCount
	w.checkAndKill()
	return w.KilledCount - beforeCount
}
