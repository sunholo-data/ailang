# M-R6: Clock & Net Effects

**Status**: 📋 Planned
**Priority**: P1 (STRETCH GOAL - Clock recommended, Net optional)
**Estimated**: 700 LOC total (250 Clock + 450 Net)
**Duration**: 3 days (if time permits)
**Dependencies**: M-R2 (Effect system runtime from v0.2.0)
**Blocks**: Real-world programs (timing, HTTP APIs, web scraping)

## Problem Statement

**Current State**: Only IO and FS effects available. No time operations or network access.

```ailang
-- ❌ BROKEN in v0.2.0
-- No way to get current time
-- No way to sleep/delay
-- No way to fetch HTTP resources
```

**Business Impact**:
- Can't build web scrapers
- Can't implement rate limiting
- Can't integrate with REST APIs
- Can't build time-based logic (cron jobs, TTLs)

## Goals

### Primary Goals (Must Achieve - Clock Only)
1. **Clock effect implemented**: `now()`, `sleep()` operations
2. **Capability enforcement**: Requires `--caps Clock`
3. **Examples work**: Time-based examples pass
4. **Stdlib integration**: `std/clock` module available

### Secondary Goals (Stretch - Net Effect)
5. **Net effect implemented**: `httpGet()`, `httpPost()` operations
6. **Security sandbox**: Block localhost, private IPs, file:// by default
7. **Timeout enforcement**: Global 30s timeout, configurable
8. **Domain allowlist**: `--net-allow` flag for explicit domains

**Decision Point**: If timeline is tight, **ship Clock only** and defer Net to v0.3.1.

## Design

### Part 1: Clock Effect (MUST HAVE)

**API Design**:
```ailang
import std/clock (now, sleep)

-- Get current Unix timestamp (milliseconds)
now() -> int ! {Clock}

-- Sleep for specified milliseconds
sleep(ms: int) -> () ! {Clock}
```

**Use Cases**:
```ailang
-- Measure execution time
let start = now();
let result = expensive_computation();
let elapsed = now() - start;
println("Took " ++ show(elapsed) ++ "ms")

-- Rate limiting
sleep(1000);  -- Wait 1 second
fetchNextBatch()

-- Timeout implementation
let deadline = now() + 5000;
loop_until(\. now() < deadline, fetch)
```

**Implementation**:
```go
// internal/effects/clock.go
func clockNow(ctx *EffContext, args []Value) (Value, error) {
    if !ctx.HasCap("Clock") {
        return nil, NewCapabilityError("Clock", "now()")
    }

    // Use deterministic time if AILANG_SEED set
    if seed := os.Getenv("AILANG_SEED"); seed != "" {
        // Virtual time based on seed (for testing)
        return IntValue(ctx.virtualTime), nil
    }

    // Real wall-clock time
    return IntValue(time.Now().UnixMilli()), nil
}

func clockSleep(ctx *EffContext, args []Value) (Value, error) {
    if !ctx.HasCap("Clock") {
        return nil, NewCapabilityError("Clock", "sleep()")
    }

    ms, ok := args[0].(IntValue)
    if !ok {
        return nil, fmt.Errorf("sleep: expected int, got %T", args[0])
    }

    if ms < 0 {
        return nil, fmt.Errorf("sleep: negative duration %d", ms)
    }

    // Use virtual sleep if in deterministic mode
    if seed := os.Getenv("AILANG_SEED"); seed != "" {
        ctx.virtualTime += int64(ms)
        return UnitValue{}, nil
    }

    // Real sleep
    time.Sleep(time.Duration(ms) * time.Millisecond)
    return UnitValue{}, nil
}
```

**Deterministic Mode** (for testing/AI training):
- `AILANG_SEED=42` enables virtual time
- `now()` returns virtual timestamp
- `sleep()` advances virtual time (no actual sleep)
- Reproducible execution for benchmarks

### Part 2: Net Effect (STRETCH GOAL)

**API Design**:
```ailang
import std/net (httpGet, httpPost)

-- HTTP GET request
httpGet(url: string) -> string ! {Net}

-- HTTP POST request
httpPost(url: string, body: string) -> string ! {Net}
```

**Security Model**:

1. **Deny by Default**:
   - Requires `--caps Net` to enable
   - All requests blocked without capability

2. **Protocol Restrictions**:
   - Allow: `http://`, `https://`
   - Block: `file://`, `ftp://`, `data://`

3. **Localhost Protection**:
   - Block: `localhost`, `127.0.0.1`, `::1`
   - Block: Private IPs (10.x, 192.168.x, 172.16-31.x)
   - Configurable via `--net-allow-localhost`

