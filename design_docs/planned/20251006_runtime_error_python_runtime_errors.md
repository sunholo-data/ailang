# PYTHON: Runtime Errors

**Discovered**: AI Eval Analysis - 2025-10-06
**Frequency**: 18 failures across 3 benchmark(s)
**Priority**: P1 (High Priority)
**Estimated**: 975 LOC, 3.5 days
**Category**: runtime_error
**Impact**: high

## Problem Statement





**Last Updated**: 2025-10-25 (merged 9 new failures)
## Evidence from AI Eval

**Affected Benchmarks**: fizzbuzz, records_person, recursion_factorial

**Models Affected**: claude-sonnet-4-5, gpt-4o-mini

**Failure Rate**: 9/31 (29.0%)

### Example Failures


**Error 1:**
```
  File "/var/folders/kv/nxj052r50g7813x2blwc74jw0000gn/T/eval_3002195128.py", line 1
    module benchmark/solution
           ^^^^^^^^^
SyntaxError: invalid syntax

```

**Generated Code:**
```python
module benchmark/solution

import std/io (println)

export func fizzBuzzValue(n: int) -> string {
  if n % 15 == 0
  then "FizzBuzz"
  else if n % 3 == 0
  then "Fizz"
  else if n % 5 == 0
  then "Buzz"
  else show(n)
}

export func fizzBuzzLoop(current: int, end: int) -> () ! {IO} {
  if current > end
  then ()
  else {
    println(fizzBuzzValue(current));
    fizzBuzzLoop(current + 1, end)
  }
}

export func main() -> () ! {IO} {
  fizzBuzzLoop(1, 100)
}
```

---

**Error 2:**
```
  File "/var/folders/kv/nxj052r50g7813x2blwc74jw0000gn/T/eval_3678032186.py", line 1
    module benchmark/solution
           ^^^^^^^^^
SyntaxError: invalid syntax

```

**Generated Code:**
```python
module benchmark/solution

import std/io (println)

export func fizzBuzz(n: int) -> string {
  if n % 15 == 0
  then "FizzBuzz"
  else if n % 3 == 0
  then "Fizz"
  else if n % 5 == 0
  then "Buzz"
  else show(n)
}

export func loop(current: int, end: int) -> () ! {IO} {
  if current > end
  then ()
  else {
    println(fizzBuzz(current));
    loop(current + 1, end)
  }
}

export func main() -> () ! {IO} {
  loop(1, 100)
}
```

---

**Error 3:**
```
  File "/var/folders/kv/nxj052r50g7813x2blwc74jw0000gn/T/eval_3148940054.py", line 1
    module benchmark/solution
           ^^^^^^^^^
SyntaxError: invalid syntax

```

**Generated Code:**
```python
module benchmark/solution

import std/io (println)

export func fizzBuzz(n: int) -> string {
  if n % 15 == 0
  then "FizzBuzz"
  else if n % 3 == 0
  then "Fizz"
  else if n % 5 == 0
  then "Buzz"
  else show(n)
}

export func loop(i: int, end: int) -> () ! {IO} {
  if i > end
  then ()
  else {
    println(fizzBuzz(i));
    loop(i + 1, end)
  }
}

export func main() -> () ! {IO} {
  loop(1, 100)
}
```


### Additional Examples (Latest Analysis)

**Error 1:**
```
  File "/var/folders/kv/nxj052r50g7813x2blwc74jw0000gn/T/eval_3002195128.py", line 1
    module benchmark/solution
           ^^^^^^^^^
SyntaxError: invalid syntax

```

**Generated Code:**
```python
module benchmark/solution

import std/io (println)

export func fizzBuzzValue(n: int) -> string {
  if n % 15 == 0
  then "FizzBuzz"
  else if n % 3 == 0
  then "Fizz"
  else if n % 5 == 0
  then "Buzz"
  else show(n)
}

export func fizzBuzzLoop(current: int, end: int) -> () ! {IO} {
  if current > end
  then ()
  else {
    println(fizzBuzzValue(current));
    fizzBuzzLoop(current + 1, end)
  }
}

export func main() -> () ! {IO} {
  fizzBuzzLoop(1, 100)
}
```

---

