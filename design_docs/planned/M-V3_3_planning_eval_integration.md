# M-V3.3: Planning Integration with M-EVAL-LOOP

**Status**: Planned
**Priority**: P1 (High Value - Improves eval success rates)
**Estimated Effort**: 2-3 days (~800-1,200 LOC)
**Dependencies**: M-V3.2 (Planning & Scaffolding) ✅, M-EVAL-LOOP Milestones 1-4 ✅
**Target Release**: v0.3.3

---

## Problem Statement

Currently, AI agents in eval benchmarks generate code directly without validating their architecture first. This leads to:

1. **Architecture mistakes discovered late**: Agents write code that won't work (e.g., circular dependencies, invalid module structure)
2. **Wasted tokens**: Failed attempts consume tokens on invalid approaches
3. **Lower success rates**: Architecture errors are harder to fix than syntax errors
4. **No learning from mistakes**: Each eval starts fresh without architectural guidance

**Evidence from M-EVAL-LOOP data:**
- 35.3% of failures are compilation errors (PAR_*, TC_*, MOD_* errors)
- Many failures involve module structure issues (MOD_001)
- Repair success rate is lower for architecture errors vs syntax errors

---

## Solution: Proactive Planning in Eval Loop

**Key Insight**: AI agents should create and validate a plan BEFORE generating code.

### Workflow Changes

**Current (v0.3.2):**
```
Benchmark → Teaching Prompt → Generate Code → Compile → Execute
                ↓ (if error)
              Repair Prompt → Generate Fixed Code → Retry
```

**Proposed (v0.3.3):**
```
Benchmark → Teaching Prompt + Planning Guidance
            ↓
         Generate Plan JSON
            ↓
         Validate Plan (:propose)
            ↓ (if valid)
         Generate Code from Plan
            ↓
         Compile → Execute
            ↓ (if error)
         Repair Prompt → Fix Code OR Fix Plan → Retry
```

### Benefits

1. **Early error detection**: Catch architecture errors before code generation
2. **Reduced token waste**: Invalid plans are much smaller than invalid code
3. **Better repair**: Plan validation errors provide clear guidance
4. **Learning opportunity**: Agents learn architectural patterns from validated plans
5. **Metrics visibility**: Track plan validation success rates separately

---

## Design

### 1. Enhanced Teaching Prompt (Use v0.3.2 ✅)

