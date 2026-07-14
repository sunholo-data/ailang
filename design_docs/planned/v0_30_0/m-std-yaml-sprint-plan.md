# Sprint Plan: M-STD-YAML — `std/yaml` YAML→JSON Bridge

## Summary

Ship a pure, WASM-portable `std/yaml` module that turns YAML into AILANG's existing `Json` ADT via a single Go builtin (`_yaml_to_json`) plus a thin `.ail` wrapper. Unblocks Stage 0 of the `cph-uni-stx-bench` STX harness.

**Duration:** 1 day (~7 hours)
**Dependencies:** `std/json` (existing), `gopkg.in/yaml.v3 v3.0.1` (already vendored). **All external premises verified live** — no blocking unknowns.
**Risk Level:** Low

## Current Status Analysis

### Completed Recently (velocity anchor)
- ✅ M-STDLIB-HTML (v0.19.1): native HTML5 parser builtin + `std/html.ail` wrapper — the identical Go-builtin + `.ail`-wrapper pattern this sprint follows, but *larger* (native parser + own ADT). This sprint is strictly smaller: bridge-to-JSON, no new ADT.
- ✅ M-STD-DEFLATE-ZLIB (v0.16.0): stdlib module with Go builtins, comparable shape.

### Velocity
- This is a ~350 LOC sprint (impl + tests + docs) — well under a single day at recent stdlib-module velocity. HTML (a much bigger module) landed as a single focused effort.

### Remaining from Design Doc
- ⏳ M1 builtin: ~270 LOC (impl + Go tests)
- ⏳ M2 module: ~70 LOC (`.ail` + integration test)
- ⏳ M3 docs/example/WASM verify: ~130 lines docs + example

## Pre-Verified Premises (do NOT re-litigate)

These were checked live during design and are frozen for the sprint:

- `gopkg.in/yaml.v3 v3.0.1` is in `go.mod`/`go.sum` — **no dependency work**.
- `GOOS=js GOARCH=wasm go build` of the yaml.v3→`encoding/json` path **succeeds (exit 0)** — WASM constraint satisfiable.
- Builtin name `_yaml_to_json` is **free** (grep clean across `internal/`, `cmd/`).
- `std/embed.go` uses `//go:embed *.ail` — a new `std/yaml.ail` is **auto-embedded**; **no resolver/embed registration code needed** (delete that task if the executor sees it).
- Cross-module `import std/json (Json, decode)` is **proven** (`std/jwt.ail`, `std/sem.ail` already do it).
- Round-trip verified: `name: STX\nitems:\n - a\n - b\ncount: 3` → `{"count":3,"items":["a","b"],"name":"STX"}`.

## Proposed Milestones

### Milestone 1: `_yaml_to_json` Go builtin
**Goal:** One pure builtin `string -> Result[string,string]`: `yaml.Unmarshal` into `interface{}`, then `json.Marshal`. No silent coercion — any marshal error returns `Err`.
**Estimated:** ~120 LOC impl + ~150 LOC tests = ~270 LOC
**Duration:** ~3h

**Tasks:**
- Create `internal/builtins/yaml.go` mirroring `internal/builtins/json_decode.go`: `registerYAMLToJSON()`, `yamlToJSONImpl`, `makeYAMLToJSONType()` (type `string -> Result[string,string]`), `init()` registration, `BuiltinMetadata` (Since `v0.30.0`, `StabilityStable`, tags `yaml,parsing,data,result`, module `std/yaml`).
- Impl body: `var v interface{}; yaml.Unmarshal([]byte(s), &v)` → on err `Err("yaml: "+e)`; else `json.Marshal(v)` → on err `Err(...)`; else `Ok(string(b))`.
- Create `internal/builtins/yaml_test.go`: happy path, nested maps, sequences, all scalar types (int/float/bool/null), empty→`"null"`, flow-style equals block-style, **non-string map key → Err**, **malformed indentation → Err**.

**Acceptance Criteria:**
- [x] `ailang builtins list` shows `_yaml_to_json` as `[pure] std/yaml`.
- [x] Non-string-key / NaN inputs return `Err`, never a coerced value.
- [x] `go test ./internal/builtins/ -run YAML` green.
- [x] `make lint` clean on new file.

**Risks:**
- `json.Marshal` rejects a real catalogue construct → **Mitigation:** that's the intended loud-fail; add the 79 KB fixture (or subset) in M2 to confirm the requester's data actually converts.

### Milestone 2: `std/yaml.ail` module + integration test
**Goal:** Expose `yamlToJson` and `decode` (composed with `std/json.decode`, reusing `Json`).
**Estimated:** ~40 LOC module + ~30 LOC `.ail` test = ~70 LOC
**Duration:** ~2h

