# M-ENV: Environment Variable Support

**Status**: Planned
**Target**: v0.4.0
**Priority**: P1 (High - completes Net enhancements milestone)
**Estimated**: 3 days (~24 hours with security enhancements)
**Dependencies**: None (Net capability pattern already established)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes hardcoded secrets from source files, eliminates need to pass secrets through CLI args |
| Preserve Semantic Clarity | + | +1 | Explicit `Env` capability shows exactly when code reads environment, effect-typed in signatures |
| Increase Determinism | 0 | 0 | Env vars can change between runs, but this is expected/controllable (use --env flags for testing) |
| Lower Token Cost | + | +1 | Removes need to hardcode or explain API keys in prompts, cleaner examples |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- Cannot read environment variables from AILANG code
- API keys and secrets must be hardcoded in source files (security risk)
- No way to configure programs for different environments (dev/staging/prod)
- AI-generated code cannot integrate with standard deployment practices (12-factor apps)

**Impact:**
- 🚫 **Cannot call AI APIs with authentication** (OpenAI, Claude, Gemini require API keys)
- 🚫 **Security risk**: Secrets committed to source control
- 🚫 **Non-standard deployment**: Can't follow 12-factor app methodology
- 🚫 **Harder AI code generation**: Models must generate insecure hardcoded credentials

**Blocker for v0.4.0:**
The [v0_4_0_net_enhancements.md](v0_4_0_net_enhancements.md) design doc lists environment variables as Feature 2 of 3 for completing Net enhancements. Currently:
- ✅ Feature 1: Custom HTTP headers (implemented in v0.3.8)
- ❌ Feature 2: Environment variables (NOT implemented - this doc)
- ✅ Feature 3: JSON parsing (implemented in v0.3.14-v0.3.22)

## Goals

**Primary Goal:** Enable AILANG programs to securely read environment variables with explicit capability tracking.

**Success Metrics:**
- ✅ `getEnv(name)` and `hasEnv(name)` functions work in AILANG code
- ✅ Requires explicit `--caps Env` grant (capability security)
- ✅ AI-generated code can call OpenAI/Claude/Gemini APIs with authentication
- ✅ Examples demonstrate secure API key handling
- ✅ Zero security vulnerabilities (no env var enumeration, audit logging optional)

## Solution Design

### Overview

Add a new `Env` capability and two builtin functions (`getEnv`, `hasEnv`) that allow reading specific environment variables. Follow the established pattern from `Net`, `IO`, `FS` capabilities, but with enhanced security and determinism features to avoid regrets at v1.

**Key design decisions:**
1. **Opt-in security**: Requires explicit `--caps Env` grant
2. **Allowlist granularity**: Optional `--allow-env` restricts which keys can be read (least-privilege)
3. **Snapshot semantics**: Env captured at program start, frozen for deterministic behavior
4. **No enumeration**: Cannot list all env vars, must know names
5. **Secrets hygiene**: Automatic redaction of sensitive values in errors/logs/debug output
6. **Effect-typed**: Functions return `! {Env}` effect in signatures (marked impure for optimizer)
7. **DX helpers**: `getEnvOr` helper for handling empty/missing vars gracefully

### Architecture

**Components:**

1. **Effect Implementation** (`internal/effects/env.go`)
   - `envGetEnv(ctx, args)` - Read env var from snapshot, return string (empty if not set)
   - `envHasEnv(ctx, args)` - Check if env var exists in snapshot, return bool
   - Capability check enforcement (`ctx.HasCap("Env")`)
   - **Allowlist enforcement**: Check `ctx.EnvAllowlist.Contains(name)` before reading
   - **Redaction system**: Mask sensitive values in error messages

2. **Snapshot System** (`internal/effects/context.go`)
   - `EnvSnapshot map[string]string` - Frozen copy of environment at program start
   - `EnvAllowlist map[string]bool` - Set of permitted env var names (nil = allow all)
   - Populated once during EffContext creation, immutable thereafter

