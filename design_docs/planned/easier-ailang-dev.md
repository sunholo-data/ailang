Implementation Friction Log: v0.3.8 HTTP + JSON Feature
Overview
Adding HTTP headers + JSON encoding required touching 8+ files across 4 subsystems. Here are the pain points and what would help.
1. Builtin Registration is Scattered Across 4 Places
The Problem
To add a single builtin like _net_httpRequest, I had to update:
internal/effects/net.go - Runtime implementation (300 LOC)
internal/runtime/builtins.go - Runtime registration wrapper
br.builtins["_net_httpRequest"] = &eval.BuiltinFunction{...}
internal/builtins/registry.go - Metadata for arity checking
Registry["_net_httpRequest"] = &BuiltinMeta{Name: "_net_httpRequest", NumArgs: 4, IsPure: false}
internal/link/builtin_module.go - Two places:
Type signature (lines 310-344)
Export list (line 385)
effectBuiltins := []string{..., "_net_httpRequest"}
Time Cost
2+ hours debugging "undefined global variable from $builtin"
Root cause: Forgot to add to hardcoded export list (step 4)
What Would Help
Option A: Convention-based auto-discovery
// In internal/effects/net.go
//go:builtin _net_httpRequest(method: String, url: String, headers: List[...], body: String) -> Result[...] ! {Net}
func netHTTPRequest(ctx *EffContext, args []eval.Value) (eval.Value, error) {
    // Compiler extracts type signature from comment
    // Auto-registers in all 4 places
}
Option B: Single registration point
// internal/builtins/registry.go
func init() {
    RegisterEffectBuiltin("_net_httpRequest", BuiltinSpec{
        Module: "Net",
        OpName: "httpRequest",
        NumArgs: 4,
        Type: func() types.Type {
            return makeHttpRequestType() // Type DSL
        },
        Impl: effects.netHTTPRequest, // Direct reference
    })
}
Benefit: Add builtin in ONE place instead of four.
2. Type System Construction is Verbose and Fragile
The Problem
Building the type for Result[HttpResponse, NetError] required:
// 35 lines of nested struct construction
headerRecordType := &types.TRecord{
    Fields: map[string]types.Type{
        "name":  strType,
        "value": strType,
    },
}
headerListType := &types.TList{Element: headerRecordType}
httpResponseType := &types.TRecord{
    Fields: map[string]types.Type{
        "status":  &types.TCon{Name: "Int"},
        "headers": headerListType,
        "body":    strType,
        "ok":      &types.TCon{Name: "Bool"},
    },
}
netErrorType := &types.TCon{Name: "NetError"}
resultAppType := &types.TApp{
    Constructor: &types.TCon{Name: "Result"},
    Args:        []types.Type{httpResponseType, netErrorType},
}
Errors I made:
Used TVar instead of TApp (wrong)
Used Elem instead of Element (compile error)
Used []RecordField instead of map[string]Type (compile error)
Time Cost
30 minutes trial-and-error with type constructors
Had to read internal/types/types.go multiple times
What Would Help
Option A: Type DSL
typeSpec := ParseType(`
    String -> String -> List[{name: String, value: String}] -> String 
    -> Result[{status: Int, headers: List[...], body: String, ok: Bool}, NetError] 
    ! {Net}
`)
Option B: Builder API
T := types.Builder()
T.Func(
    T.String(), T.String(), 
    T.List(T.Record("name", T.String(), "value", T.String())),
    T.String(),
).Returns(
    T.App("Result", 
        T.Record("status", T.Int(), "headers", ..., "body", T.String(), "ok", T.Bool()),
        T.Con("NetError")
    )
).Effects("Net")
Benefit: Readable, type-safe, catches errors at compile time.
3. No Feedback When Builtin Registration Fails
The Problem
When I forgot to add _net_httpRequest to the export list:
Error: undefined global variable: _net_httpRequest from $builtin
What this error doesn't tell me:
WHERE to add it (which file?)
THAT it's a registration issue (sounds like a typo)
HOW to check what's registered (ailang builtins list?)
Time Cost
1 hour grepping for "undefined global variable" and tracing error through type checker
What Would Help
Better error message:
Error: Builtin '_net_httpRequest' not found in $builtin module.

