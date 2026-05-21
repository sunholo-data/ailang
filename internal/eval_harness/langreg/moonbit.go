package langreg

import (
	"context"
	"fmt"

	promptpkg "github.com/sunholo-data/ailang/internal/prompt"
)

// Compile-time check: moonbitLang implements Language.
var _ Language = (*moonbitLang)(nil)

func init() { Register(&moonbitLang{}) }

type moonbitLang struct{}

func (m *moonbitLang) Name() string        { return "moonbit" }
func (m *moonbitLang) DisplayName() string { return "MoonBit" }
func (m *moonbitLang) FileExt() string     { return ".mbt" }

func (m *moonbitLang) SolutionFilename() string { return "solution.mbt" }

func (m *moonbitLang) PromptTemplatePath() string {
	return ""
}

func (m *moonbitLang) TaskTemplatePath() string {
	return "internal/eval_harness/templates/agent_task_moonbit.txt"
}

func (m *moonbitLang) LoadSyntaxRef(_ string) (string, string, error) {
	content, err := promptpkg.LoadPrompt("moonbit")
	if err != nil {
		return m.DefaultPrompt(), "default", nil
	}
	return content, "moonbit", nil
}

func (m *moonbitLang) DefaultPrompt() string {
	return "You are an expert MoonBit programmer. Write idiomatic MoonBit code; the file will be executed via `moon run solution.mbt`."
}

// NewRunner returns a runner for MoonBit (concrete type: *eval_harness.MoonbitRunner).
func (m *moonbitLang) NewRunner(_ context.Context, spec interface{}, _ string) (interface{}, error) {
	if newMoonbitRunner == nil {
		return nil, fmt.Errorf("langreg: moonbit runner factory not registered; call langreg.SetMoonbitRunnerFactory")
	}
	return newMoonbitRunner(spec), nil
}

// newMoonbitRunner is wired by eval_harness via SetMoonbitRunnerFactory to avoid a circular import.
var newMoonbitRunner func(spec interface{}) interface{}

// SetMoonbitRunnerFactory registers the factory used by NewRunner.
func SetMoonbitRunnerFactory(f func(spec interface{}) interface{}) {
	newMoonbitRunner = f
}