3. **Builtin Registration** (`internal/builtins/env.go`)
   - Register `_env_getEnv` and `_env_hasEnv` with M-DX1 registry
   - Type signatures with `Env` effect
   - **Mark as impure** in metadata (reads ambient state, no CSE across snapshots)
   - Complete metadata (descriptions, params, returns, examples, security notes)

4. **Standard Library** (`std/env.ail`)
   - Export wrapper functions with documentation
   - Type signatures: `getEnv(string) -> string ! {Env}`, `hasEnv(string) -> bool ! {Env}`
   - **Helper**: `getEnvOr(name, default) -> string ! {Env}` for empty/missing handling

5. **CLI Integration** (`cmd/ailang/run.go` and `cmd/ailang/repl.go`)
   - Parse `--caps Env` flag (coarse grant)
   - Parse `--allow-env KEY1,KEY2` or `--allow-env-file path.txt` (granular allowlist)
   - Parse `--env KEY=VALUE` (inject without mutating OS)
   - Parse `--env-file .env` (dotenv-style loader, optional)
   - Parse `--env-snapshot path.json` (load exact snapshot)
   - Parse `--write-env-snapshot path.json` (save snapshot for reproducibility)
   - Build EnvSnapshot and EnvAllowlist, pass to EffContext

6. **Redaction System** (`internal/effects/redact.go`, new)
   - Pattern matching: `(?i)(key|secret|token|password|credential)`
   - Redact values in:
     - Error messages from builtins
     - Debug logs (`--debug-compile`, evaluator traces)
     - Panic/crash reports
   - Control with `AILANG_REDACT_ENV=off` for local debugging

### Implementation Plan

**Phase 1: Core Effect Implementation** (~4 hours)
- [ ] Create `internal/effects/env.go` with `envGetEnv` and `envHasEnv`
- [ ] Implement capability checking (fail if `Env` not granted)
- [ ] Read from `ctx.EnvSnapshot` instead of `os.Getenv` directly
- [ ] Handle edge cases (non-existent vars return empty string/"" for getEnv, false for hasEnv)
- [ ] Add validation (args must be strings)
- [ ] Unit tests for basic effect functions (with mock EffContext)

