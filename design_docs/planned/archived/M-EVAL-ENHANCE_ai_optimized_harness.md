# M-EVAL-ENHANCE: AI-Optimized Evaluation Harness

**Status**: 📋 Planned
**Version**: 1.0
**Author**: AI-Assisted (Claude + User)
**Date**: 2025-10-05
**Milestone**: v0.3.0-alpha3+

---

## Executive Summary

The AILANG eval harness has proven invaluable for validating language features (M-R5 records) and measuring AI code generation efficiency. This enhancement plan makes it **the definitive tool for developing an AI-optimized programming language** by:

1. **Making results AI-consumable** - JSONL format, structured insights, error taxonomy
2. **Enabling rapid iteration** - Self-repair loops, prompt versioning, capability inference
3. **Generating marketing material** - HTML dashboards, comparison matrices, success metrics
4. **Ensuring reproducibility** - Prompt hashing, env fingerprinting, binary versioning
5. **Tracking language evolution** - Feature coverage, capability maturity, syntax confusion patterns

**Key Insight**: AILANG's eval harness is not just for benchmarking - it's a **feedback loop for AI-human collaboration** in language design.

---

## Motivation: Why This Matters

### Current State (v0.3.0-alpha2)
✅ **Works well:**
- Multi-model support (OpenAI, Gemini, Claude)
- Language comparison (AILANG vs Python)
- Separated input/output token tracking
- Execution timing (startup + compute)
- Structured JSON results

⚠️ **Pain points:**
- Results require manual JSON parsing by AI
- No failure pattern analysis
- Prompt evolution not tracked (hard to A/B test)
- Single-shot only (no retry with error feedback)
- No human-friendly visualization
- Missing reproducibility metadata
- Capability detection is manual

### Vision: AI-First Language Development

**The Problem**: Traditional language development optimizes for *human* programmers. AILANG optimizes for *AI-assisted* programming.

**The Solution**: The eval harness becomes a **co-development tool** where:
1. AI generates code → Harness measures success
2. Harness categorizes failures → AI learns patterns
3. Prompt evolves → Harness A/B tests versions
4. Language improves → Harness validates with self-repair metrics

**Example Workflow** (future):
```bash
# 1. Run baseline eval
ailang eval --model gpt-4o-mini --all-benchmarks

# 2. Analyze failures
cat eval_results/insights.json
# Output: "70% of AILANG failures are TC_REC_001 (record update syntax)"

# 3. Update prompt to clarify record update is NOT implemented
vim prompts/v0.3.0.md

# 4. Re-run with self-repair
ailang eval --model gpt-4o-mini --self-repair --all-benchmarks

# 5. Generate report
make eval-report
# Opens HTML dashboard: AILANG 85% → 95% success after prompt fix!

# 6. Commit improvement
git add prompts/v0.3.0.md eval_results/
git commit -m "Prompt v0.3.0 revision 2: Clarify record update syntax (95% success)"
```

---

## Design Goals

### 1. AI-Friendly Data Formats
**Goal**: Make eval results trivially consumable by AI assistants (Claude, GPT, Gemini)

**Why**: Currently, AI must parse JSON files one-by-one. A single JSONL file allows:
- Sequential reading without file system navigation
- Easy filtering (`grep "compile_ok.*false"`)
- Direct ingestion by AI training pipelines

**Formats**:
- `summary.jsonl` - One line per benchmark result (easy sequential parsing)
- `matrix.json` - Pivoted view for comparison (AILANG vs Python across models)
- `insights.json` - High-level analysis (success rates, token efficiency, error patterns)

### 2. Self-Repair Loop
**Goal**: Measure language UX by testing if AI can fix its own errors

**Why**: Single-shot success measures "Does AI know AILANG?" But **real-world coding is iterative**. Self-repair measures "Can AI *learn* AILANG from error messages?"

**Mechanism**:
```
1. AI generates code
2. If fails: Extract error type (TC_*, PAR_*, CAP_*)
3. Inject error-specific guidance into prompt
4. AI retries ONCE
5. Metrics track: first_attempt_ok, second_attempt_ok, repair_success
```

**Example**:
```
First attempt: TC_REC_001 (record update syntax)
Repair guidance: "Record update {r | field: val} is NOT implemented. Use new record literal."
Second attempt: ✅ Success
Metrics: first_attempt_ok=false, repair_success=true, repair_category=TC_REC_001
```

### 3. Prompt Versioning & A/B Testing
**Goal**: Track prompt evolution to measure what teaching strategies work

**Why**: We're learning how to *teach* AILANG to AI. Versioning lets us:
- Compare "v0.3.0 rev1" vs "v0.3.0 rev2" prompts
- Measure impact of adding examples vs syntax rules
- Identify which models benefit from which prompt styles

**Metadata**:
```json
{
  "prompt_version": "v0.3.0-rev2",
  "prompt_hash": "sha256:abc123...",
  "prompt_size_chars": 4521,
  "prompt_tokens_estimated": 1200
}
```

### 4. Reproducibility
**Goal**: Ensure results are scientifically valid

**Why**: AI models update, AILANG evolves, stdlib changes. We need:
- Exact prompt used
- AILANG binary version
- Stdlib version
- Model API version
- Seed for determinism

**Fingerprint**:
```json
{
  "ailang_version": "v0.3.0-alpha2",
  "binary_hash": "sha256:def456...",
  "stdlib_hash": "sha256:ghi789...",
  "model_api": "openai/gpt-4o-mini-2024-07-18",
  "seed": 42,
  "timestamp": "2025-10-05T20:52:00Z"
}
```

### 5. Human-Readable Dashboards
**Goal**: Make results accessible to non-technical stakeholders

**Why**: Marketing, investor relations, language design discussions need:
- Visual comparison of AILANG vs Python
- Success rate trends over time
- Token efficiency charts
- Error category breakdowns

**Output**: HTML dashboard with Chart.js visualizations

---

## Implementation Plan

### Phase A: Core Infrastructure (1-2 days)

#### A1: Extended RunMetrics Schema
**File**: `internal/eval_harness/metrics.go`

**Current**:
```go
type RunMetrics struct {
    ID            string
    Lang          string
    Model         string
    Seed          int64
    InputTokens   int
    OutputTokens  int
    TotalTokens   int
    CostUSD       float64
    CompileOk     bool
    RuntimeOk     bool
    StdoutOk      bool
    DurationMs    int64
    CompileMs     int64
    ExecuteMs     int64
    ErrorCategory string
    Stderr        string
    Timestamp     time.Time
    Code          string
}
```

