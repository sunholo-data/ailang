# Agent Benchmark Solutions: Multi-File & Advanced Validation

**Problem**: Current eval harness expects single file (`benchmark/solution.ail`), but some agent benchmarks need multi-file solutions or advanced validation.

**This document provides 3 solution approaches with increasing complexity.**

---

## Current Harness Constraints

From `internal/eval_harness/agent_runner.go` and `templates/agent_task_ailang.txt`:

**Hard Constraints**:
1. ✅ Solution must be at: `benchmark/solution.ail` (or `solution.py`)
2. ✅ Validation: `ailang run --entry main --caps <caps> benchmark/solution.ail`
3. ✅ Success: `stdout` must match `expected_stdout` EXACTLY
4. ✅ Workspace: Agent gets isolated `/tmp/ailang_eval/<benchmark>_<lang>_<pid>/`

**Opportunities**:
- ✅ Agent CAN create additional files in workspace
- ✅ Workspace is empty except for `benchmark/solution.ail` placeholder
- ✅ Agent has Bash, Read, Write, Edit, Grep tools
- ✅ `DEBUG_AGENT=1` preserves workspace for inspection

---

## Solution 1: Multi-File in Single Directory (Works NOW) ✅

**Status**: Ready to use, no harness changes needed

**How It Works**:
- Main solution at `benchmark/solution.ail`
- Agent creates additional modules at `benchmark/data.ail`, `benchmark/storage.ail`, etc.
- Main solution imports from same directory
- Harness runs main solution, which loads other modules

**Example Benchmark**: `multi_module_imports`

```yaml
id: multi_module_imports
description: "Multi-file module system with imports and effects"
languages: ["ailang", "python"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "hard"
expected_gain: "very_high"
task_prompt: |
  Create a multi-module program in <LANG> that demonstrates imports and effect composition.

  **IMPORTANT**: Create these files in the benchmark/ directory:
  - benchmark/solution.ail (main entry point)
  - benchmark/data.ail (data types module)
  - benchmark/storage.ail (storage functions module)

  File 1: benchmark/data.ail
  - Module declaration: module benchmark/data
  - Define User record: { name: string, age: int, email: string }
  - Define validateEmail(email: string) -> bool (checks for "@")
  - Export: User, validateEmail

  File 2: benchmark/storage.ail
  - Module declaration: module benchmark/storage
  - Import User from benchmark/data
  - Define saveUser(user: User, filename: string) !: FS -> ()
    - Encode user to JSON string
    - Write to file
  - Define loadUser(filename: string) !: FS -> User
    - Read file
    - Parse JSON to User
    - Return User
  - Export: saveUser, loadUser

  File 3: benchmark/solution.ail (main entry point)
  - Module declaration: module benchmark/solution
  - Import User, validateEmail from benchmark/data
  - Import saveUser, loadUser from benchmark/storage
  - Main function:
    1. Create test user: { name: "Alice", age: 30, email: "alice@example.com" }
    2. Validate email, print: "Email valid: {result}"
    3. Save to "user.json"
    4. Load back from "user.json"
    5. Print: "Loaded: {name}, age {age}"

  Requirements:
  - All three files must have correct module declarations
  - Imports must reference correct module paths
  - Effects must propagate correctly (FS in storage → main)
  - Final solution runs via: ailang run --entry main --caps IO,FS benchmark/solution.ail

expected_stdout: |
  Email valid: true
  Loaded: Alice, age 30
```

**Agent Instructions** (add to prompt):
```
CRITICAL: Create separate files for each module:
1. Write benchmark/data.ail with: module benchmark/data
2. Write benchmark/storage.ail with: module benchmark/storage
3. Write benchmark/solution.ail with: module benchmark/solution

DO NOT combine into one file! The benchmark tests multi-file imports.

Verify by running:
  ailang run --entry main --caps IO,FS benchmark/solution.ail
```

**Validation**: No changes needed - harness runs `benchmark/solution.ail` which imports others

**Status**: ✅ **Works NOW** - just need clear benchmark phrasing

---

## Solution 2: File Existence Validation (Small Harness Change) 🔧

**Status**: Requires small Go changes (~50 LOC)

**Use Case**: Benchmarks that produce output files, not just stdout

**Example**: CSV to JSON converter, config file generator