**Phase 2: Snapshot System** (~4 hours)
- [ ] Add `EnvSnapshot map[string]string` to `EffContext` in `internal/effects/context.go`
- [ ] Add `EnvAllowlist map[string]bool` to `EffContext` (nil = allow all)
- [ ] Populate snapshot once at EffContext creation from `os.Environ()`
- [ ] Parse `--env KEY=VALUE` injections (override OS values)
- [ ] Parse `--env-snapshot path.json` loader (override everything)
- [ ] Implement `--write-env-snapshot path.json` saver (for reproducibility)
- [ ] Case-insensitive handling on Windows (use `strings.EqualFold` for lookups)
- [ ] Unit tests for snapshot immutability (external env changes don't affect snapshot)

**Phase 3: Allowlist System** (~3 hours)
- [ ] Parse `--allow-env KEY1,KEY2,KEY3` CSV format in CLI
- [ ] Parse `--allow-env-file path.txt` (one key per line, ignore empty/comments)
- [ ] Enforce allowlist in `envGetEnv` and `envHasEnv`:
  ```go
  if ctx.EnvAllowlist != nil && !ctx.EnvAllowlist[name] {
      return nil, fmt.Errorf("E_ENV_NOT_ALLOWED: environment variable %q is not allow-listed; run with --allow-env %s", name, name)
  }
  ```
- [ ] Allowlist enforcement tests (allowed key → ok, disallowed key → deterministic error)

**Phase 4: Redaction System** (~3 hours)
- [ ] Create `internal/effects/redact.go` with `RedactSensitive(key, value string) string`
- [ ] Pattern matching: `(?i)(key|secret|token|password|credential)` → redact value
- [ ] Redact all allow-listed keys (assume sensitive if explicitly controlled)
- [ ] Hook into error formatting: wrap builtin errors with redaction
- [ ] Hook into debug logs: redact in `--debug-compile` output
- [ ] Control with `AILANG_REDACT_ENV=off` environment variable
- [ ] Unit tests: verify no raw secrets in error messages or logs

**Phase 5: Builtin Registration** (~2 hours)
- [ ] Create `internal/builtins/env.go`
- [ ] Register `_env_getEnv` with BuiltinSpec (NumArgs=1, Effect="Env", **IsPure=false**)
- [ ] Register `_env_hasEnv` with BuiltinSpec (NumArgs=1, Effect="Env", **IsPure=false**)
- [ ] Build type signatures using Type Builder DSL
- [ ] Add comprehensive metadata (M-DX1.11 spec) including security notes
- [ ] Document case-sensitivity behavior (POSIX vs Windows)
- [ ] Validate with `ailang doctor builtins`

**Phase 6: Standard Library & DX Helper** (~2 hours)
- [ ] Create `std/env.ail` module
- [ ] Export `getEnv(name: string) -> string ! {Env}` wrapper
- [ ] Export `hasEnv(name: string) -> bool ! {Env}` wrapper
- [ ] Add `getEnvOr(name: string, default: string) -> string ! {Env}` helper:
  ```ailang
  export func getEnvOr(name: string, default: string) -> string ! {Env} {
    if hasEnv(name) then {
      let v = getEnv(name);
      if v == "" then default else v
    } else {
      default
    }
  }
  ```
- [ ] Add comprehensive documentation with security notes and idioms
- [ ] Integration test (parse and compile std/env.ail)

**Phase 7: Comprehensive Testing** (~4 hours)
- [ ] **Allowlist tests**:
  - `TestEnv_Allowlist_AllowsExplicitKey`
  - `TestEnv_Allowlist_BlocksNonListedKey`
  - `TestEnv_Allowlist_NilMeansAllowAll`
- [ ] **Snapshot tests**:
  - `TestEnv_Snapshot_FrozenAcrossRun` (mutate OS, confirm no effect)
  - `TestEnv_CLIInject_OverridesOS` (`--env KEY=VALUE`)
  - `TestEnv_Snapshot_File_RoundTrip` (save + load)
- [ ] **Redaction tests**:
  - `TestEnv_Redaction_InErrorsAndLogs`
  - `TestEnv_Redaction_DisableWithEnvVar`
- [ ] **Platform tests**:
  - `TestEnv_Windows_CaseInsensitiveKeys` (skip on non-Windows)
  - `TestEnv_UTF8_NamesAndValues`
- [ ] **Basic tests** (from original plan):
  - `TestEnvGetEnv_ExistingVar`, `TestEnvGetEnv_MissingVar`, `TestEnvGetEnv_EmptyVar`
  - `TestEnvHasEnv_ExistingVar`, `TestEnvHasEnv_MissingVar`
  - `TestEnvRequiresCapability`, `TestEnvWrongArgType`, `TestEnvWrongArgCount`
- [ ] **REPL tests**: Verify `--caps Env` required, precise errors without leaking values

**Phase 8: Examples & Integration** (~2 hours)
- [ ] Update OpenAI example to use `getEnv("OPENAI_API_KEY")` with allowlist
- [ ] Update Gemini example to use `getEnv("GEMINI_API_KEY")` with allowlist
- [ ] Add config example using snapshot + injection for dev/staging/prod
- [ ] Manual testing with all CLI flags

**Phase 9: Documentation** (~2 hours)
- [ ] Update CHANGELOG.md with v0.4.0 entry (all features documented)
- [ ] Update CLAUDE.md with Env capability and all flags
- [ ] Update teaching prompt (prompts/v0.4.0.md) with env var examples
- [ ] Add to docs/CAPABILITIES.md with operational knobs table
- [ ] Update v0_4_0_net_enhancements.md status (Feature 2 complete)
- [ ] Document recommended idioms (allowlist, snapshot, redaction)

### Files to Modify/Create

**New files:**
- `internal/effects/env.go` - Effect implementation with allowlist enforcement (~180 LOC)
- `internal/effects/redact.go` - Redaction system for secrets hygiene (~120 LOC)
- `internal/builtins/env.go` - Builtin registration with security metadata (~120 LOC)
- `std/env.ail` - Standard library wrapper with `getEnvOr` helper (~80 LOC)
- `internal/effects/env_test.go` - Comprehensive unit tests (~350 LOC)
- `internal/effects/redact_test.go` - Redaction system tests (~100 LOC)
- `examples/openai_with_env.ail` - Example with API key + allowlist (~100 LOC)
- `examples/env_config.ail` - Example showing snapshot + injection (~80 LOC)

**Modified files:**
- `internal/effects/context.go` - Add EnvSnapshot + EnvAllowlist fields (~40 LOC)
- `cmd/ailang/run.go` - Parse all env flags, build snapshot/allowlist (~80 LOC)
- `cmd/ailang/repl.go` - Add env flag parsing for REPL (~40 LOC)
- `internal/errors/errors.go` - Hook redaction into error formatting (~30 LOC)
- `CHANGELOG.md` - Add v0.4.0 section with all features (~50 LOC)
- `CLAUDE.md` - Document Env capability and operational knobs (~40 LOC)
- `docs/CAPABILITIES.md` - Add Env capability documentation (~60 LOC, create if needed)

**Total estimated new code:** ~1,450 LOC (including comprehensive tests and security features)

## Examples

### Example 1: Reading API Key

**Before (INSECURE):**
```ailang
-- ❌ Hardcoded secret in source file
let apiKey = "sk-proj-abc123...xyz789"  -- NEVER DO THIS!

let headers = [{name: "Authorization", value: "Bearer " ++ apiKey}]
httpRequest("POST", "https://api.openai.com/v1/chat/completions", headers, body)
```

**After (SECURE):**
```ailang
import std/env (getEnv)
import std/net (httpRequest)

-- ✅ Read from environment
func callOpenAI(prompt: string) -> string ! {Net, Env} {
  let apiKey = getEnv("OPENAI_API_KEY");
  let headers = [{name: "Authorization", value: "Bearer " ++ apiKey}];
  let body = "{\"model\":\"gpt-4\",\"messages\":[{\"role\":\"user\",\"content\":\"" ++ prompt ++ "\"}]}";

  match httpRequest("POST", "https://api.openai.com/v1/chat/completions", headers, body) {
    Ok(resp) => resp.body,
    Err(Transport(msg)) => "Error: " ++ msg,
    Err(DisallowedHost(host)) => "Blocked: " ++ host,
    Err(InvalidHeader(hdr)) => "Invalid header: " ++ hdr,
    Err(BodyTooLarge(size)) => "Response too large: " ++ size
  }
}

export func main() -> () ! {IO, Net, Env} {
  println(callOpenAI("Hello, GPT-4!"))
}
```

**Usage (with allowlist for security):**
```bash
export OPENAI_API_KEY="sk-proj-..."
ailang run --caps IO,Net,Env --allow-env OPENAI_API_KEY --entry main openai_example.ail
# If key is missing or not allowed, error is clear and doesn't leak value
```

**Production usage (with snapshot for reproducibility):**
```bash
# Create snapshot once
ailang run --caps Env --allow-env OPENAI_API_KEY --write-env-snapshot openai.env.json --entry main openai_example.ail

# Later runs use exact snapshot (no surprises from env changes)
ailang run --caps IO,Net,Env --env-snapshot openai.env.json --entry main openai_example.ail
```

### Example 2: Conditional Configuration

**Use case:** Different behavior for dev/staging/prod environments

```ailang
import std/env (getEnv, hasEnv)
import std/io (println)

func getApiBaseUrl() -> string ! {Env} {
  if hasEnv("AILANG_ENV") then {
    let env = getEnv("AILANG_ENV");
    if env == "prod" then "https://api.production.com"
    else if env == "staging" then "https://api.staging.com"
    else "http://localhost:8080"
  } else {
    "http://localhost:8080"  -- Default to local dev
  }
}

export func main() -> () ! {IO, Env} {
  println("API URL: " ++ getApiBaseUrl())
}
```

**Usage (with injection for testing):**
```bash
# Development (default - no env var set)
ailang run --caps IO,Env --entry main config.ail
# API URL: http://localhost:8080

# Staging (inject without mutating OS)
ailang run --caps IO,Env --env AILANG_ENV=staging --entry main config.ail
# API URL: https://api.staging.com

# Production (inject specific value)
ailang run --caps IO,Env --env AILANG_ENV=prod --entry main config.ail
# API URL: https://api.production.com

# Or use actual OS env (snapshot still applies)
export AILANG_ENV=prod
ailang run --caps IO,Env --entry main config.ail
# API URL: https://api.production.com
```

### Example 3: Missing Environment Variable Handling (with getEnvOr)

**Using the DX helper:**
```ailang
import std/env (getEnvOr)
import std/io (println)

-- Simplified: getEnvOr handles empty/missing gracefully
export func main() -> () ! {IO, Env} {
  let apiKey = getEnvOr("API_KEY", "default-key-for-dev");
  let timeout = getEnvOr("TIMEOUT_MS", "5000");

  println("Using API key: " ++ apiKey);
  println("Timeout: " ++ timeout ++ "ms")
}
```

**Manual pattern (if you don't want std/env dependency):**
```ailang
import std/env (hasEnv, getEnv)
import std/io (println)

func loadConfig() -> string ! {IO, Env} {
  if hasEnv("API_KEY") then {
    let key = getEnv("API_KEY");
    if key == "" then {
      println("Warning: API_KEY is set but empty");
      "default-key"
    } else {
      key
    }
  } else {
    println("Error: API_KEY environment variable not set");
    println("Please set: export API_KEY=your-key-here");
    ""  -- Return empty string, caller handles error
  }
}

export func main() -> () ! {IO, Env} {
  let config = loadConfig();
  if config == "" then {
    println("Configuration failed - exiting")
  } else {
    println("Configuration loaded successfully")
  }
}
```

**Recommended idiom:** Use `getEnvOr` for optional config, manual checks for required config.

## Success Criteria

**Core functionality:**
- [ ] `getEnv(name)` returns string value from snapshot, or "" if not set
- [ ] `hasEnv(name)` returns true if var exists in snapshot, false otherwise
- [ ] Both functions require `Env` capability (fail with clear error if missing)
- [ ] Cannot enumerate all env vars (security: no `listEnv()` function)
- [ ] Type signatures include `! {Env}` effect, marked impure in metadata

**Security features:**
- [ ] Allowlist enforcement: `--allow-env` restricts readable keys
- [ ] Allowlist error messages are actionable (suggest `--allow-env KEY`)
- [ ] Redaction: sensitive values never appear in errors/logs/panics
- [ ] No env var enumeration possible (runtime or reflection)

**Determinism features:**
- [ ] Snapshot captured once at program start, immutable thereafter
- [ ] External env changes during execution don't affect reads
- [ ] `--env KEY=VALUE` injection overrides OS values
- [ ] `--env-snapshot path.json` provides exact reproducible environment
- [ ] `--write-env-snapshot` captures snapshot for later replay

**DX features:**
- [ ] `getEnvOr(name, default)` helper works correctly
- [ ] Works in REPL, modules, and files with same semantics
- [ ] Case-insensitive on Windows, case-sensitive on POSIX

**Integration:**
- [ ] OpenAI example works with real API key + allowlist
- [ ] Gemini example works with real API key + allowlist
- [ ] Config example demonstrates snapshot + injection

**Quality:**
- [ ] All 20+ tests passing (unit + integration + security + platform)
- [ ] Documentation updated (CHANGELOG, CLAUDE.md, teaching prompt, CAPABILITIES.md)
- [ ] `ailang doctor builtins` passes validation
- [ ] No regressions in existing tests

## Testing Strategy

**Unit tests (`internal/effects/env_test.go` - ~350 LOC):**

*Basic functionality:*
- `TestEnvGetEnv_ExistingVar` - Read existing env var from snapshot
- `TestEnvGetEnv_MissingVar` - Missing var returns empty string
- `TestEnvGetEnv_EmptyVar` - Empty var returns "" (distinguish via hasEnv)
- `TestEnvHasEnv_ExistingVar` - Var exists returns true
- `TestEnvHasEnv_MissingVar` - Missing var returns false
- `TestEnvRequiresCapability` - Fails without Env capability
- `TestEnvWrongArgType` - Fails if arg is not string
- `TestEnvWrongArgCount` - Fails if arg count != 1

*Allowlist tests:*
- `TestEnv_Allowlist_AllowsExplicitKey` - Allowed key works
- `TestEnv_Allowlist_BlocksNonListedKey` - Disallowed key gives actionable error
- `TestEnv_Allowlist_NilMeansAllowAll` - No allowlist = all keys allowed
- `TestEnv_Allowlist_FileLoader` - `--allow-env-file` parsing

*Snapshot tests:*
- `TestEnv_Snapshot_FrozenAcrossRun` - External mutations don't affect snapshot
- `TestEnv_CLIInject_OverridesOS` - `--env KEY=VALUE` takes precedence
- `TestEnv_Snapshot_File_RoundTrip` - Save + load produces identical behavior
- `TestEnv_Snapshot_File_OverridesAll` - Snapshot beats injections and OS

*Redaction tests (`internal/effects/redact_test.go` - ~100 LOC):*
- `TestEnv_Redaction_InErrorsAndLogs` - Secrets never in output
- `TestEnv_Redaction_DisableWithEnvVar` - `AILANG_REDACT_ENV=off` works
- `TestEnv_Redaction_PatternMatching` - Key/secret/token/password detected
- `TestEnv_Redaction_AllowlistImpliesSecret` - Allow-listed keys always redacted

*Platform tests:*
- `TestEnv_Windows_CaseInsensitiveKeys` - Windows case-folding (skip on non-Windows)
- `TestEnv_UTF8_NamesAndValues` - Unicode support

**Integration tests:**
- Parse and compile `std/env.ail` successfully (including `getEnvOr`)
- Import `std/env` in other modules
- End-to-end: Inject var, run program, verify output matches expectation
- REPL: Verify `--caps Env` required, error messages don't leak values

**Security tests:**
- Verify no way to enumerate env vars (no `listEnv` or reflection)
- Verify allowlist bypass impossible (try reflection, try numeric iteration)
- Verify capability enforcement (cannot read without `--caps Env`)

**Manual testing:**
```bash
# Test basic getEnv with snapshot
export TEST_VAR="hello"
ailang repl --caps Env
> import std/env (getEnv)
> getEnv("TEST_VAR")
"hello"

# Change OS env externally (snapshot should not change)
# In another terminal: export TEST_VAR="changed"
> getEnv("TEST_VAR")
"hello"  -- Still original value (snapshot frozen)

# Test allowlist enforcement
ailang repl --caps Env --allow-env ALLOWED_KEY
> import std/env (getEnv)
> getEnv("ALLOWED_KEY")
""  -- Works (even though not set)
> getEnv("SECRET_KEY")
Error: E_ENV_NOT_ALLOWED: environment variable "SECRET_KEY" is not allow-listed; run with --allow-env SECRET_KEY

# Test redaction
> getEnv("API_KEY")  -- Assume API_KEY is in allowlist but not set
Error: ... (error message does NOT contain API_KEY value even if it existed)

# Test injection
ailang run --caps Env --env TEST=injected --entry main test.ail
# Uses "injected" even if OS has different value

# Test capability requirement
ailang repl  # NO --caps Env
> import std/env (getEnv)
> getEnv("TEST_VAR")
Error: E_ENV_CAP_MISSING: Env capability not granted
```

## Non-Goals

**Not in this feature:**
- ❌ **Setting environment variables** (`setEnv`) - OS-level operation, use shell
- ❌ **Enumerating all env vars** (`listEnv`) - Security risk, not needed
- ❌ **Environment variable expansion** (`$VAR` syntax) - Use getEnv explicitly
- ❌ **Default values in getEnv** - Caller handles with if/else (keeps API simple)
- ❌ **Typed env vars** (parse int/bool) - Use getEnv + string conversions
- ❌ **Env var validation** (required vars) - Application-level concern
- ❌ **Dotenv file support** (`.env` files) - Use shell: `export $(cat .env)`

**Rationale:**
- Keep feature minimal and secure (principle of least privilege)
- Follow Unix philosophy: do one thing well (read env vars)
- Application logic handles validation, parsing, defaults

## Operational Knobs

**CLI flags for environment variable control:**

| Flag | Arguments | Purpose | Example |
|------|-----------|---------|---------|
| `--caps Env` | - | Grant coarse Env capability | `ailang run --caps Env ...` |
| `--allow-env` | CSV of keys | Restrict readable env vars (least-privilege) | `--allow-env OPENAI_API_KEY,AILANG_ENV` |
| `--allow-env-file` | Path to file | Load allowlist from file (one key per line) | `--allow-env-file .env.allow` |
| `--env` | KEY=VALUE | Inject env var (overrides OS, can repeat) | `--env API_KEY=test123 --env ENV=dev` |
| `--env-file` | Path to file | Load env vars from dotenv file (optional) | `--env-file .env.test` |
| `--env-snapshot` | Path to JSON | Load exact env snapshot (overrides all) | `--env-snapshot prod.env.json` |
| `--write-env-snapshot` | Path to JSON | Save env snapshot after loading | `--write-env-snapshot out.env.json` |

**Environment variables (control AILANG behavior):**

| Variable | Values | Purpose |
|----------|--------|---------|
| `AILANG_REDACT_ENV` | `off` / `on` (default: `on`) | Disable redaction for local debugging |

**Precedence order (highest to lowest):**
1. `--env-snapshot` (exact snapshot, wins over everything)
2. `--env KEY=VALUE` (per-run injection)
3. `--env-file` (bulk injection from file)
4. OS environment variables (captured at program start)

**Allowlist behavior:**
- If `--allow-env` or `--allow-env-file` provided: Only listed keys readable
- If neither provided: All keys readable (nil allowlist = allow all)
- Allowlist applies regardless of snapshot/injection source

**Case sensitivity:**
- POSIX systems: Case-sensitive (FOO != foo)
- Windows: Case-insensitive (FOO == foo, uses case-fold for lookups)

**File formats:**

*.env.allow* (allowlist file):
```
# Comments and empty lines ignored
OPENAI_API_KEY
GEMINI_API_KEY
AILANG_ENV
# One key per line
```

*.env* (dotenv file, optional in v0.4.0):
```
# Standard dotenv format
API_KEY=value-here
ENV=production
```

*snapshot.json* (env snapshot):
```json
{
  "API_KEY": "sk-proj-...",
  "ENV": "production",
  "PATH": "/usr/bin:/bin"
}
```

**Static analysis (future):**
- If `getEnv("LITERAL")` called with string literal, can extract declared reads
- Enables `ailang doctor env` to validate allowlists against actual usage
- Powers CI policy: "only these keys are allowed in production"

## Timeline

**Day 1** (8 hours):
- Morning (4h): Phase 1 - Core effect implementation + basic tests
- Afternoon (4h): Phase 2 - Snapshot system + CLI parsing

**Day 2** (8 hours):
- Morning (3h): Phase 3 - Allowlist system + enforcement
- Midday (3h): Phase 4 - Redaction system + security tests
- Afternoon (2h): Phase 5 - Builtin registration + validation

**Day 3** (8 hours):
- Morning (2h): Phase 6 - Standard library + getEnvOr helper
- Midday (4h): Phase 7 - Comprehensive testing (20+ tests)
- Afternoon (2h): Phases 8-9 - Examples + documentation

**Total: ~24 hours across 3 days**

**Buffer:** +4 hours for unexpected issues (platform differences, edge cases, debugging)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Security: Env var enumeration** | High | No `listEnv` function, only read specific vars by name; comprehensive tests |
| **Security: Allowlist bypass** | High | Enforce in every read; test reflection/iteration attempts; comprehensive tests |
| **Security: Secrets in logs/errors** | High | Redaction system with pattern matching + allowlist; `AILANG_REDACT_ENV=off` for debug |
| **Determinism: TOCTOU races** | Medium | Snapshot captured once at start, frozen; tests verify immutability |
| **Determinism: Reproducibility** | Medium | `--env-snapshot` + `--write-env-snapshot` enable exact replay; JSON round-trip tested |
| **Platform: Windows case sensitivity** | Low | Case-folding on Windows (`strings.EqualFold`); platform-specific tests |
| **DX: Empty vs missing vars** | Low | `hasEnv` distinguishes; `getEnvOr` helper handles both; documented idiom |
| **CRLF injection in env values** | Low | Values are opaque strings, user validates if needed |
| **Capability bypass** | High | Strict enforcement in effect functions, comprehensive tests, no reflection holes |
| **Performance: Snapshot size** | Low | Env is small (<1KB typically), one-time copy; acceptable overhead |

**Critical security invariants:**
- ✅ **Cannot read env without Env capability** - Enforced in `envGetEnv` and `envHasEnv`
- ✅ **Cannot enumerate env vars** - No API to list all keys, no reflection, tested against bypass
- ✅ **Cannot bypass allowlist** - Enforced on every read, nil = allow all, tested
- ✅ **Secrets never in output** - Redaction system intercepts errors/logs/panics
- ✅ **Snapshot immutable** - Frozen at creation, external changes ignored, tested
- ✅ **Precedence deterministic** - Snapshot > injection > file > OS, well-defined, tested

## References

- **Original design**: [v0_4_0_net_enhancements.md](v0_4_0_net_enhancements.md) (Feature 2)
- **M-DX1 builtin system**: [implemented/v0_3_10/M-DX1_developer_experience.md](../../implemented/v0_3_10/M-DX1_developer_experience.md)
- **Net capability pattern**: `internal/effects/net.go` (existing example)
- **Type Builder DSL**: `internal/types/builder.go`
- **12-factor app methodology**: https://12factor.net/config (external reference)
- **Go os.Getenv docs**: https://pkg.go.dev/os#Getenv

## Future Work

**Potential v0.5.0+ enhancements:**
- **Audit logging**: Log all env var accesses for security review (opt-in)
- **Env var schemas**: Declare required/optional vars in module manifests
- **Type-safe env vars**: `getEnvInt`, `getEnvBool` with `Result` return for parsing errors
- **Static analysis**: `ailang doctor env` to validate allowlists against actual usage
- **CI policy enforcement**: Fail build if non-allowed keys accessed
- **Dotenv file loader**: `--env-file` implementation (wire is ready, just needs parser)
- **Env var validation**: Custom validators in module manifest

**Not planned:**
- Environment variable modification (out of scope for AILANG's deterministic model)
- Dynamic env var changes during execution (snapshot is immutable by design)
- Process-level environment manipulation (`setenv`, `unsetenv` at OS level)

---

**Document created**: 2025-10-30
**Last updated**: 2025-10-30
**Author**: Claude (design-doc-creator skill)
**Version**: v2.0 (enhanced with security & determinism features)

**Changelog:**
- v1.0 (2025-10-30): Initial draft with basic getEnv/hasEnv
- v2.0 (2025-10-30): Added allowlist, snapshot semantics, redaction, getEnvOr helper, comprehensive security features (based on review feedback)
