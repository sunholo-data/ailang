# M-EVAL: Comprehensive AI Evaluation Harness

**Status**: 📋 Planned (replaces M-EVAL-ENHANCE + M-EVAL2-Agentic)
**Version**: 2.0 (Unified Design)
**Author**: AI-Assisted (Claude + User)
**Date**: 2025-10-05
**Milestone**: v0.3.0-alpha3 (Phases A-C) → v0.4.0 (Phase D)

---

## Executive Summary

The AILANG eval harness is **the definitive tool for developing an AI-optimized programming language**. This comprehensive design combines:

1. **AI-friendly infrastructure** (JSONL formats, insights, reproducibility)
2. **Single-shot self-repair** (1 retry with error-specific guidance)
3. **Multi-turn agentic evaluation** (up to 5 turns, conversation-based)
4. **Production tool integration** (Claude Code CLI, Gemini CLI)
5. **Marketing & visualization** (HTML dashboards, comparison matrices)

**Progressive Enhancement Philosophy:**
```
Phase A: Core Infrastructure (schema, formats, versioning)
    ↓
Phase B: Single-Shot Self-Repair (error taxonomy, 1 retry)
    ↓
Phase C: Dashboards & Marketing (HTML, charts, insights)
    ↓
Phase D: Multi-Turn Agentic (agent abstraction, CLI integration)
```

**Key Insight**: Start simple (1 retry), validate error taxonomy, then scale to multi-turn production agents.

---

## Table of Contents