4. **Timeout Enforcement**:
   - Default: 30 seconds
   - Configurable: `--net-timeout=60s`
   - Per-request override: `httpGet(url, timeout=10000)` (future)

5. **Domain Allowlist** (optional):
   - `--net-allow=api.example.com,data.source.org`
   - If set, only listed domains allowed
   - Wildcard support: `*.example.com`

**Implementation**:
```go
// internal/effects/net.go
func netHttpGet(ctx *EffContext, args []Value) (Value, error) {
    if !ctx.HasCap("Net") {
        return nil, NewCapabilityError("Net", "httpGet()")
    }

    url, ok := args[0].(StringValue)
    if !ok {
        return nil, fmt.Errorf("httpGet: expected string, got %T", args[0])
    }

    // Security checks
    if err := validateURL(string(url)); err != nil {
        return nil, fmt.Errorf("httpGet: %w", err)
    }

    if !isAllowedDomain(string(url), ctx.allowedDomains) {
        return nil, fmt.Errorf("httpGet: domain not in allowlist: %s", url)
    }

    // Make request with timeout
    client := &http.Client{
        Timeout: ctx.netTimeout,
    }

    resp, err := client.Get(string(url))
    if err != nil {
        return nil, fmt.Errorf("httpGet: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("httpGet: read body: %w", err)
    }

    return StringValue(body), nil
}

func validateURL(urlStr string) error {
    u, err := url.Parse(urlStr)
    if err != nil {
        return fmt.Errorf("invalid URL: %w", err)
    }

    // Check protocol
    switch u.Scheme {
    case "http", "https":
        // OK
    default:
        return fmt.Errorf("unsupported protocol: %s (only http/https allowed)", u.Scheme)
    }

    // Check localhost
    if isLocalhost(u.Host) {
        return fmt.Errorf("localhost access blocked (use --net-allow-localhost to enable)")
    }

    // Check private IPs
    if isPrivateIP(u.Host) {
        return fmt.Errorf("private IP access blocked: %s", u.Host)
    }

    return nil
}
```

**Error Messages**:
```
Net Error: Domain not in allowlist
  URL: https://untrusted.example.com
  Allowed: api.example.com, data.source.org

  Hint: Add domain with --net-allow=untrusted.example.com

Net Error: Localhost access blocked
  URL: http://localhost:8080

  Localhost is blocked by default for security.
  To enable: --net-allow-localhost

Net Error: Request timeout
  URL: https://slow.example.com/api
  Timeout: 30s

  The request exceeded the 30 second timeout.
  Hint: Increase with --net-timeout=60s
```

## Implementation Plan

### Day 1: Clock Effect (~250 LOC)

**Files to Create/Modify**:
- `internal/effects/clock.go` (~150 LOC new file)
- `stdlib/std/clock.ail` (~20 LOC new file)
- `internal/effects/clock_test.go` (~50 LOC new file)
- `internal/effects/context.go` (~30 LOC - add virtual time)

**Tasks**:
1. Implement `clockNow()` and `clockSleep()` builtins
2. Add virtual time support for deterministic mode
3. Create `std/clock` stdlib module
4. Unit tests: real time, virtual time, edge cases
5. Example: `examples/micro_clock_measure.ail`

**Test Cases**:
```go
// internal/effects/clock_test.go
func TestClockNow(t *testing.T) {
    ctx := NewEffContext()
    ctx.Grant("Clock")

    // Real time mode
    before := time.Now().UnixMilli()
    result, err := clockNow(ctx, []Value{})
    after := time.Now().UnixMilli()

    assert.NoError(t, err)
    assert.GreaterOrEqual(t, result.(IntValue), IntValue(before))
    assert.LessOrEqual(t, result.(IntValue), IntValue(after))
}

func TestClockSleep(t *testing.T) {
    ctx := NewEffContext()
    ctx.Grant("Clock")

    start := time.Now()
    _, err := clockSleep(ctx, []Value{IntValue(100)})
    elapsed := time.Since(start).Milliseconds()

    assert.NoError(t, err)
    assert.GreaterOrEqual(t, elapsed, int64(100))
    assert.Less(t, elapsed, int64(150))  // Allow 50ms variance
}

func TestClockVirtualTime(t *testing.T) {
    os.Setenv("AILANG_SEED", "42")
    defer os.Unsetenv("AILANG_SEED")

    ctx := NewEffContext()
    ctx.Grant("Clock")
    ctx.virtualTime = 1000

    // Virtual now
    result, _ := clockNow(ctx, []Value{})
    assert.Equal(t, IntValue(1000), result)

    // Virtual sleep (no actual delay)
    clockSleep(ctx, []Value{IntValue(500)})
    result, _ = clockNow(ctx, []Value{})
    assert.Equal(t, IntValue(1500), result)
}
```

