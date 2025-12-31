package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Stop stops a running daemon
func (d *Daemon) Stop() error {
	status, err := d.Status()
	if err != nil {
		return err
	}

	if !status.Running {
		return fmt.Errorf("daemon is not running")
	}

	// Send SIGTERM to the process
	process, err := os.FindProcess(status.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}

	// Wait for process to exit (with timeout)
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if !d.isProcessRunning(status.PID) {
			return nil
		}
	}

	// Force kill if still running
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	return nil
}

// Close releases resources held by the daemon.
// This must be called when done with the daemon to release file handles.
// On Windows, this is required before the log file can be deleted.
func (d *Daemon) Close() error {
	if d.logFile != nil {
		if err := d.logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
		d.logFile = nil
	}
	return nil
}

// Status returns the current daemon status
func (d *Daemon) Status() (*Status, error) {
	status := &Status{
		Running:  false,
		TasksRun: d.tasksRun,
	}

	// Read PID file
	pid, err := d.readPIDFile()
	if err != nil {
		// No PID file means not running
		return status, nil
	}

	// Check if process is running
	if d.isProcessRunning(pid) {
		status.Running = true
		status.PID = pid

		// Try to get start time from process (simplified - just use current time minus uptime estimate)
		if !d.startedAt.IsZero() {
			status.StartedAt = d.startedAt
			status.Uptime = time.Since(d.startedAt).Round(time.Second).String()
		}
	} else {
		// Stale PID file, clean up
		_ = os.Remove(d.config.PIDFile)
	}

	// Get actual task stats from database (in-memory counter resets on restart)
	store, err := NewSQLiteStore(filepath.Join(d.config.StateDir, "coordinator.db"))
	if err == nil {
		defer store.Close()
		stats, err := store.GetTaskStats(context.Background())
		if err == nil {
			status.TasksRun = stats.CompletedTasks
			status.PendingTasks = stats.PendingTasks
			status.RunningTasks = stats.RunningTasks
			status.PendingApprovals = stats.PendingApprovals
			status.FailedTasks = stats.FailedTasks
			status.TotalCost = stats.TotalCost
			status.TotalTokens = stats.TotalTokens
		}
	}

	return status, nil
}

// StatusJSON returns status as JSON string
func (d *Daemon) StatusJSON() (string, error) {
	status, err := d.Status()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// writePIDFile writes the current process ID to the PID file
func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(d.config.PIDFile, []byte(strconv.Itoa(pid)), 0644)
}

// readPIDFile reads the PID from the PID file
func (d *Daemon) readPIDFile() (int, error) {
	data, err := os.ReadFile(d.config.PIDFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %w", err)
	}

	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running
func (d *Daemon) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, FindProcess succeeds even for non-existent PIDs.
		// We check by trying to terminate with signal 0 via Windows API
		// or by running tasklist. The simplest approach is to use tasklist.
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		// If process exists, output contains the process info
		// If not, it contains "INFO: No tasks are running..."
		return !strings.Contains(string(output), "INFO:")
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// cleanup removes PID file and performs other cleanup
func (d *Daemon) cleanup() {
	// Stop GitHub approval watcher (M-COORD-GITHUB-AUTO-ROUTING)
	if d.approvalWatcher != nil {
		d.approvalWatcher.Stop()
		d.logger.Println("GitHub approval watcher stopped")
	}

	// Mark agent as idle in dashboard
	if err := d.unregisterAgent(); err != nil {
		d.logger.Printf("Failed to unregister agent: %v", err)
	}

	if err := os.Remove(d.config.PIDFile); err != nil && !os.IsNotExist(err) {
		d.logger.Printf("Failed to remove PID file: %v", err)
	}
	d.logger.Println("Cleanup complete")
}

// registerAgent registers the coordinator as an agent in the collaboration hub
func (d *Daemon) registerAgent() error {
	if d.msgStore == nil {
		return nil // No store available
	}

	db := d.msgStore.DB()
	if db == nil {
		return nil
	}

	now := time.Now().Unix()

	// Register/update agent
	_, err := db.Exec(`
		INSERT INTO agents (id, label, status, created_at, updated_at, last_active_at, config_json)
		VALUES ('coordinator', 'Coordinator Daemon', 'running', ?, ?, ?, '{}')
		ON CONFLICT(id) DO UPDATE SET status='running', updated_at=?, last_active_at=?
	`, now, now, now, now, now)

	if err != nil {
		return err
	}

	// Create instance history entry
	d.instanceID = fmt.Sprintf("coord_%d", now)
	_, err = db.Exec(`
		INSERT INTO instance_history (id, agent_id, instance_id, started_at)
		VALUES (?, 'coordinator', ?, ?)
	`, d.instanceID, d.instanceID, now)

	if err != nil {
		d.logger.Printf("Warning: Failed to record instance history: %v", err)
	}

	d.logger.Println("Registered as agent in collaboration hub")
	return nil
}

// unregisterAgent marks the coordinator as idle in the collaboration hub
func (d *Daemon) unregisterAgent() error {
	if d.msgStore == nil {
		return nil
	}

	db := d.msgStore.DB()
	if db == nil {
		return nil
	}

	now := time.Now().Unix()

	// Update agent status
	_, err := db.Exec(`
		UPDATE agents SET status='idle', updated_at=? WHERE id='coordinator'
	`, now)

	// Complete instance history entry
	if d.instanceID != "" {
		_, histErr := db.Exec(`
			UPDATE instance_history SET ended_at=?, exit_code=0
			WHERE id=?
		`, now, d.instanceID)
		if histErr != nil {
			d.logger.Printf("Warning: Failed to update instance history: %v", histErr)
		}
	}

	return err
}

// IncrementTasksRun increments the tasks run counter
func (d *Daemon) IncrementTasksRun() {
	d.tasksRun++
}

// GetLogger returns the daemon's logger
func (d *Daemon) GetLogger() *log.Logger {
	return d.logger
}

// GetContext returns the daemon's context
func (d *Daemon) GetContext() context.Context {
	return d.ctx
}

// GetActiveTaskMetrics returns metrics for all currently running tasks
func (d *Daemon) GetActiveTaskMetrics() []*ResourceMetrics {
	if d.resourceRegistry == nil {
		return nil
	}
	return d.resourceRegistry.GetAllMetrics()
}

// GetResourceRegistry returns the resource tracker registry for external access
func (d *Daemon) GetResourceRegistry() *ResourceTrackerRegistry {
	return d.resourceRegistry
}
