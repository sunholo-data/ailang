# Sprint Plan: M-EVAL-LANG-JSGO

**Sprint 2 of M-EVAL-EXPAND** — JavaScript + Go language support in the eval harness

**Status**: Planned  
**Target**: v0.15.1  
**Duration**: 1.5 days  
**Risk**: Low  
**Design doc**: [M-EVAL-EXPAND](../v0_13_0/m-eval-expand-harnesses-languages.md)  
**Depends on**: M-EVAL-LANGREG ✅ (langreg package live, all switch-lang dispatch replaced)

---

## Goal

Add JavaScript (Node.js) and Go as first-class eval languages so that
`ailang eval-suite --benchmarks fizzbuzz --langs javascript,go` runs end-to-end
and produces `RunResult` records comparable to Python and AILANG runs.

After this sprint:
- `langreg.Get("javascript")` and `langreg.Get("go")` return valid descriptors
- `JSRunner` and `GoRunner` execute code in temp dirs using `node` and `go run`
- 10 starter benchmarks declare 4 languages: `["python", "ailang", "javascript", "go"]`
- 20 reference solutions under `examples/reference/<bench>/main.{js,go}` produce
  byte-identical output to each benchmark's `expected_stdout`

---

## Toolchain

- Node.js: `node v25.5.0` (available in PATH)
- Go: `go 1.25.0` (available in PATH)

---

## Milestones

### M1 — JS/Go langreg descriptors + task templates (~160 LOC)

**Files to create:**
- `internal/eval_harness/langreg/javascript.go`
- `internal/eval_harness/langreg/go.go`
- `internal/eval_harness/templates/agent_task_javascript.txt`
- `internal/eval_harness/templates/agent_task_go.txt`

**Pattern**: mirror `python.go` and `ailang.go`. Key values:

| Field | JavaScript | Go |
|-------|-----------|-----|
| Name | `"javascript"` | `"go"` |
| DisplayName | `"JavaScript"` | `"Go"` |
| FileExt | `".js"` | `".go"` |
| SolutionFilename | `"solution.js"` | `"solution.go"` |
| DefaultPrompt | modern ES2023+ Node.js | idiomatic Go stdlib |

**⚠️ Fix required in existing tests**: `langreg_test.go` uses `"javascript"` as the
expected-unknown language. Change to `"typescript"` before registering JS.

**Acceptance criteria:**
- [ ] `langreg.Get("javascript")` returns non-nil, no error
- [ ] `langreg.Get("go")` returns non-nil, no error
- [ ] `langreg.Names()` returns `["ailang", "go", "javascript", "python"]` (sorted)
- [ ] `TestRegistry_UnknownLanguage` updated to use `"typescript"` as unknown
- [ ] `go test ./internal/eval_harness/langreg/...` passes

---

### M2 — JSRunner + GoRunner + factory wiring (~250 LOC)

**Files to modify:**
- `internal/eval_harness/runner.go` (add JSRunner, GoRunner) — check line count first; split to `runner_js_go.go` if > 800 lines
- `internal/eval_harness/langreg/javascript.go` — add `SetJSRunnerFactory`
- `internal/eval_harness/langreg/go.go` — add `SetGoRunnerFactory`
- `internal/eval_harness/langreg_wire.go` — wire both factories in `init()`

**JSRunner**: write to `solution.js`, execute `node solution.js`  
**GoRunner**: write to `solution.go` (prepend `package main` if absent), execute `go run solution.go`

Both runners follow the `PythonRunner` pattern exactly:
- `os.MkdirTemp` for isolation
- Write `spec.InputFiles` if present
- `cmd.Dir = tmpDir` for relative file access
- `SetProcessGroup(cmd)` + `KillProcessGroup` on timeout
- `LimitedWriter` for stdout/stderr

