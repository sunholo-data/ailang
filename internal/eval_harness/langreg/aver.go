package langreg

import (
	"context"
	"fmt"

	promptpkg "github.com/sunholo-data/ailang/internal/prompt"
)

// Compile-time check: averLang implements Language.
var _ Language = (*averLang)(nil)

func init() { Register(&averLang{}) }

type averLang struct{}

func (a *averLang) Name() string        { return "aver" }
func (a *averLang) DisplayName() string { return "Aver" }
func (a *averLang) FileExt() string     { return ".av" }

func (a *averLang) SolutionFilename() string { return "solution.av" }

func (a *averLang) PromptTemplatePath() string {
	return ""
}

func (a *averLang) TaskTemplatePath() string {
	return "internal/eval_harness/templates/agent_task_aver.txt"
}

func (a *averLang) LoadSyntaxRef(_ string) (string, string, error) {
	content, err := promptpkg.LoadPrompt("aver")
	if err != nil {
		return a.DefaultPrompt(), "default", nil
	}
	return content, "aver", nil
}

func (a *averLang) DefaultPrompt() string {
	return "You are an expert Aver programmer. Write idiomatic Aver code; the file will be executed via `aver run solution.av`."
}

// NewRunner returns a runner for Aver (concrete type: *eval_harness.AverRunner).
func (a *averLang) NewRunner(_ context.Context, spec interface{}, _ string) (interface{}, error) {
	if newAverRunner == nil {
		return nil, fmt.Errorf("langreg: aver runner factory not registered; call langreg.SetAverRunnerFactory")
	}
	return newAverRunner(spec), nil
}

// newAverRunner is wired by eval_harness via SetAverRunnerFactory to avoid a circular import.
var newAverRunner func(spec interface{}) interface{}

// SetAverRunnerFactory registers the factory used by NewRunner.
func SetAverRunnerFactory(f func(spec interface{}) interface{}) {
	newAverRunner = f
}
