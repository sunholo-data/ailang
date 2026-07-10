# Sprint Retrospective: M-DOGFOOD-FIXES

## Summary
- **Sprint**: M-DOGFOOD-FIXES (3 design docs from the deontic-package dogfooding session)
- **Duration**: ~2.5h wall clock (estimated: 1 day) — parallel execution
- **Execution mode**: Parallel, 1 wave of 3 + 1 sequential fix round
- **Milestones**: 3/3 passed (M1 required one fix round)
- **Models**: sonnet (M1 + fix), opus (M2, M3) — orchestrated/integrated by fable. First deliberate right-sizing experiment.

## Milestone Timing

| Milestone | Model | Est. LOC | Actual (approx) | Agent time | Tokens | Status |
|-----------|-------|----------|-----------------|------------|--------|--------|
| M1 named tests | sonnet | 300 | ~530 (2 commits) | 25 min | 162K | ✅ after fix round |
| M1 fix round | sonnet | — | ~525 | 80 min | 179K | ✅ |
| M2 #327 SCC fix | opus | 350 | ~385 | 25 min | 139K | ✅ first pass |
| M3 diagnostics | opus | 400 | ~645 | 24 min | 169K | ✅ first pass |

## Parallelization Results
- Wave of 3 in isolated worktrees; zero file overlap by design → **zero merge conflicts**.
- Speedup ≈ 3× for the wave; the M1 fix round serialized the tail.

## Friction Encountered
1. **Sub-agent self-tests are necessary but not sufficient.** M1 reported 184 green tests yet
   panicked on the first REAL package (deontic) — it skipped the package-shaped fixture its
   design doc explicitly required. The integrator's acceptance-against-real-artifacts gate is
   where the defect surfaced. Lesson: acceptance criteria referencing concrete external
   artifacts (like "deontic must report 5/5") must be in the AGENT PROMPT as a hard gate with
   the exact command, not just in the design doc.
2. **Scope drift needs review, not prohibition.** M1's fix legitimately needed pipeline
   changes (PackageDir knob) outside its stated file scope; a transient out-of-scope
   eval-layer edit was self-reverted. Integrator diff-review of out-of-scope files caught
   both cheaply.
3. **Model right-sizing verdict:** opus handled both compiler-internals milestones flawlessly
   first-pass, including root-causing a bug that had resisted earlier investigation (SCC
   non-exhaustive traversal). Sonnet was adequate for the known-root-cause milestone but
   missed a required fixture. Frontier (fable) was not needed for execution; it earned its
   keep only at integration review.
4. **Fix round on dev (not worktree):** needed the merged state; acceptable, but the agent's
   first commit swept an unstaged integrator edit (changelog) into its commit — harmless
   here, but integrators should keep the tree clean while agents run on shared branches.
5. **Bonus depth from the fix round:** chasing the panic surfaced a real non-determinism
   (bare-name env injection: std/list.concat vs std/string.concat last-write-wins over Go
   map iteration + broken LetRec self-recursion) — fixed with deterministic three-pass
   binding. Verified 20 consecutive deterministic runs.

## Recommendations for Next Sprint
- Put concrete external acceptance commands (with expected output) directly in sub-agent
  prompts; treat design-doc acceptance lists as necessary-not-sufficient.
- Keep opus as default for parser/typechecker/elaborator milestones; sonnet for scoped
  runner/CLI/docs work; escalate only on demonstrated failure.
- Integrator hygiene: no unstaged edits on a branch a sub-agent is committing to.
- The retro itself was nearly forgotten (user prompt caught it) — add it to the integrator's
  closing checklist alongside sprint-JSON completion.
