# Evaluation Dashboard Reliability

**Status**: Planned
**Created**: 2025-10-15
**Priority**: High
**Area**: M-EVAL-LOOP (Benchmarking & Evaluation)

## Problem Statement

The current process for updating the benchmark dashboard (`docs/static/benchmarks/latest.json`) is flaky, error-prone, and destroys historical data. Multiple incidents during v0.3.9 release:

1. **History destruction**: `ailang eval-report --format=json` regenerates JSON from scratch, losing carefully curated historical entries
2. **Manual intervention required**: Had to use Python script to manually calculate stats and add v0.3.9 to history array
3. **Multiple sources of truth**: `baseline.json`, `performance_tables/*.json`, and `latest.json` contain different/incomplete data
4. **No validation**: Easy to corrupt JSON structure with manual edits
5. **Version mismatches**: Git describe output (v0.3.7-46-g2cfa80a) vs actual version (v0.3.9) causes confusion

### Incident Timeline (v0.3.9)

```
1. Initial state: latest.json has v0.3.8 with 4 historical versions
2. Run: ailang eval-report --format=json > latest.json
   Result: History reduced to 2 versions (v0.3.7-46, v0.3.7-45)
   Lost: v0.3.8, v0.3.7-1, v0.3.6-24-mini, v0.3.6-24
3. Restore: git checkout HEAD~1 -- latest.json
4. Manual fix: Python script to calculate stats and insert v0.3.9
5. Final: Dashboard working, but process untrustworthy
```

### Current Flaky Workflow

```bash
# ❌ WRONG - This destroys history!
make eval-baseline
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=json > docs/static/benchmarks/latest.json
# Result: All historical data lost

# ✅ CURRENT WORKAROUND - Manual Python script
# 1. Restore old JSON
git checkout HEAD~1 -- docs/static/benchmarks/latest.json

# 2. Calculate stats manually
find eval_results/baselines/v0.3.9 -name "*_ailang_*.json" -exec jq -r 'select(.stdout_ok == true)' {} \; | wc -l
# ... repeat for python, total, etc.

# 3. Edit JSON with Python script
python3 << 'EOF'
import json
# ... 20 lines of manual stat calculation and JSON manipulation
EOF
```

## Root Cause Analysis

### Why `eval-report --format=json` Destroys History

**File**: `internal/eval_analysis/reporter.go` (assumed location)

The JSON format generator:
1. Scans `eval_results/baselines/*` for available baselines
2. Builds history array from **only** the baselines it finds
3. Overwrites entire JSON file with regenerated data

**Missing baselines problem**:
- v0.3.8 baseline was never committed to git (local only)
- v0.3.7-1, v0.3.6-24, etc. were deleted or never existed locally
- Generator can only see v0.3.9 and v0.3.7-45 (incomplete)
- Result: History shrinks from 5 versions to 2

### Multiple Sources of Truth

**1. Baseline metadata** (`eval_results/baselines/VERSION/baseline.json`):
```json
{
  "version": "v0.3.7-46-g2cfa80a",  // Git describe, not actual version!
  "total_runs": 126,
  "success_count": 20,  // WRONG! (This is success_count from jq, not actual)
  "fail_count": 106
}
```

**2. Performance table** (`eval_results/performance_tables/VERSION.json`):
```json
{
  "version": "v0.3.9",
  "languageStats": null,  // Missing!
  "successRate": null,    // Missing!
  "totalRuns": null       // Missing!
}
```

**3. Dashboard JSON** (`docs/static/benchmarks/latest.json`):
```json
{
  "version": "v0.3.9",
  "history": [
    {
      "version": "v0.3.9",
      "successRate": 0.5873,
      "languageStats": {
        "ailang": {"success_rate": 0.460, "total_runs": 63},
        "python": {"success_rate": 0.714, "total_runs": 63}
      }
    }
  ]
}
```

**All three have different data!** No single source of truth.

## Design Requirements

### 1. History Preservation

**MUST preserve historical data when updating dashboard JSON.**

```go
// ✅ CORRECT APPROACH
func GenerateDashboardJSON(baselineDir, version string) error {
    // 1. Read existing latest.json (if exists)
    existing := readExistingJSON("docs/static/benchmarks/latest.json")

    // 2. Calculate stats for new version from baseline
    newEntry := calculateStatsFromBaseline(baselineDir, version)

    // 3. Check if version already exists in history
    if !existing.HasVersion(version) {
        // 4. Append new entry to history (prepend for reverse chronological)
        existing.History = append([]HistoryEntry{newEntry}, existing.History...)
    } else {
        // Update existing entry
        existing.UpdateVersion(version, newEntry)
    }

    // 5. Update current version pointer
    existing.Version = version

    // 6. Validate JSON structure
    if err := existing.Validate(); err != nil {
        return err
    }

    // 7. Write atomically
    return writeJSONAtomic("docs/static/benchmarks/latest.json", existing)
}
```