### Day 2: Net Effect (~300 LOC)

**Files to Create/Modify**:
- `internal/effects/net.go` (~200 LOC new file)
- `internal/effects/net_security.go` (~80 LOC new file)
- `stdlib/std/net.ail` (~20 LOC new file)

**Tasks**:
1. Implement `netHttpGet()` and `netHttpPost()` builtins
2. Implement URL validation and security checks
3. Add localhost and private IP blocking
4. Add timeout enforcement
5. Create `std/net` stdlib module

**Security Functions**:
```go
// internal/effects/net_security.go
func isLocalhost(host string) bool {
    h, _, _ := net.SplitHostPort(host)
    if h == "" { h = host }

    return h == "localhost" ||
           h == "127.0.0.1" ||
           h == "::1" ||
           h == "0.0.0.0"
}

func isPrivateIP(host string) bool {
    h, _, _ := net.SplitHostPort(host)
    if h == "" { h = host }

    ip := net.ParseIP(h)
    if ip == nil { return false }

    return ip.IsLoopback() ||
           ip.IsPrivate() ||
           ip.IsLinkLocalUnicast()
}

func isAllowedDomain(urlStr string, allowed []string) bool {
    if len(allowed) == 0 {
        return true  // No allowlist = all domains OK
    }

    u, _ := url.Parse(urlStr)
    hostname := u.Hostname()

    for _, pattern := range allowed {
        if matchDomain(hostname, pattern) {
            return true
        }
    }
    return false
}

func matchDomain(hostname, pattern string) bool {
    // Exact match
    if hostname == pattern {
        return true
    }

    // Wildcard match: *.example.com
    if strings.HasPrefix(pattern, "*.") {
        suffix := pattern[1:] // Remove *
        return strings.HasSuffix(hostname, suffix)
    }

    return false
}
```

### Day 3: CLI Integration & Testing (~150 LOC)

**Files to Modify**:
- `cmd/ailang/main.go` (~30 LOC)
- `internal/effects/context.go` (~20 LOC)
- `internal/effects/net_test.go` (~50 LOC new file)
- `examples/micro_net_fetch.ail` (~20 LOC new file)
- `examples/micro_clock_rate_limit.ail` (~30 LOC new file)

**CLI Flags**:
```go
// cmd/ailang/main.go
var (
    netAllowList      = flag.String("net-allow", "", "Allowed domains (comma-separated)")
    netAllowLocalhost = flag.Bool("net-allow-localhost", false, "Allow localhost access")
    netTimeout        = flag.Duration("net-timeout", 30*time.Second, "HTTP request timeout")
)

func setupEffContext(caps []string) *effects.EffContext {
    ctx := effects.NewEffContext()

    for _, cap := range caps {
        ctx.Grant(cap)
    }

    // Net-specific config
    if slices.Contains(caps, "Net") {
        if *netAllowList != "" {
            ctx.allowedDomains = strings.Split(*netAllowList, ",")
        }
        ctx.netTimeout = *netTimeout
        ctx.allowLocalhost = *netAllowLocalhost
    }

    return ctx
}
```

**Integration Tests**:
```go
// internal/effects/net_test.go
func TestNetHttpGet_Security(t *testing.T) {
    tests := []struct {
        name    string
        url     string
        errMsg  string
    }{
        {"localhost_blocked", "http://localhost:8080", "localhost access blocked"},
        {"private_ip", "http://192.168.1.1", "private IP access blocked"},
        {"file_protocol", "file:///etc/passwd", "unsupported protocol"},
        {"data_url", "data:text/plain,hello", "unsupported protocol"},
    }

    ctx := NewEffContext()
    ctx.Grant("Net")

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            _, err := netHttpGet(ctx, []Value{StringValue(tt.url)})
            assert.Error(t, err)
            assert.Contains(t, err.Error(), tt.errMsg)
        })
    }
}
```

**Example Files**:
```ailang
// examples/micro_clock_measure.ail
module examples/micro_clock_measure

import std/clock (now, sleep)
import std/io (println)

export func measure_sleep() -> int ! {Clock, IO} {
  let start = now();
  sleep(100);  -- Sleep 100ms
  let elapsed = now() - start;
  println("Slept for " ++ show(elapsed) ++ "ms");
  elapsed
}

export func main() -> int ! {Clock, IO} {
  measure_sleep()
}
```

