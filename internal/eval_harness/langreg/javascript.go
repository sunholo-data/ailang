package langreg

import (
	"context"
	"fmt"
)

// Compile-time check: jsLang implements Language.
var _ Language = (*jsLang)(nil)

func init() { Register(&jsLang{}) }

type jsLang struct{}

func (j *jsLang) Name() string        { return "javascript" }
func (j *jsLang) DisplayName() string { return "JavaScript" }
func (j *jsLang) FileExt() string     { return ".js" }

func (j *jsLang) SolutionFilename() string { return "solution.js" }

func (j *jsLang) PromptTemplatePath() string {
	return ""
}

func (j *jsLang) TaskTemplatePath() string {
	return "internal/eval_harness/templates/agent_task_javascript.txt"
}

func (j *jsLang) LoadSyntaxRef(_ string) (string, string, error) {
	return j.DefaultPrompt(), "default", nil
}

func (j *jsLang) DefaultPrompt() string {
	return "You are an expert JavaScript programmer. Write modern ES2023+ Node.js code using only the standard library (no npm packages)."
}

// NewRunner returns a runner for JavaScript (concrete type: *eval_harness.JSRunner).
// spec must be *eval_harness.BenchmarkSpec (passed as interface{} to avoid circular import).
func (j *jsLang) NewRunner(_ context.Context, spec interface{}, _ string) (interface{}, error) {
	if newJSRunner == nil {
		return nil, fmt.Errorf("langreg: javascript runner factory not registered; call langreg.SetJSRunnerFactory")
	}
	return newJSRunner(spec), nil
}

// newJSRunner is wired by eval_harness via SetJSRunnerFactory to avoid a circular import.
var newJSRunner func(spec interface{}) interface{}

// SetJSRunnerFactory registers the factory used by NewRunner.
// Must be called before any NewRunner call — eval_harness does this in its init.
func SetJSRunnerFactory(f func(spec interface{}) interface{}) {
	newJSRunner = f
}