**Enhanced**:
```go
type RunMetrics struct {
    // Existing fields...

    // Prompt Provenance (NEW)
    PromptVersion       string  `json:"prompt_version"`        // "v0.3.0-rev2"
    PromptHash          string  `json:"prompt_hash"`           // "sha256:abc123..."
    PromptSizeChars     int     `json:"prompt_size_chars"`     // 4521
    PromptTokensEst     int     `json:"prompt_tokens_est"`     // 1200

    // Reproducibility (NEW)
    AILangVersion       string  `json:"ailang_version"`        // "v0.3.0-alpha2"
    BinaryHash          string  `json:"binary_hash"`           // "sha256:def456..."
    StdlibHash          string  `json:"stdlib_hash"`           // "sha256:ghi789..."
    ModelAPI            string  `json:"model_api"`             // "openai/gpt-4o-mini-2024-07-18"
    EnvFingerprint      string  `json:"env_fingerprint"`       // "darwin_arm64_go1.22.1"

    // Self-Repair (NEW)
    FirstAttemptOk      bool    `json:"first_attempt_ok"`      // Did first attempt succeed?
    RepairAttempted     bool    `json:"repair_attempted"`      // Did we retry?
    RepairSuccess       bool    `json:"repair_success"`        // Did retry fix it?
    RepairCategory      string  `json:"repair_category"`       // Error type that triggered repair
    RepairGuidance      string  `json:"repair_guidance"`       // Guidance injected for repair

    // Capability Detection (NEW)
    DeclaredCaps        []string `json:"declared_caps"`        // ["IO", "FS"]
    InferredCaps        []string `json:"inferred_caps"`        // ["IO"] (detected in code)
    CapsMismatch        bool     `json:"caps_mismatch"`        // true if declared != inferred

    // Code Quality (NEW)
    CodeSizeBytes       int      `json:"code_size_bytes"`      // 342
    CodeLines           int      `json:"code_lines"`           // 18
    CodeComments        int      `json:"code_comments"`        // 0

    // Existing fields unchanged...
}
```

**Helpers**:
```go
// CalculatePromptHash computes SHA-256 hash of prompt text
func CalculatePromptHash(prompt string) string {
    h := sha256.Sum256([]byte(prompt))
    return fmt.Sprintf("sha256:%x", h)
}

// GetEnvFingerprint returns OS_ARCH_GO_VERSION
func GetEnvFingerprint() string {
    return fmt.Sprintf("%s_%s_%s", runtime.GOOS, runtime.GOARCH, runtime.Version())
}

// GetBinaryHash computes SHA-256 of ailang binary
func GetBinaryHash() (string, error) {
    execPath, err := os.Executable()
    if err != nil {
        return "", err
    }
    data, err := os.ReadFile(execPath)
    if err != nil {
        return "", err
    }
    h := sha256.Sum256(data)
    return fmt.Sprintf("sha256:%x", h), nil
}

// GetStdlibHash computes SHA-256 of stdlib/ directory contents
func GetStdlibHash() (string, error) {
    // Walk stdlib/, concatenate file hashes, hash result
    // Implementation: hash(sort(hash(file1) + hash(file2) + ...))
}

// InferCapabilities detects capabilities from generated code
func InferCapabilities(code string, lang string) []string {
    var caps []string
    if lang == "ailang" {
        if strings.Contains(code, "println") || strings.Contains(code, "print") {
            caps = append(caps, "IO")
        }
        if strings.Contains(code, "readFile") || strings.Contains(code, "writeFile") {
            caps = append(caps, "FS")
        }
    }
    return caps
}
```

#### A2: AI-Friendly Data Formats
**File**: `tools/report_eval.sh` (enhanced)

**New Format 1: summary.jsonl**
```jsonl
{"id":"records_person","lang":"ailang","model":"gpt-4o-mini","output_tokens":106,"compile_ok":true,"runtime_ok":true,"stdout_ok":true,"duration_ms":12,"prompt_version":"v0.3.0-rev1","first_attempt_ok":true}
{"id":"records_person","lang":"python","model":"gpt-4o-mini","output_tokens":125,"compile_ok":true,"runtime_ok":true,"stdout_ok":true,"duration_ms":51,"prompt_version":"python-v1","first_attempt_ok":true}
{"id":"recursion_fibonacci","lang":"ailang","model":"gpt-4o-mini","output_tokens":83,"compile_ok":true,"runtime_ok":true,"stdout_ok":true,"duration_ms":56,"prompt_version":"v0.3.0-rev1","first_attempt_ok":true}
{"id":"recursion_fibonacci","lang":"python","model":"gpt-4o-mini","output_tokens":76,"compile_ok":true,"runtime_ok":true,"stdout_ok":true,"duration_ms":44,"prompt_version":"python-v1","first_attempt_ok":true}
```

**Why JSONL?**
- AI can read sequentially without JSON parsing overhead
- Easy grep/awk filtering: `grep '"compile_ok":false' summary.jsonl`
- Append-only (no need to rewrite entire file)
- Streaming-friendly for large eval runs

**New Format 2: matrix.json**
```json
{
  "meta": {
    "generated_at": "2025-10-05T21:30:00Z",
    "ailang_version": "v0.3.0-alpha2",
    "models": ["gpt-4o-mini", "gemini-2.0-flash-exp", "claude-sonnet-4.5"],
    "benchmarks": ["records_person", "recursion_fibonacci", "recursion_factorial"]
  },
  "comparison": {
    "records_person": {
      "ailang": {
        "gpt-4o-mini": {"success": true, "output_tokens": 106, "duration_ms": 12},
        "gemini-2.0-flash-exp": {"success": true, "output_tokens": 98, "duration_ms": 15},
        "claude-sonnet-4.5": {"success": true, "output_tokens": 102, "duration_ms": 18}
      },
      "python": {
        "gpt-4o-mini": {"success": true, "output_tokens": 125, "duration_ms": 51},
        "gemini-2.0-flash-exp": {"success": true, "output_tokens": 118, "duration_ms": 53},
        "claude-sonnet-4.5": {"success": true, "output_tokens": 122, "duration_ms": 68}
      }
    }
  },
  "aggregates": {
    "ailang": {
      "avg_output_tokens": 95.67,
      "avg_duration_ms": 15.0,
      "success_rate": 1.0
    },
    "python": {
      "avg_output_tokens": 121.67,
      "avg_duration_ms": 57.33,
      "success_rate": 1.0
    }
  }
}
```