### 2. Single Source of Truth

**Baseline directory is the canonical source. Everything else is derived.**

```
eval_results/baselines/v0.3.9/
├── adt_option_ailang_gpt5-mini_*.json     # Individual run results
├── adt_option_python_gpt5-mini_*.json
├── ... (126 total result files)
└── baseline.json                           # Metadata (version, timestamp, models)

# Generated from baseline (derived data):
├── eval_results/performance_tables/v0.3.9.json  # Full performance matrix
└── docs/static/benchmarks/latest.json           # Dashboard data + history
```

**Baseline metadata improvements**:
```json
{
  "version": "v0.3.9",           // Actual version, not git describe
  "git_describe": "v0.3.7-46-g2cfa80a",  // Keep git info separate
  "timestamp": "2025-10-15T22:52:07Z",
  "models": ["gpt5-mini", "claude-haiku-4-5", "gemini-2-5-flash"],
  "total_runs": 126,
  "languages": ["python", "ailang"]
  // Don't include success_count here - calculate from result files
}
```

### 3. Validation

**JSON structure must be validated before writing.**

```go
type DashboardJSON struct {
    Version   string         `json:"version"`
    Timestamp string         `json:"timestamp"`
    History   []HistoryEntry `json:"history"`
    Benchmarks map[string]BenchmarkData `json:"benchmarks"`
    Models    map[string]ModelData `json:"models"`
}

func (d *DashboardJSON) Validate() error {
    if d.Version == "" {
        return errors.New("version is required")
    }

    if len(d.History) == 0 {
        return errors.New("history must have at least one entry")
    }

    // Validate each history entry
    for i, entry := range d.History {
        if err := entry.Validate(); err != nil {
            return fmt.Errorf("history[%d]: %w", i, err)
        }
    }

    // Ensure versions are unique
    seen := make(map[string]bool)
    for _, entry := range d.History {
        if seen[entry.Version] {
            return fmt.Errorf("duplicate version in history: %s", entry.Version)
        }
        seen[entry.Version] = true
    }

    return nil
}
```

### 4. Atomic Writes

**Never leave corrupted JSON files.**

```go
func writeJSONAtomic(path string, data interface{}) error {
    // 1. Marshal JSON with proper formatting
    jsonBytes, err := json.MarshalIndent(data, "", "  ")
    if err != nil {
        return err
    }

    // 2. Write to temporary file
    tmpPath := path + ".tmp"
    if err := os.WriteFile(tmpPath, jsonBytes, 0644); err != nil {
        return err
    }

    // 3. Validate temporary file is valid JSON
    if err := validateJSONFile(tmpPath); err != nil {
        os.Remove(tmpPath)
        return err
    }

    // 4. Atomic rename
    return os.Rename(tmpPath, path)
}
```

## Proposed Solution

### Phase 1: Fix `ailang eval-report --format=json`

**Location**: `internal/eval_analysis/reporter.go` or `cmd/ailang/eval_report.go`

**Changes**:
1. Add `--append` flag (make it default behavior)
2. Read existing `latest.json` before writing
3. Merge new version data into existing history
4. Add JSON validation before writing
5. Use atomic writes

**New behavior**:
```bash
# Update dashboard with new version (preserves history)
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=json --output docs/static/benchmarks/latest.json

# Output:
# Loading existing dashboard data...
# ✓ Found existing history with 4 versions
# Calculating stats for v0.3.9...
# ✓ 126 runs (74 passed, 52 failed)
# ✓ AILANG: 46.0% (29/63)
# ✓ Python: 71.4% (45/63)
# Adding v0.3.9 to history...
# ✓ Validated JSON structure
# ✓ Dashboard updated: 5 versions in history
```

### Phase 2: Unified Metadata

**Fix baseline metadata to use actual version**:

```bash
# In tools/eval_baseline.sh:

# Before (WRONG):
VERSION="${1:-$(git describe --tags --always 2>/dev/null || echo "dev")}"

# After (CORRECT):
VERSION="${1}"  # MUST be specified explicitly
if [ -z "$VERSION" ]; then
    echo "Error: Version is required"
    echo "Usage: make eval-baseline VERSION=v0.3.9"
    exit 1
fi
GIT_DESCRIBE="$(git describe --tags --always 2>/dev/null || echo "unknown")"
```

