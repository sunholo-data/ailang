# AILANG Benchmark Audit & Agent Benchmark Recommendations

**Date**: 2025-10-29
**Context**: Analysis for selecting agent benchmarks and proposing new benchmarks
**Current Version**: v0.3.23 (agent success rate: 95% on 5 benchmarks)

---

## Executive Summary

Based on audit of 38 existing benchmarks, AILANG's current capabilities (v0.3.23), and v0.4 roadmap, this document provides:

1. **Categorization** of existing benchmarks by difficulty and current capability support
2. **Recommended agent benchmark suite** (15-20 benchmarks)
3. **New benchmark proposals** (8 new benchmarks for agent mode)

**Key Findings**:
- **5 current agent benchmarks** are too easy (95% success rate)
- **11 vision benchmarks** test AILANG differentiators (effects, determinism, totality)
- **v0.4 roadmap** will enable normalization, import suggestions, effect checking
- **Agent mode** needs longer, more complex tasks (3-10 minute completion time)

---

## Part 1: Existing Benchmark Categorization

### Category A: 100% Success (Smoke Tests) ✅

**Characteristics**: Both AILANG and Python should pass reliably, fast completion (<30s)

| Benchmark | Difficulty | Why It Works | Agent Value |
|-----------|------------|--------------|-------------|
| `fizzbuzz` | Easy | Simple control flow, no complex features | **Keep** - baseline sanity check |
| `recursion_factorial` | Easy | Basic recursion, well-known pattern | **Keep** - recursion test |
| `recursion_fibonacci` | Easy | Basic recursion, well-known pattern | **Keep** - recursion test |
| `simple_print` | Easy | Minimal IO, single effect | **Keep** - minimal viable program |
| `records_person` | Medium | Simple record types, field access | **Keep** - record basics |
| `list_operations` | Easy | List construction, pattern matching | **Keep** - list fundamentals |
| `string_manipulation` | Easy | String concat, comparisons | **Keep** - string basics |
| `nested_records` | Medium | Nested field access | **Keep** - realistic data structures |

**Recommendation**: Keep all 8 as **smoke tests** in agent suite.
**Expected Agent Success**: 95-100%
**Value**: Fast feedback that basic AILANG understanding works

---

### Category B: Hard But Achievable (Agent Differentiators) 🎯

**Characteristics**: Require iteration, debugging, effect management. Agent should outperform 0-shot.

| Benchmark | Difficulty | AILANG Challenge | Agent Advantage | Roadmap Unlock |
|-----------|------------|------------------|-----------------|----------------|
| `higher_order_functions` | Medium | Function composition, currying | Multi-turn refinement | None needed |
| `pattern_matching_complex` | Hard | Nested patterns, Tree ADT, guards | Compiler feedback helps | None needed |
| `record_update` | Medium | Update syntax `{r \| field: value}` | Syntax unfamiliar to LLMs | None needed |
| `effect_composition` | Hard | IO+FS propagation, pure/effectful separation | Effect debugging | v0.4 effect assertion |
| `effect_tracking_io_fs` | Hard | Multiple effects, correct signatures | Effect type errors | v0.4 effect assertion |
| `effect_pure_separation` | Medium | Pure vs effectful function design | Semantic understanding | v0.4 effect assertion |
| `exhaustive_pattern_matching` | Medium | Cover all ADT cases, no crashes | Compiler exhaustiveness checks | None needed |
| `type_safe_record_access` | Medium | Static field checking vs Python runtime errors | Type-driven debugging | None needed |
| `explicit_state_threading` | Hard | Pass state through functions (no globals) | Semantic understanding | None needed |
| `deterministic_list_transform` | Medium | Canonical list operations | Normalization feedback | v0.4 normalize |
| `referential_transparency` | Medium | Same input → same output | Purity understanding | v0.4 determinism checker |

**Recommendation**: Include **all 11** in agent suite.
**Expected Agent Success**: 60-80% (vs 30-50% 0-shot)
**Value**: Shows agent mode's value on harder problems

---