**Why matrix.json?**
- Single file for AI to understand all comparisons
- Easy to generate charts (models × benchmarks × languages)
- Aggregates pre-calculated for quick insights

**New Format 3: insights.json**
```json
{
  "meta": {
    "generated_at": "2025-10-05T21:30:00Z",
    "ailang_version": "v0.3.0-alpha2",
    "total_runs": 18
  },
  "key_findings": {
    "token_efficiency": {
      "ailang_avg_output": 95.67,
      "python_avg_output": 121.67,
      "reduction_pct": 21.3,
      "verdict": "AILANG generates 21% fewer output tokens"
    },
    "execution_speed": {
      "ailang_avg_ms": 15.0,
      "python_avg_ms": 57.33,
      "speedup": 3.82,
      "verdict": "AILANG executes 3.8x faster (due to faster startup)"
    },
    "success_rates": {
      "ailang_first_attempt": 1.0,
      "ailang_with_repair": 1.0,
      "python_first_attempt": 1.0,
      "python_with_repair": 1.0
    },
    "error_patterns": [
      {
        "category": "TC_REC_001",
        "description": "Record update syntax {r | field: val} not implemented",
        "count": 0,
        "models_affected": []
      }
    ],
    "prompt_efficiency": {
      "avg_prompt_tokens_ailang": 1200,
      "avg_prompt_tokens_python": 150,
      "note": "AILANG requires teaching prompt; will be optimized via fine-tuning"
    }
  },
  "recommendations": [
    "✅ M-R5 records implementation validated - 100% success across all models",
    "✅ Token efficiency advantage confirmed (21% reduction)",
    "✅ Execution speed advantage confirmed (3.8x faster startup)",
    "📊 Consider fine-tuning models on AILANG syntax to reduce prompt tokens"
  ]
}
```

**Why insights.json?**
- AI can read high-level summary without parsing all metrics
- Human-readable recommendations
- Marketing-ready bullet points

#### A3: Capability Auto-Detection
**File**: `internal/eval_harness/runner.go`

**Current**: Capabilities manually specified in benchmark YAML:
```yaml
caps: ["IO", "FS"]
```

**Enhancement**: Auto-detect capabilities from generated code, compare to declared:
```go
func (r *AILangRunner) Run(code string, declaredCaps []string) (*RunResult, error) {
    // Infer capabilities from code
    inferredCaps := InferCapabilities(code, "ailang")

    // Check for mismatch
    capsMismatch := !slicesEqual(declaredCaps, inferredCaps)

    // If mismatch, log warning
    if capsMismatch {
        log.Printf("⚠️  Capability mismatch: declared=%v inferred=%v", declaredCaps, inferredCaps)
    }

    // Run with inferred capabilities (override declared)
    actualCaps := inferredCaps
    if len(declaredCaps) > 0 {
        // If benchmark specifies caps, respect them (for testing)
        actualCaps = declaredCaps
    }

    // Execute code with actual capabilities
    // ...

    result.DeclaredCaps = declaredCaps
    result.InferredCaps = inferredCaps
    result.CapsMismatch = capsMismatch

    return result, nil
}
```

**Why?**
- Catches AI errors: "Forgot to add IO capability but code uses println"
- Validates benchmark specs: "YAML says IO but code doesn't use it"
- Future: Auto-generate capability declarations in benchmarks

#### A4: Prompt Versioning System
**File**: `prompts/versions.json`

**New Metadata File**:
```json
{
  "prompts": {
    "ailang": {
      "current": "v0.3.0-rev2",
      "versions": {
        "v0.2.0": {
          "file": "v0.2.0.md",
          "hash": "sha256:abc123...",
          "created": "2025-09-28T10:00:00Z",
          "features": ["recursion", "blocks", "effects"]
        },
        "v0.3.0-rev1": {
          "file": "v0.3.0.md",
          "hash": "sha256:def456...",
          "created": "2025-10-05T12:00:00Z",
          "features": ["records"],
          "notes": "Initial v0.3.0 prompt - includes record update syntax (incorrect)"
        },
        "v0.3.0-rev2": {
          "file": "v0.3.0.md",
          "hash": "sha256:ghi789...",
          "created": "2025-10-05T18:00:00Z",
          "features": ["records"],
          "notes": "Clarified record update NOT implemented - improved success rate"
        }
      }
    },
    "python": {
      "current": "python-v1",
      "versions": {
        "python-v1": {
          "file": "python.md",
          "hash": "sha256:jkl012...",
          "created": "2025-10-05T12:00:00Z"
        }
      }
    }
  }
}
```

**Usage**:
```bash
# Use specific prompt version
ailang eval --benchmark records_person --prompt-version v0.3.0-rev1

# Compare prompt versions
ailang eval --benchmark records_person --prompt-version v0.3.0-rev1 --output-dir eval_results/rev1
ailang eval --benchmark records_person --prompt-version v0.3.0-rev2 --output-dir eval_results/rev2
tools/compare_prompts.sh eval_results/rev1 eval_results/rev2

# Output:
# Prompt v0.3.0-rev1: 85% success (6/7 benchmarks)
# Prompt v0.3.0-rev2: 95% success (20/21 benchmarks)
# Improvement: +10% success rate
```

**Implementation**:
```go
// LoadPromptVersion loads a specific prompt version
func LoadPromptVersion(lang, version string) (string, error) {
    meta, err := loadVersionsMetadata("prompts/versions.json")
    if err != nil {
        return "", err
    }

    langMeta, ok := meta.Prompts[lang]
    if !ok {
        return "", fmt.Errorf("no prompts for language %s", lang)
    }

    verMeta, ok := langMeta.Versions[version]
    if !ok {
        return "", fmt.Errorf("prompt version %s not found", version)
    }

    path := filepath.Join("prompts", verMeta.File)
    data, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }

    // Verify hash
    actualHash := CalculatePromptHash(string(data))
    if actualHash != verMeta.Hash {
        return "", fmt.Errorf("prompt hash mismatch: expected %s, got %s", verMeta.Hash, actualHash)
    }

    return string(data), nil
}
```

---

### Phase B: Analysis & Self-Repair (2-3 days)

#### B1: Error Taxonomy System
**File**: `internal/eval_harness/errors.go` (NEW)