**Tasks:**
- Create `std/yaml.ail`: `module std/yaml`, `import std/json (Json, decode)`, `import std/result (Result, Ok, Err)`. Export `yamlToJson(s: string) -> Result[string, string] = _yaml_to_json(s)` and `decode(s: string) -> Result[Json, string]` as `match yamlToJson(s) { Ok(j) => decode(j), Err(e) => Err(e) }` (alias the json import if the name `decode` collides — e.g. `import std/json (Json, decode as jsonDecode)`).
- Doc comments matching `std/json.ail` style.
- Create `tests/yaml_bridge_test.ail`: `decode` round-trip, and assert `yamlToJson` output fed to `std/json.decode` equals decoding the equivalent JSON.
- **No** `std/embed.go` or resolver edits (auto-embed confirmed).

**Acceptance Criteria:**
- [x] `ailang docs std/yaml` lists `yamlToJson` and `decode` with signatures.
- [x] `ailang check std/yaml.ail` clean.
- [x] `.ail` integration test passes.
- [x] `decode` returns the same `Json` ADT as `std/json` (constructor identity confirmed in test).

**Risks:**
- `decode` name collision with the imported `std/json.decode` → **Mitigation:** import alias (`decode as jsonDecode`). Verify in `ailang check`.

### Milestone 3: Example, docs, WASM verify
**Goal:** Runnable example, reference docs, index entry, changelog, WASM build check.
**Estimated:** ~25 LOC example + ~80 lines docs
**Duration:** ~2h

**Tasks:**
- Create `examples/yaml_config.ail` (from design doc Example 1, adapted to a self-contained YAML string so it needs no external file): `decode` a YAML string, pull fields via `std/json` accessors, print. Must pass `make verify-examples`.
- Create `docs/docs/reference/stdlib/yaml.md` mirroring the html/json reference page.
- Add `std/yaml` row to the Stdlib Index `docs/docs/reference/stdlib.md`.
- CHANGELOG.md entry under v0.30.0 (category: stdlib).
- Run WASM build verify — must succeed.

**Acceptance Criteria:**
- [x] `ailang run examples/runnable/yaml_config.ail` prints expected output (`title: Fysik A` / `year: 2026`); manifest entry added + `validate_manifest --ci` green.
- [x] WASM build exit 0 — verified on `./cmd/wasm` (real browser binary) and `./internal/builtins`. NOTE: `go build ./...` for wasm fails on PRE-EXISTING `_unix.go` host packages (`internal/executor/motoko`, `internal/eval_harness` use `Setpgid`), which are never part of the WASM binary; the correct target is `./cmd/wasm`.
- [x] `std/yaml` added to stdlib index + new `std-yaml.md` reference page (docs_search picks these up on next index).
- [x] CHANGELOG + stdlib index updated.

**Risks:**
- Example uses a construct not yet supported → **Mitigation:** `ailang check` the example before committing (per verify-by-running rule).

## Success Metrics
- Test coverage: new builtin ≥ 80% (Go unit tests cover all edge cases).
- Examples passing: `examples/yaml_config.ail` verified via `make verify-examples`.
- Documentation: `docs/docs/reference/stdlib/yaml.md` + `stdlib.md` index + CHANGELOG.
- All tests passing: ✅ (`go test ./...` + `.ail` integration)
- All linting passing: ✅ (`make lint`)
- WASM: `GOOS=js GOARCH=wasm go build ./...` ✅

## Dependencies
- `std/json` (`Json` ADT + `decode`) — existing, no changes.
- `gopkg.in/yaml.v3` — vendored, no changes.

## Open Questions
- **Acceptance fixture:** ideally test against the requester's actual 79 KB catalogue (or a representative slice). If not obtainable, a hand-authored fixture covering nested maps + sequences + all scalar types is sufficient. *(agent may proceed with a synthetic fixture; note it in the implementation report.)*
- **`encode` (Json→YAML):** explicitly deferred to Future Work — do NOT add a second builtin this sprint.

## Notes
- **Systemic-fix compliance:** one Go builtin serves both public APIs; `decode` is pure-AILANG composition, minimizing frozen core surface.
- **No-silent-fallback compliance:** unconvertible YAML returns `Err`, never a guessed value.
- Single-document only in v1 (multi-doc `decodeAll` is Future Work) — yaml.v3 `Unmarshal` reads the first document; the executor should not attempt a decoder loop.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_30_0/m-std-yaml-sprint-plan.md`