### Category C: Currently Impossible (Roadmap-Blocked) ⏳

**Characteristics**: Missing language features or stdlib. Will improve with v0.4+

| Benchmark | Blocker | Roadmap Fix | Agent Value Now | Agent Value Post-Fix |
|-----------|---------|-------------|-----------------|---------------------|
| `json_parse` | Missing JSON decode | **v0.3.22 fixed** | Include now | High |
| `json_encode` | Missing encode() | **v0.3.22 fixed** | Include now | High |
| `api_call_json` | Missing Net effect + JSON | v0.4.0 Net enhancements | Skip | High (post-v0.4) |
| `list_comprehension` | No stdlib map/filter/fold | v0.4.0 stdlib | Skip | Medium (post-v0.4) |
| `error_handling` | No Result/Either ADT | v0.4.0 stdlib | Skip | High (post-v0.4) |
| `cli_args` | No args parsing | v0.4.0 stdlib | Skip | Medium (post-v0.4) |
| `pipeline` | Stdin handling unclear | v0.4.0 IO enhancements | Skip | Medium (post-v0.4) |
| `canonical_normalization` | No normalizer | v0.4.0 normalize tool | Skip | Very High (post-v0.4) |

**Recommendation**: **Skip** roadmap-blocked benchmarks for now. Re-add in v0.4 agent suite.
**Expected Agent Success**: <20% (language limitations, not agent limitations)
**Value**: False negative - would unfairly penalize agent mode

---

### Category D: Vision-Specific Benchmarks 🌟

**Characteristics**: Test AILANG's differentiating features vs Python

| Benchmark | Tests | Keep for Agent? | Notes |
|-----------|-------|-----------------|-------|
| `immutable_data_structures` | Immutable updates vs mutations | **Yes** | Semantic understanding |
| `no_runtime_crashes_option` | Option types prevent null errors | **Yes** | Type safety |
| `print_with_show` | Show typeclass usage | **Yes** | Type-directed printing |
| `print_missing_effect` | Effect signature correctness | **Yes** | Effect system |

**Recommendation**: Include **all 4**.
**Expected Agent Success**: 70-90%
**Value**: Demonstrates AILANG's unique value propositions

---

## Part 2: Recommended Agent Benchmark Suite (v0.3.24+)

### Tier 1: Smoke Tests (8 benchmarks, <1 min each)
Fast sanity checks that basic AILANG works:
- `fizzbuzz`, `recursion_factorial`, `recursion_fibonacci`, `simple_print`
- `records_person`, `list_operations`, `string_manipulation`, `nested_records`

### Tier 2: Agent Differentiators (11 benchmarks, 1-3 min each)
Where agent mode shows clear value over 0-shot:
- `higher_order_functions`, `pattern_matching_complex`, `record_update`
- `effect_composition`, `effect_tracking_io_fs`, `effect_pure_separation`
- `exhaustive_pattern_matching`, `type_safe_record_access`
- `explicit_state_threading`, `deterministic_list_transform`, `referential_transparency`

### Tier 3: Vision Features (4 benchmarks, 1-2 min each)
AILANG's unique selling points:
- `immutable_data_structures`, `no_runtime_crashes_option`
- `print_with_show`, `print_missing_effect`

### Tier 4: JSON/Encoding (2 benchmarks, 2-4 min each)
Newly unblocked by v0.3.22:
- `json_parse`, `json_encode`

**Total: 25 benchmarks** (current agent suite: 5)

**Expected Performance**:
- Tier 1 (Smoke): 95-100% agent success
- Tier 2 (Differentiators): 60-80% agent success (vs 30-50% 0-shot)
- Tier 3 (Vision): 70-90% agent success
- Tier 4 (JSON): 50-70% agent success (new feature, less familiar)

**Estimated Runtime**: 25-50 minutes total (depends on agent turn count)

---

## Part 3: New Agent-Focused Benchmark Proposals

### Design Principles for Agent Benchmarks

