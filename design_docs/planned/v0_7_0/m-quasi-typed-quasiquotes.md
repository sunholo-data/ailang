# M-QUASI: Typed Quasiquotes (String Templates)

**Status**: Planned
**Target**: v0.4.2 (or defer to v0.5.0)
**Priority**: P1 (Medium - High value for web/API applications)
**Estimated**: 3-4 weeks (~100 hours)
**Dependencies**: v0.4.0 complete, consider deferring to v0.5.0

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Replaces manual string concatenation with template literals |
| Preserve Semantic Clarity | + | +1 | Type annotations make interpolations explicit (${x: SafeText}) |
| Increase Determinism | + | +1 | Typed interpolations prevent injection attacks at compile time |
| Lower Token Cost | 0 | 0 | Neutral - replaces one pattern with another |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Current State:**
- Web/API examples require manual string concatenation
- SQL queries vulnerable to injection attacks (no type safety)
- HTML templating requires manual escaping (XSS vulnerabilities)
- JSON construction is verbose and error-prone

**Impact:**
- **Example file blocked**: `examples/experimental/web_api.ail` cannot run
- **Security**: No compile-time protection against injection attacks
- **DX**: Verbose string construction reduces readability
- **AI code gen**: Models struggle with proper escaping/interpolation

**Example of current pain:**
```ailang
-- Current: Manual string concatenation (error-prone)
let query = "SELECT * FROM users WHERE id = " ++ show(id)  -- SQL INJECTION!
let html = "<div>" ++ escapeHTML(name) ++ "</div>"  -- Must remember to escape
let json = "{\"name\": \"" ++ escapeJSON(name) ++ "\"}"  -- Quotes must be escaped
```

## Goals

**Primary Goal:** Provide type-safe string templates with compile-time interpolation checking and automatic escaping

**Success Metrics:**
- `web_api.ail` example works correctly with SQL, HTML, JSON quasiquotes
- Zero SQL injection or XSS vulnerabilities in generated code
- 50% reduction in string construction boilerplate
- AI models can generate safe web code (measured via M-EVAL)

## Solution Design

### Overview

Add typed quasiquote syntax with three forms:

1. **SQL Quasiquotes**: Type-safe database queries with parameterization
2. **HTML Quasiquotes**: XSS-safe HTML templating with automatic escaping
3. **JSON Quasiquotes**: Structured JSON construction with type checking

Each quasiquote form:
- Has a **prefix** (e.g., `sql`, `html`, `json`)
- Uses **triple-quoted strings** for multi-line support
- Requires **type annotations** on interpolations (e.g., `${x: int}`)
- Provides **compile-time validation** of interpolations
- Applies **context-appropriate escaping** automatically

### Architecture

**Components:**

1. **Lexer Extensions** (`internal/lexer/lexer.go`)
   - Recognize quasiquote prefixes: `sql"""`, `html"""`, `json{`
   - Tokenize interpolations: `${expr: type}`
   - Handle nested quotes and escaping

2. **Parser Extensions** (`internal/parser/parser.go`, `internal/ast/ast.go`)
   - Parse quasiquote expressions
   - Extract interpolation expressions and types
   - Build QuasiquoteExpr AST node

3. **Type Checker** (`internal/types/typechecker.go`)
   - Validate interpolation types match annotations
   - Check return type matches quasiquote kind (SQL, HTML, JSON)
   - Ensure escaping functions are available

4. **Elaborator** (`internal/elaborate/elaborate.go`)
   - Transform quasiquotes to function calls
   - Insert escaping/parameterization calls
   - Generate SQL parameterized queries

5. **Runtime Support** (`internal/builtins/quasi.go` - NEW)
   - SQL parameter binding functions
   - HTML escaping (SafeText type)
   - JSON encoding helpers

### Syntax Design

**1. SQL Quasiquotes**

```ailang
-- Syntax: sql"""...${ expr: type }...""" : SQL[QueryType]
let query = sql"""
  SELECT id, name, email, created
  FROM users
  WHERE id = ${id: int}
    AND status = ${status: string}
""" : SQL[Query[User]]

-- Elaborates to (conceptually):
let query = SQL.Query[User](
  "SELECT id, name, email, created FROM users WHERE id = $1 AND status = $2",
  [id, status]  -- Parameterized (safe from injection)
)
```