Hint: Did you forget to add it to internal/link/builtin_module.go:GetBuiltinInterface()?

Registered Net builtins:
  - _net_httpGet
  - _net_httpPost

To see all builtins: ailang builtins list
Validation on startup:
$ ailang doctor builtins
✓ 45 builtins registered in eval.Builtins
✓ 45 builtins have type signatures
✗ Missing from GetBuiltinInterface(): _net_httpRequest
  → Add to internal/link/builtin_module.go:385
4. AILANG Syntax Surprises
The Problem
Several syntax issues in examples/ai_call.ail: Issue 1: Can't call functions in if conditions
-- ❌ FAILS: "function arity mismatch: 1 vs 2"
if eq_String(apiKey, "<YOUR_API_KEY>") then ...

-- ✅ WORKS: Extract to variable
let isPlaceholder = eq_String(apiKey, "<YOUR_API_KEY>");
if isPlaceholder then ...
Issue 2: Nullary constructors need special syntax
-- ❌ FAILS: Parse error
let null = jn();

-- ✅ WORKS: Direct constructor
let null = JNull;
Issue 3: Block expressions need explicit ()
-- ❌ FAILS: Type error
if x then { println("hi") } else { println("bye") }

-- ✅ WORKS: Explicit unit
if x then { println("hi"); () } else { println("bye"); () }
Time Cost
20 minutes debugging "arity mismatch" errors that were actually parser issues
What Would Help
Option A: Better error messages
Error: Cannot call function 'eq_String' directly in if-condition (parser limitation)

Workaround: Extract to a let-binding first:
  let cond = eq_String(a, b);
  if cond then ...
Option B: Fix the parser
Allow function calls in if-conditions
Infer unit () at end of blocks automatically
Option C: Teach prompt Update prompts/v0.3.8.md with:
## Common Pitfalls

1. **Functions in if-conditions**: Extract to variable first
2. **Nullary constructors**: Use `Constructor` not `constructor()`
3. **Block return values**: Add `()` at end if returning unit
5. Testing Workflow is Manual
The Problem
To test the implementation, I had to:
# 1. Rebuild (every time)
make quick-install

# 2. Write test file
vim test_json.ail

# 3. Run manually
ailang run --caps IO --entry main test_json.ail

# 4. Check output by eye
# 5. Delete test file
rm test_json.ail
No way to:
Unit test builtins in isolation
Verify type signatures without full compile
Test with mock HTTP responses (hermetic)
Time Cost
45 minutes of rebuild-test-check cycles
What Would Help
Option A: Builtin test harness
// internal/effects/net_test.go
func TestNetHTTPRequest(t *testing.T) {
    ctx := NewMockEffContext(NetCap)
    
    result, err := netHTTPRequest(ctx, []eval.Value{
        StringVal("POST"),
        StringVal("https://example.com"),
        ListVal(RecordVal("name", "Auth", "value", "Bearer X")),
        StringVal("{}"),
    })
    
    assert.IsOk(result) // Result[HttpResponse, NetError]
    resp := result.(*TaggedValue).Fields[0].(*RecordValue)
    assert.Equal(200, resp.Fields["status"])
}
Option B: AILANG test syntax
-- test_json.ail
test "encode null" {
  assert_eq(encode(JNull), "null")
}

test "encode object" {
  let obj = jo([kv("a", js("x"))]);
  assert_eq(encode(obj), "{\"a\":\"x\"}")
}
Run: ailang test test_json.ail
6. Documentation is Scattered
The Problem
To understand how to add a builtin, I had to read:
CLAUDE.md - High-level structure
internal/effects/net.go - Example implementation
internal/eval/builtins.go - Pure builtins pattern
internal/runtime/builtins.go - Effect builtins pattern
internal/link/builtin_module.go - Type registration
internal/builtins/registry.go - Metadata
No single document explaining the full flow.
What Would Help
Add: docs/ADDING_BUILTINS.md
# Adding a Builtin Function

