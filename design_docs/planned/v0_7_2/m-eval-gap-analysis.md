# M-EVAL-GAP: AILANG vs Python Parity Analysis

**Status**: In Progress - v0.6.3 Tested
**Version**: v0.6.2 → v0.6.5
**Date**: 2026-01-05
**Goal**: Achieve AILANG parity with Python in agent mode evals

---

## v0.6.3 Test Results (2026-01-05)

**Changes made:**
1. ✅ Added `floatToInt`/`intToFloat` wrappers to `std/math.ail`
2. ✅ Created prompt v0.6.3 with explicit stdlib documentation (std/math, std/string)

**Test Results:**

| Benchmark | v0.6.2 | v0.6.3 | Turns | Result |
|-----------|--------|--------|-------|--------|
| `json_parse` | ❌ 14 turns | ✅ **Pass** | 47 | Agent succeeded with new prompt |
| `config_file_parser` | ❌ 55 turns | ❌ 26 turns | 26 | **Improvement** but still fails |

**Key Findings:**

1. **Agent discoverability improved** - In v0.6.3, agent uses `floatToInt(port_f)` (knows the function exists from prompt), but **forgets to import `std/math`**. In v0.6.2, agent implemented its own broken recursive version.

2. **`json_parse` now passes** - 47 turns is high but successful. Agent correctly handles nested Option matching.

3. **Import forgetting is the new gap** - Agent sees function in prompt, uses it, but omits import statement.

**Next Actions (v0.6.4):**

| Priority | Action | Expected Impact |
|----------|--------|-----------------|
| P0 | Add "REMEMBER to import" warning in prompt | Fix config_file_parser |
| P0 | Add JSON helper functions (`getString`, `getNumber`) | Reduce json_parse turns from 47 to <20 |
| P1 | Auto-import `std/math` in prelude | Eliminate import forgetting |
| P2 | Fix Gemini executor bug | Enable real Gemini benchmarking |

---

## Executive Summary

Analysis of v0.6.2 eval baselines reveals:

| Mode | Model | AILANG | Python | Gap |
|------|-------|--------|--------|-----|
| Standard | Haiku | 56.5% | 58.7% | **2.2%** |
| Standard | Opus | 69.5% | 80.4% | 10.9% |
| Agent | Haiku | **80.7%** | 100% | 19.3% |
| Agent | Gemini Flash 3 | 69.2% | 100% | 30.8% |