**Features:**
- **Parameterization**: Interpolations become `$1`, `$2`, etc. with separate params array
- **Type checking**: `${id: int}` ensures `id` has type `int`
- **Return type**: `: SQL[Query[User]]` specifies expected result type
- **SQL injection impossible**: Parameters never concatenated into query string

**2. HTML Quasiquotes**

```ailang
-- Syntax: html"""...${ expr: SafeText }..."""
func renderUserPage(user: User) -> Html[Page] {
  html"""
    <!DOCTYPE html>
    <html>
      <head>
        <title>User: ${user.name: SafeText}</title>
      </head>
      <body>
        <div class="user-profile">
          <h1>${user.name: SafeText}</h1>
          <p>Email: ${user.email: SafeText}</p>
        </div>
      </body>
    </html>
  """
}

-- Elaborates to (conceptually):
HTML.Page([
  "<html><head><title>User: ",
  escapeHTML(user.name),
  "</title></head><body><div class=\"user-profile\"><h1>",
  escapeHTML(user.name),
  "</h1><p>Email: ",
  escapeHTML(user.email),
  "</p></div></body></html>"
])
```

**Features:**
- **Automatic escaping**: `${x: SafeText}` calls `escapeHTML(x)` automatically
- **XSS prevention**: All interpolations must be `SafeText` (forces explicit escaping)
- **Type-level safety**: `Html[Page]` return type enforces well-formed HTML
- **Multi-line support**: Triple quotes allow natural HTML formatting

**3. JSON Quasiquotes**

```ailang
-- Syntax: json{ "key": ${expr} }
let response = json{
  "id": ${user.id},
  "name": ${user.name},
  "email": ${user.email},
  "created": ${formatDate(user.created)}
}

-- Elaborates to:
JSON.Object([
  ("id", JSON.Number(user.id)),
  ("name", JSON.String(user.name)),
  ("email", JSON.String(user.email)),
  ("created", JSON.String(formatDate(user.created)))
])
```

**Features:**
- **Type inference**: Interpolations infer JSON type from AILANG type
- **Structured**: Returns `JSONValue` ADT (not just string)
- **Composable**: Can nest json{} inside json{}

### Implementation Plan

**Phase 1: Lexer & Parser** (~25 hours)
- [ ] Add quasiquote prefix tokens (`sql`, `html`, `json`)
- [ ] Lex triple-quoted strings with interpolations
- [ ] Parse `${ expr : type }` syntax
- [ ] Build QuasiquoteExpr AST node
- [ ] Unit tests for lexer/parser

**Phase 2: Type Checking** (~20 hours)
- [ ] Type check interpolation expressions
- [ ] Validate interpolation types match annotations
- [ ] Check quasiquote return types
- [ ] Unit tests for type checking

**Phase 3: SQL Quasiquotes** (~20 hours)
- [ ] Elaborate SQL quasiquotes to parameterized queries
- [ ] Generate parameter binding code
- [ ] SQL type checking (Query[T], Insert[T], etc.)
- [ ] Integration with DB effect (if exists, else stub)
- [ ] Unit + integration tests

**Phase 4: HTML Quasiquotes** (~20 hours)
- [ ] Implement SafeText type and escapeHTML builtin
- [ ] Elaborate HTML quasiquotes to escaped concatenation
- [ ] Html[Page] type checking
- [ ] XSS prevention tests (verify escaping works)
- [ ] Integration tests

**Phase 5: JSON Quasiquotes** (~10 hours)
- [ ] Elaborate JSON quasiquotes to JSONValue construction
- [ ] Type inference for interpolations
- [ ] Integration with std/json module
- [ ] Unit + integration tests

**Phase 6: Documentation & Examples** (~5 hours)
- [ ] Update web_api.ail to work correctly
- [ ] Add quasiquote examples to docs
- [ ] Update teaching prompt with quasiquote syntax
- [ ] CHANGELOG entry