1. [Motivation & Vision](#motivation--vision)
2. [Design Goals](#design-goals)
3. [Architecture Overview](#architecture-overview)
4. [Phase A: Core Infrastructure (1-2 days)](#phase-a-core-infrastructure-1-2-days)
5. [Phase B: Single-Shot Self-Repair (2-3 days)](#phase-b-single-shot-self-repair-2-3-days)
6. [Phase C: Dashboards & Marketing (ongoing)](#phase-c-dashboards--marketing-ongoing)
7. [Phase D: Multi-Turn Agentic (2-3 weeks)](#phase-d-multi-turn-agentic-2-3-weeks)
8. [Unified Metrics Schema](#unified-metrics-schema)
9. [Timeline & Success Metrics](#timeline--success-metrics)
10. [Risk Assessment](#risk-assessment)
11. [References](#references)

---

## Motivation & Vision

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
- No multi-turn iteration (real-world coding is iterative)
- No human-friendly visualization
- Missing reproducibility metadata
- Capability detection is manual

### Vision: AI-First Language Development

**The Problem**: Traditional language development optimizes for *human* programmers. AILANG optimizes for *AI-assisted* programming.

**The Solution**: The eval harness becomes a **co-development tool** where:

1. **Baseline Testing** (existing)
   - AI generates code → Harness measures success
   - Single-shot evaluation validates language features

2. **Self-Repair Loop** (Phase B)
   - First attempt fails → Extract error category
   - Inject error-specific guidance → Retry once
   - Measures: "Can AI *learn* AILANG from error messages?"

3. **Multi-Turn Agentic** (Phase D)
   - Full conversation loop (up to 5 turns)
   - Cumulative token cost tracking
   - Production agent integration (Claude Code, Gemini CLI)
   - Measures: "What's the *total* cost of AI-assisted development?"

4. **Continuous Improvement** (all phases)
   - Harness categorizes failures → AI learns patterns
   - Prompt evolves → Harness A/B tests versions
   - Language improves → Metrics validate impact

---

## Design Goals

### 1. AI-Friendly Data Formats
**Goal**: Make eval results trivially consumable by AI assistants

**Formats**:
- `summary.jsonl` - One line per run (sequential reading)
- `matrix.json` - Pivoted comparison (benchmarks × models × languages)
- `insights.json` - High-level analysis (success rates, error patterns, recommendations)

**Why**: AI can read 1 JSONL file instead of parsing 10+ individual JSON files

### 2. Progressive Repair Strategies
**Goal**: Measure language UX at increasing complexity levels

**Strategy Levels**:
1. **Single-shot** (baseline) - Does AI know AILANG?
2. **Self-repair** (1 retry) - Can AI learn from error messages?
3. **Multi-turn** (3-5 turns) - What's the total debugging cost?
4. **Production agents** (external CLIs) - How do real tools perform?

### 3. Prompt Versioning & A/B Testing
**Goal**: Track prompt evolution scientifically

**Features**:
- SHA-256 hash of prompts for reproducibility
- Version metadata (`v0.3.0-rev1`, `v0.3.0-rev2`)
- A/B testing: `compare_prompts.sh rev1 rev2`
- Measures: "Which teaching strategy works best?"

### 4. Reproducibility
**Goal**: Ensure results are scientifically valid

**Fingerprints**:
- AILANG binary hash (SHA-256)
- Stdlib hash (directory tree)
- Model API version
- Prompt hash
- Environment (OS, arch, Go version)
- Seed for determinism

### 5. Error Taxonomy
**Goal**: Systematic categorization of failures

**Categories**:
- `TC_REC_001`: Record update syntax not implemented
- `PAR_003`: Missing semicolon in block
- `CAP_001`: Missing capability declaration
- `RUN_002`: Recursion depth exceeded
- `LOG_001`: Wrong output

**Why**: Powers self-repair guidance and insights generation

### 6. Human-Readable Dashboards
**Goal**: Make results accessible to non-technical stakeholders

**Outputs**:
- HTML dashboard with Chart.js visualizations
- Marketing-ready snippets (success rates, token efficiency)
- Turn-by-turn breakdown for agentic runs

---

## Architecture Overview

### Agent Abstraction (Strategy Pattern)

```
┌──────────────┐
│ Eval Command │  ailang eval --agent <type> --benchmark <name>
└───────┬──────┘
        │
        ▼
  ┌─────────────┐
  │ Agent       │  interface { Run(task) -> result }
  │ Interface   │
  └─────┬───────┘
        │
   ┌────┴───────────────┬──────────────┬──────────────┐
   │                    │              │              │
   ▼                    ▼              ▼              ▼
┌─────────┐      ┌──────────┐   ┌───────────┐  ┌──────────┐
│ API     │      │ SelfRep  │   │ Loop      │  │ Claude   │
│ Agent   │      │ Agent    │   │ Agent     │  │ Code CLI │
│ (base)  │      │ (1 retry)│   │ (5 turns) │  │ (extern) │
└─────────┘      └──────────┘   └───────────┘  └──────────┘
```

### Data Flow

```
Benchmark YAML → Agent → Code Generation → Execution → Metrics
                   │                                      │
                   └──────── Multi-turn feedback ────────┘
                                (if agent supports)

Metrics → JSONL + Matrix + Insights → Dashboard + Marketing
```

---

## Phase A: Core Infrastructure (1-2 days)

**Goal**: Extend schema, add AI-friendly formats, enable prompt versioning

### A1: Unified RunMetrics Schema

**File**: `internal/eval_harness/metrics.go`

**Extended Schema**:
```go
type RunMetrics struct {
    // ===== EXISTING FIELDS =====
    ID            string    `json:"id"`
    Lang          string    `json:"lang"`
    Model         string    `json:"model"`
    Seed          int64     `json:"seed"`
    InputTokens   int       `json:"input_tokens"`
    OutputTokens  int       `json:"output_tokens"`
    TotalTokens   int       `json:"total_tokens"`
    CostUSD       float64   `json:"cost_usd"`
    CompileOk     bool      `json:"compile_ok"`
    RuntimeOk     bool      `json:"runtime_ok"`
    StdoutOk      bool      `json:"stdout_ok"`
    DurationMs    int64     `json:"duration_ms"`
    CompileMs     int64     `json:"compile_ms"`
    ExecuteMs     int64     `json:"execute_ms"`
    ErrorCategory string    `json:"error_category"`
    Stderr        string    `json:"stderr"`
    Timestamp     time.Time `json:"timestamp"`
    Code          string    `json:"code"`

    // ===== PHASE A: PROVENANCE (NEW) =====
    PromptVersion   string `json:"prompt_version"`      // "v0.3.0-rev2"
    PromptHash      string `json:"prompt_hash"`         // "sha256:abc123..."
    PromptSizeChars int    `json:"prompt_size_chars"`   // 4521
    PromptTokensEst int    `json:"prompt_tokens_est"`   // 1200 (estimated)
    AILangVersion   string `json:"ailang_version"`      // "v0.3.0-alpha2"
    BinaryHash      string `json:"binary_hash"`         // "sha256:def456..."
    StdlibHash      string `json:"stdlib_hash"`         // "sha256:ghi789..."
    ModelAPI        string `json:"model_api"`           // "openai/gpt-4o-mini-2024-07-18"
    EnvFingerprint  string `json:"env_fingerprint"`     // "darwin_arm64_go1.22.1"

    // ===== PHASE A: CAPABILITY DETECTION (NEW) =====
    DeclaredCaps []string `json:"declared_caps"`        // ["IO", "FS"] from YAML
    InferredCaps []string `json:"inferred_caps"`        // ["IO"] detected in code
    CapsMismatch bool     `json:"caps_mismatch"`        // true if declared != inferred

    // ===== PHASE A: CODE QUALITY (NEW) =====
    CodeSizeBytes int `json:"code_size_bytes"`          // 342
    CodeLines     int `json:"code_lines"`               // 18
    CodeComments  int `json:"code_comments"`            // 0

    // ===== PHASE B: SELF-REPAIR (NEW) =====
    AgentType       string `json:"agent_type"`          // "api", "self-repair", "loop", "claude-code"
    FirstAttemptOk  bool   `json:"first_attempt_ok"`    // Did first attempt succeed?
    RepairAttempted bool   `json:"repair_attempted"`    // Did we retry?
    RepairSuccess   bool   `json:"repair_success"`      // Did retry fix it?
    RepairCategory  string `json:"repair_category"`     // Error type that triggered repair
    RepairGuidance  string `json:"repair_guidance"`     // Guidance injected for repair

    // ===== PHASE D: MULTI-TURN AGENTIC (NEW) =====
    Turns           int      `json:"turns"`              // Number of conversation turns
    TokensPerTurn   []int    `json:"tokens_per_turn"`    // [180, 150, 120] tokens each turn
    ErrorHistory    []string `json:"error_history"`      // All errors encountered
    TurnTimestamps  []string `json:"turn_timestamps"`    // When each turn completed
    ConversationLog string   `json:"conversation_log"`   // Full conversation (optional)
}
```

**Helpers** (Phase A):
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
    // Walk stdlib/, collect file hashes, hash concatenation
    var hashes []string
    err := filepath.Walk("stdlib", func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() {
            return err
        }
        data, err := os.ReadFile(path)
        if err != nil {
            return err
        }
        h := sha256.Sum256(data)
        hashes = append(hashes, fmt.Sprintf("%x", h))
        return nil
    })
    if err != nil {
        return "", err
    }
    sort.Strings(hashes) // Deterministic order
    combined := strings.Join(hashes, "")
    finalHash := sha256.Sum256([]byte(combined))
    return fmt.Sprintf("sha256:%x", finalHash), nil
}

// InferCapabilities detects capabilities from generated code
func InferCapabilities(code string, lang string) []string {
    var caps []string
    if lang == "ailang" {
        // Simple string matching (Phase A)
        // Phase D: Replace with AST parsing for accuracy
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

### A2: AI-Friendly Data Formats

**Files**: `tools/generate_formats.sh` (NEW)

**Format 1: summary.jsonl**
```jsonl
{"id":"records_person","lang":"ailang","model":"gpt-4o-mini","agent_type":"api","output_tokens":106,"compile_ok":true,"runtime_ok":true,"stdout_ok":true,"duration_ms":12,"prompt_version":"v0.3.0-rev1","first_attempt_ok":true,"turns":1}
{"id":"records_person","lang":"python","model":"gpt-4o-mini","agent_type":"api","output_tokens":125,"compile_ok":true,"runtime_ok":true,"stdout_ok":true,"duration_ms":51,"prompt_version":"python-v1","first_attempt_ok":true,"turns":1}
```

**Format 2: matrix.json**
```json
{
  "meta": {
    "generated_at": "2025-10-05T21:30:00Z",
    "ailang_version": "v0.3.0-alpha2",
    "models": ["gpt-4o-mini", "gemini-2.0-flash-exp", "claude-sonnet-4.5"],
    "benchmarks": ["records_person", "recursion_fibonacci"],
    "agent_types": ["api", "self-repair", "loop"]
  },
  "comparison": {
    "records_person": {
      "ailang": {
        "gpt-4o-mini": {
          "api": {"success": true, "output_tokens": 106, "turns": 1},
          "self-repair": {"success": true, "output_tokens": 212, "turns": 2},
          "loop": {"success": true, "output_tokens": 450, "turns": 3}
        }
      },
      "python": {
        "gpt-4o-mini": {
          "api": {"success": true, "output_tokens": 125, "turns": 1}
        }
      }
    }
  },
  "aggregates": {
    "ailang": {
      "api": {"avg_output_tokens": 95, "avg_turns": 1.0, "success_rate": 1.0},
      "self-repair": {"avg_output_tokens": 200, "avg_turns": 1.5, "success_rate": 0.95},
      "loop": {"avg_output_tokens": 420, "avg_turns": 3.2, "success_rate": 0.90}
    },
    "python": {
      "api": {"avg_output_tokens": 120, "avg_turns": 1.0, "success_rate": 1.0}
    }
  }
}
```

**Format 3: insights.json**
```json
{
  "meta": {
    "generated_at": "2025-10-05T21:30:00Z",
    "ailang_version": "v0.3.0-alpha2",
    "total_runs": 24
  },
  "key_findings": {
    "token_efficiency": {
      "ailang_api_avg": 95,
      "ailang_self_repair_avg": 200,
      "ailang_loop_avg": 420,
      "python_api_avg": 120,
      "verdict": "AILANG single-shot: 21% fewer tokens. Multi-turn: 250% more tokens (learning curve)"
    },
    "success_rates": {
      "ailang_first_attempt": 0.60,
      "ailang_with_repair": 0.95,
      "ailang_with_loop": 0.90,
      "python_first_attempt": 1.0
    },
    "error_patterns": [
      {"category": "TC_REC_001", "count": 8, "repair_success_rate": 0.87},
      {"category": "PAR_003", "count": 3, "repair_success_rate": 1.0}
    ]
  },
  "recommendations": [
    "✅ Self-repair effective for syntax errors (87% success)",
    "⚠️ Multi-turn needed for complex logic (3.2 turns avg)",
    "📊 Prompt v0.3.0-rev2 reduces first-attempt failures by 40%"
  ]
}
```

**Implementation**:
```bash
#!/usr/bin/env bash
# tools/generate_formats.sh

RESULTS_DIR="${1:-eval_results}"

# Generate summary.jsonl
find "$RESULTS_DIR" -name "*.json" -not -name "summary.jsonl" | \
  xargs jq -c '{id,lang,model,agent_type,output_tokens,compile_ok,runtime_ok,stdout_ok,duration_ms,prompt_version,first_attempt_ok,turns}' \
  > "$RESULTS_DIR/summary.jsonl"

# Generate matrix.json and insights.json
go run tools/generate_insights.go "$RESULTS_DIR"

echo "✓ Generated: summary.jsonl, matrix.json, insights.json"
```

### A3: Prompt Versioning System

**File**: `prompts/versions.json` (NEW)

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
          "notes": "Clarified record update NOT implemented"
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
tools/compare_prompts.sh v0.3.0-rev1 v0.3.0-rev2
```

### A4: Capability Auto-Detection

**File**: `internal/eval_harness/runner.go` (enhanced)

```go
func (r *AILangRunner) Run(code string, declaredCaps []string) (*RunResult, error) {
    // Infer capabilities from code
    inferredCaps := InferCapabilities(code, "ailang")

    // Check for mismatch
    capsMismatch := !slicesEqual(declaredCaps, inferredCaps)

    if capsMismatch {
        log.Printf("⚠️  Capability mismatch: declared=%v inferred=%v", declaredCaps, inferredCaps)
    }

    // Use inferred capabilities (override declared)
    actualCaps := inferredCaps

    // Execute code with actual capabilities
    result := r.execute(code, actualCaps)

    result.DeclaredCaps = declaredCaps
    result.InferredCaps = inferredCaps
    result.CapsMismatch = capsMismatch

    return result, nil
}
```

---

## Phase B: Single-Shot Self-Repair (2-3 days)

**Goal**: Add error taxonomy and 1-retry self-repair mechanism

### B1: Error Taxonomy System

**File**: `internal/eval_harness/errors.go` (NEW)

```go
package eval_harness

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

// CategorizeError maps error message to taxonomy code
func CategorizeError(stderr string, compileOk, runtimeOk, stdoutOk bool) string {
    switch {
    case !compileOk:
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

// RepairGuidance maps error codes to retry prompts
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

### B2: Self-Repair Agent

**File**: `internal/eval_harness/agent.go` (NEW - agent interface)

```go
package eval_harness

import (
    "context"
    "time"
)

// Agent represents an AI code generation agent
type Agent interface {
    // Run executes a benchmark task and returns results
    Run(ctx context.Context, task BenchmarkTask) (*AgentResult, error)

    // Name returns the agent type identifier
    Name() string
}

// BenchmarkTask defines a coding task
type BenchmarkTask struct {
    Prompt      string
    Lang        string
    ExpectedOut string
    Workspace   string
    Caps        []string
    MaxTurns    int
    Timeout     time.Duration
}

// AgentResult contains execution results
type AgentResult struct {
    Success         bool
    Turns           int
    TotalTokens     int
    TokensPerTurn   []int
    DurationMs      int64
    FinalCode       string
    ErrorHistory    []string
    TurnTimestamps  []string
    FirstAttemptOk  bool
    RepairAttempted bool
    RepairSuccess   bool
    RepairCategory  string
    RepairGuidance  string
}
```

**File**: `internal/eval_harness/self_repair_agent.go` (NEW)

```go
package eval_harness

import (
    "context"
    "fmt"
    "time"
)

// SelfRepairAgent performs single-shot with optional 1 retry
type SelfRepairAgent struct {
    model      string
    apiClient  APIClient
    enableSelfRepair bool
}

func NewSelfRepairAgent(model string, enableSelfRepair bool) *SelfRepairAgent {
    return &SelfRepairAgent{
        model:            model,
        apiClient:        NewAPIClient(),
        enableSelfRepair: enableSelfRepair,
    }
}

func (a *SelfRepairAgent) Name() string {
    if a.enableSelfRepair {
        return "self-repair"
    }
    return "api"
}

func (a *SelfRepairAgent) Run(ctx context.Context, task BenchmarkTask) (*AgentResult, error) {
    result := &AgentResult{}

    // First attempt
    firstAttempt, err := a.runSingleAttempt(ctx, task, "")
    if err != nil {
        return nil, err
    }

    result.TotalTokens = firstAttempt.Tokens
    result.TokensPerTurn = []int{firstAttempt.Tokens}
    result.Turns = 1
    result.FirstAttemptOk = firstAttempt.Success
    result.TurnTimestamps = []string{time.Now().Format(time.RFC3339)}

    // If first attempt succeeded, we're done
    if firstAttempt.Success {
        result.Success = true
        result.FinalCode = firstAttempt.Code
        result.DurationMs = firstAttempt.DurationMs
        return result, nil
    }

    // First attempt failed
    result.ErrorHistory = append(result.ErrorHistory, firstAttempt.Stderr)
    errorCategory := CategorizeError(firstAttempt.Stderr, firstAttempt.CompileOk, firstAttempt.RuntimeOk, firstAttempt.StdoutOk)

    // If self-repair disabled, stop here
    if !a.enableSelfRepair {
        result.Success = false
        result.FinalCode = firstAttempt.Code
        result.DurationMs = firstAttempt.DurationMs
        return result, nil
    }

    // Get repair guidance
    guidance, ok := RepairGuidance[errorCategory]
    if !ok {
        // No known guidance for this error
        result.Success = false
        result.FinalCode = firstAttempt.Code
        result.DurationMs = firstAttempt.DurationMs
        return result, nil
    }

    // Retry with guidance
    result.RepairAttempted = true
    result.RepairCategory = errorCategory
    result.RepairGuidance = guidance

    secondAttempt, err := a.runSingleAttempt(ctx, task, guidance)
    if err != nil {
        return nil, err
    }

    result.TotalTokens += secondAttempt.Tokens
    result.TokensPerTurn = append(result.TokensPerTurn, secondAttempt.Tokens)
    result.Turns = 2
    result.TurnTimestamps = append(result.TurnTimestamps, time.Now().Format(time.RFC3339))
    result.DurationMs = firstAttempt.DurationMs + secondAttempt.DurationMs

    if secondAttempt.Success {
        result.Success = true
        result.RepairSuccess = true
        result.FinalCode = secondAttempt.Code
    } else {
        result.Success = false
        result.RepairSuccess = false
        result.FinalCode = secondAttempt.Code
        result.ErrorHistory = append(result.ErrorHistory, secondAttempt.Stderr)
    }

    return result, nil
}

func (a *SelfRepairAgent) runSingleAttempt(ctx context.Context, task BenchmarkTask, repairGuidance string) (*AttemptResult, error) {
    // Build prompt
    prompt := task.Prompt
    if repairGuidance != "" {
        prompt = prompt + "\n\n## Previous Attempt Failed\n" + repairGuidance
    }

    start := time.Now()

    // Generate code
    genResult, err := a.apiClient.Generate(ctx, a.model, prompt)
    if err != nil {
        return nil, err
    }

    // Execute code
    runResult := executeCode(task.Lang, genResult.Code, task.ExpectedOut, task.Caps)

    return &AttemptResult{
        Code:      genResult.Code,
        Tokens:    genResult.OutputTokens,
        Success:   runResult.CompileOk && runResult.RuntimeOk && runResult.StdoutOk,
        CompileOk: runResult.CompileOk,
        RuntimeOk: runResult.RuntimeOk,
        StdoutOk:  runResult.StdoutOk,
        Stderr:    runResult.Stderr,
        DurationMs: time.Since(start).Milliseconds(),
    }, nil
}

type AttemptResult struct {
    Code       string
    Tokens     int
    Success    bool
    CompileOk  bool
    RuntimeOk  bool
    StdoutOk   bool
    Stderr     string
    DurationMs int64
}
```

### B3: Update CLI

**File**: `cmd/ailang/eval.go` (enhanced)

```go
var (
    agentType  string
    selfRepair bool
)

func init() {
    evalCmd.Flags().StringVar(&agentType, "agent", "api", "Agent type: api, self-repair, loop, claude-code")
    evalCmd.Flags().BoolVar(&selfRepair, "self-repair", false, "Enable self-repair (shorthand for --agent self-repair)")
}

func runEval(cmd *cobra.Command, args []string) error {
    // Resolve agent type
    if selfRepair {
        agentType = "self-repair"
    }

    // Create agent
    var agent Agent
    switch agentType {
    case "api":
        agent = NewSelfRepairAgent(model, false)
    case "self-repair":
        agent = NewSelfRepairAgent(model, true)
    case "loop":
        // Phase D
        return fmt.Errorf("loop agent not yet implemented (Phase D)")
    case "claude-code":
        // Phase D
        return fmt.Errorf("claude-code agent not yet implemented (Phase D)")
    default:
        return fmt.Errorf("unknown agent type: %s", agentType)
    }

    // Load benchmark
    spec, err := LoadBenchmark(benchmarkName)
    if err != nil {
        return err
    }

    // Build task
    task := BenchmarkTask{
        Prompt:      spec.PromptForLanguage(lang),
        Lang:        lang,
        ExpectedOut: spec.ExpectedOutput,
        Caps:        spec.Caps,
        MaxTurns:    maxTurns,
        Timeout:     timeout,
    }

    // Run agent
    result, err := agent.Run(context.Background(), task)
    if err != nil {
        return err
    }

    // Convert to RunMetrics and save
    metrics := agentResultToMetrics(result, spec, agent.Name())
    saveMetrics(metrics)

    // Print summary
    printEvalSummary(metrics)

    return nil
}

func agentResultToMetrics(result *AgentResult, spec *BenchmarkSpec, agentName string) *RunMetrics {
    // Populate all fields from Phase A + Phase B
    return &RunMetrics{
        ID:              spec.ID,
        Lang:            spec.Lang,
        Model:           model,
        Seed:            seed,
        AgentType:       agentName,
        OutputTokens:    result.TotalTokens,
        CompileOk:       result.Success,
        RuntimeOk:       result.Success,
        StdoutOk:        result.Success,
        DurationMs:      result.DurationMs,
        Code:            result.FinalCode,
        Turns:           result.Turns,
        TokensPerTurn:   result.TokensPerTurn,
        ErrorHistory:    result.ErrorHistory,
        TurnTimestamps:  result.TurnTimestamps,
        FirstAttemptOk:  result.FirstAttemptOk,
        RepairAttempted: result.RepairAttempted,
        RepairSuccess:   result.RepairSuccess,
        RepairCategory:  result.RepairCategory,
        RepairGuidance:  result.RepairGuidance,
        PromptVersion:   getPromptVersion(),
        PromptHash:      getPromptHash(),
        AILangVersion:   getAILangVersion(),
        BinaryHash:      getBinaryHash(),
        StdlibHash:      getStdlibHash(),
        // ... etc
    }
}
```

---

## Phase C: Dashboards & Marketing (Ongoing)

**Goal**: Human-readable visualizations and marketing materials

### C1: HTML Dashboard

**File**: `tools/dashboard_template.html` (NEW)

*(See M-EVAL-ENHANCE doc for full template)*

**Key Charts**:
- Token usage by benchmark (bar chart)
- Success rates by agent type (bar chart)
- Turn distribution (histogram)
- Error category breakdown (pie chart)

**Generator**:
```bash
#!/usr/bin/env bash
# tools/generate_dashboard.sh

RESULTS_DIR="${1:-eval_results}"
go run tools/generate_dashboard.go "$RESULTS_DIR"
echo "✓ Dashboard: $RESULTS_DIR/dashboard.html"
```

### C2: Marketing Material Generator

**File**: `tools/generate_marketing.sh` (NEW)

**Output**:
```markdown
# AILANG Evaluation Results

## Single-Shot Performance
- ✅ **95% first-attempt success** (AILANG)
- 📉 **21% fewer tokens** than Python
- ⚡ **3.8x faster execution**

## Self-Repair Performance
- 🔧 **87% repair success** for syntax errors
- 📊 **1.5 turns average** to working code
- 💡 Error taxonomy covers 80% of failures

## Multi-Turn Performance (Phase D)
- 🔄 **3.2 turns average** for complex tasks
- 💰 **420 tokens total** (vs 395 for Python)
- 🎯 **90% eventual success**
```

---

## Phase D: Multi-Turn Agentic (2-3 weeks)

**Goal**: Full conversation loop with production agent integration

### D1: Loop Agent (Multi-Turn)

**File**: `internal/eval_harness/loop_agent.go` (NEW)

```go
package eval_harness

import (
    "context"
    "fmt"
    "time"
)

// LoopAgent performs multi-turn conversation (up to maxTurns)
type LoopAgent struct {
    model    string
    maxTurns int
    apiClient APIClient
}

func NewLoopAgent(model string, maxTurns int) *LoopAgent {
    return &LoopAgent{
        model:     model,
        maxTurns:  maxTurns,
        apiClient: NewAPIClient(),
    }
}

func (a *LoopAgent) Name() string {
    return "loop"
}

func (a *LoopAgent) Run(ctx context.Context, task BenchmarkTask) (*AgentResult, error) {
    result := &AgentResult{
        Turns: 0,
        TokensPerTurn: []int{},
        ErrorHistory: []string{},
        TurnTimestamps: []string{},
    }

    // Initialize conversation
    conversation := []Message{
        {Role: "user", Content: task.Prompt},
    }

    for turn := 1; turn <= a.maxTurns; turn++ {
        turnStart := time.Now()

        // Generate code
        genResult, err := a.apiClient.GenerateWithConversation(ctx, a.model, conversation)
        if err != nil {
            return nil, err
        }

        result.TotalTokens += genResult.OutputTokens
        result.TokensPerTurn = append(result.TokensPerTurn, genResult.OutputTokens)
        result.Turns = turn
        result.TurnTimestamps = append(result.TurnTimestamps, time.Now().Format(time.RFC3339))

        code := extractCode(genResult.Content)

        // Test the code
        runResult := executeCode(task.Lang, code, task.ExpectedOut, task.Caps)

        if turn == 1 {
            result.FirstAttemptOk = runResult.CompileOk && runResult.RuntimeOk && runResult.StdoutOk
        }

        // Check if successful
        if runResult.CompileOk && runResult.RuntimeOk && runResult.StdoutOk {
            result.Success = true
            result.FinalCode = code
            result.DurationMs += time.Since(turnStart).Milliseconds()
            return result, nil
        }

        // Failed - prepare feedback for next turn
        result.ErrorHistory = append(result.ErrorHistory, runResult.Stderr)

        feedback := fmt.Sprintf(
            "The code failed with error:\n%s\n\nExpected output:\n%s\n\nActual output:\n%s\n\nPlease fix the code.",
            runResult.Stderr,
            task.ExpectedOut,
            runResult.Stdout,
        )

        // Add to conversation
        conversation = append(conversation,
            Message{Role: "assistant", Content: genResult.Content},
            Message{Role: "user", Content: feedback},
        )

        result.DurationMs += time.Since(turnStart).Milliseconds()
    }

    // Failed after maxTurns
    result.Success = false
    return result, nil
}

type Message struct {
    Role    string
    Content string
}
```

### D2: Claude Code CLI Agent

**File**: `internal/eval_harness/claude_agent.go` (NEW)

```go
package eval_harness

import (
    "context"
    "fmt"
    "os/exec"
    "path/filepath"
    "time"
)

// ClaudeCodeAgent integrates with Anthropic's Claude Code CLI
type ClaudeCodeAgent struct {
    binaryPath string
    apiKey     string
}

func NewClaudeCodeAgent(binaryPath, apiKey string) *ClaudeCodeAgent {
    return &ClaudeCodeAgent{
        binaryPath: binaryPath,
        apiKey:     apiKey,
    }
}

func (a *ClaudeCodeAgent) Name() string {
    return "claude-code"
}

func (a *ClaudeCodeAgent) Run(ctx context.Context, task BenchmarkTask) (*AgentResult, error) {
    // Create isolated session directory
    sessionDir := filepath.Join(task.Workspace, "claude_session")
    os.MkdirAll(sessionDir, 0755)
    defer os.RemoveAll(sessionDir)

    // Write task to file
    taskFile := filepath.Join(sessionDir, "task.txt")
    os.WriteFile(taskFile, []byte(task.Prompt), 0644)

    // Write expected output
    expectedFile := filepath.Join(sessionDir, "expected.txt")
    os.WriteFile(expectedFile, []byte(task.ExpectedOut), 0644)

    // Spawn Claude Code CLI
    cmd := exec.CommandContext(ctx,
        a.binaryPath,
        "--session-dir", sessionDir,
        "--lang", task.Lang,
        "--task-file", taskFile,
        "--test-file", expectedFile,
        "--max-turns", fmt.Sprintf("%d", task.MaxTurns),
    )

    // Set API key
    cmd.Env = append(cmd.Env, "ANTHROPIC_API_KEY="+a.apiKey)

    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    start := time.Now()
    err := cmd.Run()
    duration := time.Since(start)

    // Parse output
    tokens := parseClaudeTokenUsage(stdout.String())
    turns := parseClaudeTurnCount(stdout.String())
    tokensPerTurn := parseClaudeTokensPerTurn(stdout.String())

    // Read final code
    finalCode := readFinalCode(sessionDir, task.Lang)

    return &AgentResult{
        Success:        err == nil,
        Turns:          turns,
        TotalTokens:    tokens,
        TokensPerTurn:  tokensPerTurn,
        DurationMs:     duration.Milliseconds(),
        FinalCode:      finalCode,
        FirstAttemptOk: turns == 1 && err == nil,
    }, nil
}

func parseClaudeTokenUsage(output string) int {
    // Parse "Total tokens: 450" from Claude Code CLI output
    // Implementation depends on actual CLI output format
    return 0
}

func parseClaudeTurnCount(output string) int {
    // Parse "Turns: 3" from Claude Code CLI output
    return 0
}

func parseClaudeTokensPerTurn(output string) []int {
    // Parse per-turn token usage
    return []int{}
}

func readFinalCode(sessionDir, lang string) string {
    // Read generated code file (e.g., main.ail or main.py)
    var filename string
    if lang == "ailang" {
        filename = "main.ail"
    } else {
        filename = "main.py"
    }
    data, _ := os.ReadFile(filepath.Join(sessionDir, filename))
    return string(data)
}
```

### D3: Gemini CLI Agent

**File**: `internal/eval_harness/gemini_agent.go` (NEW)

*(Similar pattern to ClaudeCodeAgent)*

### D4: Update CLI for Phase D

```bash
# Multi-turn agent loop (built-in)
ailang eval --agent loop --benchmark fizzbuzz --max-turns 5

# Claude Code CLI
ailang eval --agent claude-code --benchmark fizzbuzz --claude-path /usr/local/bin/claude

# Gemini CLI
ailang eval --agent gemini --benchmark fizzbuzz --gemini-path /usr/local/bin/gemini

# Compare all agents
ailang eval --agent all --benchmark fizzbuzz
```

---

## Unified Metrics Schema

**Final Schema** (all phases combined):

```go
type RunMetrics struct {
    // ===== CORE (v0.3.0-alpha2) =====
    ID            string    `json:"id"`
    Lang          string    `json:"lang"`
    Model         string    `json:"model"`
    Seed          int64     `json:"seed"`
    InputTokens   int       `json:"input_tokens"`
    OutputTokens  int       `json:"output_tokens"`
    TotalTokens   int       `json:"total_tokens"`
    CostUSD       float64   `json:"cost_usd"`
    CompileOk     bool      `json:"compile_ok"`
    RuntimeOk     bool      `json:"runtime_ok"`
    StdoutOk      bool      `json:"stdout_ok"`
    DurationMs    int64     `json:"duration_ms"`
    CompileMs     int64     `json:"compile_ms"`
    ExecuteMs     int64     `json:"execute_ms"`
    ErrorCategory string    `json:"error_category"`
    Stderr        string    `json:"stderr"`
    Timestamp     time.Time `json:"timestamp"`
    Code          string    `json:"code"`

    // ===== PHASE A: PROVENANCE =====
    PromptVersion   string `json:"prompt_version"`
    PromptHash      string `json:"prompt_hash"`
    PromptSizeChars int    `json:"prompt_size_chars"`
    PromptTokensEst int    `json:"prompt_tokens_est"`
    AILangVersion   string `json:"ailang_version"`
    BinaryHash      string `json:"binary_hash"`
    StdlibHash      string `json:"stdlib_hash"`
    ModelAPI        string `json:"model_api"`
    EnvFingerprint  string `json:"env_fingerprint"`

    // ===== PHASE A: CAPABILITIES =====
    DeclaredCaps []string `json:"declared_caps"`
    InferredCaps []string `json:"inferred_caps"`
    CapsMismatch bool     `json:"caps_mismatch"`

    // ===== PHASE A: CODE QUALITY =====
    CodeSizeBytes int `json:"code_size_bytes"`
    CodeLines     int `json:"code_lines"`
    CodeComments  int `json:"code_comments"`

    // ===== PHASE B: AGENT TYPE =====
    AgentType string `json:"agent_type"` // "api", "self-repair", "loop", "claude-code", "gemini"

    // ===== PHASE B: SELF-REPAIR =====
    FirstAttemptOk  bool   `json:"first_attempt_ok"`
    RepairAttempted bool   `json:"repair_attempted"`
    RepairSuccess   bool   `json:"repair_success"`
    RepairCategory  string `json:"repair_category"`
    RepairGuidance  string `json:"repair_guidance"`

    // ===== PHASE D: MULTI-TURN =====
    Turns          int      `json:"turns"`
    TokensPerTurn  []int    `json:"tokens_per_turn"`
    ErrorHistory   []string `json:"error_history"`
    TurnTimestamps []string `json:"turn_timestamps"`
}
```

**Compatibility**:
- Phase A metrics work with existing single-shot eval
- Phase B adds self-repair fields (optional)
- Phase D adds multi-turn fields (optional)
- All agents populate same schema

---

## Timeline & Success Metrics

### Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase A** | 1-2 days | Extended schema, JSONL/matrix/insights formats, prompt versioning, capability detection |
| **Phase B** | 2-3 days | Error taxonomy, self-repair agent, CLI integration |
| **Phase C** | Ongoing | HTML dashboard, marketing generator |
| **Phase D** | 2-3 weeks | Loop agent, Claude Code CLI, Gemini CLI, multi-turn analysis |

**Total**: ~4-6 days (Phases A-B) + ongoing (C) + 2-3 weeks (D)

### Success Metrics

**Phase A:**
- ✅ All eval results have provenance fields (hash, version, fingerprint)
- ✅ JSONL generation works (1 file instead of 10+)
- ✅ Capability mismatch detection works (warns on inferred != declared)

**Phase B:**
- ✅ Error taxonomy covers 80%+ of failures
- ✅ Self-repair success rate: 50%+ for known errors
- ✅ Repair guidance effective (track category → success rate)

**Phase C:**
- ✅ Dashboard generated successfully
- ✅ Marketing snippets accurate and usable
- ✅ Non-technical stakeholders can understand results

**Phase D:**
- ✅ Loop agent supports 3-5 turns
- ✅ Claude Code CLI integration works
- ✅ Gemini CLI integration works
- ✅ Multi-turn metrics accurate (tokens per turn, cumulative cost)

---

## Risk Assessment

### Technical Risks

| Risk | Phase | Impact | Mitigation |
|------|-------|--------|------------|
| Prompt hashing instability (whitespace changes) | A | Low | Normalize before hashing |
| Self-repair false positives | B | Low | Track repair category effectiveness |
| Error taxonomy incomplete | B | Medium | Use "UNKNOWN" category, review regularly |
| Capability inference inaccurate | A | Medium | Phase D: Replace string matching with AST parsing |
| CLI tools have breaking changes | D | High | Version pin, test on CI |
| Multi-turn timeout before success | D | Medium | Configurable max turns, partial credit |

### Process Risks

| Risk | Phase | Impact | Mitigation |
|------|-------|--------|------------|
| Complexity creep | C | Low | Keep Phase C minimal (JSONL > fancy UI) |
| Backward compatibility | All | Low | Make new fields optional |
| Phase D depends on Phase B validation | D | Medium | Don't start D until B error taxonomy tested |

---

## Usage Examples

### Phase A-B: Single-Shot & Self-Repair

```bash
# Baseline (existing)
ailang eval --benchmark records_person --model gpt-4o-mini

# Self-repair (Phase B)
ailang eval --benchmark records_person --model gpt-4o-mini --self-repair

# Specific prompt version (Phase A)
ailang eval --benchmark records_person --prompt-version v0.3.0-rev1

# Generate reports (Phase A-C)
make eval-all
make eval-insights
make eval-dashboard
open eval_results/dashboard.html
```

### Phase D: Multi-Turn Agentic

```bash
# Built-in loop agent (3 turns)
ailang eval --agent loop --benchmark fizzbuzz --max-turns 3

# Claude Code CLI
ailang eval --agent claude-code --benchmark fizzbuzz

# Gemini CLI
ailang eval --agent gemini --benchmark fizzbuzz

# Compare all agents
ailang eval --agent all --benchmark fizzbuzz --seed 42
```

### Comparison Workflow

```bash
# 1. Run baseline
ailang eval --agent api --all-benchmarks --seed 42

# 2. Run self-repair
ailang eval --agent self-repair --all-benchmarks --seed 42

# 3. Run multi-turn
ailang eval --agent loop --all-benchmarks --seed 42 --max-turns 5

# 4. Generate comparison report
tools/compare_agents.sh api self-repair loop
```

**Output**:
```markdown
# Agent Comparison

| Agent | Avg Tokens | Avg Turns | Success Rate | Avg Duration |
|-------|-----------|-----------|--------------|--------------|
| api         | 95  | 1.0 | 60% | 15ms |
| self-repair | 200 | 1.5 | 95% | 28ms |
| loop        | 420 | 3.2 | 90% | 82ms |

**Insights:**
- Self-repair best balance (95% success, 2x tokens)
- Loop needed for complex tasks (3.2 turns avg)
- API baseline useful for known-good prompts
```

---

## File Structure (After All Phases)

```
ailang/
├── internal/
│   └── eval_harness/
│       ├── metrics.go               # UNIFIED SCHEMA (all phases)
│       ├── agent.go                 # Agent interface (Phase B)
│       ├── self_repair_agent.go     # Phase B
│       ├── loop_agent.go            # Phase D
│       ├── claude_agent.go          # Phase D
│       ├── gemini_agent.go          # Phase D
│       ├── errors.go                # Error taxonomy (Phase B)
│       ├── ai_client.go             # API client
│       ├── runner.go                # Code execution
│       └── spec.go                  # Benchmark loading
│
├── prompts/
│   ├── versions.json                # Phase A
│   ├── v0.2.0.md
│   ├── v0.3.0.md
│   └── python.md
│
├── tools/
│   ├── generate_formats.sh          # JSONL, matrix, insights (Phase A)
│   ├── generate_insights.go         # Insights generator (Phase A)
│   ├── generate_dashboard.sh        # HTML dashboard (Phase C)
│   ├── generate_marketing.sh        # Marketing snippets (Phase C)
│   ├── dashboard_template.html      # Phase C
│   ├── compare_prompts.sh           # Phase A
│   └── compare_agents.sh            # Phase D
│
├── eval_results/
│   ├── *.json                       # Individual runs
│   ├── summary.jsonl                # Phase A
│   ├── matrix.json                  # Phase A
│   ├── insights.json                # Phase A
│   ├── leaderboard.md               # Existing
│   └── dashboard.html               # Phase C
│
└── cmd/ailang/
    └── eval.go                      # CLI with agent selection
```

---

## References

### Design Documents (Merged)
- ~~M-EVAL-ENHANCE~~ → This document (Phases A-C)
- ~~M-EVAL2-Agentic~~ → This document (Phase D)
- [M-EVAL: AI Benchmarking](../implemented/v0_3_0/M-EVAL_ai_benchmarking.md) (baseline)
- [M-R5: Records Implementation](../20251005/M-R5_records.md)

### External Inspiration
- **OpenAI Evals**: Multi-model benchmarking
- **HumanEval**: Code generation benchmarks
- **Anthropic's Constitutional AI**: Error-guided refinement
- **Claude Code CLI**: Production AI coding tool
- **Gemini CLI**: Google's AI coding assistant

---

## Changelog

**v2.0 (2025-10-05)** - Unified Design
- Merged M-EVAL-ENHANCE and M-EVAL2-Agentic
- Progressive enhancement: A→B→C→D
- Unified RunMetrics schema
- Agent abstraction (API, SelfRepair, Loop, External CLIs)
- Timeline: 4-6 days (A-C) + 2-3 weeks (D)

**v1.0 (2025-10-05)** - Separate Designs (superseded)
- M-EVAL-ENHANCE: Single-retry self-repair
- M-EVAL2-Agentic: Multi-turn iteration

---

**Status**: 📋 Ready for implementation (Phase A → B → C → D)

**Next Steps**:
1. Review unified design
2. Start Phase A (core infrastructure)
3. Validate error taxonomy in Phase B
4. Iterate on Phase C dashboards
5. Begin Phase D after Phase B validation complete
