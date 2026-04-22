package langreg

import (
	"context"
	"fmt"

	promptpkg "github.com/sunholo-data/ailang/internal/prompt"
)

// Compile-time check: pythonLang implements Language.
var _ Language = (*pythonLang)(nil)

func init() { Register(&pythonLang{}) }

type pythonLang struct{}

func (p *pythonLang) Name() string        { return "python" }
func (p *pythonLang) DisplayName() string { return "Python 3" }
func (p *pythonLang) FileExt() string     { return ".py" }

func (p *pythonLang) SolutionFilename() string { return "solution.py" }

func (p *pythonLang) PromptTemplatePath() string {
	return "internal/eval_harness/templates/agent_prompt_python.txt"
}

func (p *pythonLang) TaskTemplatePath() string {
	return "internal/eval_harness/templates/agent_task_python.txt"
}

func (p *pythonLang) LoadSyntaxRef(_ string) (string, string, error) {
	// Python prompt is stored as "python" version in versions.json.
	// version arg is ignored — Python has no versioned prompts.
	content, err := promptpkg.LoadPrompt("python")
	if err != nil {
		return p.DefaultPrompt(), "default", nil
	}
	return content, "python", nil
}

func (p *pythonLang) DefaultPrompt() string {
	return "You are an expert Python programmer. Write clean, idiomatic Python code."
}

// NewRunner returns a runner for Python (concrete type: *eval_harness.PythonRunner).
// spec must be *eval_harness.BenchmarkSpec (passed as interface{} to avoid circular import).
func (p *pythonLang) NewRunner(_ context.Context, spec interface{}, _ string) (interface{}, error) {
	if newPythonRunner == nil {
		return nil, fmt.Errorf("langreg: python runner factory not registered; call langreg.SetPythonRunnerFactory")
	}
	return newPythonRunner(spec), nil
}

// newPythonRunner is wired by eval_harness via SetPythonRunnerFactory to avoid a circular import.
var newPythonRunner func(spec interface{}) interface{}

// SetPythonRunnerFactory registers the factory used by NewRunner.
// Must be called before any NewRunner call — eval_harness does this in its init.
func SetPythonRunnerFactory(f func(spec interface{}) interface{}) {
	newPythonRunner = f
}
