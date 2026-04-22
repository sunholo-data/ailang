# langreg — Language Registry for the AILANG Eval Harness

`langreg` is the single registration point for languages supported by
`ailang eval-suite`. Each language registers itself via `init()`, so
adding a new language requires **one new file** and zero edits to existing
switch statements.

## Adding a new language in 5 steps

### 1. Create `<language>.go` in this package

```go
package langreg

import (
    "context"
    "fmt"
)

var _ Language = (*jsLang)(nil)

func init() { Register(&jsLang{}) }

type jsLang struct{}

func (j *jsLang) Name() string        { return "javascript" }
func (j *jsLang) DisplayName() string { return "JavaScript" }
func (j *jsLang) FileExt() string     { return ".js" }

func (j *jsLang) SolutionFilename() string  { return "solution.js" }
func (j *jsLang) PromptTemplatePath() string { return "internal/eval_harness/templates/agent_prompt_js.txt" }
func (j *jsLang) TaskTemplatePath() string   { return "internal/eval_harness/templates/agent_task_js.txt" }

func (j *jsLang) LoadSyntaxRef(_ string) (string, string, error) {
    return j.DefaultPrompt(), "default", nil
}

func (j *jsLang) DefaultPrompt() string {
    return "You are an expert JavaScript programmer. Write clean, modern ES2023+ code."
}

func (j *jsLang) NewRunner(_ context.Context, spec interface{}, _ string) (interface{}, error) {
    if newJSRunner == nil {
        return nil, fmt.Errorf("langreg: javascript runner factory not registered")
    }
    return newJSRunner(spec), nil
}

var newJSRunner func(spec interface{}) interface{}

func SetJSRunnerFactory(f func(spec interface{}) interface{}) { newJSRunner = f }
```

### 2. Add template files

Create `internal/eval_harness/templates/agent_prompt_js.txt` and
`agent_task_js.txt` following the existing Python/AILANG templates.

### 3. Implement a runner

Create `internal/eval_harness/js_runner.go` (or similar) implementing
`LanguageRunner`. It must satisfy:
- `Run(ctx, taskDir, solutionPath string) RunResult`
- `Language() string` returning `"javascript"`

### 4. Wire the factory

In `internal/eval_harness/langreg_wire.go`, add inside `init()`:

```go
langreg.SetJSRunnerFactory(func(spec interface{}) interface{} {
    if bs, ok := spec.(*BenchmarkSpec); ok {
        return NewJSRunnerWithSpec(bs)
    }
    return NewJSRunner()
})
```

### 5. Add the blank import

In `internal/eval_harness/provider_executor.go` (or wherever language
registrations are gathered), add:

```go
_ "github.com/sunholo-data/ailang/internal/eval_harness/langreg"
```

(Only needed if the package isn't already transitively imported.)

---

## Architecture notes

- **Circular import avoided**: `langreg` lives inside `eval_harness/langreg/`
  but cannot import `eval_harness`. Runner factories are injected via
  `Set*RunnerFactory` functions, which `langreg_wire.go` calls from its `init()`.
- **`NewRunner` returns `interface{}`**: callers in `eval_harness` type-assert
  to `LanguageRunner`. This keeps `langreg` free of `eval_harness` types.
- **`LoadSyntaxRef` returns the resolved version ID**: use
  `promptpkg.LoadPromptWithVersion` (not `LoadPrompt`) so the actual active
  version key is returned, not the string `"active"`.