**Error 2:**
```
  File "/var/folders/kv/nxj052r50g7813x2blwc74jw0000gn/T/eval_3678032186.py", line 1
    module benchmark/solution
           ^^^^^^^^^
SyntaxError: invalid syntax

```

**Generated Code:**
```python
module benchmark/solution

import std/io (println)

export func fizzBuzz(n: int) -> string {
  if n % 15 == 0
  then "FizzBuzz"
  else if n % 3 == 0
  then "Fizz"
  else if n % 5 == 0
  then "Buzz"
  else show(n)
}

export func loop(current: int, end: int) -> () ! {IO} {
  if current > end
  then ()
  else {
    println(fizzBuzz(current));
    loop(current + 1, end)
  }
}

export func main() -> () ! {IO} {
  loop(1, 100)
}
```

---

---


## Root Cause Analysis

The M-EVAL harness lacks robust language detection and over-relies on code fence labels and/or file extensions. When a model emits an AILANG program in a fenced block labeled "python", the harness saves it as a .py file and invokes Python. Because AILANG programs begin with "module", include "export func", "->" arrows in type annotations, and effect annotations "! {IO}", this is syntactically invalid Python and fails at runtime with SyntaxError. The AILANG implementation itself is unaffected; what’s missing is a content-based language detector and cross-runner guardrails in the evaluation pipeline to prevent misrouting.

## Proposed Solution

Introduce a robust, content-based language detector in M-EVAL that scores code snippets across supported languages and chooses the most probable runner regardless of the fence label. The detector should recognize AILANG by structural signatures: leading "module" declaration, "export func", "import std/...", Hindley-Milner style type arrows "->", effect annotations "! {IO}", algebraic data type keywords, and block/semicolon usage. Make routing decisions based on scored features, with a configurable override flag and structured metrics.

In addition, harden non-AILANG runners (especially Python) with a pre-exec guard that checks for strong AILANG signatures and returns a rerouteable detection error instead of attempting execution. Update the code block extractor to rewrite mislabeled fences to "ailang" when the detector’s confidence exceeds a threshold. This design centralizes detection in one place, is non-invasive to the AILANG compiler/runtime, and aligns with the repository’s M-EVAL architecture.

### Implementation Approach

- Implement a multi-signal language detector with weighted features for AILANG (and other languages) and expose a DetectLanguage(content) -> (lang, confidence, features) API.
- Integrate detection into the code extraction and runner selection flow, overriding fence labels when detector confidence is high for AILANG.
- Add a protective preflight to Python (and other) runners to detect AILANG signatures and return a typed RerouteSuggested error.
- Update M-EVAL configuration to enable detection by default, with env/CLI flags to tune thresholds and logging of detection metrics.
- Improve error messaging to include a clear explanation when rerouting occurs, with breadcrumbs for debugging.
- Add unit and integration tests covering mislabelled AILANG snippets and ensure they route to the AILANG runner.
- Add new regression benchmarks for fizzbuzz, records_person, recursion_factorial explicitly mislabeled as python.
- Update developer docs/prompts to prefer "```ailang" fences; add lints in CI that fail on persistent mislabeling for AILANG corpora.

## Technical Design

### API Changes

- New M-EVAL public function: detect.DetectLanguage(content) -> (lang string, confidence float64, features map[string]float64)
- New CLI flags: --lang-detect, --lang-detect-threshold
- New env vars: AILANG_LANG_DETECT=1, AILANG_DETECT_THRESHOLD=0.85

### Type System Changes

None

### Runtime Changes

None (AILANG runtime unchanged). Non-AILANG runners gain preflight guards; AILANG runner accepts content without extension when detector says AILANG.

## Implementation Plan


1. **1. Content-based Language Detector (~220 LOC, 1.0 day) - New pkg providing token/regex features, scoring, and confidence for AILANG and other supported langs.** (~TBD LOC, TBD)
   

2. **2. Integrate Detector in Extractor (~90 LOC, 0.5 day) - Modify code block extraction to consult detector and optionally rewrite mislabeled fences to ailang.** (~TBD LOC, TBD)
   

3. **3. Runner Routing Orchestrator Update (~110 LOC, 0.5 day) - Centralize routing: prefer detector over fence when confidence >= threshold; add telemetry.** (~TBD LOC, TBD)
   