**Benchmark Spec Addition**:
```yaml
id: csv_to_json_converter
# ... existing fields ...
validation:
  files_must_exist:
    - "users.json"
  json_schema:
    users.json:
      type: "array"
      items:
        type: "object"
        required: ["name", "age", "email"]
expected_stdout: |
  Converted 3 valid rows to users.json
```

**Implementation** (`internal/eval_harness/agent_runner.go`):

```go
// Add to BenchmarkSpec
type BenchmarkSpec struct {
    // ... existing fields ...
    Validation *ValidationConfig `yaml:"validation,omitempty"`
}

type ValidationConfig struct {
    FilesMustExist []string               `yaml:"files_must_exist,omitempty"`
    JSONSchema     map[string]interface{} `yaml:"json_schema,omitempty"`
}

// After running solution, validate files exist
func validateAgentSolution(workspace string, spec *BenchmarkSpec) error {
    if spec.Validation == nil {
        return nil
    }

    for _, file := range spec.Validation.FilesMustExist {
        path := filepath.Join(workspace, file)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            return fmt.Errorf("validation failed: required file not found: %s", file)
        }
    }

    // TODO: Add JSON schema validation if needed

    return nil
}
```

**Effort**: ~2 hours (add validation config, file checks, tests)

**Value**: Enables benchmarks that test file generation (configs, CSVs, JSON exports)

---

## Solution 3: Custom Validation Scripts (Medium Harness Change) 🔨

**Status**: Requires moderate Go changes (~150 LOC)

**Use Case**: Complex validation (e.g., directory structure, multiple outputs, semantic checks)

**Example**: Recursive directory listing, log analyzer with stats file

**Benchmark Spec Addition**:
```yaml
id: recursive_directory_listing
# ... existing fields ...
validation:
  script: "benchmarks/validators/directory_listing.sh"
  # OR inline:
  inline_script: |
    #!/bin/bash
    # Check directory structure
    [ -f benchmark/file_list.txt ] || exit 1
    # Check file count
    lines=$(wc -l < benchmark/file_list.txt)
    [ "$lines" -eq 5 ] || exit 1
    exit 0
expected_stdout: |
  Found 5 files
```

**Implementation** (`internal/eval_harness/validators.go` - NEW):

```go
type ValidationConfig struct {
    FilesMustExist []string `yaml:"files_must_exist,omitempty"`
    Script         string   `yaml:"script,omitempty"`
    InlineScript   string   `yaml:"inline_script,omitempty"`
}

func RunValidationScript(workspace string, config *ValidationConfig) error {
    var scriptContent string
    if config.InlineScript != "" {
        scriptContent = config.InlineScript
    } else if config.Script != "" {
        data, err := os.ReadFile(config.Script)
        if err != nil {
            return fmt.Errorf("failed to read validation script: %w", err)
        }
        scriptContent = string(data)
    } else {
        return nil // No script validation
    }

    // Write to temp file
    tmpFile, err := os.CreateTemp("", "validator_*.sh")
    if err != nil {
        return err
    }
    defer os.Remove(tmpFile.Name())

    if err := os.WriteFile(tmpFile.Name(), []byte(scriptContent), 0755); err != nil {
        return err
    }

    // Run with workspace as CWD
    cmd := exec.Command("bash", tmpFile.Name())
    cmd.Dir = workspace
    output, err := cmd.CombinedOutput()

    if err != nil {
        return fmt.Errorf("validation script failed: %s\nOutput: %s", err, output)
    }

    return nil
}
```

**Effort**: ~4 hours (validator execution, security checks, tests)

**Value**: Maximum flexibility - can validate anything scriptable

---

## Recommended Approach: Phased Implementation

### Phase 1: Rephrase Benchmarks (Immediate) ✅

**What**: Use Solution 1 (multi-file in same directory)

**Benchmarks Ready NOW**:
1. ✅ `multi_module_imports` - 3 files in `benchmark/`
2. ✅ `config_file_parser` - Creates `app_config.json`, validates, single solution file
3. ✅ `log_file_analyzer` - Creates `app.log`, analyzes, single solution file
4. ✅ `state_machine_traffic_light` - Single file with ADTs
5. ✅ `tree_transformation_pipeline` - Single file with recursive functions

**Action**: Write benchmarks with clear multi-file instructions