**Good Agent Benchmarks Should**:
1. **Take 3-10 minutes** to complete (long enough to need iteration)
2. **Require multiple subtasks** (planning, implementation, debugging)
3. **Have clear success criteria** (deterministic stdout or file output)
4. **Test realistic use cases** (not just language exercises)
5. **Benefit from compiler feedback** (error messages guide debugging)
6. **Scale with complexity** (simple version works, then add features)

**Bad Agent Benchmarks**:
- ❌ Too easy (agent wastes time on trivial tasks)
- ❌ Too vague (unclear when "done")
- ❌ Non-deterministic (random output, time-dependent)
- ❌ Require external resources (databases, APIs not in stdlib)

---

### Proposal 1: Multi-File Module System 🆕

**ID**: `multi_module_imports`
**Difficulty**: Hard
**Expected Time**: 5-8 minutes
**Why Agent Mode**: Requires creating multiple files, coordinating imports, debugging module resolution

**Task**:
```yaml
id: multi_module_imports
description: "Multi-file module system with imports and effects"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "hard"
expected_gain: "very_high"
task_prompt: |
  Create a multi-file program in <LANG> with the following structure:

  File 1: data.ail (or data.py)
  - Define a User record type: { name: string, age: int, email: string }
  - Define a function validateEmail(email: string) -> bool
  - Export both

  File 2: storage.ail (or storage.py)
  - Import User from data module
  - Define saveUser(user: User, filename: string) that writes user to JSON file (FS effect)
  - Define loadUser(filename: string) that reads user from JSON file (FS effect)
  - Export both functions

  File 3: main.ail (or main.py)
  - Import User, validateEmail from data module
  - Import saveUser, loadUser from storage module
  - Create test user: { name: "Alice", age: 30, email: "alice@example.com" }
  - Validate email, print result
  - Save user to "user.json"
  - Load user back, print: "Loaded: {name}, age {age}"

  Requirements:
  - Proper module imports
  - Effect signatures (IO + FS in storage and main)
  - JSON encoding/decoding
  - All files must work together

expected_stdout: |
  Email valid: true
  Loaded: Alice, age 30
```

**Agent Challenges**:
- Creating multiple files
- Coordinating imports across files
- Effect propagation (FS in storage → main)
- JSON encoding/decoding
- File path handling

**Success Metrics**:
- Agent success: 40-60% (complex coordination)
- 0-shot success: <10% (too many moving parts)
- **Agent superiority**: 30-50pp improvement

---

### Proposal 2: State Machine Implementation 🆕

**ID**: `state_machine_traffic_light`
**Difficulty**: Hard
**Expected Time**: 6-10 minutes
**Why Agent Mode**: Requires ADT design, exhaustive pattern matching, state threading

**Task**:
```yaml
id: state_machine_traffic_light
description: "State machine with ADTs and exhaustive pattern matching"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "hard"
expected_gain: "very_high"
task_prompt: |
  Implement a traffic light state machine in <LANG>:

  1. Define a State ADT:
     - Green { timer: int }
     - Yellow { timer: int }
     - Red { timer: int }

  2. Define Event ADT:
     - Tick
     - Reset

  3. Implement transition(state: State, event: Event) -> State:
     - Green + Tick: if timer > 0, decrement timer, else → Yellow(3)
     - Yellow + Tick: if timer > 0, decrement timer, else → Red(10)
     - Red + Tick: if timer > 0, decrement timer, else → Green(20)
     - Any + Reset → Green(20)
     - Must be exhaustive (all cases handled)

  4. Implement show_state(state: State) -> string:
     - Green → "GREEN (Xsec)"
     - Yellow → "YELLOW (Xsec)"
     - Red → "RED (Xsec)"

  5. Main simulation:
     - Start: Green(20)
     - Apply events: [Tick × 25, Reset, Tick × 5]
     - Print state after each event

  Requirements:
  - Algebraic data types
  - Exhaustive pattern matching (no missing cases)
  - Explicit state threading (no mutations)
  - Clear output format

expected_stdout: |
  GREEN (19sec)
  GREEN (18sec)
  ...
  GREEN (1sec)
  YELLOW (3sec)
  YELLOW (2sec)
  YELLOW (1sec)
  RED (10sec)
  ...
  GREEN (20sec)
  GREEN (19sec)
  ...
```

