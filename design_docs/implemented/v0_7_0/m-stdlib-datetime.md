# M-STDLIB-DATETIME: Date and Time Standard Library

**Status:** Planned
**Target:** v0.7.0
**Priority:** P1 (Medium)
**Estimated:** 3-4 days
**Dependencies:** None

## Problem Statement

AILANG's current time support is minimal - `std/clock` provides only `now()` (Unix timestamp in ms) and `sleep()`. This blocks porting date-heavy dashboard code like heatmap grid generation, which requires:

- Date extraction (year, month, day, weekday)
- Date arithmetic (add/subtract days)
- Date formatting (ISO 8601)
- Date comparison and diffing

The heatmap grid generator (`internal/server/handlers_controlplane.go:258-331`) uses all of these operations, making it the primary motivating use case.

## Goals

**Primary Goal:** Enable AILANG to manipulate dates for calendar-based visualizations.

**Success Metrics:**
- Port `buildHeatmapGrid` function to AILANG
- All date operations deterministic (pure functions on timestamps)
- Performance acceptable for dashboard use (<10ms for 365 days of data)
- Clear separation: effectful time source vs pure timestamp math

## Solution Design

### Key Design Decision: Effects Split

**Principle:** Effect `! {Clock}` means "reads or advances ambient time state". Pure timestamp math should NOT require effects.

| Category | Effect | Rationale |
|----------|--------|-----------|
| `now()`, `sleep()` | `! {Clock}` | Reads/advances ambient time |
| `year(ts)`, `addDays(ts, n)`, `formatISODate(ts)` | Pure | Deterministic transform of input |

This keeps effect signatures clean and maintains composability.

### Module Structure

**Two modules with clear separation:**

```
std/clock    - Effectful time source (existing, minimal additions)
std/datetime - Pure timestamp operations (NEW)
```

#### std/clock (Effectful - Existing Module)

```ailang
module std/clock

-- Get current UTC timestamp in milliseconds
-- ALWAYS returns UTC, regardless of host timezone
export func now() -> int ! {Clock} = _clock_now()

-- Sleep/advance time (virtual time aware)
export func sleep(ms: int) -> () ! {Clock} = _clock_sleep(ms)
```

**Implementation note:** `_clock_now` uses `time.Now().UTC().UnixMilli()` to ensure UTC regardless of host configuration.

#### std/datetime (Pure - New Module)

```ailang
module std/datetime

-- ═══════════════════════════════════════════════════════════
-- DATE EXTRACTION (pure, no effects)
-- All functions interpret timestamp as UTC milliseconds
-- ═══════════════════════════════════════════════════════════

export func year(ts: int) -> int = _dt_year(ts)
export func month(ts: int) -> int = _dt_month(ts)      -- 1-12
export func day(ts: int) -> int = _dt_day(ts)          -- 1-31
export func weekday(ts: int) -> int = _dt_weekday(ts)  -- 0=Sun, 6=Sat
export func hour(ts: int) -> int = _dt_hour(ts)        -- 0-23
export func minute(ts: int) -> int = _dt_minute(ts)    -- 0-59
export func second(ts: int) -> int = _dt_second(ts)    -- 0-59

-- ═══════════════════════════════════════════════════════════
-- DATE ARITHMETIC (pure)
-- Uses Go time.AddDate semantics for edge cases
-- e.g., Jan 31 + 1 month = Feb 28/29
-- ═══════════════════════════════════════════════════════════

export func addDays(ts: int, days: int) -> int = _dt_addDays(ts, days)
export func addMonths(ts: int, months: int) -> int = _dt_addMonths(ts, months)
export func addYears(ts: int, years: int) -> int = _dt_addYears(ts, years)

-- Difference in whole days (a - b, truncated)
export func diffDays(a: int, b: int) -> int = _dt_diffDays(a, b)

-- ═══════════════════════════════════════════════════════════
-- DATE BOUNDARIES (pure)
-- ═══════════════════════════════════════════════════════════

export func startOfDay(ts: int) -> int = _dt_startOfDay(ts)
export func startOfWeek(ts: int) -> int = _dt_startOfWeek(ts)  -- Monday
export func startOfMonth(ts: int) -> int = _dt_startOfMonth(ts)

-- ═══════════════════════════════════════════════════════════
-- CONSTRUCTION (pure)
-- ═══════════════════════════════════════════════════════════

export func makeDate(year: int, month: int, day: int) -> int =
  _dt_make(year, month, day, 0, 0, 0)

export func makeDateTime(year: int, month: int, day: int, hour: int, minute: int, second: int) -> int =
  _dt_make(year, month, day, hour, minute, second)

-- ═══════════════════════════════════════════════════════════
-- FORMATTING (pure, fixed formats - no raw Go layouts)
-- All output is UTC, no timezone suffix ambiguity
-- ═══════════════════════════════════════════════════════════

-- ISO 8601 date: "2026-01-15"
export func formatISODate(ts: int) -> string = _dt_formatISODate(ts)

-- RFC 3339: "2026-01-15T14:30:00Z"
export func formatRFC3339(ts: int) -> string = _dt_formatRFC3339(ts)

-- Short month name: "Jan", "Feb", ...
export func formatMonthShort(ts: int) -> string = _dt_formatMonthShort(ts)

-- Full weekday name: "Monday", "Tuesday", ...
export func formatWeekdayFull(ts: int) -> string = _dt_formatWeekdayFull(ts)

-- ═══════════════════════════════════════════════════════════
-- PARSING (pure, returns Option - no panics)
-- Interprets input as UTC
-- ═══════════════════════════════════════════════════════════

-- Parse ISO date: "2026-01-15" -> Some(ts) or None
export func parseISODate(s: string) -> Option[int] = _dt_parseISODate(s)

-- Parse RFC 3339: "2026-01-15T14:30:00Z" -> Some(ts) or None
export func parseRFC3339(s: string) -> Option[int] = _dt_parseRFC3339(s)
```