**Update baseline.json structure**:
```json
{
  "version": "v0.3.9",  // Explicit version (not git describe)
  "git_describe": "v0.3.7-46-g2cfa80a",
  "git_commit": "2cfa80a...",
  "timestamp": "2025-10-15T22:52:07Z",
  "models": ["gpt5-mini", "claude-haiku-4-5", "gemini-2-5-flash"],
  "languages": ["python", "ailang"],
  "total_runs": 126
}
```

### Phase 3: Stat Calculation from Result Files

**Always calculate success rates from actual result files, never cache in metadata.**

```go
func CalculateStatsFromBaseline(baselineDir string) (*VersionStats, error) {
    // 1. Read all result JSON files
    files, err := filepath.Glob(filepath.Join(baselineDir, "*.json"))

    stats := &VersionStats{
        ByLanguage: make(map[string]LanguageStats),
    }

    for _, file := range files {
        if filepath.Base(file) == "baseline.json" {
            continue  // Skip metadata
        }

        result, err := loadResultFile(file)
        if err != nil {
            return nil, err
        }

        // Track by language
        langStats := stats.ByLanguage[result.Language]
        langStats.TotalRuns++
        if result.StdoutOk {
            langStats.SuccessCount++
        }
        stats.ByLanguage[result.Language] = langStats

        stats.TotalRuns++
        if result.StdoutOk {
            stats.SuccessCount++
        }
    }

    // Calculate success rates
    stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalRuns)
    for lang, langStats := range stats.ByLanguage {
        langStats.SuccessRate = float64(langStats.SuccessCount) / float64(langStats.TotalRuns)
        stats.ByLanguage[lang] = langStats
    }

    return stats, nil
}
```

## Implementation Plan

### Step 1: Add History Preservation (High Priority)

**File**: `cmd/ailang/eval_report.go`

```go
// Add before generating report
func loadExistingDashboard(path string) (*DashboardJSON, error) {
    data, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        // First time - create empty structure
        return &DashboardJSON{History: []HistoryEntry{}}, nil
    }
    if err != nil {
        return nil, err
    }

    var dashboard DashboardJSON
    if err := json.Unmarshal(data, &dashboard); err != nil {
        return nil, fmt.Errorf("invalid existing dashboard JSON: %w", err)
    }

    return &dashboard, nil
}

func appendVersionToHistory(dashboard *DashboardJSON, newEntry HistoryEntry) {
    // Check if version already exists
    for i, entry := range dashboard.History {
        if entry.Version == newEntry.Version {
            // Update existing entry
            dashboard.History[i] = newEntry
            return
        }
    }

    // Prepend (reverse chronological order)
    dashboard.History = append([]HistoryEntry{newEntry}, dashboard.History...)
}
```

**Estimated effort**: 2-3 hours

### Step 2: Add Validation (Medium Priority)

**File**: `internal/eval_analysis/dashboard.go` (new file)

```go
package eval_analysis

type DashboardJSON struct { /* ... */ }
type HistoryEntry struct { /* ... */ }

func (d *DashboardJSON) Validate() error { /* ... */ }
func (h *HistoryEntry) Validate() error { /* ... */ }
```

**Estimated effort**: 1-2 hours

### Step 3: Fix Baseline Metadata (Medium Priority)

**Files**:
- `tools/eval_baseline.sh`
- `Makefile` (eval-baseline target)

**Changes**:
```bash
# Makefile
eval-baseline: build
	@if [ -z "$(VERSION)" ]; then \
		echo "Error: VERSION is required"; \
		echo "Usage: make eval-baseline VERSION=v0.3.9"; \
		exit 1; \
	fi
	@VERSION=$(VERSION) ./tools/eval_baseline.sh
```

**Estimated effort**: 1 hour

### Step 4: Documentation (Low Priority)

Update `CLAUDE.md` with correct workflow:

```markdown
## Updating Benchmark Dashboard

**After creating a baseline:**

```bash
# 1. Create baseline with explicit version
make eval-baseline VERSION=v0.3.9

# 2. Update dashboard (preserves history automatically)
ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=json --output docs/static/benchmarks/latest.json

# 3. Verify history preserved
cat docs/static/benchmarks/latest.json | jq '.history | map(.version)'

# 4. Commit
git add eval_results/baselines/v0.3.9 docs/static/benchmarks/latest.json
git commit -m "Add v0.3.9 benchmark results"
```

**DO NOT**:
- ❌ Redirect output: `ailang eval-report ... > latest.json` (use --output instead)
- ❌ Manually edit JSON files (use the tool)
- ❌ Delete old baselines from git (they're needed for history)
```

**Estimated effort**: 30 minutes

## Testing Strategy

### Unit Tests