**Agent Challenges**:
- Designing ADTs
- Writing exhaustive pattern matches
- State threading without mutation
- String formatting with show()

**Success Metrics**:
- Agent success: 50-70% (ADTs + exhaustiveness)
- 0-shot success: 20-30% (pattern matching errors)
- **Agent superiority**: 30-40pp improvement

---

### Proposal 3: Tree Transformation Pipeline 🆕

**ID**: `tree_transformation_pipeline`
**Difficulty**: Hard
**Expected Time**: 7-12 minutes
**Why Agent Mode**: Recursive data structures, higher-order functions, composition

**Task**:
```yaml
id: tree_transformation_pipeline
description: "Recursive tree transformations with higher-order functions"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]
difficulty: "hard"
expected_gain: "very_high"
task_prompt: |
  Implement a binary tree transformation pipeline in <LANG>:

  1. Define Tree ADT:
     - Leaf { value: int }
     - Node { left: Tree, value: int, right: Tree }

  2. Implement mapTree(fn: int -> int, tree: Tree) -> Tree:
     - Recursively apply fn to all values

  3. Implement foldTree(fn: int -> int -> int, acc: int, tree: Tree) -> int:
     - In-order fold over tree values

  4. Implement filterTree(pred: int -> bool, tree: Tree) -> Tree:
     - Keep only values where pred(value) is true
     - Restructure tree to maintain valid Tree structure

  5. Implement treeDepth(tree: Tree) -> int:
     - Calculate maximum depth

  6. Test with example tree:
     ```
           5
          / \
         3   8
        /   / \
       1   7   9
     ```

  Operations:
  - Map: double all values → [2,6,10,14,16,18]
  - Filter: keep only evens → [2,6,10,14,18]
  - Fold: sum all values
  - Depth: calculate depth (3)

  Requirements:
  - Recursive ADT operations
  - Higher-order functions
  - Pattern matching on Tree
  - Print results of each operation

expected_stdout: |
  Original: 1, 3, 5, 7, 8, 9
  Doubled: 2, 6, 10, 14, 16, 18
  Evens only: 2, 6, 10, 14, 18
  Sum: 32
  Depth: 3
```

**Agent Challenges**:
- Recursive data structures
- Higher-order function design
- Pattern matching on nested ADTs
- Maintaining tree invariants during filtering

**Success Metrics**:
- Agent success: 40-60% (recursion + HOFs)
- 0-shot success: 15-25% (complex recursion errors)
- **Agent superiority**: 25-35pp improvement

---

### Proposal 4: HTTP Request Handler (Post-v0.4) 🔮

**ID**: `http_request_handler`
**Difficulty**: Very Hard
**Expected Time**: 10-15 minutes
**Why Agent Mode**: Effect composition, error handling, JSON parsing

**Task**:
```yaml
id: http_request_handler
description: "HTTP request with JSON parsing and error handling"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO", "Net"]
difficulty: "very_hard"
expected_gain: "very_high"
task_prompt: |
  Build an HTTP client that fetches and processes user data in <LANG>:

  1. Fetch user data from API: GET https://jsonplaceholder.typicode.com/users/1

  2. Parse JSON response to extract:
     - name
     - email
     - company.name

  3. Error handling:
     - Network errors → print "Network error: {details}"
     - JSON parse errors → print "Parse error: {details}"
     - Missing fields → print "Missing field: {field}"

  4. On success:
     - Print: "User: {name}"
     - Print: "Email: {email}"
     - Print: "Company: {company}"

  5. Type safety:
     - Use Result type for errors
     - No exceptions/crashes

  Requirements:
  - Net effect for HTTP
  - JSON decoding with error handling
  - Result type usage
  - Proper effect signatures

expected_stdout: |
  User: Leanne Graham
  Email: Sincere@april.biz
  Company: Romaguera-Crona
```