**Example Template**:
```yaml
task_prompt: |
  Create a multi-file solution in <LANG>:

  <LANG=AILANG> Structure:
  - benchmark/solution.ail (module benchmark/solution) - main entry
  - benchmark/data.ail (module benchmark/data) - types
  - benchmark/helpers.ail (module benchmark/helpers) - utilities

  Verify with:
  ailang run --entry main --caps <caps> benchmark/solution.ail

  <LANG=PYTHON> Structure:
  - solution.py (main entry)
  - data.py (types)
  - helpers.py (utilities)

  Verify with:
  python3 solution.py
```

**Effort**: 2-3 hours to write 6 benchmarks

**Timeline**: This week

---

### Phase 2: Add File Validation (v0.3.25) 🔧

**What**: Implement Solution 2 (file existence checks)

**Enables New Benchmarks**:
1. `csv_to_json_converter` - Validates `users.json` exists
2. `config_generator` - Validates config file structure
3. `http_request_handler` - Validates response log exists

**Implementation**:
- Add `validation` field to `BenchmarkSpec`
- Implement `validateAgentSolution()` in `agent_runner.go`
- Add tests for validation logic

**Effort**: ~3 hours (design + impl + tests)

**Timeline**: Next sprint (v0.3.25)

---

### Phase 3: Custom Validators (v0.4.0+) 🔨

**What**: Implement Solution 3 (validation scripts)

**Enables Advanced Benchmarks**:
1. `recursive_directory_listing` - Validates directory structure
2. `code_generator` - Validates generated code compiles
3. `test_generator` - Validates tests pass

**Implementation**:
- Create `internal/eval_harness/validators.go`
- Add script execution with security sandboxing
- Document validator script API

**Effort**: ~6 hours (design + impl + security + tests)

**Timeline**: Post-v0.4.0 (deferred)

---

## Immediate Action Plan

### Week 1: Write Phase 1 Benchmarks

**Priority order** (easiest → hardest):

1. **csv_to_json_converter** (2 hours)
   - Single file: `benchmark/solution.ail`
   - Creates: `users.csv` input, produces stdout
   - NO file validation needed (just stdout)

2. **config_file_parser** (2 hours)
   - Single file: `benchmark/solution.ail`
   - Creates: `app_config.json`, reads it, produces stdout
   - NO file validation needed (just stdout)

3. **log_file_analyzer** (2 hours)
   - Single file: `benchmark/solution.ail`
   - Creates: `app.log`, analyzes it, produces stdout
   - NO file validation needed (just stdout)

4. **multi_module_imports** (3 hours)
   - Three files: `data.ail`, `storage.ail`, `solution.ail`
   - Tests module system
   - Requires clear multi-file instructions

5. **state_machine_traffic_light** (3 hours)
   - Single file: `benchmark/solution.ail`
   - Complex ADTs + pattern matching
   - Requires careful prompt

6. **tree_transformation_pipeline** (4 hours)
   - Single file: `benchmark/solution.ail`
   - Hardest: recursive HOFs + ADTs
   - Requires detailed examples

**Total**: 16 hours (~2 days)

---

### Week 2: Validate & Tune

1. **Run agent benchmarks** (4 hours)
   ```bash
   ailang eval-suite --agent \
     --models claude-sonnet-4-5 \
     --benchmarks csv_to_json_converter,config_file_parser,log_file_analyzer,multi_module_imports,state_machine_traffic_light,tree_transformation_pipeline \
     --output eval_results/new_benchmarks_test
   ```

2. **Analyze results** (2 hours)
   - Agent success rates
   - Common failure modes
   - Prompt improvements needed

3. **Tune prompts** (4 hours)
   - Add clarifications based on failures
   - Improve multi-file instructions
   - Add more examples if needed

4. **Re-run validation** (2 hours)
   - Verify improvements
   - Compare to 0-shot baseline
   - Document success metrics

**Total**: 12 hours (~1.5 days)

---

## Revised Benchmark Prompts (Solution 1 Ready)

### Multi-Module Imports (Clearer Version)