```go
func TestHistoryPreservation(t *testing.T) {
    // Setup: Create existing dashboard with 3 versions
    existing := &DashboardJSON{
        Version: "v0.3.8",
        History: []HistoryEntry{
            {Version: "v0.3.8", SuccessRate: 0.65},
            {Version: "v0.3.7", SuccessRate: 0.60},
            {Version: "v0.3.6", SuccessRate: 0.55},
        },
    }

    // Add new version
    newEntry := HistoryEntry{Version: "v0.3.9", SuccessRate: 0.59}
    appendVersionToHistory(existing, newEntry)

    // Verify
    assert.Equal(t, 4, len(existing.History))
    assert.Equal(t, "v0.3.9", existing.History[0].Version)
    assert.Equal(t, "v0.3.8", existing.History[1].Version)
}

func TestDuplicateVersionUpdate(t *testing.T) {
    existing := &DashboardJSON{
        History: []HistoryEntry{
            {Version: "v0.3.8", SuccessRate: 0.65},
        },
    }

    // Update existing version (e.g., rerun benchmark)
    updated := HistoryEntry{Version: "v0.3.8", SuccessRate: 0.68}
    appendVersionToHistory(existing, updated)

    // Should update, not duplicate
    assert.Equal(t, 1, len(existing.History))
    assert.Equal(t, 0.68, existing.History[0].SuccessRate)
}
```

### Integration Tests

```bash
# Test full workflow
test_dashboard_update() {
    # Setup
    cp docs/static/benchmarks/latest.json /tmp/backup.json

    # Run report (should preserve history)
    ailang eval-report eval_results/baselines/v0.3.9 v0.3.9 --format=json --output docs/static/benchmarks/latest.json

    # Verify history count increased
    BEFORE=$(jq '.history | length' /tmp/backup.json)
    AFTER=$(jq '.history | length' docs/static/benchmarks/latest.json)

    if [ "$AFTER" -le "$BEFORE" ]; then
        echo "FAIL: History count did not increase ($BEFORE -> $AFTER)"
        exit 1
    fi

    echo "PASS: History preserved ($BEFORE -> $AFTER versions)"
}
```

## Success Metrics

### Before (Current State)

- ❌ `eval-report --format=json` destroys 3 historical versions
- ❌ Manual Python script required (20 lines)
- ❌ 4 manual steps to update dashboard
- ❌ No validation (JSON can be corrupted)
- ❌ Multiple sources of truth (3 different files)

### After (Target State)

- ✅ `eval-report --format=json` preserves all historical versions
- ✅ Fully automated (1 command)
- ✅ JSON validation prevents corruption
- ✅ Single source of truth (baseline directory)
- ✅ Atomic writes (no partial/corrupted files)

## Migration Path

### For v0.3.9 (Current Release)

**Short-term fix (already done)**:
- Manual Python script to add v0.3.9 to history
- Document in commit message that this was a manual workaround

**Commit message**:
```
Add v0.3.9 to benchmark dashboard (manual workaround)

WARNING: This was added via manual Python script because
`ailang eval-report --format=json` destroys historical data.

See design_docs/planned/eval-dashboard-reliability.md for fix.

Stats:
- Total: 126 runs, 58.7% success
- AILANG: 29/63 (46.0%)
- Python: 45/63 (71.4%)
```

### For v0.3.10 (Next Release)

**Long-term fix (implement design)**:
1. Implement history preservation in `eval-report`
2. Add validation
3. Update Makefile to require explicit VERSION
4. Test with v0.3.10 release

## Open Questions

1. **Should we retroactively fix baseline.json files?**
   - Option A: Leave them as-is (git history shows the issue)
   - Option B: Create script to fix all baseline.json files in git
   - **Recommendation**: Option A (don't rewrite history)

2. **Should we commit all historical baselines to git?**
   - Pro: Full reproducibility, history preservation
   - Con: Large git repo size (~100KB per baseline × N versions)
   - **Recommendation**: Yes, commit baselines (they're the source of truth)

3. **Should dashboard JSON be generated or manually curated?**
   - Current: Mix of both (error-prone)
   - Proposed: Generated from baselines (append-only)
   - **Recommendation**: Generated, but preserve existing history on first migration

## References

- **Incident**: v0.3.9 dashboard update (2025-10-15)
- **Related**: CLAUDE.md "M-EVAL-LOOP" section
- **Files**:
  - `cmd/ailang/eval_report.go`
  - `tools/eval_baseline.sh`
  - `docs/static/benchmarks/latest.json`
  - `internal/eval_analysis/`

## Priority Justification

**High Priority** because:
1. Affects every release (v0.3.10, v0.3.11, ...)
2. Data corruption risk (losing historical benchmarks)
3. Manual workarounds are error-prone and time-consuming
4. Blocks trustworthy performance tracking over time

**Estimated total effort**: 4-6 hours to implement + test