**Error Categories**:
```go
const (
    // Type checker errors
    ErrorTC_REC_001 = "TC_REC_001" // Record update syntax not implemented
    ErrorTC_REC_002 = "TC_REC_002" // Record field not found
    ErrorTC_TYP_001 = "TC_TYP_001" // Type mismatch
    ErrorTC_TYP_002 = "TC_TYP_002" // Unbound type variable

    // Parser errors
    ErrorPAR_001 = "PAR_001" // Syntax error
    ErrorPAR_002 = "PAR_002" // Unexpected token
    ErrorPAR_003 = "PAR_003" // Missing semicolon in block

    // Capability errors
    ErrorCAP_001 = "CAP_001" // Missing capability (IO, FS)
    ErrorCAP_002 = "CAP_002" // Capability not declared in signature

    // Runtime errors
    ErrorRUN_001 = "RUN_001" // Panic / crash
    ErrorRUN_002 = "RUN_002" // Infinite recursion (depth exceeded)

    // Logic errors
    ErrorLOG_001 = "LOG_001" // Wrong output
    ErrorLOG_002 = "LOG_002" // Missing output
)
```

**Error Detection**:
```go
// CategorizeError maps error message to taxonomy code
func CategorizeError(stderr string, compileOk, runtimeOk, stdoutOk bool) string {
    switch {
    case !compileOk:
        // Parse stderr for specific patterns
        if strings.Contains(stderr, "record update") {
            return ErrorTC_REC_001
        }
        if strings.Contains(stderr, "field not found") {
            return ErrorTC_REC_002
        }
        if strings.Contains(stderr, "syntax error") {
            return ErrorPAR_001
        }
        if strings.Contains(stderr, "missing semicolon") {
            return ErrorPAR_003
        }
        if strings.Contains(stderr, "capability") {
            return ErrorCAP_001
        }
        return "COMPILE_ERROR_UNKNOWN"

    case !runtimeOk:
        if strings.Contains(stderr, "panic") {
            return ErrorRUN_001
        }
        if strings.Contains(stderr, "recursion depth") {
            return ErrorRUN_002
        }
        return "RUNTIME_ERROR_UNKNOWN"

    case !stdoutOk:
        return ErrorLOG_001

    default:
        return "NONE"
    }
}
```

**Error Guidance Map**:
```go
var RepairGuidance = map[string]string{
    ErrorTC_REC_001: `
⚠️ ERROR: Record update syntax {r | field: val} is NOT yet implemented in AILANG v0.3.0.
INSTEAD: Create a new record literal with all fields:
  {name: r.name, age: 31, city: r.city}
`,
    ErrorPAR_003: `
⚠️ ERROR: Block expressions require semicolons between statements.
SYNTAX: { stmt1; stmt2; stmt3 }
EXAMPLE: { println("Hello"); println("World"); 42 }
`,
    ErrorCAP_001: `
⚠️ ERROR: Your code uses IO/FS operations but the function signature doesn't declare effects.
FIX: Add ! {IO} or ! {FS} to function signature:
  func main() -> () ! {IO} { ... }
`,
}
```

#### B2: Single-Shot Self-Repair Loop
**File**: `cmd/ailang/eval.go` (enhanced)

**New Flag**:
```go
var selfRepair bool
rootCmd.PersistentFlags().BoolVar(&selfRepair, "self-repair", false, "Enable single-shot self-repair on errors")
```

**Implementation**:
```go
func runEval(spec *BenchmarkSpec, lang, model string, seed int64) *RunMetrics {
    // First attempt
    metrics := runSingleAttempt(spec, lang, model, seed, "")
    metrics.FirstAttemptOk = metrics.CompileOk && metrics.RuntimeOk && metrics.StdoutOk

    // If first attempt failed and self-repair enabled
    if !metrics.FirstAttemptOk && selfRepair {
        fmt.Printf("  ⚠️  First attempt failed: %s\n", metrics.ErrorCategory)

        // Get repair guidance
        guidance, ok := RepairGuidance[metrics.ErrorCategory]
        if ok {
            fmt.Printf("  🔧 Attempting self-repair with error-specific guidance...\n")

            // Retry with guidance injected into prompt
            repairMetrics := runSingleAttempt(spec, lang, model, seed, guidance)

            // Update metrics
            metrics.RepairAttempted = true
            metrics.RepairSuccess = repairMetrics.CompileOk && repairMetrics.RuntimeOk && repairMetrics.StdoutOk
            metrics.RepairCategory = metrics.ErrorCategory
            metrics.RepairGuidance = guidance

            // Use repair metrics if successful
            if metrics.RepairSuccess {
                fmt.Printf("  ✅ Self-repair succeeded!\n")
                // Keep first attempt data, but mark as repaired
                metrics.CompileOk = repairMetrics.CompileOk
                metrics.RuntimeOk = repairMetrics.RuntimeOk
                metrics.StdoutOk = repairMetrics.StdoutOk
                metrics.Code = repairMetrics.Code
                metrics.OutputTokens += repairMetrics.OutputTokens
                metrics.TotalTokens += repairMetrics.TotalTokens
            } else {
                fmt.Printf("  ❌ Self-repair failed\n")
            }
        } else {
            fmt.Printf("  ⚠️  No repair guidance for error category: %s\n", metrics.ErrorCategory)
        }
    }

    return metrics
}

func runSingleAttempt(spec *BenchmarkSpec, lang, model string, seed int64, repairGuidance string) *RunMetrics {
    // Generate prompt
    prompt := spec.PromptForLanguage(lang)

    // Inject repair guidance if provided
    if repairGuidance != "" {
        prompt = prompt + "\n\n## Previous Attempt Failed\n" + repairGuidance
    }

    // Generate code
    result, err := agent.Generate(prompt, model, seed)
    // ... rest of evaluation logic
}
```

**Self-Repair Metrics**:
```json
{
  "first_attempt_ok": false,
  "repair_attempted": true,
  "repair_success": true,
  "repair_category": "TC_REC_001",
  "repair_guidance": "⚠️ ERROR: Record update syntax...",
  "output_tokens": 106,  // First attempt
  "total_tokens": 212    // First + repair attempt
}
```

**Analysis**:
- `first_attempt_ok=true`: AI knows AILANG well
- `repair_success=true`: AI can learn from error messages (good UX)
- `repair_attempted=false`: No known repair guidance for this error (taxonomy gap)

