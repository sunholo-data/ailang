package langreg

import (
	"context"
	"fmt"

	promptpkg "github.com/sunholo-data/ailang/internal/prompt"
)

func init() { Register(&ailangLang{}) }

type ailangLang struct{}

func (a *ailangLang) Name() string        { return "ailang" }
func (a *ailangLang) DisplayName() string { return "AILANG" }
func (a *ailangLang) FileExt() string     { return ".ail" }

func (a *ailangLang) SolutionFilename() string { return "solution.ail" }

func (a *ailangLang) PromptTemplatePath() string {
	// AILANG uses the default template (empty string = use fallback).
	return ""
}

func (a *ailangLang) TaskTemplatePath() string {
	return "internal/eval_harness/templates/agent_task_ailang.txt"
}

// LoadSyntaxRef loads the versioned AILANG teaching prompt.
// version="" loads the active version; a specific version ID loads that version.
// Returns (content, versionUsed, error).
func (a *ailangLang) LoadSyntaxRef(version string) (string, string, error) {
	content, versionUsed, err := promptpkg.LoadPromptWithVersion(version)
	if err != nil {
		return a.DefaultPrompt(), "default", nil
	}
	return content, versionUsed, nil
}

func (a *ailangLang) DefaultPrompt() string {
	// Single source of truth: internal/prompt.
	content, err := promptpkg.LoadPrompt("")
	if err == nil {
		return content
	}
	return "You are writing code in AILANG, a functional programming language."
}

// NewRunner returns a runner for AILANG (concrete type: *eval_harness.AILANGRunner).
// spec must be *eval_harness.BenchmarkSpec (passed as interface{} to avoid circular import).
func (a *ailangLang) NewRunner(ctx context.Context, spec interface{}, taskID string) (interface{}, error) {
	if newAILANGRunner == nil {
		return nil, fmt.Errorf("langreg: ailang runner factory not registered; call langreg.SetAILANGRunnerFactory")
	}
	return newAILANGRunner(ctx, spec, taskID), nil
}

// newAILANGRunner is wired by eval_harness via SetAILANGRunnerFactory to avoid a circular import.
var newAILANGRunner func(ctx context.Context, spec interface{}, taskID string) interface{}

// SetAILANGRunnerFactory registers the factory used by NewRunner.
// Must be called before any NewRunner call — eval_harness does this in its init.
func SetAILANGRunnerFactory(f func(ctx context.Context, spec interface{}, taskID string) interface{}) {
	newAILANGRunner = f
}
