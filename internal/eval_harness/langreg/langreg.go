// Package langreg is the language registry for the AILANG eval harness.
// Each supported language (python, ailang, javascript, go, ...) registers
// itself via init() → Register(). Callers use Get(name) instead of switch
// statements, so adding a new language is one file + one init() call.
package langreg

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Language is the per-language descriptor used by the eval harness.
// Implement this interface and call Register() from init() to add a new language.
type Language interface {
	// Name returns the canonical registry key ("python", "ailang", "javascript", "go").
	Name() string
	// DisplayName returns the human-readable name for prompt placeholders ("Python 3", "AILANG").
	DisplayName() string
	// FileExt returns the solution file extension (".py", ".ail", ".js", ".go").
	FileExt() string
	// SolutionFilename returns the expected output filename ("solution.py", "solution.ail").
	SolutionFilename() string
	// PromptTemplatePath returns the agent_prompt template file path.
	// Empty string means use the AILANG fallback template.
	PromptTemplatePath() string
	// TaskTemplatePath returns the agent_task template file path.
	TaskTemplatePath() string
	// LoadSyntaxRef loads the teaching / syntax reference prompt.
	// version is the requested prompt version ("" = active/default).
	// Returns (content, versionUsed, error).
	LoadSyntaxRef(version string) (string, string, error)
	// DefaultPrompt returns a minimal fallback prompt when LoadSyntaxRef fails.
	DefaultPrompt() string
	// NewRunner constructs a language-specific runner.
	// Returns a value satisfying eval_harness.LanguageRunner; callers type-assert.
	// spec is *eval_harness.BenchmarkSpec (passed as interface{} to avoid circular import).
	// Returns (runner interface{}, error).
	NewRunner(ctx context.Context, spec interface{}, taskID string) (interface{}, error)
}

var (
	mu       sync.RWMutex
	registry = map[string]Language{}
)

// Register adds a Language implementation to the registry.
// Calling Register with the same name twice is a no-op (idempotent).
// Panics if lang is nil.
func Register(lang Language) {
	if lang == nil {
		panic("langreg: Register called with nil Language")
	}
	mu.Lock()
	defer mu.Unlock()
	if _, exists := registry[lang.Name()]; !exists {
		registry[lang.Name()] = lang
	}
}

// Get returns the Language for the given name, or an error if not registered.
func Get(name string) (Language, error) {
	mu.RLock()
	defer mu.RUnlock()
	lang, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("langreg: language %q not registered (registered: %v)", name, names())
	}
	return lang, nil
}

// MustGet returns the Language for the given name, panicking if not registered.
// Use only in test helpers or init paths where the language is guaranteed registered.
func MustGet(name string) Language {
	lang, err := Get(name)
	if err != nil {
		panic(err)
	}
	return lang
}

// Names returns all registered language names, sorted alphabetically.
func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	return names()
}

// names returns sorted names without acquiring the lock (caller must hold mu).
func names() []string {
	out := make([]string, 0, len(registry))
	for k := range registry {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
