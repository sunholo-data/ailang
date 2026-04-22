# Sprint Plan: M-EVAL-LANGREG (Language Registry)

**Status**: Ready for execution
**Target**: v0.15.0
**Estimated**: 2 days
**Priority**: P1 — prerequisite for JS/Go language support (M-EVAL-LANG-JSGO)
**Source design doc**: [m-eval-expand-harnesses-languages.md](../v0_13_0/m-eval-expand-harnesses-languages.md) §Sprint 1

---

## Context

M-EXEC-EXPAND (complete) delivered Codex + opencode executors — Sprints 3 and 4 of M-EVAL-EXPAND.
This sprint delivers Sprint 1: the language registry that replaces 9 `switch lang` dispatch sites
across 4 files with a single `langreg.Get(lang)` call. Once landed, Sprint 2 (JS/Go) is a
mechanical exercise: add `javascript.go` + `go.go`, extend benchmark `languages:` arrays, done.

**Pre-existing foundation in `runner.go`:** `LanguageRunner` interface already exists:
```go
type LanguageRunner interface {
    Run(code string, timeout time.Duration) (*RunResult, error)
    Language() string
}
```
The langreg `Language` interface wraps this and adds prompt/template/filename dispatch.

---

## Dispatch Sites to Replace

9 sites across 4 files:

| File | Line | What it dispatches |
|------|------|--------------------|
| `agent_prompt.go` | 15 | Template path selection (`agent_prompt_python.txt` vs `agent_prompt.txt`) |
| `agent_prompt.go` | 109 | `LoadActiveSyntaxReference` → ailang prompt vs python prompt loader |
| `agent_prompt.go` | 342 | `LoadPromptAndVersion` → ailang active/versioned prompt vs python prompt |
| `agent_prompt.go` | 383 | `LoadTaskPromptTemplate` → `agent_task_python.txt` vs `agent_task_ailang.txt` |
| `agent_runner.go` | 524 | `getSolutionFilename` → `solution.py` vs `solution.ail` |
| `runner.go` | 485 | `GetRunnerWithContext` → `NewPythonRunnerWithSpec` vs `NewAILANGRunnerWithTask` |
| `spec.go` | 148 | `PromptForLanguage` — `if lang == "ailang"` special path |
| `spec.go` | 194 | `PromptForLanguage` — `langName` display string (`"Python 3"` vs `"AILANG"`) |
| `spec.go` | 207 | `getDefaultPrompt` — language-specific default prompt |

---

## Milestones

### M1 — langreg Package (Day 1)

**Goal**: Define the `Language` interface and register `python` + `ailang` concretes.

**New files:**
- `internal/eval_harness/langreg/langreg.go` (~120 LOC) — interface + registry
- `internal/eval_harness/langreg/python.go` (~100 LOC) — Python impl
- `internal/eval_harness/langreg/ailang.go` (~130 LOC) — AILANG impl (special prompt loading)
- `internal/eval_harness/langreg/langreg_test.go` (~160 LOC) — registry + contract tests

**Interface design:**

```go
package langreg

// Language is the per-language descriptor used by the eval harness.
// Add a new language by implementing this interface and calling Register().
type Language interface {
    // Name returns the canonical lang key ("python", "ailang", "javascript", "go").
    Name() string
    // DisplayName returns the human-readable name for prompts ("Python 3", "AILANG").
    DisplayName() string
    // FileExt returns the solution file extension (".py", ".ail", ".js", ".go").
    FileExt() string
    // SolutionFilename returns the expected output file ("solution.py", "solution.ail").
    SolutionFilename() string
    // PromptTemplatePath returns the agent_prompt template file path.
    // Empty string means use the fallback AILANG template.
    PromptTemplatePath() string
    // TaskTemplatePath returns the agent_task template file path.
    TaskTemplatePath() string
    // LoadSyntaxRef loads the teaching / syntax reference prompt for this language.
    LoadSyntaxRef(version string) (content string, versionUsed string, err error)
    // DefaultPrompt returns a minimal fallback prompt when LoadSyntaxRef fails.
    DefaultPrompt() string
    // NewRunner constructs a LanguageRunner (from eval_harness.LanguageRunner) for this language.
    NewRunner(ctx context.Context, spec interface{}, taskID string) (interface{}, error)
}

// Get returns the Language for the given key, or an error if not registered.
func Get(name string) (Language, error)

// Register registers a Language implementation. Called from init() in each lang file.
func Register(lang Language)

// Names returns all registered language keys, sorted.
func Names() []string
```

**Acceptance criteria:**
- `langreg.Get("python")` and `langreg.Get("ailang")` return non-nil without error
- `langreg.Get("javascript")` returns descriptive error (not registered yet)
- `langreg.Names()` returns `["ailang", "python"]` (sorted)
- `TestLanguageContract` verifies all registered langs implement every method non-panicking
- `TestRegistry_Idempotent` verifies double-Register is a no-op (not a panic)
- `make test ./internal/eval_harness/langreg/...` passes
- `make lint` clean

**Estimated LOC**: 510 (new files)

---

### M2 — Replace Dispatch Sites (Day 2, morning)

**Goal**: Replace all 9 switch sites with `langreg.Get(lang)` calls. Zero behavioral change.

**Approach per site:**