**Acceptance criteria:**
- [ ] `GetRunnerWithContext(ctx, "javascript", spec, "")` → `*JSRunner`
- [ ] `GetRunnerWithContext(ctx, "go", spec, "")` → `*GoRunner`
- [ ] `JSRunner.Run("console.log(42)", 10s)` → stdout `"42\n"`, exitCode 0
- [ ] `GoRunner.Run("package main\nimport \"fmt\"\nfunc main(){fmt.Println(42)}", 10s)` → `"42\n"`
- [ ] Unit tests for both runners in `runner_test.go`
- [ ] `go build ./internal/eval_harness/...` passes
- [ ] `make lint` passes

---

### M3 — Benchmark YAML language updates (~10 LOC)

Add `"javascript"` and `"go"` to `languages:` in all 10 starter benchmarks:

```yaml
languages: ["python", "ailang", "javascript", "go"]
```

**Benchmarks**: fizzbuzz, recursion_fibonacci, graph_bfs, binary_tree_sum,
balanced_parens, csv_to_json_converter, expression_evaluator, gcd_lcm,
fold_reduce, higher_order_functions

**Acceptance criteria:**
- [ ] All 10 YAMLs have `languages: ["python", "ailang", "javascript", "go"]`
- [ ] `ailang eval-suite --benchmarks fizzbuzz --langs javascript --dry-run` exits 0

---

### M4 — Reference solutions (~400 LOC, 20 files)

Create `examples/reference/<bench>/main.{js,go}` for all 10 benchmarks.

**Rules:**
- Stdlib only — no npm packages, no Go modules (just `package main`)
- Each file must produce byte-identical output to the benchmark's `expected_stdout`
- Go files: `package main` + only stdlib imports
- JS files: Node.js built-ins only (`fs`, `path`, no `require('npm-pkg')`)

**Expected outputs:**

| Benchmark | Expected stdout |
|-----------|----------------|
| fizzbuzz | FizzBuzz 1–100 (100 lines) |
| recursion_fibonacci | `6765` |
| graph_bfs | `1\n2\n3\n4\n5` |
| binary_tree_sum | `31` |
| balanced_parens | `true\nfalse\nfalse\ntrue` |
| csv_to_json_converter | `Converted 3 valid rows to users.json` |
| expression_evaluator | `49` |
| gcd_lcm | `6\n144` |
| fold_reduce | `Sum: 30\nProduct: 3840\nMax: 11` |
| higher_order_functions | `Result: 14` |

**Acceptance criteria:**
- [ ] `node examples/reference/fizzbuzz/main.js` produces exact expected output
- [ ] `go run examples/reference/fizzbuzz/main.go` produces exact expected output
- [ ] Same for all 10 benchmarks × 2 languages = 20 verifications

---

### M5 — Verification + CHANGELOG (~80 LOC)

**Test**: Add `TestReferenceSolutions_JS` and `TestReferenceSolutions_Go` in
`internal/eval_harness/reference_solutions_test.go` that run all 20 reference
solution files and compare output to benchmark YAML `expected_stdout`.

**CHANGELOG**: Entry under `[Unreleased] v0.15.1`.

**Acceptance criteria:**
- [ ] `go test ./internal/eval_harness/... -run TestReferencesolutions` passes
- [ ] CHANGELOG updated
- [ ] `make test` passes
- [ ] `make lint` passes

---

## Success Metrics

- `langreg.Names()` = `["ailang", "go", "javascript", "python"]`
- `ailang eval-suite --benchmarks fizzbuzz --langs javascript --dry-run` routes correctly
- `ailang eval-suite --benchmarks fizzbuzz --langs go --dry-run` routes correctly
- All 20 reference solutions pass verification
- `make ci` passes (excluding pre-existing covdata issue)

## Non-Goals

- TypeScript, Rust, or other languages (registry makes them easy later)
- npm or Go module dependencies in reference solutions
- Porting all 50+ benchmarks (10 starter benchmarks only)
- Full agent eval run against cloud models (that's smoke testing, not this sprint)
