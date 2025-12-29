package coordinator

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.PollInterval != 30*time.Second {
		t.Errorf("expected poll interval 30s, got %v", cfg.PollInterval)
	}

	if cfg.MaxWorktrees != 3 {
		t.Errorf("expected max worktrees 3, got %d", cfg.MaxWorktrees)
	}

	if cfg.LogFile == "" {
		t.Error("expected log file path to be set")
	}

	if cfg.PIDFile == "" {
		t.Error("expected PID file path to be set")
	}
}

func TestNewDaemon(t *testing.T) {
	// Use temp directory for tests
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	if daemon == nil {
		t.Fatal("daemon is nil")
	}

	if daemon.logger == nil {
		t.Error("logger is nil")
	}

	if daemon.ctx == nil {
		t.Error("context is nil")
	}
}

func TestDaemonPIDFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	// Write PID file
	err = daemon.writePIDFile()
	if err != nil {
		t.Fatalf("failed to write PID file: %v", err)
	}

	// Verify PID file exists
	if _, err := os.Stat(cfg.PIDFile); os.IsNotExist(err) {
		t.Error("PID file was not created")
	}

	// Read PID file
	pid, err := daemon.readPIDFile()
	if err != nil {
		t.Fatalf("failed to read PID file: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("expected PID %d, got %d", os.Getpid(), pid)
	}
}

func TestDaemonStatus(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	// Initially not running
	status, err := daemon.Status()
	if err != nil {
		t.Fatalf("failed to get status: %v", err)
	}

	if status.Running {
		t.Error("daemon should not be running initially")
	}
}

func TestDaemonStatusJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	jsonStr, err := daemon.StatusJSON()
	if err != nil {
		t.Fatalf("failed to get status JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("status JSON is empty")
	}

	// Should contain expected fields
	if !containsSubstring(jsonStr, "running") {
		t.Error("status JSON missing 'running' field")
	}
}

func TestDaemonCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	// Write PID file
	err = daemon.writePIDFile()
	if err != nil {
		t.Fatalf("failed to write PID file: %v", err)
	}

	// Cleanup should remove PID file
	daemon.cleanup()

	if _, err := os.Stat(cfg.PIDFile); !os.IsNotExist(err) {
		t.Error("PID file should be removed after cleanup")
	}
}

func TestIsProcessRunning(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	// Current process should be running
	if !daemon.isProcessRunning(os.Getpid()) {
		t.Error("current process should be detected as running")
	}

	// Non-existent PID should not be running
	if daemon.isProcessRunning(999999) {
		t.Error("non-existent PID should not be detected as running")
	}
}

func TestDaemonIncrementTasksRun(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	daemon, err := NewDaemon(cfg)
	if err != nil {
		t.Fatalf("failed to create daemon: %v", err)
	}

	if daemon.tasksRun != 0 {
		t.Error("tasks run should start at 0")
	}

	daemon.IncrementTasksRun()
	if daemon.tasksRun != 1 {
		t.Error("tasks run should be 1 after increment")
	}

	daemon.IncrementTasksRun()
	daemon.IncrementTasksRun()
	if daemon.tasksRun != 3 {
		t.Error("tasks run should be 3 after three increments")
	}
}