#### B3: Insights Generator
**File**: `tools/generate_insights.go` (NEW)

**Purpose**: Analyze all eval results and generate `insights.json`

**Implementation**:
```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
)

type Insights struct {
    Meta         Meta                `json:"meta"`
    KeyFindings  KeyFindings         `json:"key_findings"`
    ErrorPatterns []ErrorPattern     `json:"error_patterns"`
    Recommendations []string        `json:"recommendations"`
}

func main() {
    resultsDir := os.Args[1] // eval_results/

    // Load all JSON results
    metrics := loadAllMetrics(resultsDir)

    // Analyze
    insights := Insights{
        Meta: Meta{
            GeneratedAt:   time.Now(),
            AILangVersion: metrics[0].AILangVersion,
            TotalRuns:     len(metrics),
        },
        KeyFindings: analyzeKeyFindings(metrics),
        ErrorPatterns: analyzeErrorPatterns(metrics),
        Recommendations: generateRecommendations(metrics),
    }

    // Write insights.json
    data, _ := json.MarshalIndent(insights, "", "  ")
    os.WriteFile(filepath.Join(resultsDir, "insights.json"), data, 0644)
}

func analyzeKeyFindings(metrics []RunMetrics) KeyFindings {
    var ailangOutputTokens, pythonOutputTokens int
    var ailangDuration, pythonDuration int64
    var ailangFirstAttempt, pythonFirstAttempt int
    var ailangRepairSuccess, pythonRepairSuccess int

    for _, m := range metrics {
        if m.Lang == "ailang" {
            ailangOutputTokens += m.OutputTokens
            ailangDuration += m.DurationMs
            if m.FirstAttemptOk {
                ailangFirstAttempt++
            }
            if m.RepairSuccess {
                ailangRepairSuccess++
            }
        } else {
            pythonOutputTokens += m.OutputTokens
            pythonDuration += m.DurationMs
            if m.FirstAttemptOk {
                pythonFirstAttempt++
            }
            if m.RepairSuccess {
                pythonRepairSuccess++
            }
        }
    }

    ailangCount := len(metrics) / 2

    return KeyFindings{
        TokenEfficiency: TokenEfficiency{
            AILangAvg: float64(ailangOutputTokens) / float64(ailangCount),
            PythonAvg: float64(pythonOutputTokens) / float64(ailangCount),
            ReductionPct: (1 - float64(ailangOutputTokens)/float64(pythonOutputTokens)) * 100,
        },
        ExecutionSpeed: ExecutionSpeed{
            AILangAvg: float64(ailangDuration) / float64(ailangCount),
            PythonAvg: float64(pythonDuration) / float64(ailangCount),
            Speedup: float64(pythonDuration) / float64(ailangDuration),
        },
        SuccessRates: SuccessRates{
            AILangFirstAttempt: float64(ailangFirstAttempt) / float64(ailangCount),
            PythonFirstAttempt: float64(pythonFirstAttempt) / float64(ailangCount),
        },
    }
}

func analyzeErrorPatterns(metrics []RunMetrics) []ErrorPattern {
    errorCounts := make(map[string]int)
    errorModels := make(map[string]map[string]bool)

    for _, m := range metrics {
        if m.ErrorCategory != "NONE" && m.ErrorCategory != "" {
            errorCounts[m.ErrorCategory]++
            if errorModels[m.ErrorCategory] == nil {
                errorModels[m.ErrorCategory] = make(map[string]bool)
            }
            errorModels[m.ErrorCategory][m.Model] = true
        }
    }

    var patterns []ErrorPattern
    for category, count := range errorCounts {
        var models []string
        for model := range errorModels[category] {
            models = append(models, model)
        }

        patterns = append(patterns, ErrorPattern{
            Category:    category,
            Description: getErrorDescription(category),
            Count:       count,
            Models:      models,
        })
    }

    return patterns
}

func generateRecommendations(metrics []RunMetrics) []string {
    var recs []string

    // Check success rate
    successCount := 0
    for _, m := range metrics {
        if m.StdoutOk {
            successCount++
        }
    }
    successRate := float64(successCount) / float64(len(metrics))

    if successRate >= 0.9 {
        recs = append(recs, "✅ High success rate - language feature validated")
    } else {
        recs = append(recs, fmt.Sprintf("⚠️ Success rate only %.0f%% - review error patterns", successRate*100))
    }

    // Check token efficiency
    // ... (implementation similar to KeyFindings)

    return recs
}
```

**Usage**:
```bash
make eval-insights
# Generates eval_results/insights.json

cat eval_results/insights.json | jq '.key_findings.token_efficiency'
```

---

### Phase C: Dashboard & Visualization (Ongoing)

#### C1: HTML Dashboard
**File**: `tools/generate_dashboard.sh` (NEW)

**Purpose**: Generate human-readable HTML report with charts

**Template** (`tools/dashboard_template.html`):
```html
<!DOCTYPE html>
<html>
<head>
    <title>AILANG Eval Dashboard</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body { font-family: system-ui, -apple-system, sans-serif; margin: 2rem; }
        .metric { display: inline-block; margin: 1rem; padding: 1rem; border: 1px solid #ccc; }
        .metric-value { font-size: 2rem; font-weight: bold; }
        .chart-container { width: 600px; height: 400px; margin: 2rem 0; }
    </style>
</head>
<body>
    <h1>AILANG vs Python Evaluation</h1>
    <p><strong>Version:</strong> {{AILANG_VERSION}} | <strong>Date:</strong> {{DATE}}</p>

    <h2>Key Metrics</h2>
    <div class="metric">
        <div>Token Efficiency</div>
        <div class="metric-value">{{TOKEN_REDUCTION}}%</div>
        <div>reduction</div>
    </div>
    <div class="metric">
        <div>Success Rate</div>
        <div class="metric-value">{{SUCCESS_RATE}}%</div>
        <div>first attempt</div>
    </div>
    <div class="metric">
        <div>Execution Speed</div>
        <div class="metric-value">{{SPEEDUP}}x</div>
        <div>faster</div>
    </div>

    <h2>Token Usage by Benchmark</h2>
    <div class="chart-container">
        <canvas id="tokenChart"></canvas>
    </div>

    <h2>Success Rates by Model</h2>
    <div class="chart-container">
        <canvas id="successChart"></canvas>
    </div>

    <script>
        // Token chart
        new Chart(document.getElementById('tokenChart'), {
            type: 'bar',
            data: {
                labels: {{BENCHMARK_NAMES}},
                datasets: [{
                    label: 'AILANG',
                    data: {{AILANG_TOKENS}},
                    backgroundColor: 'rgba(75, 192, 192, 0.6)'
                }, {
                    label: 'Python',
                    data: {{PYTHON_TOKENS}},
                    backgroundColor: 'rgba(255, 99, 132, 0.6)'
                }]
            },
            options: {
                responsive: true,
                scales: { y: { beginAtZero: true } }
            }
        });

        // Success chart
        new Chart(document.getElementById('successChart'), {
            type: 'bar',
            data: {
                labels: {{MODEL_NAMES}},
                datasets: [{
                    label: 'AILANG First Attempt',
                    data: {{AILANG_SUCCESS}},
                    backgroundColor: 'rgba(75, 192, 192, 0.6)'
                }, {
                    label: 'AILANG After Repair',
                    data: {{AILANG_REPAIR_SUCCESS}},
                    backgroundColor: 'rgba(54, 162, 235, 0.6)'
                }]
            },
            options: {
                responsive: true,
                scales: { y: { beginAtZero: true, max: 100 } }
            }
        });
    </script>
</body>
</html>
```