4. **4. Python Runner Preflight Guard (~60 LOC, 0.25 day) - Detect AILANG signatures and return RerouteSuggested error before exec.** (~TBD LOC, TBD)
   

5. **5. AILANG Runner Acceptance Hardenings (~40 LOC, 0.25 day) - Tolerate inputs without .ail extension when content is AILANG; improve diagnostics.** (~TBD LOC, TBD)
   

6. **6. Config/Flags and Logging (~70 LOC, 0.25 day) - Add AILANG_LANG_DETECT, AILANG_DETECT_THRESHOLD env vars and CLI flags; structured logs for detections.** (~TBD LOC, TBD)
   

7. **7. Unit Tests: Detector and Guards (~180 LOC, 0.75 day) - Positive/negative cases, edge cases around mixed content and low-confidence signals.** (~TBD LOC, TBD)
   

8. **8. Integration Tests: Mislabelled Benchmarks (~120 LOC, 0.5 day) - End-to-end runs for fizzbuzz, records_person, recursion_factorial mislabeled as python.** (~TBD LOC, TBD)
   

9. **9. Docs and Prompt Templates (~35 LOC, 0.25 day) - Update README/CONTRIBUTING and prompt kit to use ```ailang fences; add evaluator notes.** (~TBD LOC, TBD)
   

10. **10. CI Wiring and Metrics (~50 LOC, 0.25 day) - Ensure detection is enabled in CI; publish detection rate and reroute counts.** (~TBD LOC, TBD)
   


## Testing Strategy

### Unit Tests

- Detector identifies AILANG with high confidence for:
  - module header + export func + effect annotation
  - no fence, raw text AILANG
  - mislabeled python fence
- Detector does not falsely classify:
  - Real Python with async defs and typing annotations
  - JSON, Markdown noise
- Python runner preflight returns RerouteSuggested for AILANG signatures.
- Orchestrator routes to AILANG when confidence >= threshold and preserves original language when confidence < threshold.
- Configurable threshold behavior tested at multiple levels.

### Integration Tests

- Run fizzbuzz, records_person, recursion_factorial with fences:
  - ```python mislabeled
  - ```ailang correct
  - No fence, raw text
  All must route to AILANG and pass.
- Mixed-file batch where some true Python coexists with AILANG; ensure no cross-language contamination.
- Telemetry snapshot includes detection counts, average confidence, and reroutes.

### New Benchmarks

- mislabel_fizzbuzz_python_fence.ail.txt: AILANG code wrapped in ```python.
- mislabel_records_person_python_fence.ail.txt.
- mislabel_recursion_factorial_python_fence.ail.txt.
- multi_lang_mixture_batch.json: interleaved AILANG and Python snippets to validate detector precision and recall.

## Success Criteria


- [ ] 0 runtime_error occurrences from Python runner on AILANG sources across CI and local runs.

- [ ] All fizzbuzz, records_person, recursion_factorial examples pass when mislabeled as python.

- [ ] Detector precision >= 0.98 and recall >= 0.98 for AILANG on test corpus.

- [ ] No regression in non-AILANG language routing (all existing tests remain green).

- [ ] Detection enabled by default with documented flags; telemetry exposed in CI.

- [ ] Developer docs and prompt templates updated to prefer ```ailang fences.

- [ ] Bench coverage updated; new mislabel benchmarks pass.


## References

- **Similar Features**: See design_docs/implemented/ for reference implementations
- **Design Docs**: CLAUDE.md, README.md, design_docs/planned/v0_4_0_net_enhancements.md
- **AILANG Architecture**: See CLAUDE.md, README.md

## Estimated Impact

**Before Fix**:
- AI success rate: 72.7%%
- Token efficiency: Models sometimes emit longer prompts/workarounds to force correct fences; extra retries due to misrouting inflate tokens.

**After Fix** (projected):
- AI success rate: 86.4% (projected; +13.7 pp by eliminating 9 of 18 remaining example failures)%
- Token efficiency: 1-3% fewer tokens on average by removing retries and allowing looser fence requirements; reduced error recovery overhead.
"""
print(doc)

---

*Generated by ailang eval-analyze on 2025-10-06 12:03:06*
*Model: gpt5*