```ailang
// examples/micro_net_fetch.ail
module examples/micro_net_fetch

import std/net (httpGet)
import std/io (println)

export func fetch_example() -> string ! {Net, IO} {
  let html = httpGet("https://example.com");
  println("Fetched " ++ show(length(html)) ++ " bytes");
  html
}

export func main() -> string ! {Net, IO} {
  fetch_example()
}
```

## Acceptance Criteria

### Clock Effect (MUST SHIP)
- [ ] `now()` returns current Unix timestamp (ms)
- [ ] `sleep(ms)` blocks for specified duration
- [ ] Virtual time works in deterministic mode (`AILANG_SEED`)
- [ ] Requires `--caps Clock`, fails without
- [ ] `std/clock` module available

### Net Effect (STRETCH GOAL)
- [ ] `httpGet(url)` fetches HTTP/HTTPS resources
- [ ] `httpPost(url, body)` sends POST requests
- [ ] Localhost blocked by default
- [ ] Private IPs blocked by default
- [ ] file://, ftp://, data:// protocols rejected
- [ ] 30s timeout enforced (configurable)
- [ ] `--net-allow` allowlist works
- [ ] Requires `--caps Net`, fails without

### Code Quality
- [ ] 100% test coverage for Clock effect
- [ ] 100% test coverage for Net security checks
- [ ] Clear error messages with hints
- [ ] Examples documented and passing

## Risks & Mitigations

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Net security vulnerabilities** | Critical | Medium | **Defer Net to v0.3.1 if security can't be hardened** |
| **SSRF attacks** | High | Medium | Block localhost, private IPs, validate all URLs |
| **Timeout bypass** | Medium | Low | Enforce at HTTP client level, not user code |
| **Deterministic time bugs** | Low | Low | Comprehensive testing of virtual time |
| **Timeline pressure** | Medium | High | **Ship Clock only, defer Net to v0.3.1** |

**CRITICAL DECISION**: If Week 2 timeline is tight, **cut Net effect entirely** and ship Clock only.

## Testing Strategy

### Unit Tests (~100 LOC)
- `internal/effects/clock_test.go`
  - Real time: `now()`, `sleep()`
  - Virtual time: deterministic mode
  - Edge cases: negative sleep, capability denied

- `internal/effects/net_test.go` (if Net shipped)
  - Security: localhost, private IP, protocols
  - Timeout: slow requests
  - Domain allowlist: matching logic

### Integration Tests
- `examples/micro_clock_measure.ail` - Clock effect demo
- `examples/micro_net_fetch.ail` - Net effect demo (if shipped)
- `examples/micro_clock_rate_limit.ail` - Real-world timing

### Security Tests (Net only)
- SSRF prevention: localhost, 127.0.0.1, private IPs
- Protocol validation: file://, ftp://, data://
- Timeout enforcement: mock slow server
- Domain allowlist: wildcard matching

## Success Metrics

| Metric | Clock Target | Net Target (Stretch) |
|--------|-------------|----------------------|
| **Examples fixed** | +2-3 | +2-3 |
| **Security issues** | N/A | 0 |
| **Test coverage** | 100% | 100% |
| **Effects available** | IO, FS, Clock (3) | + Net (4) |

## Scope Guardrails

**IF timeline is tight (end of Week 2)**:

1. **Priority 1**: Ship Clock effect
   - Simple, low-risk implementation
   - Enables timing and rate limiting
   - No security concerns

2. **Priority 2**: Defer Net effect to v0.3.1
   - Security complexity is high
   - SSRF prevention requires careful design
   - Can ship incrementally in next release

**Release Decision Tree**:
```
Timeline OK? → Ship Clock + Net
Timeline tight? → Ship Clock only, defer Net
Timeline very tight? → Defer both to v0.3.1
```

## Future Work (Deferred)

**v0.3.1 - Net Effect** (if deferred):
- Full implementation with security hardening
- SSRF prevention audit
- Rate limiting support

**v0.4.0 - Extended Clock**:
- Timezone support: `nowInZone(tz)`
- Date parsing: `parseDate(str, fmt)`
- Time arithmetic: `addDays(ts, n)`

**v0.4.0 - Extended Net**:
- Request headers: `httpGetWithHeaders(url, headers)`
- Response status: `httpGetWithStatus(url) -> (status, body)`
- Streaming: `httpStream(url, callback)`
- WebSocket support

**v0.5.0 - Async Effects**:
- `async/await` syntax
- Concurrent HTTP requests
- Effect composition

## References

- **v0.2.0**: Effect system foundation (M-R2)
- **Design Doc**: `design_docs/planned/v0_3_0_implementation_plan.md`
- **Security**: OWASP SSRF prevention guide
- **Prior Art**: Node.js fetch, Python requests, Go http.Client
