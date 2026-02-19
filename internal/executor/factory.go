package executor

import (
	"fmt"
	"os"
	"sync"
)

// ExecutorFactory creates and manages executor instances
type ExecutorFactory struct {
	mu        sync.RWMutex
	executors map[string]Executor
	builders  map[string]ExecutorBuilder
	config    *Config
}

// ExecutorBuilder is a function that creates an executor from config
type ExecutorBuilder func(cfg *Config) (Executor, error)

// Config holds configuration for all executors
type Config struct {
	// Default executor to use when none specified
	DefaultExecutor string

	// Claude configuration
	ClaudePath       string   // Path to claude CLI binary
	ClaudeModel      string   // Default Claude model
	ClaudeTools      []string // Allowed tools for Claude
	ClaudePermission string   // Permission mode (bypassPermissions, etc.)

	// Gemini configuration
	GeminiPath  string // Path to gemini CLI binary
	GeminiModel string // Default Gemini model
	GeminiTools []string

	// Common settings
	TimeoutSeconds int
	WorkspaceDir   string
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		DefaultExecutor:  "gemini", // Gemini 3 Flash is the default
		ClaudePath:       "claude",
		ClaudeModel:      "haiku",
		ClaudeTools:      []string{"Bash", "Read", "Write", "Edit", "Grep", "Glob"},
		ClaudePermission: "bypassPermissions",
		GeminiPath:       "gemini",
		GeminiModel:      "gemini-3-flash-preview",
		GeminiTools:      []string{},
		TimeoutSeconds:   300,
		WorkspaceDir:     os.TempDir(),
	}
}

// NewFactory creates a new executor factory
func NewFactory(cfg *Config) *ExecutorFactory {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &ExecutorFactory{
		executors: make(map[string]Executor),
		builders:  make(map[string]ExecutorBuilder),
		config:    cfg,
	}
}

// Register adds an executor builder to the factory
func (f *ExecutorFactory) Register(name string, builder ExecutorBuilder) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.builders[name] = builder
}

// GetExecutor returns an executor by name, creating it if needed
func (f *ExecutorFactory) GetExecutor(name string) (Executor, error) {
	f.mu.RLock()
	if exec, ok := f.executors[name]; ok {
		f.mu.RUnlock()
		return exec, nil
	}
	f.mu.RUnlock()

	// Need to create the executor
	f.mu.Lock()
	defer f.mu.Unlock()

	// Double-check after acquiring write lock
	if exec, ok := f.executors[name]; ok {
		return exec, nil
	}

	builder, ok := f.builders[name]
	if !ok {
		return nil, fmt.Errorf("unknown executor: %s", name)
	}

	exec, err := builder(f.config)
	if err != nil {
		return nil, fmt.Errorf("failed to create executor %s: %w", name, err)
	}

	f.executors[name] = exec
	return exec, nil
}

// GetDefault returns the default executor
func (f *ExecutorFactory) GetDefault() (Executor, error) {
	// Check AILANG_EXECUTOR environment variable first
	if envExecutor := os.Getenv("AILANG_EXECUTOR"); envExecutor != "" {
		return f.GetExecutor(envExecutor)
	}
	return f.GetExecutor(f.config.DefaultExecutor)
}

// ListAvailable returns names of all registered executors
func (f *ExecutorFactory) ListAvailable() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	names := make([]string, 0, len(f.builders))
	for name := range f.builders {
		names = append(names, name)
	}
	return names
}

// UpdateConfig applies a mutation to the factory's config.
// Must be called before GetExecutor() for the executor to use the updated config.
func (f *ExecutorFactory) UpdateConfig(fn func(*Config)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f.config)
}

// Close closes all created executors
func (f *ExecutorFactory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var firstErr error
	for name, exec := range f.executors {
		if err := exec.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to close executor %s: %w", name, err)
		}
	}
	f.executors = make(map[string]Executor)
	return firstErr
}

// Global factory instance
var globalFactory *ExecutorFactory
var globalFactoryOnce sync.Once

// GlobalFactory returns the global executor factory
func GlobalFactory() *ExecutorFactory {
	globalFactoryOnce.Do(func() {
		globalFactory = NewFactory(nil)
	})
	return globalFactory
}

// SetGlobalFactory sets the global factory (for testing)
func SetGlobalFactory(f *ExecutorFactory) {
	globalFactory = f
}