### Builtin Implementation (Reduced Surface)

**Only 6 core builtins needed** (vs 11+ in original design):

| Builtin | Type | Go Implementation |
|---------|------|-------------------|
| `_dt_parts` | `int -> {year:int, month:int, day:int, weekday:int, hour:int, minute:int, second:int}` | Extract all parts in one call |
| `_dt_make` | `(int, int, int, int, int, int) -> int` | `time.Date(y,m,d,h,mi,s,0,time.UTC).UnixMilli()` |
| `_dt_add` | `(int, int, int, int) -> int` | `t.AddDate(years, months, days).UnixMilli()` |
| `_dt_diffDays` | `(int, int) -> int` | `int((a-b) / (24*60*60*1000))` |
| `_dt_formatISODate` | `int -> string` | `t.Format("2006-01-02")` |
| `_dt_parseISODate` | `string -> Option[int]` | Returns `Some(ts)` or `None` |

**Wrapper functions** in `std/datetime.ail` provide the ergonomic API:

```ailang
-- Wrappers use _dt_parts for extraction
export func year(ts: int) -> int =
  let parts = _dt_parts(ts);
  parts.year

export func addDays(ts: int, days: int) -> int =
  _dt_add(ts, 0, 0, days)

export func startOfDay(ts: int) -> int =
  let p = _dt_parts(ts);
  _dt_make(p.year, p.month, p.day, 0, 0, 0)

export func startOfWeek(ts: int) -> int =
  let wd = weekday(ts);
  -- Monday = 1, so: Sun(0)->6, Mon(1)->0, Tue(2)->1, ...
  let daysBack = if wd == 0 then 6 else wd - 1;
  startOfDay(addDays(ts, -daysBack))
```

### Timezone Handling

**Hard requirement: UTC everywhere, mechanically enforced.**

| Operation | Timezone |
|-----------|----------|
| `now()` | Returns `time.Now().UTC().UnixMilli()` |
| `formatISODate(ts)` | Formats as UTC (no suffix needed for date-only) |
| `formatRFC3339(ts)` | Formats with "Z" suffix (UTC) |
| `parseISODate(s)` | Interprets as UTC |
| `parseRFC3339(s)` | Accepts "Z" or explicit offset, converts to UTC |

**No local time leakage:** Tests verify functions are invariant under `TZ` environment variable changes.

**Future (v0.8.0+):** If needed, add explicit timezone parameter:
```ailang
export func formatInTimezone(ts: int, tz: string) -> string  -- "America/New_York"
```

### Format Ergonomics

**Why no raw Go layouts:**
- Go's reference time (`Mon Jan 2 15:04:05 MST 2006`) is non-obvious
- AI agents will guess strftime (`%Y-%m-%d`) or ISO tokens
- Fixed formats eliminate prompt burden and bugs

**Provided formats cover dashboard needs:**
| Function | Output Example | Use Case |
|----------|----------------|----------|
| `formatISODate` | `"2026-01-15"` | Heatmap cell keys |
| `formatRFC3339` | `"2026-01-15T14:30:00Z"` | API timestamps |
| `formatMonthShort` | `"Jan"` | Month labels |
| `formatWeekdayFull` | `"Monday"` | Week alignment |

**Future (v0.7.1+):** Add format enum if more patterns needed:
```ailang
type DateFormat = ISO_DATE | ISO_DATETIME | RFC3339 | MONTH_SHORT | WEEKDAY_FULL
export func format(ts: int, fmt: DateFormat) -> string
```