**Generator Script**:
```bash
#!/usr/bin/env bash
# tools/generate_dashboard.sh

set -euo pipefail

RESULTS_DIR="${1:-eval_results}"
INSIGHTS="${RESULTS_DIR}/insights.json"
TEMPLATE="tools/dashboard_template.html"
OUTPUT="${RESULTS_DIR}/dashboard.html"

# Extract values from insights.json
TOKEN_REDUCTION=$(jq -r '.key_findings.token_efficiency.reduction_pct' "$INSIGHTS")
SUCCESS_RATE=$(jq -r '.key_findings.success_rates.ailang_first_attempt * 100' "$INSIGHTS")
SPEEDUP=$(jq -r '.key_findings.execution_speed.speedup' "$INSIGHTS")
AILANG_VERSION=$(jq -r '.meta.ailang_version' "$INSIGHTS")
DATE=$(jq -r '.meta.generated_at' "$INSIGHTS")

# Extract chart data from matrix.json
MATRIX="${RESULTS_DIR}/matrix.json"
BENCHMARK_NAMES=$(jq -r '.meta.benchmarks | @json' "$MATRIX")
AILANG_TOKENS=$(jq -r '[.comparison[] | .ailang | to_entries[] | .value.output_tokens] | @json' "$MATRIX")
PYTHON_TOKENS=$(jq -r '[.comparison[] | .python | to_entries[] | .value.output_tokens] | @json' "$MATRIX")
MODEL_NAMES=$(jq -r '.meta.models | @json' "$MATRIX")

# Generate HTML
sed "s|{{AILANG_VERSION}}|${AILANG_VERSION}|g" "$TEMPLATE" | \
sed "s|{{DATE}}|${DATE}|g" | \
sed "s|{{TOKEN_REDUCTION}}|${TOKEN_REDUCTION}|g" | \
sed "s|{{SUCCESS_RATE}}|${SUCCESS_RATE}|g" | \
sed "s|{{SPEEDUP}}|${SPEEDUP}|g" | \
sed "s|{{BENCHMARK_NAMES}}|${BENCHMARK_NAMES}|g" | \
sed "s|{{AILANG_TOKENS}}|${AILANG_TOKENS}|g" | \
sed "s|{{PYTHON_TOKENS}}|${PYTHON_TOKENS}|g" | \
sed "s|{{MODEL_NAMES}}|${MODEL_NAMES}|g" \
> "$OUTPUT"

echo "✓ Dashboard generated: $OUTPUT"
echo "Open in browser: open $OUTPUT"
```

**Usage**:
```bash
make eval-dashboard
# Generates eval_results/dashboard.html

open eval_results/dashboard.html
```

#### C2: Marketing Material Generator
**File**: `tools/generate_marketing.sh` (NEW)

**Purpose**: Generate social media-ready snippets

**Output**:
```markdown
# AILANG vs Python: AI Code Generation Efficiency

**TL;DR**: AILANG generates **21% fewer tokens** and executes **3.8x faster** than Python when AI writes the code.

## Key Metrics

- ✅ **100% Success Rate** across GPT-4o-mini, Gemini 2.0, Claude 4.5
- 📉 **21% Token Reduction** (95 tokens vs 121 for Python)
- ⚡ **3.8x Faster Execution** (15ms vs 57ms average)
- 🎯 **First-Attempt Success** on all benchmarks

## Why This Matters

AILANG is designed for **AI-assisted programming**. When Claude, GPT, or Gemini write AILANG code:
- They write **less code** (fewer tokens = lower API costs)
- It runs **faster** (optimized runtime)
- It's **more correct** (type safety catches errors early)

## Example: Record Types

**Python** (125 tokens):
```python
def main():
    alice = {"name": "Alice", "age": 30, "city": "NYC"}
    bob = {"name": "Bob", "age": 25, "city": "SF"}
    print(f"{alice['name']}, {alice['age']}, {alice['city']}")
    print(f"{bob['name']}, {bob['age']}, {bob['city']}")
```

**AILANG** (106 tokens, 15% reduction):
```ailang
export func main() -> () ! {IO} {
  let alice = {name: "Alice", age: 30, city: "NYC"};
  let bob = {name: "Bob", age: 25, city: "SF"};
  println(alice.name ++ ", " ++ show(alice.age) ++ ", " ++ alice.city);
  println(bob.name ++ ", " ++ show(bob.age) ++ ", " ++ bob.city)
}
```

**Result**: Fewer tokens, type-safe field access, explicit effects.

---

*Generated from AILANG eval harness v0.3.0-alpha2*
```

**Usage**:
```bash
tools/generate_marketing.sh eval_results/ > MARKETING.md
```

---

## File Structure (After Implementation)