**Key Finding**: Agent mode dramatically improves AILANG success (56.5% → 80.7% for Haiku), but 10-16 benchmarks still fail due to:
1. **Timeout issues** (1m limit too short for complex repairs)
2. **JSON API confusion** (agents struggle with `Json` type)
3. **Missing stdlib functions** (agents hallucinate functions that don't exist)

---

## Detailed Failure Analysis

### Agent Mode Failures by Model

#### Claude Haiku (10 failures, 80.7% success)

| Benchmark | Turns | Root Cause |
|-----------|-------|------------|
| `api_call_json` | 15 | Timeout - Net capability complexity |
| `cli_args` | 34 | Runtime error - file I/O confusion |
| `config_file_parser` | 22 | Timeout - missing stdlib functions |
| `csv_to_json_converter` | 28 | Timeout - type confusion with strings |
| `json_parse` | 14 | Timeout - `Json` type confusion |
| `json_transform` | 13 | Timeout - JSON API misunderstanding |
| `log_file_analyzer` | 17 | Timeout - string parsing complexity |
| `pipeline` | 17 | Timeout - effect composition |
| `symbolic_diff` | 13 | Timeout - ADT complexity |
| `type_unify` | 14 | Timeout - recursive ADT |

**Pattern**: Most failures are timeouts after 13-34 turns. Agent understands AILANG but gets stuck iterating on type errors.

#### Gemini Flash 3 (16 failures, 69.2% success)

| Benchmark | Turns | Root Cause |
|-----------|-------|------------|
| All 16 | 1 | **Immediate timeout** - agent infra issue |

**Pattern**: Gemini shows `turns: 1` for all failures - the agent eval infrastructure may not be properly handling Gemini CLI.

---

## Root Cause Categories

### Category 1: Timeout Configuration (40% of failures)

**Problem**: 1-minute timeout too short for complex benchmarks.

**Evidence**:
- `csv_to_json_converter`: 28 turns before timeout
- `cli_args`: 34 turns before timeout
- Agent is making progress but runs out of time

**Solution**:
- Increase timeout to 3-5 minutes for agent mode
- Add benchmark-specific timeout overrides for complex tasks

### Category 2: JSON API Teaching Gap (30% of failures)

**Problem**: Prompt doesn't clearly explain `Json` type vs `string`.

**Evidence** (from json_parse transcript):
```
[TURN 8] I'm passing a string to get, but get expects a Json
[TURN 9] Wait, I'm still confused about the types
[TURN 11] The type error is clear: get expects a Json but I'm passing a string
[TURN 14] Let me just try to look at what AILANG considers the JSON type
```

Agent spends 14 turns confused about whether JSON values are `string` or `Json` type.

**Solution**: Add explicit JSON examples to prompt:
```ailang
-- JSON values have type Json, not string
let parsed: Result[Json, string] = decode("{\"name\":\"Alice\"}")
match parsed {
  Ok(obj) => {  -- obj has type Json
    match get(obj, "name") {  -- get takes Json, returns Option[Json]
      Some(nameJson) => match asString(nameJson) {  -- convert Json to string
        Some(name) => print(name),
        None => ()
      },
      None => ()
    }
  },
  Err(e) => print("Parse error: " ++ e)
}
```

### Category 3: Missing Stdlib Functions (20% of failures)

**Problem**: Agents hallucinate stdlib functions that don't exist.

**Hallucinated functions**:
- `_string_slice` / `stringSlice` - substring extraction
- `contains` - string containment check
- `_cast_float` / `_itof` / `round` - numeric conversions
- `floatToInt` - float to int conversion

**Current stdlib gaps**:
| Function | Exists? | Workaround |
|----------|---------|------------|
| `stringSlice(s, start, end)` | No | Use `_string_to_list` + `_list_slice` |
| `contains(s, sub)` | No | Use `_string_indexOf(s, sub) >= 0` |
| `floatToInt(f)` | Partial | `_float_to_int` exists (v0.5.11) |
| `round(f)` | No | No workaround |

**Solution**: Add commonly-needed string functions to `std/string`:
```ailang
-- Proposed additions to std/string
export func slice(s: string, start: int, end: int) -> string
export func contains(s: string, sub: string) -> bool
export func startsWith(s: string, prefix: string) -> bool
export func endsWith(s: string, suffix: string) -> bool
```

### Category 4: Gemini Agent Infrastructure - CRITICAL BUG (30% of failures)

**Problem**: Gemini Flash 3 agent runs are NOT using Gemini CLI - they're using Claude Code!

**Root Cause Analysis**:

1. **Wrong Runner Used**: v0.6.2 evals used `RunAgentBenchmark()` which hardcodes Claude Code:
   ```go
   // agent_runner.go:225
   Executor: "claude", // Legacy runner always uses Claude Code
   ```

2. **Multi-Executor Not Invoked**: The correct `RunAgentBenchmarkWithExecutor()` in
   `agent_runner_multi.go` supports multiple executors but was never called.

3. **models.yml Shows Intent But Not Reality**:
   ```yaml
   gemini-3-flash:
     agent_cli: "gemini"  # Configured but NOT USED
     agent_model_name: "gemini-3-flash-preview"
   ```

   Comment in models.yml even says: `"(not yet implemented)"` for Gemini CLI

**Evidence**:
```json
{
  "model": "gemini-3-flash",
  "agent_turns": 1,
  "stderr": "timeout after 1m0s\n\n=== Claude Session Transcript ==="  // <-- CLAUDE, not Gemini!
}
```

Even successful Gemini runs show "Claude Session Transcript" with `turns: 1`.

**Impact**: All "Gemini Flash 3" agent results are actually Claude Code runs with the wrong
model configuration. The 69.2% success rate is meaningless for Gemini evaluation.

**Solution** (High Priority):

1. **Use `RunAgentBenchmarkWithExecutor`** in eval suite:
   - Update `cmd/ailang/eval.go` to use multi-executor runner
   - Pass model name to select correct executor from models.yml

2. **Verify Gemini CLI Works**:
   ```bash
   gemini -m gemini-3-flash-preview --output-format json "Hello"
   ```

3. **Add Integration Test**:
   ```go
   func TestGeminiExecutor(t *testing.T) {
       exec, _ := executor.GlobalFactory().GetExecutor("gemini")
       // Verify it calls gemini CLI, not claude
   }
   ```

---

## Recommended Actions

### Phase 1: Infrastructure Fixes (v0.6.3) - CRITICAL

**P0: Fix Gemini Agent Executor** (blocks all Gemini agent eval)

1. **Wire up multi-executor in eval suite**:
   - File: `cmd/ailang/eval.go`
   - Change: Use `RunAgentBenchmarkWithExecutor()` instead of `RunAgentBenchmark()`
   - Test: `DEBUG_AGENT=1 ailang eval-suite --models gemini-3-flash --agent-mode`

2. **Verify Gemini CLI executor works**:
   - File: `internal/executor/gemini/gemini.go`
   - Test: `go test ./internal/executor/gemini/...`

3. **Re-run Gemini agent baseline** after fix

**P1: ~~Increase Agent Timeout~~ - WILL NOT FIX MOST FAILURES**

**Analysis Update**: Examined transcripts show models are **stuck in loops**, not "almost done":

```
[config_file_parser - 22 turns]
Turn 9:  "floatToInt type error - let me check what casting function is available"
Turn 13: "Let me check the builtins..."
Turn 14+: Keeps trying workarounds that compile but fail logically
Final code: Broken convertToInt using `(x % 1.0)` - doesn't work

[csv_to_json_converter - 28 turns]
Turn 11: "I'm trying to pattern match on strings as if they're lists"
Turn 12+: Rewrites but can't fix fundamental misunderstanding
```

**Real Root Causes** (timeout won't help):
1. **Missing `floatToInt`** - model keeps searching for it, can't find, tries broken workarounds
2. **String ≠ character list** - model tries `match str { [a, b, c] => ... }` which doesn't work
3. **Missing string operations** - `slice`, `contains`, `charAt` don't exist

**Actual Fix**: Add stdlib functions (see Phase 3), not longer timeouts.

**Per-benchmark timeouts**: May be useful for genuinely complex tasks (e.g., `type_unify` which requires deep recursion), but current 60s is reasonable for most benchmarks. The issue is capability gaps, not time constraints.

### Phase 2: Prompt Improvements (v0.6.3)

**P0: Add JSON API Examples** (fixes 30% of failures)

Add to `prompts/v0.6.3.md`:

```ailang
-- IMPORTANT: JSON values have type `Json`, not `string`!
-- The `decode` function returns Result[Json, string]
-- Use accessor functions to extract typed values

import std/json (decode, get, asString, asNumber, asArray)

func example() -> () ! {IO} = {
  -- Parse JSON string into Json value
  let jsonStr = "{\"name\":\"Alice\",\"age\":30}";
  match decode(jsonStr) {
    Ok(obj) => {
      -- obj has type Json
      -- get(obj, key) returns Option[Json]
      match get(obj, "name") {
        Some(nameJson) => {
          -- nameJson has type Json, convert to string
          match asString(nameJson) {
            Some(name) => print("Name: " ++ name),
            None => print("name is not a string")
          }
        },
        None => print("no name field")
      }
    },
    Err(e) => print("Parse error: " ++ e)
  }
}
```

### Phase 3: Add Missing Wrappers + Document in Prompt (v0.6.3-v0.6.4)

**Discovery**: Most stdlib wrappers already exist! Agent just doesn't know about them.

**Already in stdlib** (agent transcript shows it couldn't find these):

| Function | Exists in stdlib? | Agent looked for |
|----------|-------------------|------------------|
| `substring(s, start, end)` | ✅ std/string | `stringSlice` |
| `contains(hay, needle)` | ✅ std/string | `contains` |
| `trim(s)` | ✅ std/string | `trim` |
| `split(s, delim)` | ✅ std/string | `split` |
| `floor(x)` | ✅ std/math (returns float) | - |
| `ceil(x)` | ✅ std/math (returns float) | - |
| `round(x)` | ✅ std/math (returns float) | - |

**ONLY missing wrappers** (cause of stuck loops):

| Function | Builtin exists? | Wrapper needed |
|----------|-----------------|----------------|
| `floatToInt(x) -> int` | ✅ `_float_to_int` | ❌ No wrapper in stdlib |
| `intToFloat(x) -> float` | ✅ `_int_to_float` | ❌ No wrapper in stdlib |

**Minimal fix** (just 2 lines in std/math.ail):
```ailang
export pure func floatToInt(x: float) -> int = _float_to_int(x)
export pure func intToFloat(x: int) -> float = _int_to_float(x)
```

**Prompt fix** (document what's available):
```
## String Functions (import std/string)
substring(s, start, end), contains(hay, needle), trim(s), split(s, delim)
toLower(s), toUpper(s), find(hay, needle), stringToInt(s), stringToFloat(s)

## Math Functions (import std/math)
floor(x), ceil(x), round(x)          -- return float
floatToInt(x), intToFloat(x)         -- type conversion (v0.6.3+)
abs_Float(x), abs_Int(x), sqrt(x), pow(x, y)
sin(x), cos(x), tan(x), log(x), exp(x)
```

**Agent was 1 function away from success** - just needed `floatToInt`!

### Phase 4: Prompt A/B Testing (v0.6.5)

1. **Create benchmark-specific prompts** for the 10 hardest cases
2. **Measure turns-to-success** as primary metric (not just pass/fail)
3. **Compare Haiku vs Opus** on same prompts to identify model-specific issues

---

## Success Metrics

| Metric | Current (v0.6.2) | Target (v0.6.5) | Notes |
|--------|------------------|-----------------|-------|
| Haiku Agent Success | 80.7% | **90%+** | +5 benchmarks |
| Gemini Flash Agent Success | 69.2%* | **TBD** | *Invalid - needs re-run |
| Average turns (success) | ~10 | **<8** | Efficiency metric |
| "floatToInt not found" loops | ~3 | **0** | After stdlib fix |
| JSON-related failures | 6 | **0-2** | After prompt fix |

*Gemini results are invalid because agent eval used Claude Code CLI instead of Gemini CLI.

## Revised Priority Actions

1. **Add `floatToInt`/`intToFloat` to std/math.ail** (2 lines) - unblocks config_file_parser
2. **Document stdlib in prompt** - unblocks agents who can't find existing functions
3. **Fix Gemini executor** - enables real Gemini benchmarking
4. **Add JSON examples to prompt** - reduces type confusion

---

## Appendix: Benchmark-Specific Analysis

### Hardest Benchmarks (0% AILANG success across all models)

| Benchmark | Python | Issue | Fix |
|-----------|--------|-------|-----|
| `config_file_parser` | 100% | Missing `stringSlice` | Add to stdlib |
| `csv_to_json_converter` | 100% | String parsing, `contains` | Add to stdlib |
| `error_handling` | 100% | Type annotation confusion | Better examples |

### Benchmarks Where Opus Succeeds but Haiku Fails

| Benchmark | Opus | Haiku | Gap Reason |
|-----------|------|-------|------------|
| `canonical_normalization` | Pass | WRONG_LANG | Haiku writes Python-like |
| `fold_reduce` | Pass | WRONG_LANG | Same |
| `graph_bfs` | Pass | WRONG_LANG | Same |
| `json_parse` | Pass | Timeout | Haiku slower to learn |
| `merge_sort` | Pass | PAR_001 | Syntax errors |

**Insight**: Haiku's main gaps are:
1. Syntax confusion (writes Python-like code) → **Prompt examples**
2. Slower iteration (times out) → **Longer timeout**
3. Missing functions → **Stdlib additions**
