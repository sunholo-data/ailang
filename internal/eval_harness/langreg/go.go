package langreg

import (
	"context"
	"fmt"
)

// Compile-time check: goLang implements Language.
var _ Language = (*goLang)(nil)

func init() { Register(&goLang{}) }

type goLang struct{}

func (g *goLang) Name() string        { return "go" }
func (g *goLang) DisplayName() string { return "Go" }
func (g *goLang) FileExt() string     { return ".go" }

func (g *goLang) SolutionFilename() string { return "solution.go" }

func (g *goLang) PromptTemplatePath() string {
	return ""
}

func (g *goLang) TaskTemplatePath() string {
	return "internal/eval_harness/templates/agent_task_go.txt"
}

func (g *goLang) LoadSyntaxRef(_ string) (string, string, error) {
	return g.DefaultPrompt(), "default", nil
}

func (g *goLang) DefaultPrompt() string {
	return "You are an expert Go programmer. Write idiomatic Go code using only the standard library (no external modules)."
}

// NewRunner returns a runner for Go (concrete type: *eval_harness.GoRunner).
// spec must be *eval_harness.BenchmarkSpec (passed as interface{} to avoid circular import).
func (g *goLang) NewRunner(_ context.Context, spec interface{}, _ string) (interface{}, error) {
	if newGoRunner == nil {
		return nil, fmt.Errorf("langreg: go runner factory not registered; call langreg.SetGoRunnerFactory")
	}
	return newGoRunner(spec), nil
}

// newGoRunner is wired by eval_harness via SetGoRunnerFactory to avoid a circular import.
var newGoRunner func(spec interface{}) interface{}

// SetGoRunnerFactory registers the factory used by NewRunner.
// Must be called before any NewRunner call — eval_harness does this in its init.
func SetGoRunnerFactory(f func(spec interface{}) interface{}) {
	newGoRunner = f
}