**Agent Challenges**:
- HTTP request construction
- JSON parsing with nested fields
- Error handling with Result
- Effect composition (Net + IO)

**Success Metrics** (post-v0.4):
- Agent success: 30-50% (complex effects + errors)
- 0-shot success: <10% (too many failure points)
- **Agent superiority**: 20-40pp improvement

**Note**: Blocked until v0.4.0 (Net effect enhancements)

---

### Proposal 5: Configuration File Parser 🆕

**ID**: `config_file_parser`
**Difficulty**: Hard
**Expected Time**: 8-12 minutes
**Why Agent Mode**: FS effects, JSON parsing, validation, error reporting

**Task**:
```yaml
id: config_file_parser
description: "Parse and validate JSON configuration file"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "hard"
expected_gain: "high"
task_prompt: |
  Build a configuration file parser in <LANG>:

  1. Create a JSON config file "app_config.json":
     {
       "app_name": "MyApp",
       "version": "1.0.0",
       "port": 8080,
       "features": ["logging", "auth", "api"]
     }

  2. Define Config record type:
     { app_name: string, version: string, port: int, features: List[string] }

  3. Implement loadConfig(filename: string) -> Result[Config, string]:
     - Read file (FS effect)
     - Parse JSON
     - Validate:
       * port must be 1024-65535
       * version must match "X.Y.Z" format
       * features list not empty
     - Return Result (Ok or Err with message)

  4. Main:
     - Load config
     - On success: print "Loaded {app_name} v{version} on port {port} with N features"
     - On error: print "Config error: {message}"

  Requirements:
  - FS effect for file reading
  - JSON parsing
  - Validation logic
  - Result type error handling
  - Clear error messages

expected_stdout: |
  Loaded MyApp v1.0.0 on port 8080 with 3 features
```

**Agent Challenges**:
- File I/O with FS effect
- JSON parsing and field extraction
- Validation logic
- Result type usage
- Error message construction

**Success Metrics**:
- Agent success: 50-70% (multiple effects + validation)
- 0-shot success: 20-30% (effect errors common)
- **Agent superiority**: 30-40pp improvement

---

### Proposal 6: Log File Analyzer 🆕

**ID**: `log_file_analyzer`
**Difficulty**: Hard
**Expected Time**: 6-10 minutes
**Why Agent Mode**: String processing, FS effects, aggregation

**Task**:
```yaml
id: log_file_analyzer
description: "Parse log file and compute statistics"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "hard"
expected_gain: "high"
task_prompt: |
  Analyze a log file and compute statistics in <LANG>:

  1. Create log file "app.log":
     2024-10-29 10:00:00 INFO User logged in: alice
     2024-10-29 10:01:15 ERROR Database connection failed
     2024-10-29 10:02:30 INFO User logged in: bob
     2024-10-29 10:03:45 WARNING High memory usage: 85%
     2024-10-29 10:05:00 ERROR API timeout: /users endpoint
     2024-10-29 10:06:15 INFO User logged out: alice

  2. Parse log entries:
     - Extract: timestamp, level (INFO/ERROR/WARNING), message

  3. Compute statistics:
     - Total entries
     - Breakdown by level: INFO: X, ERROR: Y, WARNING: Z
     - List of unique users mentioned

  4. Output format:
     Total log entries: N
     INFO: X (X%)
     ERROR: Y (Y%)
     WARNING: Z (Z%)
     Unique users: alice, bob

  Requirements:
  - FS effect for file reading
  - String parsing (split, match)
  - Aggregation with records/lists
  - Percentage calculation (float)

expected_stdout: |
  Total log entries: 6
  INFO: 3 (50%)
  ERROR: 2 (33%)
  WARNING: 1 (17%)
  Unique users: alice, bob
```

**Agent Challenges**:
- File reading with FS effect
- String parsing and pattern extraction
- Stateful aggregation (counts, unique sets)
- Float arithmetic for percentages

