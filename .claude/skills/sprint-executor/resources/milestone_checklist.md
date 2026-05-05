# Milestone Execution Checklist

## Pre-Implementation
- [ ] Mark milestone as `in_progress` in TodoWrite
- [ ] Review milestone goals and acceptance criteria
- [ ] Identify files to create/modify
- [ ] Verify estimated LOC matches plan

## Implementation
- [ ] Write implementation code
- [ ] Follow design patterns from sprint plan
- [ ] Add inline comments for complex logic
- [ ] Keep functions small and focused (<50 lines)

## Testing
- [ ] Create/update test files (*_test.go)
- [ ] Comprehensive coverage (all acceptance criteria)
- [ ] Include edge cases and error conditions
- [ ] Test both success and failure paths
- [ ] Use realistic-complexity inputs (not just toy examples)
  - XML: include namespaces, duplicate prefixes, nested elements
  - Lists: test with >10 elements, not just 2-3
  - Maps: verify ordering doesn't affect output

## Determinism Verification (AILANG Axiom)
- [ ] For builtins marked `IsPure: true`: run tests with `-count=20`
  - `go test ./internal/builtins/ -run TestMyPureFunc -count=20`
  - A pure function MUST return identical output for identical input, every time
  - Single-pass tests can hide Go map iteration nondeterminism
- [ ] For any Go code iterating maps in pure function paths:
  - **Red flag**: `for k, v := range someMap` in code called by `IsPure: true` builtins
  - Prefer deterministic alternatives: sorted keys, explicit priority (e.g., check default first)
  - Or add multi-iteration tests that detect ordering sensitivity
- [ ] If test uses random/time-based data: verify with fixed seed or multiple runs

## Quality Verification
- [ ] Run tests: `make test` - MUST PASS
- [ ] Run linting: `make lint` - MUST PASS
- [ ] Check formatting: `make fmt-check` or run `make fmt`
- [ ] If any fail, fix immediately before proceeding

## Regression Surface Coverage (REQUIRED for shared compilation infrastructure)

**Triggered when** the milestone modifies any of:
- `internal/parser/`, `internal/lexer/`, `internal/ast/`
- `internal/types/`, `internal/elaborate/`, `internal/iface/`
- `internal/codegen/`, `internal/eval/`, `internal/vm/`
- `internal/effects/` (effect-row algebra)
- `cmd/ailang/exec.go` and other compilation entry points

**Why**: changes to shared compilation infrastructure can silently break programs that nobody on the team currently writes but external consumers do. The `M-PARSER-REFINEMENT-LOOKAHEAD` (v0.15.2) regression — `T{not LABEL}` shadowing `func ... -> bool { not f(x) }` — slipped through 3 release cycles because internal idiomatic AILANG never used the conflicting form. Cost: ~14 mis-parses in motoko_agent's `rpc.ail` + sibling files, caught only when migrating off the v0.13.0 fork.

### Hard gates — NONE of these are advisory; all must pass before milestone completion

- [ ] **Conflict Surface section was filled in the design doc.** If the design doc lacks this section but the milestone touches the listed paths, STOP and request the design author update the doc before proceeding. Do not improvise.
- [ ] **All "Programs that MUST still work" fixtures from the design doc pass.** For each fixture, write a regression test that pins the parse/typecheck/eval output. Failures here are blockers, not test churn.
- [ ] **`make verify-examples` passes.** Re-runs every `examples/runnable/*.ail` and asserts no parse/type/eval regression. Pre-existing skips are acceptable; net-new failures are not.
- [ ] **External consumer corpus check** (when CI infrastructure exists): if `make external-consumer-smoke` exists, run it. It clones pinned external repos (e.g. motoko_agent at a known commit) and runs `ailang check`. Net-new failures block the milestone.
- [ ] **AST differential** (when goldens exist): regenerate AST goldens for the example tree (`make ast-diff` or equivalent). ANY diff requires explicit justification in the commit message — "intentional change because <reason>" or "regression test added at <path>". No silent diffs.

### Discipline rules

- **Don't normalize regression-test failures**: if a previously-clean program newly fails, the default action is "this is a regression — fix the implementation," not "update the test." Only update the test if the design doc explicitly listed this as an intentional incompatibility.
- **Don't skip the gate by claiming the change is small**: a 1-line parser change has the same regression surface as a 100-line one. The trigger is "what code is touched," not "how much."
- **Surface the gate failure to the user**: if the gate trips on a previously-passing program, pause and report to the user with the program name + the new error. Don't silently weaken the milestone scope to bypass.

### Failure modes this gate catches

| Failure | Without gate | With gate |
|---------|--------------|-----------|
| New parser path claims tokens that another path was using | Lands; breaks external users; caught months later | Caught at milestone checkpoint via verify-examples or AST diff |
| Type-rule change makes a previously-typeable program fail | Lands; breaks downstream cache/iface; caught via user reports | Caught when corpus differential surfaces the change |
| Codegen change alters AST shape in unexpected ways | Lands; sibling features that depended on the old shape break | Caught when AST goldens diff |
| Effect-row algebra widens the unification rules | Lands; effect inference quietly accepts previously-rejected programs | Caught when example outputs diverge from goldens |

## Documentation
- [ ] Update CHANGELOG.md with milestone completion:
  - What was implemented
  - LOC counts (implementation + tests)
  - Key design decisions
  - Files modified/created
- [ ] Update sprint plan with completion status (✅)
- [ ] Add metrics (actual LOC vs estimated, time spent)

## Example Files (REQUIRED for new language features)
- [ ] Create example file in `examples/runnable/<feature>.ail`
- [ ] Add entry to `examples/manifest.json`:
  - `path`: relative path from examples/
  - `status`: "working"
  - `tags`: relevant tags (e.g., "records", "types")
  - `description`: brief description of what example demonstrates
- [ ] Update statistics in manifest (increment total, working counts)
- [ ] Verify searchable: `ailang examples search "<feature>"`
- [ ] Verify viewable: `ailang examples show <name>`

## Sprint JSON Update (CRITICAL)
**⚠️ The checkpoint script will remind you, but DON'T SKIP THIS!**
- [ ] Update `.ailang/state/sprints/sprint_<id>.json`:
  - Set `passes: true` (or `false` if failing)
  - Set `completed: "<ISO timestamp>"`
  - Add `notes: "<what was done>"`
- [ ] If sprint is fully complete, change `status: "completed"`
- File path shown in checkpoint output

## Pause for Breath
- [ ] Show summary of what was completed
- [ ] Show current sprint progress (X of Y milestones done)
- [ ] Show velocity (LOC/day vs planned)
- [ ] Ask user: "Ready to continue to next milestone?"
- [ ] If user says "pause" or "stop", save state and exit gracefully

## Commit (After Quality Checks Pass)
- [ ] Stage files: `git add <files>`
- [ ] Commit with descriptive message
- [ ] Include milestone name and key changes
- [ ] Push if appropriate

## Milestone Complete
- [ ] Mark milestone as `completed` in TodoWrite
- [ ] Move to next milestone or finalize sprint