### Files to Create/Modify

| File | Action | LOC Est. |
|------|--------|----------|
| `std/datetime.ail` | **Create** - pure datetime functions | ~80 |
| `internal/builtins/spec.go` | Modify - register 6 builtins | +60 |
| `internal/builtins/datetime.go` | **Create** - builtin implementations | +120 |
| `internal/builtins/datetime_test.go` | **Create** - unit tests | +150 |
| `std/clock.ail` | Modify - add UTC comment | +5 |
| `examples/datetime_demo.ail` | **Create** - example program | +40 |

**Total:** ~455 LOC

## Example Usage

### Heatmap Grid (Target Use Case)

```ailang
module dashboard/heatmap

import std/clock (now)
import std/datetime (addDays, weekday, formatISODate, diffDays, startOfWeek)
import std/list (map, foldl)

type HeatmapCell = {
  date: string,
  taskCount: int,
  cost: float,
  intensity: float
}

-- Build week-by-week grid for last N days
export func buildGrid(
  cells: [{date: string, taskCount: int, cost: float, ...}],
  days: int
) -> [[HeatmapCell]] ! {Clock} {
  let endTs = now();  -- Only effectful call
  let startTs = addDays(endTs, -days);  -- Pure
  let alignedStart = startOfWeek(startTs);  -- Pure

  -- Find max count for intensity calculation
  let maxCount = foldl(\acc c. if c.taskCount > acc then c.taskCount else acc, 0, cells);

  -- Build cell lookup map
  let cellMap = buildCellMap(cells);

  -- Generate weeks (pure recursion)
  generateWeeks(alignedStart, endTs, cellMap, maxCount)
}

-- Pure function: no effects needed
func generateWeeks(
  start: int,
  end: int,
  cellMap: {string: HeatmapCell},
  maxCount: int
) -> [[HeatmapCell]] =
  if start > end then []
  else {
    let week = generateWeek(start, cellMap, maxCount);
    let nextWeek = addDays(start, 7);
    [week] ++ generateWeeks(nextWeek, end, cellMap, maxCount)
  }
```

Note how only `buildGrid` has `! {Clock}` - all helper functions are pure.

## Testing Strategy

1. **Unit tests** for each builtin (datetime_test.go)
   - Edge cases: leap years, month boundaries, year boundaries
   - Feb 29 handling in leap vs non-leap years

2. **Timezone invariance test**
   ```go
   // Run same test with TZ=UTC, TZ=America/New_York, TZ=Asia/Tokyo
   // Results must be identical
   ```

3. **Virtual time integration test**
   ```ailang
   -- With --virtual-time=1704067200000 (2024-01-01T00:00:00Z)
   let ts = now();
   assert(formatISODate(ts) == "2024-01-01")
   ```

4. **Go parity test** - Compare AILANG vs Go output for same operations

## Open Questions

1. **Do we need ISO week numbers?**
   - `isoWeek(ts) -> {year: int, week: int}` for ISO 8601 week numbering
   - Heatmaps only need Monday alignment, not ISO weeks
   - **Decision:** Defer to v0.7.1 unless needed

2. **Result vs Option for parse errors?**
   - `Option[int]` is simpler, sufficient for "valid or not"
   - `Result[int, DateParseError]` provides diagnostics
   - **Decision:** `Option[int]` for v0.7.0, can add Result variant later

3. **Dashboard timestamps always UTC?**
   - Yes - coordinator stores UTC, API serves UTC
   - **Decision:** Hard-pin UTC, no timezone support in v0.7.0

## Success Criteria

- [ ] 6 core builtins implemented and tested
- [ ] std/datetime.ail created with wrapper functions
- [ ] All functions pure except now()/sleep()
- [ ] Timezone invariance tests pass
- [ ] Virtual time mode works correctly
- [ ] Example datetime_demo.ail runs successfully
- [ ] Heatmap can be ported to AILANG

## Alternatives Considered

### Alternative A: Everything in std/clock with Effects
**Rejected:** Pollutes effect signatures, weakens meaning of `Clock` capability.

### Alternative B: Raw Go Layout Strings
**Rejected:** Footgun for AI agents who will guess strftime patterns. Fixed formats are safer.

### Alternative C: Full Timezone Support
**Rejected for v0.7.0:** Adds complexity, leaks host configuration. UTC-only is deterministic and sufficient for dashboard.

## Related Documents

- [m-dashboard-dogfooding.md](../../implemented/v0_6_6/m-dashboard-dogfooding.md) - Motivating use case
- [std/clock.ail](../../../std/clock.ail) - Current time module
- [handlers_controlplane.go](../../../internal/server/handlers_controlplane.go) - Heatmap implementation to port