**Success Metrics**:
- Agent success: 60-80% (practical task)
- 0-shot success: 30-40% (string parsing errors)
- **Agent superiority**: 30-40pp improvement

---

### Proposal 7: CSV to JSON Converter 🆕

**ID**: `csv_to_json_converter`
**Difficulty**: Medium-Hard
**Expected Time**: 5-8 minutes
**Why Agent Mode**: FS effects, parsing, data transformation

**Task**:
```yaml
id: csv_to_json_converter
description: "Convert CSV file to JSON with validation"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "medium"
expected_gain: "high"
task_prompt: |
  Convert a CSV file to JSON in <LANG>:

  1. Create CSV file "users.csv":
     name,age,email
     Alice,30,alice@example.com
     Bob,25,bob@example.com
     Carol,35,carol@example.com

  2. Parse CSV:
     - First line is headers
     - Remaining lines are data

  3. Convert to JSON array of objects:
     [
       {"name": "Alice", "age": 30, "email": "alice@example.com"},
       {"name": "Bob", "age": 25, "email": "bob@example.com"},
       {"name": "Carol", "age": 35, "email": "carol@example.com"}
     ]

  4. Write to "users.json"

  5. Validation:
     - Age must be positive integer
     - Email must contain "@"
     - Skip invalid rows, print warning

  6. Print: "Converted N valid rows to users.json"

  Requirements:
  - FS effect (read CSV, write JSON)
  - String parsing (split by comma/newline)
  - Validation logic
  - JSON encoding

expected_stdout: |
  Converted 3 valid rows to users.json
```

**Agent Challenges**:
- CSV parsing (string split)
- Data validation
- JSON encoding
- File I/O coordination (read → transform → write)

**Success Metrics**:
- Agent success: 70-85% (practical, clear task)
- 0-shot success: 40-50% (FS effect errors)
- **Agent superiority**: 30-35pp improvement

---

### Proposal 8: Recursive Directory Listing (Post-v0.4) 🔮

**ID**: `recursive_directory_listing`
**Difficulty**: Very Hard
**Expected Time**: 8-12 minutes
**Why Agent Mode**: Recursive FS operations, tree building, filtering

**Task**:
```yaml
id: recursive_directory_listing
description: "Recursively list directory structure with filtering"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO", "FS"]
difficulty: "very_hard"
expected_gain: "very_high"
task_prompt: |
  Build a recursive directory lister in <LANG>:

  1. Create test directory structure:
     test_dir/
       file1.txt
       file2.md
       subdir1/
         file3.txt
         file4.py
       subdir2/
         file5.md

  2. Implement listDir(path: string, filter: string -> bool) -> List[string]:
     - Recursively traverse directory
     - Apply filter function to each file
     - Return list of matching file paths

  3. Filters:
     - All files: always true
     - Text files: ends with ".txt"
     - Markdown files: ends with ".md"

  4. Output:
     - Print "All files:" followed by all file paths
     - Print "Text files:" followed by .txt files
     - Print "Markdown files:" followed by .md files

  Requirements:
  - FS effect (read directory)
  - Recursive traversal
  - Higher-order filter function
  - Path handling

expected_stdout: |
  All files:
    test_dir/file1.txt
    test_dir/file2.md
    test_dir/subdir1/file3.txt
    test_dir/subdir1/file4.py
    test_dir/subdir2/file5.md
  Text files:
    test_dir/file1.txt
    test_dir/subdir1/file3.txt
  Markdown files:
    test_dir/file2.md
    test_dir/subdir2/file5.md
```

**Agent Challenges**:
- Recursive directory traversal
- FS effect for directory operations
- Higher-order filter functions
- Path manipulation

**Success Metrics** (post-v0.4):
- Agent success: 30-50% (complex FS recursion)
- 0-shot success: <15% (FS effects + recursion errors)
- **Agent superiority**: 15-35pp improvement

**Note**: Blocked until v0.4.0 (enhanced FS effect API)

---

## Part 4: Implementation Recommendations

### Phase 1: Validate Agent Suite (Immediate)