```
ailang/
├── internal/
│   └── eval_harness/
│       ├── metrics.go          # EXTENDED: Prompt/repro/repair/caps fields
│       ├── errors.go           # NEW: Error taxonomy + repair guidance
│       ├── ai_agent.go         # ENHANCED: Token separation
│       ├── runner.go           # ENHANCED: Capability auto-detection
│       └── spec.go             # EXISTING: Load benchmarks
│
├── prompts/
│   ├── versions.json           # NEW: Prompt version metadata
│   ├── v0.2.0.md               # EXISTING: v0.2.0 prompt
│   ├── v0.3.0.md               # EXISTING: v0.3.0 prompt (current)
│   └── python.md               # EXISTING: Python prompt
│
├── tools/
│   ├── report_eval.sh          # ENHANCED: Generate JSONL, matrix, insights
│   ├── generate_insights.go    # NEW: Analyze results → insights.json
│   ├── generate_dashboard.sh   # NEW: Create HTML dashboard
│   ├── generate_marketing.sh   # NEW: Marketing snippets
│   ├── dashboard_template.html # NEW: HTML template
│   └── compare_prompts.sh      # NEW: A/B test prompt versions
│
├── eval_results/               # Output directory
│   ├── *.json                  # Individual run metrics (EXTENDED schema)
│   ├── summary.csv             # EXISTING: CSV report
│   ├── summary.jsonl           # NEW: JSONL for AI consumption
│   ├── matrix.json             # NEW: Pivoted comparison view
│   ├── insights.json           # NEW: High-level analysis
│   ├── leaderboard.md          # EXISTING: Markdown report
│   └── dashboard.html          # NEW: Interactive HTML dashboard
│
├── benchmarks/
│   ├── records_person.yml      # EXISTING: Record benchmark
│   ├── recursion_fibonacci.yml # EXISTING: Recursion benchmark
│   └── ...                     # Other benchmarks
│
└── cmd/ailang/
    └── eval.go                 # ENHANCED: Self-repair loop, prompt versioning
```

---

## Usage Examples

### Basic Evaluation (No changes to existing workflow)
```bash
ailang eval --benchmark records_person --model gpt-4o-mini --seed 42
```

### Self-Repair Mode (NEW)
```bash
ailang eval --benchmark records_person --model gpt-4o-mini --self-repair
```

### Prompt Version Testing (NEW)
```bash
# Test current prompt
ailang eval --benchmark records_person --prompt-version v0.3.0-rev2

# Test previous prompt
ailang eval --benchmark records_person --prompt-version v0.3.0-rev1 --output-dir eval_results/rev1

# Compare
tools/compare_prompts.sh eval_results/ eval_results/rev1
```

### Generate All Reports (NEW)
```bash
# Run evals
make eval-all

# Generate reports
make eval-report          # CSV + Markdown (existing)
make eval-insights        # insights.json (new)
make eval-dashboard       # dashboard.html (new)
make eval-marketing       # MARKETING.md (new)

# Open dashboard
open eval_results/dashboard.html
```

### CI Integration (NEW)
```bash
# In .github/workflows/eval.yml
- name: Run Eval
  run: |
    ailang eval --all-benchmarks --model gpt-4o-mini --self-repair
    make eval-insights

- name: Check Success Rate
  run: |
    SUCCESS_RATE=$(jq -r '.key_findings.success_rates.ailang_first_attempt' eval_results/insights.json)
    if (( $(echo "$SUCCESS_RATE < 0.9" | bc -l) )); then
      echo "❌ Success rate below 90%: $SUCCESS_RATE"
      exit 1
    fi

- name: Comment on PR
  run: |
    gh pr comment ${{ github.event.pull_request.number }} --body-file eval_results/leaderboard.md
```

---

## Timeline & Effort Estimates

### Phase A: Core Infrastructure (1-2 days)
- **A1: Extended RunMetrics Schema** - 2-3 hours
  - Add new fields to `metrics.go`
  - Implement helper functions (hashing, fingerprinting)
  - Update JSON serialization
  - Update tests

- **A2: AI-Friendly Data Formats** - 3-4 hours
  - Enhance `report_eval.sh` to generate JSONL
  - Create `matrix.json` generator
  - Create `insights.json` generator (basic version)
  - Update Makefile targets

- **A3: Capability Auto-Detection** - 1-2 hours
  - Implement `InferCapabilities()` in `runner.go`
  - Add mismatch detection and logging
  - Update tests

- **A4: Prompt Versioning System** - 2-3 hours
  - Create `prompts/versions.json`
  - Implement version loading in `spec.go`
  - Add `--prompt-version` flag
  - Create `compare_prompts.sh` script

**Total Phase A**: ~8-12 hours (1-2 days)

### Phase B: Analysis & Self-Repair (2-3 days)
- **B1: Error Taxonomy System** - 4-5 hours
  - Define error constants in `errors.go`
  - Implement pattern matching in `CategorizeError()`
  - Create repair guidance map
  - Test with existing failures

- **B2: Single-Shot Self-Repair Loop** - 4-5 hours
  - Add `--self-repair` flag
  - Implement retry logic in `eval.go`
  - Update metrics to track repair attempts
  - Test with intentionally broken code

- **B3: Insights Generator** - 3-4 hours
  - Create `generate_insights.go`
  - Implement analysis functions
  - Generate recommendations
  - Test with real eval data

**Total Phase B**: ~11-14 hours (2-3 days)

### Phase C: Dashboard & Visualization (Ongoing)
- **C1: HTML Dashboard** - 4-6 hours
  - Create HTML template
  - Implement `generate_dashboard.sh`
  - Add Chart.js visualizations
  - Test with multiple benchmark runs

- **C2: Marketing Material Generator** - 2-3 hours
  - Create `generate_marketing.sh`
  - Design output format (Markdown + stats)
  - Test with real data

**Total Phase C**: ~6-9 hours (1 day)

### **Grand Total**: ~25-35 hours (~4-6 days of development)

---

## Success Metrics

### Quantitative Metrics
1. **AI Consumption Efficiency**
   - Baseline: AI must read 10+ JSON files individually
   - Target: AI reads 1 JSONL file sequentially
   - Metric: File reads per analysis

2. **Prompt Iteration Speed**
   - Baseline: Manual comparison of before/after
   - Target: Automated A/B testing with `compare_prompts.sh`
   - Metric: Time to validate prompt changes (minutes vs hours)

3. **Self-Repair Success Rate**
   - Baseline: 0% (no retry mechanism)
   - Target: 50%+ of failures fixed by single retry
   - Metric: `repair_success / (total_runs - first_attempt_ok)`

4. **Error Taxonomy Coverage**
   - Baseline: Generic "compile_error" / "runtime_error"
   - Target: 80%+ of failures mapped to specific error codes
   - Metric: Categorized errors / total errors

