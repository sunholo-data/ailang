# Agent Prompt Integration with Benchmarks and Prompts

## How It Works

The agent evaluation system integrates three key components:

### 1. Agent Prompt Templates
**Purpose**: Provides the overall structure and instructions for the agent
**Locations**:
- `internal/eval_harness/templates/agent_prompt.txt` (AILANG)
- `internal/eval_harness/templates/agent_prompt_python.txt` (Python)

**Contains**: Workflow steps, success criteria, tools available, language-specific tips

**Placeholders**:
- `{{CAPS}}` - Filled from benchmark YAML `caps` field
- `{{TIMEOUT}}` - From `AgentBenchmarkConfig.TimeoutSeconds`
- `{{MAX_ITERATIONS}}` - From `AgentBenchmarkConfig.MaxIterations`

**Language Selection**: Template chosen based on `language` parameter:
- `"ailang"` → `agent_prompt.txt`
- `"python"` → `agent_prompt_python.txt`

### 2. Benchmark YAML Files
**Purpose**: Defines the specific problem to solve
**Location**: `benchmarks/*.yml`
**Key fields**:
- `task_prompt` - Problem description (goes into README.md)
- `caps` - Required capabilities like `["IO", "FS"]`
- `expected_stdout` - Expected output
- `prompt_files` (optional) - Can specify AILANG teaching prompt version

**Example**:
```yaml
id: hello_world
task_prompt: |
  Write a program that prints "Hello, World!"
caps: ["IO"]
expected_stdout: |
  Hello, World!
```

### 3. AILANG Teaching Prompts
**Purpose**: Complete language syntax reference
**Location**: `prompts/v0.3.22.md` (and other versions)
**Used as**: Full syntax reference embedded in `syntax_reference.md`

**Loaded by**: `LoadActiveSyntaxReference()` from `prompts/versions.json`

## Data Flow

```
Benchmark YAML (spec.yml)
    ↓
    ├─ task_prompt → README.md (workspace file)
    ├─ caps → agent_prompt.txt placeholders
    └─ expected_stdout → README.md

Agent Prompt Template (agent_prompt.txt)
    ↓
    ├─ Overall instructions
    ├─ {{CAPS}} replaced with benchmark caps
    └─ {{TIMEOUT}}, {{MAX_ITERATIONS}} replaced

AILANG Teaching Prompt (prompts/v0.3.22.md)
    ↓
    └─ syntax_reference.md (workspace file)

All three → Workspace directory:
    ├─ README.md (problem + expected output)
    ├─ solution.ail (agent writes this)
    └─ syntax_reference.md (full AILANG syntax)
```

## Customization Points

### Change agent instructions
**Edit**: `internal/eval_harness/templates/agent_prompt.txt`
**Effect**: Changes workflow, tips, success criteria for ALL benchmarks

### Change problem description
**Edit**: `benchmarks/<benchmark-id>.yml` → `task_prompt` field
**Effect**: Changes specific benchmark only

### Change AILANG syntax reference
**Option 1**: Set active version in `prompts/versions.json`
**Option 2**: Benchmark can specify `prompt_files: {ailang: "prompts/v0.3.15.md"}`
**Effect**: Agent sees different AILANG syntax/features

## Example: Customizing for Complex Benchmarks

Suppose you want more detailed instructions for complex algorithmic benchmarks:

**Option 1: Update agent_prompt.txt** (affects all benchmarks):
```markdown
## Tips

- Start simple: Get basic structure working first
- **For complex algorithms**: Break into smaller helper functions
- **For recursion**: Use explicit accumulator parameters
```

**Option 2: Add hints to benchmark YAML** (affects one benchmark):
```yaml
task_prompt: |
  Write a balanced binary search tree implementation.

  Hints:
  - Use ADT: type Tree = Leaf | Node(Tree, int, Tree)
  - Implement insert recursively
  - Use pattern matching for tree structure
```

**Option 3: Create prompt variant** (for testing):
```bash
# Copy active prompt
cp prompts/v0.3.22.md prompts/v0.3.23-algo-hints.md

# Edit to add algorithm-specific guidance

# Register in versions.json
# Use in benchmark:
prompt_files:
  ailang: "prompts/v0.3.23-algo-hints.md"
```

## Code References

- `agent_prompt.go:GenerateAgentPrompt()` - Loads and processes template
- `agent_prompt.go:LoadActiveSyntaxReference()` - Loads AILANG teaching prompt
- `agent_prompt.go:PrepareWorkspaceWithSyntax()` - Creates workspace files
- `spec.go:BenchmarkSpec` - Benchmark YAML structure
- `prompt_loader.go:GetActivePrompt()` - Loads active AILANG prompt

## Testing Integration

To verify the integration works:

```bash
# 1. Check benchmark loads correctly
ailang eval-suite --models claude-sonnet-4-5 --benchmarks hello_world --dry-run

# 2. Run with agent mode (when CLI integrated)
ailang eval-suite --agent --models claude-sonnet-4-5 --benchmarks hello_world

# 3. Inspect generated workspace (for debugging)
DEBUG_WORKSPACE=1 ailang eval-suite --agent --benchmarks hello_world
# This would show: README.md, solution.ail, syntax_reference.md contents
```

## Future Enhancements

Possible improvements:
- **Benchmark-specific templates**: Allow `template: "agent_prompt_algo.txt"` in YAML
- **Conditional tips**: Include different tips based on benchmark difficulty
- **Prompt chaining**: Reference multiple prompt files for hybrid guidance
- **Template variables**: More placeholders like `{{DIFFICULTY}}`, `{{CATEGORY}}`

See M-EVAL-AGENT design doc for roadmap.
