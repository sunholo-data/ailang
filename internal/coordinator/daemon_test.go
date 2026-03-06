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
	defer daemon.Close() // Required on Windows to release log file handle

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
	defer daemon.Close() // Required on Windows to release log file handle

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
	defer daemon.Close() // Required on Windows to release log file handle

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
	defer daemon.Close() // Required on Windows to release log file handle

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
	defer daemon.Close() // Required on Windows to release log file handle

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
	defer daemon.Close() // Required on Windows to release log file handle

	// Current process should be running
	if !daemon.isProcessRunning(os.Getpid()) {
		t.Error("current process should be detected as running")
	}

	// Non-existent PID should not be running
	if daemon.isProcessRunning(999999) {
		t.Error("non-existent PID should not be detected as running")
	}
}

func TestNewDaemonCloudModeMultiWriter(t *testing.T) {
	// Verify that COORDINATOR_MODE=cloud causes logs to go to stderr too
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	// Capture stderr by replacing it temporarily
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	t.Setenv("COORDINATOR_MODE", "cloud")

	daemon, err := NewDaemon(cfg)
	if err != nil {
		os.Stderr = origStderr
		t.Fatalf("failed to create daemon: %v", err)
	}
	defer daemon.Close()

	// Write a test log message
	daemon.logger.Println("cloud-logging-test-message")

	// Restore stderr and close write end to flush
	w.Close()
	os.Stderr = origStderr

	// Read what was captured from stderr
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	r.Close()
	stderrOutput := string(buf[:n])

	if !containsSubstring(stderrOutput, "cloud-logging-test-message") {
		t.Errorf("expected stderr to contain log message in cloud mode, got: %q", stderrOutput)
	}

	// Also verify the log file got the message
	logContent, err := os.ReadFile(cfg.LogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !containsSubstring(string(logContent), "cloud-logging-test-message") {
		t.Errorf("expected log file to contain message, got: %q", string(logContent))
	}
}

func TestNewDaemonLocalModeNoStderr(t *testing.T) {
	// Verify that local mode does NOT write to stderr
	tmpDir := t.TempDir()
	cfg := &Config{
		PollInterval: time.Second,
		MaxWorktrees: 2,
		LogFile:      filepath.Join(tmpDir, "logs", "coordinator.log"),
		PIDFile:      filepath.Join(tmpDir, "state", "coordinator.pid"),
		StateDir:     filepath.Join(tmpDir, "state"),
	}

	// Capture stderr
	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	// Explicitly unset COORDINATOR_MODE (local mode)
	t.Setenv("COORDINATOR_MODE", "")

	daemon, err := NewDaemon(cfg)
	if err != nil {
		os.Stderr = origStderr
		t.Fatalf("failed to create daemon: %v", err)
	}
	defer daemon.Close()

	daemon.logger.Println("local-mode-test-message")

	w.Close()
	os.Stderr = origStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	r.Close()
	stderrOutput := string(buf[:n])

	if containsSubstring(stderrOutput, "local-mode-test-message") {
		t.Error("local mode should NOT write logs to stderr")
	}

	// But log file should have it
	logContent, err := os.ReadFile(cfg.LogFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !containsSubstring(string(logContent), "local-mode-test-message") {
		t.Error("local mode should write logs to file")
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
	defer daemon.Close() // Required on Windows to release log file handle

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
