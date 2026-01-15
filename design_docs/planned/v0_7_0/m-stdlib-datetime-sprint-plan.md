# Sprint Plan: M-STDLIB-DATETIME

## Summary
Add date/time operations to AILANG's standard library with pure timestamp functions in `std/datetime` module, enabling calendar-based dashboard features like heatmap grid generation.

**Duration:** 3 days
**Dependencies:** None
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- ✅ Dashboard dogfooding (embed API, AILANG bridge): ~400 LOC in 2 days
- ✅ M-GAP2 multi-param lambda fix: ~100 LOC in 1 day
- ✅ Stdlib gaps (repeat, head, tail): ~150 LOC in 1 day
- ✅ Control plane UI: ~500 LOC in 3 days

### Velocity
- Recent average: ~150-200 LOC/day (including tests)
- Estimated capacity: 450-600 LOC for 3-day sprint
- Sprint target: 455 LOC (fits comfortably)

### Remaining from Design Doc
- ⏳ Core builtins (6 functions): ~180 LOC
- ⏳ std/datetime.ail wrappers: ~80 LOC
- ⏳ Tests: ~150 LOC
- ⏳ Example file: ~40 LOC

## Proposed Milestones

### Milestone 1: Core Builtins
**Goal:** Implement the 6 core datetime builtins in Go
**Estimated:** 120 LOC implementation + 60 LOC tests = 180 LOC
**Duration:** 1 day

**Tasks:**
- Implement `_dt_parts(ts) -> record` - extract all date components at once
- Implement `_dt_make(y,m,d,h,mi,s) -> int` - construct timestamp from components
- Implement `_dt_add(ts, years, months, days) -> int` - date arithmetic via Go AddDate
- Implement `_dt_diffDays(a, b) -> int` - difference in whole days
- Implement `_dt_formatISODate(ts) -> string` - ISO 8601 date format
- Implement `_dt_parseISODate(s) -> Option[int]` - parse ISO date string
- Register all builtins in spec.go

**Files to modify/create:**
- `internal/builtins/datetime.go` (NEW) - builtin implementations
- `internal/builtins/datetime_test.go` (NEW) - unit tests
- `internal/builtins/spec.go` - register builtins

**Acceptance Criteria:**
- [ ] All 6 builtins registered and callable
- [ ] `_dt_parts` returns record with year/month/day/weekday/hour/minute/second
- [ ] `_dt_make` creates correct UTC timestamp
- [ ] `_dt_add` handles month boundaries correctly (Jan 31 + 1 month = Feb 28)
- [ ] `_dt_parseISODate` returns None for invalid input (no panics)
- [ ] All Go tests passing
- [ ] Linting clean

**Risks:**
- None significant - straightforward Go time package wrapping

### Milestone 2: std/datetime Module
**Goal:** Create the AILANG datetime module with wrapper functions
**Estimated:** 80 LOC implementation + 50 LOC tests = 130 LOC
**Duration:** 1 day

**Tasks:**
- Create `std/datetime.ail` with all exported functions
- Implement extraction wrappers (year, month, day, weekday, hour, minute, second)
- Implement arithmetic wrappers (addDays, addMonths, addYears, diffDays)
- Implement boundary functions (startOfDay, startOfWeek, startOfMonth)
- Implement construction functions (makeDate, makeDateTime)
- Implement format functions (formatISODate, formatRFC3339, formatMonthShort, formatWeekdayFull)
- Implement parse functions (parseISODate, parseRFC3339)
- Add module to stdlib path

**Files to create:**
- `std/datetime.ail` (NEW) - datetime module

**Acceptance Criteria:**
- [ ] `import std/datetime` works
- [ ] All exported functions callable
- [ ] `startOfWeek` aligns to Monday correctly
- [ ] Format functions produce expected output
- [ ] Parse functions return Option type correctly
- [ ] No effects required (all pure functions)

**Risks:**
- Row polymorphism for `_dt_parts` return record - verify unification works

### Milestone 3: Testing & Documentation
**Goal:** Comprehensive tests, example file, and documentation
**Estimated:** 100 LOC tests + 45 LOC example = 145 LOC
**Duration:** 1 day

**Tasks:**
- Add integration tests (AILANG calling datetime functions)
- Add timezone invariance tests (verify UTC-only)
- Add edge case tests (leap years, month boundaries, year boundaries)
- Create `examples/datetime_demo.ail` demonstrating all functions
- Update std/clock.ail with UTC documentation note
- Verify virtual time mode compatibility

**Files to create/modify:**
- `internal/builtins/datetime_test.go` - add integration tests
- `examples/datetime_demo.ail` (NEW) - example program
- `std/clock.ail` - add UTC documentation

**Acceptance Criteria:**
- [ ] Timezone invariance: same results with TZ=UTC, TZ=America/New_York
- [ ] Leap year handling: Feb 29 works in 2024, fails gracefully in 2025
- [ ] Month boundaries: Jan 31 + 1 month = Feb 28/29
- [ ] Example file runs successfully with `ailang run`
- [ ] All tests passing
- [ ] `make lint` clean

**Risks:**
- Virtual time interaction may need debugging - verify `now()` uses virtual clock

## Success Metrics
- Test coverage: >80% for datetime.go
- Examples passing: datetime_demo.ail runs successfully
- Documentation: std/clock.ail updated with UTC note
- All tests passing: ✅
- All linting passing: ✅
- Design doc acceptance criteria met

## Dependencies
- None - this is a standalone stdlib addition

## Open Questions
(Resolved in design doc)
1. ~~Result vs Option for parse errors~~ → Option[int] for v0.7.0
2. ~~ISO week numbers~~ → Deferred to v0.7.1
3. ~~Timezone support~~ → UTC-only for v0.7.0

## Notes
- All datetime functions are pure (no `! {Clock}` effect)
- Only `now()` and `sleep()` in std/clock have Clock effect
- Go's `time.AddDate` semantics for edge cases (documented)
- UTC enforced via `time.Now().UTC()` in clock handler