**Week 1-2**: Run agent mode on all 25 recommended benchmarks:
```bash
ailang eval-suite --agent \
  --models claude-sonnet-4-5,claude-haiku-4-5 \
  --benchmarks fizzbuzz,recursion_factorial,...,json_encode \
  --output eval_results/agent_suite_v1
```

**Success Criteria**:
- Agent success ≥60% overall
- Agent > 0-shot by ≥20pp on Tier 2 (Differentiators)
- Agent > 0-shot by ≥30pp on Tier 3 (Vision)
- ≥80% success on Tier 1 (Smoke tests)

**If Criteria Met**: Current suite is well-balanced, proceed to Phase 2.
**If Not Met**: Adjust difficulty mix (add more Tier 1, reduce Tier 2).

---

### Phase 2: Add New Agent Benchmarks (v0.3.24)

**Week 3-4**: Implement and validate new benchmarks ready for v0.3.x:

**Immediately Ready** (no new language features):
1. ✅ `multi_module_imports` - Tests module system
2. ✅ `state_machine_traffic_light` - Tests ADTs + pattern matching
3. ✅ `tree_transformation_pipeline` - Tests recursion + HOFs
4. ✅ `config_file_parser` - Tests FS + JSON + validation
5. ✅ `log_file_analyzer` - Tests FS + string parsing
6. ✅ `csv_to_json_converter` - Tests FS + transformation

**Implementation Order** (by estimated difficulty):
1. `csv_to_json_converter` (easiest, clear task) - **Start here**
2. `config_file_parser` (validation adds complexity)
3. `log_file_analyzer` (string parsing challenges)
4. `multi_module_imports` (multi-file coordination)
5. `state_machine_traffic_light` (ADT design complexity)
6. `tree_transformation_pipeline` (hardest, recursive HOFs)

**Validation**:
```bash
# Test each new benchmark
ailang eval-suite --agent \
  --benchmark csv_to_json_converter \
  --models claude-sonnet-4-5 \
  --output eval_results/new_benchmarks/

# Compare to 0-shot baseline
ailang eval-suite \
  --benchmark csv_to_json_converter \
  --models claude-sonnet-4-5 \
  --output eval_results/new_benchmarks_baseline/
```