## Quick Start (5 steps)

1. **Implement** in `internal/effects/my_effect.go`:
   ```go
   func myEffectOp(ctx *EffContext, args []eval.Value) (eval.Value, error) {
       // Implementation
   }
Register runtime in internal/runtime/builtins.go:
br.builtins["_my_op"] = &eval.BuiltinFunction{
    Name: "_my_op",
    Fn: func(args []eval.Value) (eval.Value, error) {
        return effects.Call(ctx, "MyEffect", "op", args)
    },
}
Add metadata in internal/builtins/registry.go:
Registry["_my_op"] = &BuiltinMeta{Name: "_my_op", NumArgs: 2, IsPure: false}
Add type signature in internal/link/builtin_module.go (line ~350)
Export in internal/link/builtin_module.go:GetBuiltinInterface() (line ~385)
Checklist
 Step 1-5 complete
 Tests in internal/effects/my_effect_test.go
 Example in examples/my_feature.ail
 Update stdlib/std/my_effect.ail if needed
 Run: make test && make verify-examples

---

## 7. **No Incremental Type Checking**

### The Problem
When I got:
Error: type unification failed at [function application at ai_call.ail:58:33]: function arity mismatch: 2 vs 3

**I couldn't:**
- Check just line 58 in isolation
- Ask "what is the type of this expression?"
- Get intermediate type inference steps

Had to binary-search comment out code until error went away.

### What Would Help

**REPL type checking:**
```ailang
λ> :type concat_String
concat_String : String -> String -> String

λ> :type concat_String("a", "b")
String

λ> :type let x = concat_String in x
Error: Cannot infer type - function needs 2 arguments, got 0

λ> :explain
concat_String has type String -> String -> String
You're trying to use it as a value without applying it to arguments
Or:
$ ailang check --explain examples/ai_call.ail:58
Line 58: _str_slice(resp.body, 0, 200)
         ^^^^^^^^^^^ builtin: String -> Int -> Int -> String
                     ^^^^^^^^^ String ✓
                                  ^ Int ✓
                                     ^^^ Int ✓
Type: String ✓

No errors on line 58
Summary: What AILANG Needs
High Priority (Biggest Impact)
Single-file builtin registration (saves 2+ hours per builtin)
Convention-based or centralized registry
Validation tool: ailang doctor builtins
Type DSL or builder API (saves 1+ hour per complex type)
Either parse type string or fluent API
Catches errors at compile time
Better error messages (saves 1+ hour per error)
Show WHERE to fix (file:line)
Show HOW to fix (example code)
Validate at startup
Medium Priority
Builtin testing harness (speeds iteration 3x)
Mock EffContext
Assert helpers for ADTs
Run without full compile
docs/ADDING_BUILTINS.md (onboarding speedup)
Step-by-step guide
Checklist format
Links to examples
REPL type queries (debugging speedup)
:type expr command
:explain for errors
Partial expression checking
Low Priority (Nice to Have)
Fix parser issues
Functions in if-conditions
Auto-infer () at block end
Better function call syntax
Teaching prompt updates
Document common pitfalls
Add "Gotchas" section
Link from error messages
Metrics
Time Spent:
Implementation: 4 hours (net.go, builtins.go, type signatures)
Debugging registration: 2 hours (scattered registration)
Debugging types: 1 hour (TApp, TRecord confusion)
Debugging syntax: 0.5 hours (if-conditions, blocks)
Total: 7.5 hours
With improvements above:
Implementation: 2 hours (single registration point, type DSL)
Debugging: 0.5 hours (better errors, validation)
Estimated: 2.5 hours (70% faster)
Recommendation
Start with #1-3 (builtin registration, type DSL, error messages) - they have the highest ROI and unblock future feature development. The others can be added incrementally as the language matures.