### Files to Modify/Create

**New files:**
- `internal/ast/quasiquote.go` - QuasiquoteExpr AST node (~150 LOC)
- `internal/elaborate/quasiquote.go` - Quasiquote elaboration (~400 LOC)
- `internal/builtins/quasi.go` - Quasiquote runtime support (~200 LOC)
- `internal/builtins/quasi_test.go` - Tests (~300 LOC)

**Modified files:**
- `internal/lexer/token.go` - Add quasiquote tokens (~20 LOC)
- `internal/lexer/lexer.go` - Lex quasiquotes (~150 LOC)
- `internal/parser/parser.go` - Parse quasiquotes (~200 LOC)
- `internal/ast/ast.go` - Add quasiquote nodes (~50 LOC)
- `internal/types/typechecker.go` - Type check quasiquotes (~100 LOC)
- `internal/elaborate/elaborate.go` - Hook quasiquote elaboration (~50 LOC)

**Total new code: ~1,620 LOC**

## Examples

### Example 1: SQL Quasiquotes (Safe from Injection)

**Before (v0.3.x - VULNERABLE):**
```ailang
-- ❌ SQL INJECTION VULNERABLE!
func getUser(id: int) -> string ! {DB} {
  let query = "SELECT * FROM users WHERE id = " ++ show(id);
  -- If id is "1 OR 1=1", this executes: SELECT * FROM users WHERE id = 1 OR 1=1
  DB.execute(query)
}
```

**After (v0.4.2 - SAFE):**
```ailang
-- ✅ PARAMETERIZED - injection impossible
func getUser(id: int) -> Result[User, string] ! {DB} {
  let query = sql"""
    SELECT id, name, email
    FROM users
    WHERE id = ${id: int}
  """ : SQL[Query[User]];

  DB.execute(query)
  -- Executes: SELECT ... WHERE id = $1 with params=[42]
  -- Even if id is malicious, it's treated as literal value
}
```

### Example 2: HTML Templates (XSS-Safe)

**Before (v0.3.x - VULNERABLE):**
```ailang
-- ❌ XSS VULNERABLE!
func renderUser(name: string) -> string {
  "<div>" ++ name ++ "</div>"
  -- If name is "<script>alert('XSS')</script>", script executes!
}
```

**After (v0.4.2 - SAFE):**
```ailang
-- ✅ AUTO-ESCAPED - XSS impossible
func renderUser(name: string) -> Html[Div] {
  html"""<div>${name: SafeText}</div>"""
  -- name is automatically escaped: &lt;script&gt;alert('XSS')&lt;/script&gt;
}
```

### Example 3: JSON Construction

**Before (v0.3.x - Verbose):**
```ailang
-- Manual JSON construction
let response = encode(
  JSONObject([
    ("id", JSONNumber(float(user.id))),
    ("name", JSONString(user.name)),
    ("email", JSONString(user.email))
  ])
)
```

**After (v0.4.2 - Concise):**
```ailang
-- Clean JSON literal syntax
let response = encode(
  json{
    "id": ${user.id},
    "name": ${user.name},
    "email": ${user.email}
  }
)
```

## Success Criteria

- [ ] SQL quasiquotes generate parameterized queries (verified immune to injection)
- [ ] HTML quasiquotes escape all interpolations (verified immune to XSS)
- [ ] JSON quasiquotes construct valid JSONValue ADTs
- [ ] `web_api.ail` example runs correctly
- [ ] Compile-time type errors for incorrect interpolation types
- [ ] AI models can generate safe web code (M-EVAL benchmark)
- [ ] All tests passing (50+ new tests for quasiquotes)
- [ ] Documentation updated (teaching prompt, examples)
- [ ] Zero regressions in existing examples

## Testing Strategy

**Unit tests:**
- Lexer correctly tokenizes `sql"""`, `html"""`, `json{`
- Parser builds correct QuasiquoteExpr AST
- Type checker validates interpolation types
- Elaboration generates correct escaped/parameterized code

**Security tests:**
- SQL injection attempts are blocked (parameterization works)
- XSS attempts are neutralized (escaping works)
- JSON special characters are escaped properly