**Success Criteria for Each New Benchmark**:
- Agent success: ≥40% (shows it's achievable)
- Agent > 0-shot by ≥20pp (shows agent adds value)
- 0-shot success: 10-30% (shows it's non-trivial)
- Avg completion time: 3-10 minutes (shows it's substantive)

---

### Phase 3: Post-v0.4 Benchmarks (Deferred)

**After v0.4.0 ships**, add benchmarks that test roadmap features:

**Blocked Until v0.4.0**:
1. `http_request_handler` - Needs Net effect enhancements
2. `recursive_directory_listing` - Needs enhanced FS API
3. `canonical_normalization` - Needs normalize tool

**New Benchmarks Enabled by v0.4.0**:
- `import_suggestion_test` - Tests `ailang suggest-imports`
- `effect_assertion_test` - Tests `ailang run --assert-effect`
- `determinism_verification` - Tests `ailang run --determinism-check`

---

## Part 5: Metrics & Success Tracking

### Benchmark Suite Health Metrics

**Balance Check**:
- Tier 1 (Smoke): 30-40% of benchmarks
- Tier 2 (Differentiators): 40-50% of benchmarks
- Tier 3 (Vision): 10-20% of benchmarks
- New/Experimental: <10% of benchmarks

**Current Recommended Suite (v0.3.24)**:
- Tier 1: 8 (32%)
- Tier 2: 11 (44%)
- Tier 3: 4 (16%)
- Tier 4: 2 (8%)
- **Total: 25 benchmarks** ✅ Well-balanced

**With New Benchmarks (v0.3.25)**:
- Tier 1: 8 (26%)
- Tier 2: 11 + 6 new = 17 (55%)
- Tier 3: 4 (13%)
- Tier 4: 2 (6%)
- **Total: 31 benchmarks** ✅ Still balanced

---

### Agent Value Metrics

**Track These Metrics Per Benchmark**:
1. **Agent Success Rate** - % of agent runs that succeed
2. **0-Shot Success Rate** - % of 0-shot runs that succeed
3. **Agent Superiority** - (Agent - 0-Shot) in percentage points
4. **Avg Agent Turns** - How many iterations agent needs
5. **Avg Agent Time** - Wall-clock time to completion
6. **Repair Success Rate** - % of failures fixed by one repair

**Good Agent Benchmark Indicators**:
- ✅ Agent Superiority ≥20pp (agent clearly better)
- ✅ Agent Success 40-80% (achievable but non-trivial)
- ✅ 0-Shot Success 10-30% (hard enough to need agent)
- ✅ Avg Agent Turns: 5-15 (enough iteration to show value)
- ✅ Avg Agent Time: 3-10 min (substantive but not excessive)

**Bad Agent Benchmark Indicators**:
- ❌ Agent Superiority <10pp (agent not adding value)
- ❌ Agent Success >95% (too easy, not differentiating)
- ❌ Agent Success <20% (too hard, language limitations)
- ❌ Avg Agent Turns >20 (benchmark too vague/hard)
- ❌ Avg Agent Time >15 min (cost-prohibitive for evals)

---

## Part 6: Next Steps

### Immediate Actions (This Week)

1. ✅ **Review this analysis** with team
2. **Decide on Phase 1 suite** (25 existing benchmarks or adjust?)
3. **Run Phase 1 validation** (`ailang eval-suite --agent` on 25 benchmarks)
4. **Analyze Phase 1 results** and confirm suite balance

### Short-Term (Next 2 Weeks)

1. **Implement new benchmarks** in priority order (csv → tree pipeline)
2. **Validate each new benchmark** (agent vs 0-shot)
3. **Update benchmark docs** (README, VISION_BENCHMARKS.md)
4. **Add new benchmarks to CI** (regression tracking)

### Medium-Term (Post-v0.4.0)

1. **Add v0.4-enabled benchmarks** (normalization, effects, Net)
2. **Re-run full agent suite** on v0.4.0
3. **Measure roadmap impact** (did new features improve agent success?)
4. **Update dashboard** with agent vs 0-shot vs repair metrics

---

## Appendix A: Benchmark Difficulty Calibration

**Easy** (1-2 min, 80-100% agent success):
- Basic syntax, single feature
- Examples: fizzbuzz, simple recursion, simple print

**Medium** (2-5 min, 60-80% agent success):
- Multiple features, some debugging
- Examples: records, nested structures, simple effects

**Hard** (5-10 min, 40-60% agent success):
- Complex features, multi-step debugging, effects composition
- Examples: effect composition, pattern matching trees, multi-module

**Very Hard** (10-15 min, 20-40% agent success):
- Novel problems, unclear requirements, multiple failure modes
- Examples: HTTP handlers, recursive FS ops, complex state machines

---

## Appendix B: Python vs AILANG Comparison

**Where Python Should Win**:
- ✅ **Familiarity**: LLMs know Python better (more training data)
- ✅ **Flexibility**: Multiple ways to solve problems
- ✅ **Stdlib**: Rich standard library

**Where AILANG Should Win**:
- ✅ **Type Safety**: Compile-time error detection
- ✅ **Effect Tracking**: Explicit IO/FS/Net in types
- ✅ **Determinism**: Referential transparency, no hidden state
- ✅ **Totality**: Exhaustive pattern matching prevents crashes

**Agent Mode Advantage**:
- ✅ **Compiler feedback**: AILANG's structured errors guide debugging
- ✅ **Effect debugging**: Missing effect signatures caught early
- ✅ **Type errors**: Clear type mismatches vs Python runtime errors

**Expected Outcome**:
- Python: Higher 0-shot success (familiarity)
- AILANG: Better agent improvement (compiler feedback)
- AILANG: Fewer total tokens (concise, structured errors)

---

**End of Analysis**

Questions? Feedback? Ready to implement Phase 1?
