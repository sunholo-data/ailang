# Sprint Plan: M-DEPRECATE-AILANG-AGENT

## Summary

Deprecate and remove the standalone `ailang-agent` binary and `internal/agent/` package, migrating the capability detection feature to the coordinator before deletion.

**Duration:** 0.5 days (~4 hours)
**Dependencies:** None (v0.6.2 coordinator already complete)
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- M-OTEL-ENHANCED-TRACING-DX: ~635 LOC in 2 days
- M-MSG-ROUTING: ~300 LOC in 1 day
- stdlib additions (sortBy, take, drop, contains): ~100 LOC

### Velocity
- Recent average: ~200-300 LOC/day for new features
- Deletion work is faster than new code
- Estimated capacity: This sprint mostly involves deletion and migration

### Remaining from Design Doc
- M1: Migrate capability detection (~350 LOC new)
- M2: Delete deprecated code (~1,760 LOC removed)
- M3: Update documentation (~50 LOC changed)

## Proposed Milestones

### Milestone 1: Migrate Capability Detection
**Goal:** Move capability detection from `internal/agent/` to `internal/coordinator/` before deletion
**Estimated:** 200 LOC implementation + 150 LOC tests = 350 LOC
**Duration:** 1.5 hours

**Tasks:**
1. Create `internal/coordinator/capability_detector.go`
   - Adapt `DetectCapabilities()` from `internal/agent/capabilities.go`
   - Add `Capability` type with FS, Net, Shell, Budget variants
   - Add `ClassifyImpact()` returning low/medium/high
   - Add `EstimateCost()` for pre-execution estimates

2. Integrate with `TaskAnalyzer`
   - Add fields to `AnalyzedTask`: `Capabilities`, `ImpactLevel`, `EstimatedCost`
   - Call `CapabilityDetector` from `Analyze()` method

3. Write tests `internal/coordinator/capability_detector_test.go`
   - Test FS keyword detection
   - Test Shell keyword detection (high risk)
   - Test Network keyword detection
   - Test impact classification
   - Test cost estimation

**Acceptance Criteria:**
- [ ] `capability_detector.go` created with all detection functions
- [ ] `AnalyzedTask` has capability fields populated
- [ ] Tests cover FS, Net, Shell, Budget detection
- [ ] `go test ./internal/coordinator/...` passes

**Risks:**
- Detection logic differences - Mitigation: Direct port from working code

### Milestone 2: Delete Deprecated Code
**Goal:** Remove `cmd/ailang-agent/` and `internal/agent/` directories
**Estimated:** 0 LOC new (~1,760 LOC removed)
**Duration:** 30 minutes

**Tasks:**
1. Delete directories:
   ```bash
   rm -rf cmd/ailang-agent/
   rm -rf internal/agent/
   rm cmd/ailang/agent.go.bak
   ```

2. Update Makefile:
   - Remove `build-agent` target (lines 43-48)
   - Remove `install-agent` target (lines 50-54)
   - Update `build-all` target
   - Update `help` text (lines 568-571)

3. Verify no broken imports:
   ```bash
   go build ./...
   go test ./...
   ```

**Acceptance Criteria:**
- [ ] `cmd/ailang-agent/` deleted
- [ ] `internal/agent/` deleted
- [ ] `cmd/ailang/agent.go.bak` deleted
- [ ] Makefile targets removed
- [ ] `go build ./...` succeeds
- [ ] `go test ./...` passes

**Risks:**
- Hidden dependencies - Mitigation: Already verified only ailang-agent uses internal/agent

### Milestone 3: Update Documentation
**Goal:** Update all references to `ailang-agent` in documentation
**Estimated:** 50 LOC changed
**Duration:** 30 minutes

**Tasks:**
1. Update `docs/docs/guides/collaboration-hub.md`:
   - Remove `make build-agent` / `make install-agent` examples
   - Remove `ailang-agent --version` verification
   - Remove `ailang-agent` CLI usage section
   - Replace with `ailang coordinator` equivalents

2. Update `CHANGELOG.md`:
   - Add v0.6.3 section
   - Document deprecation and removal
   - Reference design doc

3. Verify docs build:
   ```bash
   cd docs && npm run build
   ```

**Acceptance Criteria:**
- [ ] `collaboration-hub.md` updated with coordinator commands
- [ ] `CHANGELOG.md` has v0.6.3 deprecation notice
- [ ] `cd docs && npm run build` succeeds
- [ ] No references to `ailang-agent` in docs (except historical CHANGELOG)

**Risks:**
- Missed references - Mitigation: grep for ailang-agent after updates

### Milestone 4: Final Verification
**Goal:** Complete verification that nothing is broken
**Estimated:** 0 LOC
**Duration:** 30 minutes

**Tasks:**
1. Run full test suite:
   ```bash
   make test
   ```

2. Run coordinator-specific tests:
   ```bash
   go test ./internal/coordinator/... -v
   ```

3. Verify no stray references:
   ```bash
   grep -r "ailang-agent" --include="*.go" .
   grep -r "internal/agent" --include="*.go" .
   ```

4. Verify build and install:
   ```bash
   make build
   make install
   ```

**Acceptance Criteria:**
- [ ] All tests pass
- [ ] No references to deleted code
- [ ] Build succeeds
- [ ] Install succeeds

## Success Metrics
- Test coverage: Maintained (no regression)
- Coordinator tests: All passing
- Documentation: collaboration-hub.md updated
- All tests passing: Required
- All linting passing: Required

## Dependencies
- None (v0.6.2 coordinator work already complete)

## Open Questions
- None - straightforward deprecation

## Notes
- Code is preserved in git history for rollback if needed
- Capability detection migration is the only "new" code
- Most work is deletion and documentation updates
- Total time estimate: ~4 hours