| Site | Before | After |
|------|--------|-------|
| `agent_prompt.go:15` | `switch language { case "python": templatePath = ... }` | `lang, _ := langreg.Get(language); templatePath = lang.PromptTemplatePath()` |
| `agent_prompt.go:109` | `switch language { case "ailang": return loadAILANGPrompt() ... }` | `lang, _ := langreg.Get(language); return lang.LoadSyntaxRef("")` |
| `agent_prompt.go:342` | `switch language { case "ailang": prompt, versionUsed = ... }` | `lang, err := langreg.Get(language); prompt, versionUsed, err = lang.LoadSyntaxRef(promptVersion)` |
| `agent_prompt.go:383` | `switch language { case "python": templatePath = agent_task_python.txt }` | `lang, _ := langreg.Get(language); templatePath = lang.TaskTemplatePath()` |
| `agent_runner.go:524` | `switch language { case "python": return "solution.py" }` | `lang, _ := langreg.Get(language); return lang.SolutionFilename()` |
| `runner.go:485` | `switch lang { case "python": return NewPythonRunnerWithSpec... }` | `l, _ := langreg.Get(lang); return l.NewRunner(ctx, spec, taskID)` |
| `spec.go:148` | `if lang == "ailang" { ... } else { ... }` | `l, _ := langreg.Get(lang); if l.RequiresTeachingPrompt() { ... }` |
| `spec.go:194` | `switch lang { case "python": langName = "Python 3" }` | `l, _ := langreg.Get(lang); langName = l.DisplayName()` |
| `spec.go:207` | `switch lang { case "python": return "..." }` | `l, _ := langreg.Get(lang); return l.DefaultPrompt()` |

**Acceptance criteria:**
- All 9 sites replaced; `grep -rn "switch.*lang\|case.*\"python\"\|case.*\"ailang\"" internal/eval_harness/` returns 0 results (excluding langreg/ itself and test files that verify string names)
- `make test` passes (identical output to pre-refactor for python + ailang)
- `make lint` clean
- `ailang eval-suite --models claude-sonnet-4-6 --benchmarks fizzbuzz --langs python --dry-run` works
- `ailang eval-suite --models claude-sonnet-4-6 --benchmarks fizzbuzz --langs ailang --dry-run` works

**Estimated LOC**: ~-80 net (removal > addition at call sites)

---

### M3 — Regression Verification + CHANGELOG (Day 2, afternoon)

**Goal**: Prove zero behavioral regression; update docs.

**Steps:**
1. Run `ailang eval-suite --agent --models claude-sonnet-4-6 --benchmarks fizzbuzz --langs python,ailang` — verify results match pre-refactor baseline
2. Add `langreg_regression_test.go` (~80 LOC): table-driven test asserting `GetRunnerWithContext("python")` and `("ailang")` return runners whose `Language()` matches and `Run()` produces same output bytes as before
3. Update `CHANGELOG.md` under v0.15.0 — `langreg` language registry section
4. Add `internal/eval_harness/langreg/README.md` — how to add a new language (6-step recipe, mirroring EXECUTOR_SHAPE.md's value)

**Acceptance criteria:**
- `TestLangregRegression_Python` and `TestLangregRegression_AILANG` pass (output bytes identical)
- `CHANGELOG.md` updated
- `internal/eval_harness/langreg/README.md` documents the "new language in one file" recipe
- `make ci` passes (tests + lint; coverage-badge known pre-existing issue)

**Estimated LOC**: ~130 (regression tests + README)

---

## Summary

| Milestone | Description | LOC | Day |
|-----------|-------------|-----|-----|
| M1 | langreg package: interface + python + ailang + tests | +510 | 1 |
| M2 | Replace 9 dispatch sites in 4 files | -80 net | 2 AM |
| M3 | Regression tests + CHANGELOG + README | +130 | 2 PM |
| **Total** | | **~560 net** | **2 days** |

**Velocity basis**: M-EXEC-EXPAND shipped ~2150 LOC in ~1 session; this sprint is more surgical
(touching existing harness files), budgeted conservatively at ~300 LOC/day. 560 LOC = 2 days with buffer.

---

## What This Unlocks

After this sprint, M-EVAL-LANG-JSGO (Sprint 2) is mechanical:
```
internal/eval_harness/langreg/javascript.go   (~100 LOC)
internal/eval_harness/langreg/go.go           (~100 LOC)
```
Plus extending 10 benchmark `languages:` arrays and adding reference solutions. No harness dispatch files change.

---

## Files to Create/Modify

**New:**
- `internal/eval_harness/langreg/langreg.go`
- `internal/eval_harness/langreg/python.go`
- `internal/eval_harness/langreg/ailang.go`
- `internal/eval_harness/langreg/langreg_test.go`
- `internal/eval_harness/langreg/langreg_regression_test.go`
- `internal/eval_harness/langreg/README.md`

**Modified (dispatch sites only — no logic changes):**
- `internal/eval_harness/agent_prompt.go` — 4 switches → langreg calls
- `internal/eval_harness/agent_runner.go` — 1 switch → langreg call
- `internal/eval_harness/runner.go` — 1 switch → langreg call
- `internal/eval_harness/spec.go` — 3 locations → langreg calls
- `CHANGELOG.md` — v0.15.0 entry

**Not modified (proves clean interface):**
- `internal/eval_harness/models.yml` — no changes
- `cmd/ailang/eval_suite.go` — no changes
- Any coordinator or executor code — no changes
