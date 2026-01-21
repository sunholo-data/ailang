# Retrospective: Observability Dashboard Development

**Sprint**: M-OTEL + M-TASK-HIERARCHY (v0.6.2 - v0.6.5)
**Duration**: ~7-8 weeks (Nov 15, 2025 - Jan 8, 2026)
**Total LOC**: ~14,500 (Go: 12,670, TypeScript: ~23,761)
**Outcome**: Feature complete, but felt broken due to doc/binary sync issues

## Executive Summary

The observability dashboard implementation was technically successful but felt like a struggle due to **documentation debt** and **build/runtime synchronization issues**. The code was complete weeks before we realized it, because:

1. Design docs said "NOT DONE" when features were implemented
2. Running server binary was often older than source code
3. No automated verification that docs match implementation

## Timeline

| Week | Work Done | Perception |
|------|-----------|------------|
| 1-2 | OTEL instrumentation, provider support | "Making progress" |
| 3 | Observatory backend (7,456 LOC) | "Almost there" |
| 4 | React dashboard UI | "Just need to wire it up" |
| 5-6 | Task hierarchy linking | "Why doesn't it work?" |
| 7-8 | Debugging... actually complete | "Oh, it was working" |

## What Went Wrong

### 1. Documentation Debt (Primary Issue)

**Problem**: Sprint plans showed checkboxes like `[ ] M2: Context Propagation ❌ NOT STARTED` when the code existed in `internal/executor/environment.go`.

**Impact**:
- Wasted time re-investigating "missing" features
- Created perception of incomplete work
- Made progress invisible

**Evidence**:
```
Sprint plan (2026-01-05): "M2: Context Propagation ❌ NOT STARTED"
Actual code: internal/executor/environment.go - 193 LOC, fully tested
```

### 2. Binary/Server Sync (Secondary Issue)

**Problem**: After `make install`, the server process wasn't restarted, so it ran old code.

**Impact**:
- APIs returned 404 even though code existed
- Version endpoint returned "dev" instead of actual version
- Created illusion of broken features

**Evidence**:
```bash
# Server running old binary
curl http://localhost:1957/api/version
# {"version":"dev"}

# After restart
curl http://localhost:1957/api/observatory/traces?limit=3
# [{"trace_id":"ae48fb3c",...}]  # Works!
```

### 3. Scope Creep (Tertiary Issue)

**Problem**: Built cloud backend stubs (GCP, Jaeger, composite) before they were needed.

**Impact**:
- ~750 LOC of stub code that's mostly unimplemented
- Added complexity without immediate value
- Distracted from core functionality

**Evidence**:
```
backend_gcp.go: 210 LOC (mostly TODOs)
backend_jaeger.go: 210 LOC (mostly TODOs)
backend_composite.go: 300 LOC (working but unused)
```

## What Went Well

### 1. Implementation Quality

The actual code is robust:
- 6-level fallback chain for task ID extraction (exceeded 3-level design)
- Timestamp correlation for cross-process span linking (bonus feature)
- 36+ tests passing
- Full CRUD API with WebSocket streaming

### 2. Architecture

The separation of concerns is clean:
- `observatory/` - Backend storage and API
- `coordinator/` - Task management with observatory sync
- `executor/` - Context propagation to AI providers
- `ui/` - React components with hooks

### 3. Test Coverage

Critical paths are well tested:
- Hierarchy API tests
- Aggregation update tests
- OTLP receiver tests
- Context propagation tests

## Root Cause Analysis

### Why Did Docs Drift from Implementation?

1. **No verification step**: Marking a checkbox "done" didn't require proof
2. **Async development**: Implementation happened in bursts, docs updated later
3. **Multiple files**: 7 milestones across many files - easy to lose track
4. **No CI check**: Nothing verified docs matched code

### Why Was Server Restart Overlooked?

1. **Development habit**: `make build` runs, assume it works
2. **Long-running server**: Started days ago, forgot it was running
3. **No version check**: Server didn't warn "you're running v0.6.2, code is v0.6.5"

## Recommendations for Future Projects

### 1. Automated Doc/Code Verification

Add a CI check that scans design docs for implementation claims and verifies code exists:

```bash
# Example: verify_design_docs.sh
grep -r "Status.*DONE" design_docs/planned/ | while read line; do
  # Extract file path claims and verify they exist
  # Fail CI if docs claim completion but code missing
done
```

### 2. Server Startup Version Check

Add version validation on server start:

```go
func (s *Server) Start() error {
    binaryVersion := version.Get()
    log.Printf("Starting server v%s", binaryVersion)

    if s.lastKnownVersion != "" && s.lastKnownVersion != binaryVersion {
        log.Printf("WARNING: Version changed from %s to %s - restart required",
            s.lastKnownVersion, binaryVersion)
    }
    // ...
}
```

### 3. Make Target for Full Rebuild/Restart

```makefile
# Add to Makefile
dev-restart: quick-install ui-deploy
	@pkill -f "ailang serve" 2>/dev/null || true
	@sleep 1
	@ailang serve &
	@echo "Server restarted with latest code"
```

### 4. Smaller Scope, Verify Earlier

Instead of building 7 milestones then verifying, do:
1. Build M1 → Verify with real data → Update docs
2. Build M2 → Verify with real data → Update docs
3. etc.

### 5. Design Doc Status as Code

Use a machine-readable format that can be validated:

```yaml
# design_docs/status.yaml
m-task-hierarchy:
  m1_entity_sync:
    status: done
    implementation: internal/coordinator/observatory_sync.go
    tests: internal/coordinator/observatory_sync_test.go
    verified: 2026-01-08
```

## Metrics

| Metric | Value |
|--------|-------|
| Total commits (telemetry-focused) | 43 |
| Lines of Go code | 12,670 |
| Lines of TypeScript | 23,761 |
| Files touched | 118 |
| Databases | 3 |
| API endpoints | 30+ |
| Tests | 36+ passing |
| Time feeling "stuck" | ~2 weeks |
| Actual time code was complete | ~4 weeks |

## Conclusion

The observability dashboard was a success technically but a struggle experientially. The key lesson is:

> **Documentation is code.** If docs say "not done," the feature doesn't exist - regardless of what's in the codebase.

Future projects should treat design doc status updates with the same rigor as code commits: verified, tested, and reviewed.

---

**Written**: 2026-01-08
**Authors**: Claude Code + Human collaboration