**Integration tests:**
- `web_api.ail` runs without errors
- SQL queries execute with parameters
- HTML renders with escaped content
- JSON encodes/decodes correctly

**M-EVAL tests:**
- Add "generate safe SQL query" benchmark
- Add "generate XSS-safe HTML" benchmark
- Measure AI success rate before/after quasiquotes

## Non-Goals

**Not in this feature:**
- **GraphQL quasiquotes** - Deferred to v0.5.0+
- **Regex quasiquotes** - Deferred to v0.5.0+
- **Custom quasiquote types** - Only SQL, HTML, JSON in v0.4.2
- **Quasiquote macros** - No user-defined quasiquotes yet
- **Streaming/lazy evaluation** - Quasiquotes fully evaluate
- **Syntax highlighting** - Editor support is out of scope

## Timeline

**Week 1** (25 hours):
- Phase 1: Lexer & Parser

**Week 2** (20 hours):
- Phase 2: Type Checking

**Week 3** (30 hours):
- Phase 3: SQL Quasiquotes
- Phase 4: HTML Quasiquotes (start)

**Week 4** (25 hours):
- Phase 4: HTML Quasiquotes (complete)
- Phase 5: JSON Quasiquotes
- Phase 6: Documentation

**Total: ~100 hours across 4 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **Lexer complexity** | High | Incremental approach - one quasiquote type at a time |
| **Escaping bugs** | High | Comprehensive security tests with known exploits |
| **SQL parameterization** | Medium | Use well-tested libraries (Go's database/sql) |
| **Type system complexity** | Medium | Keep interpolation types simple (no inference in v0.4.2) |
| **AI model adoption** | Medium | Update teaching prompt with clear examples |
| **Scope creep** | High | Strict non-goals - only SQL, HTML, JSON in v0.4.2 |

## References

- **Quasiquote designs**:
  - Scala: https://docs.scala-lang.org/overviews/quasiquotes/intro.html
  - Template Haskell: https://wiki.haskell.org/Template_Haskell
  - JavaScript template literals: https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Template_literals

- **Security references**:
  - OWASP SQL Injection: https://owasp.org/www-community/attacks/SQL_Injection
  - OWASP XSS: https://owasp.org/www-community/attacks/xss/
  - Parameterized queries: https://cheatsheetseries.owasp.org/cheatsheets/Query_Parameterization_Cheat_Sheet.html

- **Example files requiring this feature**:
  - `examples/experimental/web_api.ail`

- **Related design docs**:
  - [v0_4_0_net_enhancements.md](../v0_4_0/v0_4_0_net_enhancements.md) - HTTP/JSON foundation
  - [v0.4-roadmap.md](../v0.4-roadmap.md) - Overall v0.4 plan

## Future Work

**v0.5.0+:**
- GraphQL quasiquotes (`graphql"""query { ... }"""`)
- Regex quasiquotes (`regex"^\d{3}-\d{4}$"`)
- Custom quasiquote types (user-defined)
- Quasiquote macros (compile-time expansion)
- F# / Scala-style computation expressions
- Typed CSS quasiquotes for styling

## Decision: Defer or Implement?

**Recommendation**: **Defer to v0.5.0** unless web API support is critical for v0.4.2 launch.

**Rationale:**
- **High complexity**: 100 hours, ~1,620 LOC
- **Low priority**: Only 1 blocked example file
- **Alternative exists**: Manual string construction works (just less convenient)
- **v0.4.2 focus**: Property-based testing is higher priority for AI code gen

**If implemented in v0.4.2:**
- Start with JSON quasiquotes only (simpler, more useful)
- Defer SQL/HTML to v0.5.0
- Reduces scope to ~40 hours instead of 100

---

**Document created**: 2025-10-26
**Last updated**: 2025-10-26

---

## Website Links

**Update these when this feature is implemented:**
- [Limitations page](/docs/reference/limitations) — Remove from limitations list
- [Implementation Status](/docs/reference/implementation-status) — Update status
- Move this doc from `planned/` to `implemented/`
