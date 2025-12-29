package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// Config holds daemon configuration
type Config struct {
	PollInterval time.Duration // How often to check for new messages
	MaxWorktrees int           // Maximum concurrent worktrees
	LogFile      string        // Path to log file
	PIDFile      string        // Path to PID file
	StateDir     string        // Directory for state files
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".ailang", "state")
	logsDir := filepath.Join(homeDir, ".ailang", "logs")

	return &Config{
		PollInterval: 30 * time.Second,
		MaxWorktrees: 3,
		LogFile:      filepath.Join(logsDir, "coordinator.log"),
		PIDFile:      filepath.Join(stateDir, "coordinator.pid"),
		StateDir:     stateDir,
	}
}

// Status represents the daemon's current state
type Status struct {
	Running   bool      `json:"running"`
	PID       int       `json:"pid,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
	TasksRun  int       `json:"tasks_run"`
}

// Daemon is the coordinator daemon
type Daemon struct {
	config    *Config
	logger    *log.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	startedAt time.Time
	tasksRun  int
}

// NewDaemon creates a new daemon instance
func NewDaemon(config *Config) (*Daemon, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(config.LogFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(config.PIDFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(logFile, "[coordinator] ", log.LstdFlags|log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())

	return &Daemon{
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

// Start starts the daemon
func (d *Daemon) Start() error {
	// Check if already running
	if status, _ := d.Status(); status.Running {
		return fmt.Errorf("daemon already running with PID %d", status.PID)
	}

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	d.startedAt = time.Now()
	d.logger.Printf("Daemon starting (PID: %d)", os.Getpid())

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		d.logger.Printf("Received signal: %v, shutting down...", sig)
		d.cancel()
	}()

	// Run main loop
	return d.Run()
}

// Run is the main daemon loop
func (d *Daemon) Run() error {
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	d.logger.Println("Daemon running, polling for tasks...")

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("Daemon shutting down gracefully")
			d.cleanup()
			return nil
		case <-ticker.C:
			// Poll for new tasks
			d.logger.Println("Checking for new tasks...")
			// Task polling will be implemented in watcher.go
		}
	}
}

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

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// cleanup removes PID file and performs other cleanup
func (d *Daemon) cleanup() {
	if err := os.Remove(d.config.PIDFile); err != nil && !os.IsNotExist(err) {
		d.logger.Printf("Failed to remove PID file: %v", err)
	}
	d.logger.Println("Cleanup complete")
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