```yaml
id: multi_module_imports
description: "Multi-file module system with imports and effects"
languages: ["ailang", "python"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "hard"
expected_gain: "very_high"
task_prompt: |
  Create a multi-module program in <LANG> demonstrating imports and effect composition.

  <LANG=AILANG> You MUST create THREE separate files:

  **File 1: benchmark/data.ail**
  ```ailang
  module benchmark/data

  type User = { name: string, age: int, email: string }

  func validateEmail(email: string) -> bool =
    // Check if email contains "@"
    ...
  ```

  **File 2: benchmark/storage.ail**
  ```ailang
  module benchmark/storage
  import benchmark/data (User)

  func saveUser(user: User, filename: string) !: FS -> () =
    // Encode user to JSON and write to file
    ...

  func loadUser(filename: string) !: FS -> User =
    // Read file and parse JSON to User
    ...
  ```

  **File 3: benchmark/solution.ail** (main entry point)
  ```ailang
  module benchmark/solution
  import benchmark/data (User, validateEmail)
  import benchmark/storage (saveUser, loadUser)

  func main() !: IO,FS -> () =
    let user = { name: "Alice", age: 30, email: "alice@example.com" } in
    let valid = validateEmail(user.email) in
    let _ = println("Email valid: " ++ show(valid)) in
    let _ = saveUser(user, "user.json") in
    let loaded = loadUser("user.json") in
    println("Loaded: " ++ loaded.name ++ ", age " ++ show(loaded.age))
  ```

  **CRITICAL STEPS**:
  1. Create benchmark/data.ail (NOT solution.ail yet!)
  2. Create benchmark/storage.ail
  3. Create benchmark/solution.ail (main file)
  4. Verify: `ailang run --entry main --caps IO,FS benchmark/solution.ail`
  5. Check output matches exactly

  <LANG=PYTHON>
  Create three files: data.py, storage.py, solution.py
  Follow same structure with Python imports.

  **DO NOT** combine into one file - the benchmark tests multi-file imports!

expected_stdout: |
  Email valid: true
  Loaded: Alice, age 30
```

**Key Improvements**:
- ✅ Explicit "create THREE separate files" instruction
- ✅ Shows module declaration syntax
- ✅ Clear file order (data → storage → solution)
- ✅ Verification command included
- ✅ Warning not to combine files

---

### CSV to JSON (Single File, File Creation)

```yaml
id: csv_to_json_converter
description: "Parse CSV and convert to JSON with validation"
languages: ["ailang", "python"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "medium"
expected_gain: "high"
task_prompt: |
  Convert a CSV file to JSON in <LANG>:

  <LANG=AILANG> Write to: benchmark/solution.ail

  Steps:
  1. Create input CSV file "users.csv" with content:
     ```
     name,age,email
     Alice,30,alice@example.com
     Bob,25,bob@example.com
     Carol,35,carol@example.com
     ```

  2. Parse CSV (split by lines and commas)

  3. Validate each row:
     - Age must parse to positive int
     - Email must contain "@"
     - Skip invalid rows with warning

  4. Convert valid rows to JSON array:
     ```json
     [
       {"name": "Alice", "age": 30, "email": "alice@example.com"},
       {"name": "Bob", "age": 25, "email": "bob@example.com"},
       {"name": "Carol", "age": 35, "email": "carol@example.com"}
     ]
     ```

  5. Write JSON to "users.json"

  6. Print to stdout: "Converted N valid rows to users.json"

  Example code structure:
  ```ailang
  module benchmark/solution
  import std/json (encode)

  func parseCSV(content: string) -> List[{name: string, age: int, email: string}] =
    // Split lines, parse each row, validate
    ...

  func main() !: IO,FS -> () =
    // 1. Create users.csv
    // 2. Read and parse
    // 3. Convert to JSON
    // 4. Write users.json
    // 5. Print confirmation
    ...
  ```

  Verify: `ailang run --entry main --caps IO,FS benchmark/solution.ail`

  <LANG=PYTHON>
  Write to: solution.py
  Use csv and json modules.

expected_stdout: |
  Converted 3 valid rows to users.json
```

**Key Points**:
- ✅ Single file solution (simpler)
- ✅ Creates files as part of task (tests FS effects)
- ✅ Validation via stdout only (no file validation needed in harness)
- ✅ Clear step-by-step instructions

---

## Summary

**Solution 1 (Multi-File in Same Dir)**: ✅ **Use NOW**
- Works with current harness
- 6 benchmarks ready to implement
- Just needs clear prompt phrasing
- **Timeline**: This week

**Solution 2 (File Validation)**: 🔧 Defer to v0.3.25
- Small harness change (~50 LOC)
- Enables file-output benchmarks
- **Timeline**: Next sprint

**Solution 3 (Custom Validators)**: 🔨 Defer to v0.4.0+
- Larger harness change (~150 LOC)
- Maximum flexibility
- **Timeline**: Post-v0.4.0

**Next Step**: Implement 6 Phase 1 benchmarks using Solution 1 approach

Ready to create the YAML files?