### Qualitative Metrics
1. **Human Readability**
   - Can a non-technical stakeholder understand dashboard?
   - Can marketing use generated snippets without editing?

2. **AI Insights Quality**
   - Can AI (Claude, GPT) read `insights.json` and make design recommendations?
   - Does JSONL reduce token usage in AI conversations?

3. **Developer Productivity**
   - Does self-repair reveal UX issues faster?
   - Do error categories guide documentation improvements?

---

## Risk Assessment

### Technical Risks

**Risk 1: Prompt Hashing Instability**
- **Issue**: Hash changes if whitespace/formatting changes
- **Mitigation**: Normalize prompts before hashing (trim, lowercase, etc.)
- **Severity**: Low (affects versioning, not functionality)

**Risk 2: Self-Repair False Positives**
- **Issue**: Retry succeeds by chance, not due to guidance
- **Mitigation**: Track repair category; analyze which guidance helps
- **Severity**: Low (over-reporting success, not breaking anything)

**Risk 3: Error Taxonomy Maintenance**
- **Issue**: New error types emerge as language evolves
- **Mitigation**: Use "UNKNOWN" category for unmapped errors; review regularly
- **Severity**: Medium (incomplete taxonomy reduces self-repair effectiveness)

**Risk 4: Capability Inference Accuracy**
- **Issue**: False positives (detecting "print" in string literal)
- **Mitigation**: Use AST parsing instead of string matching
- **Severity**: Medium (incorrect capability detection causes test failures)

### Process Risks

**Risk 5: Complexity Creep**
- **Issue**: Dashboard becomes too complex to maintain
- **Mitigation**: Keep Phase C minimal; prioritize JSONL/insights over fancy UI
- **Severity**: Low (Phase C is optional/ongoing)

**Risk 6: Backward Compatibility**
- **Issue**: New schema breaks existing tools
- **Mitigation**: Make new fields optional; test with old benchmarks
- **Severity**: Low (we control all tooling)

---

## Appendix: Example Data

### Example: summary.jsonl (3 runs)
```jsonl
{"id":"records_person","lang":"ailang","model":"gpt-4o-mini","seed":42,"output_tokens":106,"first_attempt_ok":true,"prompt_version":"v0.3.0-rev2","prompt_hash":"sha256:abc123","ailang_version":"v0.3.0-alpha2","duration_ms":12}
{"id":"records_person","lang":"python","model":"gpt-4o-mini","seed":42,"output_tokens":125,"first_attempt_ok":true,"prompt_version":"python-v1","prompt_hash":"sha256:def456","ailang_version":"v0.3.0-alpha2","duration_ms":51}
{"id":"recursion_fibonacci","lang":"ailang","model":"gpt-4o-mini","seed":42,"output_tokens":83,"first_attempt_ok":true,"prompt_version":"v0.3.0-rev2","prompt_hash":"sha256:abc123","ailang_version":"v0.3.0-alpha2","duration_ms":56}
```

### Example: matrix.json (2 benchmarks × 2 langs × 1 model)
```json
{
  "meta": {
    "generated_at": "2025-10-05T21:30:00Z",
    "ailang_version": "v0.3.0-alpha2",
    "models": ["gpt-4o-mini"],
    "benchmarks": ["records_person", "recursion_fibonacci"]
  },
  "comparison": {
    "records_person": {
      "ailang": {
        "gpt-4o-mini": {"success": true, "output_tokens": 106, "duration_ms": 12}
      },
      "python": {
        "gpt-4o-mini": {"success": true, "output_tokens": 125, "duration_ms": 51}
      }
    },
    "recursion_fibonacci": {
      "ailang": {
        "gpt-4o-mini": {"success": true, "output_tokens": 83, "duration_ms": 56}
      },
      "python": {
        "gpt-4o-mini": {"success": true, "output_tokens": 76, "duration_ms": 44}
      }
    }
  },
  "aggregates": {
    "ailang": {
      "avg_output_tokens": 94.5,
      "avg_duration_ms": 34.0,
      "success_rate": 1.0
    },
    "python": {
      "avg_output_tokens": 100.5,
      "avg_duration_ms": 47.5,
      "success_rate": 1.0
    }
  }
}
```

### Example: insights.json
```json
{
  "meta": {
    "generated_at": "2025-10-05T21:30:00Z",
    "ailang_version": "v0.3.0-alpha2",
    "total_runs": 4
  },
  "key_findings": {
    "token_efficiency": {
      "ailang_avg_output": 94.5,
      "python_avg_output": 100.5,
      "reduction_pct": 6.0,
      "verdict": "AILANG generates 6% fewer output tokens"
    },
    "execution_speed": {
      "ailang_avg_ms": 34.0,
      "python_avg_ms": 47.5,
      "speedup": 1.4,
      "verdict": "AILANG executes 1.4x faster"
    },
    "success_rates": {
      "ailang_first_attempt": 1.0,
      "python_first_attempt": 1.0
    },
    "error_patterns": []
  },
  "recommendations": [
    "✅ 100% success rate - excellent baseline",
    "✅ Token efficiency validated (6% reduction)",
    "📊 Consider more complex benchmarks to stress-test"
  ]
}
```

---

## References

### Design Documents
- [M-EVAL: AI Benchmarking](../implemented/v0_3_0/M-EVAL_ai_benchmarking.md)
- [M-R5: Records Implementation](../20251005/M-R5_records.md)
- [v0.2.0 Implementation Plan](v0_2_0_implementation_plan.md)

### External Inspiration
- **OpenAI Evals**: Multi-model benchmarking framework
- **HumanEval**: Code generation benchmarks for AI models
- **BIG-bench**: Collaborative benchmark for language models
- **Anthropic's Constitutional AI**: Error-guided refinement loops

### AI Feedback (Endorsements)
> **GPT-5 (hypothetical)**: "The JSONL format is ideal for streaming analysis. Add detailed run traces (token-by-token generation) for deeper debugging."

> **Gemini 2.5 Pro (hypothetical)**: "Self-repair loop is brilliant for measuring language UX. Consider multi-turn repair (not just single-shot) for production systems."

---

## Changelog

**v1.0 (2025-10-05)**
- Initial design document
- Incorporates user feedback from eval harness testing
- Adds AI feedback from GPT-5 and Gemini 2.5 Pro (hypothetical)
- Defines 3-phase implementation plan

---

**Status**: 📋 Ready for implementation
**Next Steps**: Review with team → Start Phase A → Iterate based on real-world usage