The v0.3.2 teaching prompt already includes:
- ✅ Planning workflow explanation
- ✅ Plan schema documentation
- ✅ Validation error codes (VAL_M##, VAL_T##, etc.)
- ✅ Best practices for module paths, type names, function names

**Action**: Set `v0.3.2` as active prompt in `prompts/versions.json`

### 2. Plan-First Benchmark Task Prompts

**Current task prompt structure:**
```
Write an AILANG program that <task description>.

Requirements:
- Must use module benchmark/solution
- Must export main() function
- ...
```

**New task prompt structure:**
```
Write an AILANG program that <task description>.

STEP 1: Create a plan (JSON) describing your architecture:
- Modules needed
- Types to define
- Functions to implement
- Effects required

STEP 2: Validate your plan to catch errors early

STEP 3: Generate code implementing the plan

Requirements:
- Must use module benchmark/solution
- Must export main() function
- Plan must validate before generating code
- ...
```

**Implementation:**
- Add `--plan-first` flag to eval harness
- Modify task prompt generation to include planning instructions
- AI agent generates plan JSON, then validates, then generates code

### 3. Plan Validation in Eval Harness

**New execution flow:**

```go
// In internal/eval_harness/runner.go (RepairRunner)

type EvalStep string

const (
    StepPlanGeneration  EvalStep = "plan_generation"
    StepPlanValidation  EvalStep = "plan_validation"
    StepCodeGeneration  EvalStep = "code_generation"
    StepCompilation     EvalStep = "compilation"
    StepExecution       EvalStep = "execution"
)

type StepResult struct {
    Step      EvalStep
    Success   bool
    Output    string
    Error     string
    TokensIn  int
    TokensOut int
}

// Modified RepairRunner
func (r *RepairRunner) RunWithPlanning(ctx context.Context) (*Metrics, error) {
    // Step 1: Generate plan
    planStep := r.generatePlan(ctx)
    if !planStep.Success {
        return r.failWithStep(planStep)
    }

    // Step 2: Validate plan
    validationStep := r.validatePlan(planStep.Output)
    if !validationStep.Success {
        // Try plan repair
        if r.selfRepair {
            repairedPlan := r.repairPlan(ctx, validationStep.Error)
            validationStep = r.validatePlan(repairedPlan)
        }
    }

    // Step 3: Generate code (if plan valid)
    if validationStep.Success {
        codeStep := r.generateCodeFromPlan(ctx, planStep.Output)
        // Continue with compile/execute
    }

    return r.populateMetrics()
}
```

### 4. Extended Metrics

**New metrics fields:**

```go
type Metrics struct {
    // Existing fields...

    // Plan metrics (NEW)
    PlanGenerated      bool   `json:"plan_generated"`
    PlanValid          bool   `json:"plan_valid"`
    PlanValidationErrs []string `json:"plan_validation_errors,omitempty"`
    PlanTokensIn       int    `json:"plan_tokens_in"`
    PlanTokensOut      int    `json:"plan_tokens_out"`
    PlanRepairUsed     bool   `json:"plan_repair_used"`
    PlanRepairOk       bool   `json:"plan_repair_ok"`

    // Code generation from plan (NEW)
    CodeFromPlan       bool   `json:"code_from_plan"`

    // Existing fields...
    FirstAttemptOk     bool
    RepairUsed         bool
    RepairOk           bool
}
```

### 5. Plan Repair Prompts

**When plan validation fails:**

```
Your plan has validation errors:

Error 1: [VAL_M01] Invalid module path 'Api/Core'
  Location: modules[0].path
  Fix: Use lowercase: 'api/core'

Error 2: [VAL_T01] Invalid type name 'request'
  Location: types[0].name
  Fix: Use CamelCase: 'Request'

Please generate a corrected plan addressing these errors.
```

### 6. CLI Integration

**New flags for `ailang eval`:**

```bash
# Enable plan-first workflow
ailang eval --benchmark fizzbuzz --plan-first

# Enable plan repair (separate from code repair)
ailang eval --benchmark fizzbuzz --plan-first --plan-repair

# Use v0.3.2 prompt (includes planning guidance)
ailang eval --benchmark fizzbuzz --prompt-version v0.3.2
```

### 7. A/B Testing Setup

**Experiment: Does planning improve success rates?**

**Control group (v0.3.0-hints):**
- Direct code generation (no planning)
- Self-repair on code errors
- Current baseline

**Treatment group (v0.3.2 + plan-first):**
- Plan generation → validation → code generation
- Self-repair on plan errors AND code errors
- New approach

**Metrics to compare:**
- First-attempt success rate
- Overall success rate (after repair)
- Token consumption (total)
- Average attempts per benchmark
- Error distribution (plan errors vs code errors)

---

## Implementation Plan

### Day 1: Prompt Updates & Plan Generation (0.5 days)

**Tasks:**
1. Set `v0.3.2` as active prompt in `prompts/versions.json`
2. Add `--plan-first` flag to eval harness CLI
3. Modify task prompt template to include planning instructions
4. Implement `generatePlan()` method in RepairRunner
5. Add plan extraction from AI response (JSON parsing)

**Deliverables:**
- AI agents receive planning instructions
- Plans are generated and extracted
- ~150 LOC

### Day 2: Plan Validation & Metrics (1 day)

**Tasks:**
1. Implement `validatePlan()` using existing `planning.ValidatePlan()`
2. Add plan metrics to Metrics struct
3. Implement `populatePlanMetrics()`
4. Add plan validation step to execution flow
5. Update eval output to show plan validation results
6. Add plan repair prompt generation

**Deliverables:**
- Plans are validated before code generation
- Validation errors are recorded in metrics
- Plan repair prompts generated
- ~300 LOC

### Day 3: Plan Repair & A/B Testing (1 day)

**Tasks:**
1. Implement `repairPlan()` method (generate corrected plan)
2. Add `--plan-repair` flag
3. Implement code generation from validated plan
4. Update summary tools to include plan metrics
5. Create A/B test script comparing v0.3.0-hints vs v0.3.2+plan-first
6. Run pilot A/B test on 10 benchmarks

**Deliverables:**
- Plan repair works end-to-end
- A/B testing infrastructure ready
- Pilot data shows feasibility
- ~350 LOC

### Day 4: Polish & Documentation (0.5 days)

**Tasks:**
1. Update documentation (CHANGELOG, design doc)
2. Add example plan-first eval runs
3. Write analysis script for plan metrics
4. Create visualization for plan vs code errors
5. Document A/B testing procedure

**Deliverables:**
- Complete documentation
- Ready for production A/B testing
- ~100 LOC

**Total Estimated:** ~900 LOC over 3 days

---

## Success Metrics

**Must Have ✅:**
- [ ] AI agents generate valid plans before code
- [ ] Plan validation catches at least 50% of architecture errors
- [ ] Plan repair reduces token waste by >20%
- [ ] Metrics track plan generation, validation, and repair separately
- [ ] A/B test shows measurable improvement in success rates

**Nice to Have 🎯:**
- [ ] Plan success rate > 70% on first attempt
- [ ] Plan repair success rate > 90%
- [ ] Overall benchmark success rate improves by >10%
- [ ] Token consumption per benchmark decreases by >15%

**Deferred ⏭️:**
- Interactive plan debugging (future)
- Plan learning from successful benchmarks
- Multi-step planning (decompose complex tasks)

---

## Risks & Mitigations

**Risk 1: Plans add overhead without improving success**
- **Impact**: Wasted effort, no benefit
- **Mitigation**: Pilot test on 10 benchmarks first, measure ROI
- **Fallback**: Keep plan-first optional, don't force it

**Risk 2: Plan validation false positives**
- **Impact**: Valid plans rejected, agents frustrated
- **Mitigation**: Validation errors are warnings, not blockers (warn-and-proceed mode)
- **Fallback**: Allow override with `--skip-plan-validation`

**Risk 3: AI agents ignore planning instructions**
- **Impact**: Agents skip plan step, go straight to code
- **Mitigation**: Make plan generation first response requirement, parse and validate before requesting code
- **Fallback**: Detect missing plans, provide stronger prompt

**Risk 4: Plan repair doesn't converge**
- **Impact**: Infinite plan repair loops
- **Mitigation**: Single-shot repair (same as code repair), track attempts
- **Fallback**: Fall back to direct code generation if plan fails

---

## Open Questions

1. **Should planning be mandatory or optional?**
   - Proposal: Optional via `--plan-first` flag initially, mandate after A/B test proves benefit

2. **How to handle benchmarks where planning doesn't help?**
   - Proposal: Track per-benchmark plan effectiveness, disable planning for simple benchmarks

3. **Should we scaffold code from plans or let AI generate directly?**
   - Proposal: AI generates code directly but references plan for structure (no automatic scaffolding in eval)

4. **How many plan repair attempts before giving up?**
   - Proposal: 1 repair attempt (same as code repair), then proceed with invalid plan or skip

---

## Future Enhancements (v0.3.4+)

1. **Plan Learning**: Store successful plans, use as few-shot examples
2. **Plan Templates**: Common benchmark patterns (CLI tool, API, data processor)
3. **Multi-step Planning**: Decompose complex benchmarks into sub-plans
4. **Plan Diffing**: Compare plans across attempts to identify systematic errors
5. **Interactive Planning**: AI-in-the-loop plan refinement during eval
6. **Plan Metrics Analysis**: Correlate plan characteristics with success rates

---

## References

- M-V3.2: Planning & Scaffolding Protocol (implemented)
- M-EVAL-LOOP Milestones 1-4 (implemented)
- `prompts/v0.3.2.md` (teaching prompt with planning)
- `internal/planning/validator.go` (validation logic)
- `internal/eval_harness/repair.go` (current repair runner)

---

## Appendix: Example Plan-First Eval Run

**Benchmark**: `fizzbuzz`

**Step 1: Generate Plan**
```json
{
  "schema": "ailang.plan/v1",
  "goal": "FizzBuzz implementation with IO output",
  "modules": [{
    "path": "benchmark/solution",
    "exports": ["main", "fizzbuzz"],
    "imports": ["std/io"]
  }],
  "types": [],
  "functions": [{
    "name": "fizzbuzz",
    "type": "(int) -> string",
    "effects": [],
    "module": "benchmark/solution"
  }, {
    "name": "main",
    "type": "() -> () ! {IO}",
    "effects": ["IO"],
    "module": "benchmark/solution"
  }],
  "effects": ["IO"]
}
```

**Step 2: Validate Plan**
```
✅ Plan is valid!
```

**Step 3: Generate Code**
```ailang
module benchmark/solution

import std/io (println)

export func fizzbuzz(n: int) -> string {
  if n % 15 == 0 then "FizzBuzz"
  else if n % 3 == 0 then "Fizz"
  else if n % 5 == 0 then "Buzz"
  else show(n)
}

export func main() -> () ! {IO} {
  let i = 1;
  -- (recursion for 1..100)
}
```

**Step 4: Compile & Execute**
```
✅ Success!
```

**Metrics:**
- PlanGenerated: true
- PlanValid: true
- PlanTokensIn: 250
- PlanTokensOut: 180
- CodeFromPlan: true
- FirstAttemptOk: true
- TotalTokens: 430 (vs ~600 for direct code generation)
