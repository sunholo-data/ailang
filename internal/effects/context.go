package effects

import (
	"bufio"
	"context"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/sunholo-data/ailang/internal/eval"
	"github.com/sunholo-data/ailang/internal/trace"
)

// EffContext holds runtime capability grants and environment configuration
//
// The effect context is the central runtime structure for effect execution.
// It tracks which capabilities have been granted and provides environment
// configuration for deterministic effect execution.
//
// Thread-safety: EffContext is typically created once per evaluation and
// should not be mutated concurrently.
type EffContext struct {
	Caps           map[string]Capability // Effect name → Capability grant
	Env            EffEnv                // Environment configuration
	Clock          *ClockContext         // Clock effect state (monotonic time)
	Net            *NetContext           // Net effect configuration (security settings)
	Debug          *DebugContext         // Debug effect state (logs, assertions)
	AI             *AIContext            // AI effect state (handler for AI.call)
	DOM            *DOMContext           // DOM effect state (M-COG-RUNTIME, v0.21.x)
	Msg            *MsgContext           // Msg effect state (M-COG-RUNTIME, v0.21.x)
	Cog            *CogContext           // Subscribe/drain pending-callback queue (M-COG-RUNTIME-BROWSER M4)
	SharedMem      *SharedMemContext     // SharedMem effect state (v0.5.11 M-DX15)
	SharedIndex    *SharedIndexContext   // SharedIndex effect state (v0.5.11 M-DX16)
	Contracts      *ContractContext      // Contract effect state (M-VERIFY)
	Secret         *SecretContext        // Secret effect state (M-SECRET-EFFECT, v0.26.0)
	Stream         *StreamContext        // Stream effect state (M-STREAM-BIDI)
	Process        *ProcessContext       // Process effect state (M-PROCESS)
	Budget         *BudgetContext        // Budget tracking for effect limits (v0.7.0 M-CAPABILITY-BUDGETS)
	BudgetFrames   *BudgetFrameStack     // Per-execution stack of per-invocation budget frames (M-BUDGET-SCOPING-BUG)
	BudgetReport   *BudgetReport         // Budget usage report (--budget-report flag, M-DX25)
	DisableBudgets bool                  // Bypass budget enforcement (--no-budgets flag)
	EnvSnapshot    map[string]string     // Env effect: immutable snapshot of environment variables
	EnvAllowlist   []string              // Env effect: allowed variable names (nil = allow all)
	Args           []string              // CLI arguments passed to the program (excluding program name)
	Trace          *trace.Collector      // Semantic trace collector (--emit-trace flag, M-TRACE-EXPORT)
	IOWriter       io.Writer             // Override for IO effect output (nil = os.Stdout)
	IOReader       io.Reader             // Override for IO effect input (nil = os.Stdin)
	stdinReader    *bufio.Reader         // Persistent buffered reader for readLine (lazily initialized)

	// M-BUDGET-SCOPING-BUG: re-entrancy guard for budget charging. A single
	// logical effect op must charge the budget exactly ONCE, but each effect
	// builtin passes through TWO RequireCapWithBudget call-sites: the runtime
	// builtin wrapper (internal/runtime/builtins.go) AND the builtin's own Impl
	// (either a direct call or via effects.Call). The wrapper opens a charge
	// scope around the Impl; nested RequireCapWithBudget calls while this depth
	// is > 0 do the capability check only and skip the (already-applied) budget
	// charge. Without this the per-frame @limit would count each op twice.
	budgetChargeDepth int

	// M-EFFECT-REPLAY-CONTRACTS: mode-aware Rand dispatch state. Held behind a
	// pointer (see randModeState) so it is SHARED across WithBudget scopes (same
	// logical execution) but RESET per request in Clone (per-request isolation).
	// The mutex lives inside the pointee, so neither Clone's shallow copy nor
	// WithBudget's field-by-field rebuild ever copies a lock by value.
	randMode *randModeState
	// seedSet records whether AILANG_SEED was actually present in the environment
	// (Env.Seed defaults to 0 when unset, a valid seed value, so the value alone
	// can't distinguish "unset" from "=0"). Gates the seeded-mode source: a
	// Rand[mode=seeded] draw with seedSet==false is a typed SeededModeError.
	seedSet bool

	// M-STREAM-BIDI: Function caller for stream event handlers
	// Set by the evaluator; allows effects to call AILANG functions without import cycles.
	FnCaller func(fn eval.Value, arg eval.Value) (eval.Value, error)

	// M-ITERATIVE-LIST: Multi-arg function caller for iterative builtins (e.g., foldl callback).
	// Set by the evaluator alongside FnCaller.
	FnCallerN func(fn eval.Value, args []eval.Value) (eval.Value, error)

	// OTEL effect tracing: callback injection to avoid effects→telemetry import cycle.
	// GoCtx carries the Go context with OTEL trace propagation.
	// SpanWrapper wraps each effect operation with an OTEL span (nil = no tracing).
	GoCtx       context.Context
	SpanWrapper SpanWrapperFunc
}

// BeginBudgetChargeScope marks the start of a single logical effect op that has
// already had its budget charged (M-BUDGET-SCOPING-BUG). Nested
// RequireCapWithBudget calls until the paired EndBudgetChargeScope do a
// capability check only.
func (ctx *EffContext) BeginBudgetChargeScope() {
	ctx.budgetChargeDepth++
}

// EndBudgetChargeScope closes a scope opened by BeginBudgetChargeScope.
func (ctx *EffContext) EndBudgetChargeScope() {
	if ctx.budgetChargeDepth > 0 {
		ctx.budgetChargeDepth--
	}
}

// SaveAndResetBudgetChargeScope zeroes the charge-scope depth and returns the
// previous value (M-BUDGET-SCOPING-BUG). The evaluator calls this on entry to an
// AILANG function body so that effects performed by a user callback invoked from
// inside a builtin (e.g. a Stream event handler) are charged normally — the
// caller's charge scope must not leak across an AILANG function-call boundary.
// Paired with RestoreBudgetChargeScope on exit.
func (ctx *EffContext) SaveAndResetBudgetChargeScope() int {
	prev := ctx.budgetChargeDepth
	ctx.budgetChargeDepth = 0
	return prev
}

// RestoreBudgetChargeScope restores a depth saved by SaveAndResetBudgetChargeScope.
func (ctx *EffContext) RestoreBudgetChargeScope(prev int) {
	ctx.budgetChargeDepth = prev
}

// SpanWrapperFunc wraps an effect operation with an OTEL span.
// Called by Call() if non-nil. The wrapper starts a span, calls fn(),
// sets span attributes/status, and ends the span.
// Defined here (in effects) so telemetry can implement it without import cycles.
type SpanWrapperFunc func(
	goCtx context.Context,
	effectName, opName string,
	args []eval.Value,
	fn func() (eval.Value, error),
) (eval.Value, error)

// EffEnv provides deterministic effect execution configuration
//
// The environment holds configuration from OS environment variables
// that control effect behavior:
//   - AILANG_SEED: Seed for reproducible randomness
//   - TZ: Timezone for deterministic time operations
//   - LANG: Locale for deterministic string operations
//   - AILANG_FS_SANDBOX: Root directory for sandboxed FS operations
type EffEnv struct {
	Seed    int64  // AILANG_SEED for reproducible randomness
	TZ      string // TZ for deterministic time operations
	Locale  string // LANG for deterministic string operations
	Sandbox string // Root directory for FS operations (empty = no sandbox)
}

// ClockContext provides monotonic time for Clock effect
//
// The clock context maintains a monotonic time anchor to prevent time travel bugs
// caused by NTP adjustments, DST changes, or manual clock changes.
//
// For production (AILANG_SEED unset):
//   - now() returns: epoch + time.Since(startTime)
//   - Guarantees monotonic time (never goes backwards)
//
// For testing (AILANG_SEED set):
//   - now() returns: virtual (starts at 0)
//   - sleep() advances virtual (no real delay)
//   - Fully deterministic and reproducible
type ClockContext struct {
	startTime time.Time // Process start time (monotonic anchor)
	epoch     int64     // Unix epoch at process start (ms)
	virtual   int64     // Virtual time offset (ms, for AILANG_SEED mode)
}

// NewClockContext creates a new clock context with monotonic time anchor
//
// The clock context captures the current time at creation and uses it as
// a monotonic reference point for all future time operations.
//
// Returns:
//   - A new ClockContext with startTime and epoch initialized
func NewClockContext() *ClockContext {
	now := time.Now()
	return &ClockContext{
		startTime: now,
		epoch:     now.UnixMilli(),
		virtual:   0, // Virtual time starts at epoch 0 in AILANG_SEED mode
	}
}

// NetContext provides configuration for Net effect security
//
// The net context holds security settings for HTTP requests:
//   - Timeout enforcement (default: 30s)
//   - Body size limits (default: 5MB)
//   - Redirect limits (default: 5)
//   - Protocol allowlist (https always, http opt-in)
//   - Domain allowlist (optional)
//   - Localhost override (default: blocked)
type NetContext struct {
	Timeout        time.Duration // HTTP request timeout
	MaxBytes       int64         // Max response body size
	MaxRedirects   int           // Max HTTP redirects
	AllowHTTP      bool          // Allow http:// (default: false, https only)
	AllowLocalhost bool          // Allow localhost/127.x/::1 (default: false)
	AllowMetadata  bool          // Allow cloud metadata server at 169.254.169.254 (default: false)
	AllowedDomains []string      // Domain allowlist (empty = all allowed)
	UserAgent      string        // User-Agent header
}

// NewNetContext creates a new net context with secure defaults
//
// Default configuration:
//   - Timeout: 30 seconds
//   - MaxBytes: 5 MB
//   - MaxRedirects: 5
//   - AllowHTTP: false (https only)
//   - AllowLocalhost: false (localhost blocked)
//   - AllowedDomains: empty (all domains allowed)
//   - UserAgent: "ailang/0.3.0"
//
// Returns:
//   - A new NetContext with secure defaults
func NewNetContext() *NetContext {
	return &NetContext{
		Timeout:        30 * time.Second,
		MaxBytes:       5 * 1024 * 1024, // 5 MB
		MaxRedirects:   5,
		AllowHTTP:      false,
		AllowLocalhost: false,
		AllowedDomains: []string{},
		UserAgent:      "ailang/0.3.0", // TODO: Get version dynamically
	}
}

// NewEffContext creates a new effect context with command-line arguments
//
// The context is initialized with no capabilities granted (deny-by-default)
// and environment loaded from OS environment variables.
//
// Parameters:
//   - args: Command-line arguments passed to the program (excluding program name)
//
// Returns:
//   - A new EffContext ready to use
//
// Example:
//
//	ctx := NewEffContext([]string{"arg1", "arg2"})
//	ctx.Grant(NewCapability("IO"))
//	ctx.Grant(NewCapability("FS"))
//	ctx.Grant(NewCapability("Clock"))
//	ctx.Grant(NewCapability("Net"))
//	ctx.Grant(NewCapability("Env"))
func NewEffContext(args []string) *EffContext {
	env, seedSet := loadEffEnv()
	ctx := &EffContext{
		Caps:         make(map[string]Capability),
		Env:          env,
		seedSet:      seedSet,            // M-EFFECT-REPLAY-CONTRACTS: seeded-mode gate
		Clock:        NewClockContext(),  // Initialize monotonic time anchor
		Net:          NewNetContext(),    // Initialize secure network defaults
		Secret:       NewSecretContext(), // Initialize Secret resolver (1Password CLI)
		EnvSnapshot:  captureEnvSnapshot(),
		EnvAllowlist: nil, // nil = allow all (no restrictions by default)
		Args:         args,
		BudgetFrames: NewBudgetFrameStack(), // M-BUDGET-SCOPING-BUG: per-execution frame stack
	}
	// Debug is a ghost effect — always available, no explicit --caps needed
	ctx.Grant(NewCapability("Debug"))
	ctx.Debug = NewDebugContext()
	return ctx
}

// Grant adds a capability to the context
//
// Once granted, the capability allows execution of the corresponding
// effect operations. Granting is idempotent - granting the same
// capability twice has no additional effect.
//
// Parameters:
//   - cap: The capability to grant
//
// Example:
//
//	ctx.Grant(NewCapability("IO"))
func (ctx *EffContext) Grant(cap Capability) {
	ctx.Caps[cap.Name] = cap
}

// HasCap checks if a capability is granted
//
// Parameters:
//   - name: The capability name to check (e.g., "IO", "FS")
//
// Returns:
//   - true if the capability is granted, false otherwise
//
// Example:
//
//	if ctx.HasCap("IO") {
//	    // IO operations allowed
//	}
func (ctx *EffContext) HasCap(name string) bool {
	_, ok := ctx.Caps[name]
	return ok
}

// RequireCap checks for a capability and returns an error if missing
//
// This is the primary capability check used by effect operations.
// It provides a consistent error type (CapabilityError) when a
// capability is not granted.
//
// Parameters:
//   - name: The required capability name
//
// Returns:
//   - nil if the capability is granted
//   - CapabilityError if the capability is missing
//
// Example:
//
//	if err := ctx.RequireCap("FS"); err != nil {
//	    return nil, err
//	}
//	// FS operations allowed here
func (ctx *EffContext) RequireCap(name string) error {
	if !ctx.HasCap(name) {
		return NewCapabilityError(name)
	}
	return nil
}

// RequireCapWithBudget checks for capability and budget, consuming one budget unit
//
// This combines capability checking with budget enforcement. Use this instead of
// RequireCap when budgets are configured.
//
// Parameters:
//   - name: The required capability name
//   - position: Source position for error reporting (optional)
//
// Returns:
//   - nil if the capability is granted and budget is available
//   - CapabilityError if the capability is missing
//   - BudgetExhaustedError if the budget is exhausted
//
// Example:
//
//	if err := ctx.RequireCapWithBudget("IO", "file.ail:10:5"); err != nil {
//	    return nil, err
//	}
//	// IO operation allowed here
func (ctx *EffContext) RequireCapWithBudget(name, position string) error {
	// First check capability
	if err := ctx.RequireCap(name); err != nil {
		return err
	}

	// M-BUDGET-SCOPING-BUG: re-entrancy guard. If we're already inside a charged
	// effect op (the runtime builtin wrapper opened a scope before dispatching to
	// the Impl), do the capability check only — the budget was already charged by
	// the wrapper. This makes a single logical op charge the frame exactly once.
	if ctx.budgetChargeDepth > 0 {
		return nil
	}

	// M-BUDGET-SCOPING-BUG: enforcement lives in the per-invocation frame stack.
	// The LIMIT check is applied against every active frame via the bubbling
	// charge rule. Physical usage stays tracked on ctx.Budget for --emit-trace
	// budget deltas and the observability report.
	frameScoped := !ctx.DisableBudgets && ctx.BudgetFrames != nil && ctx.BudgetFrames.HasActiveLimit(name)

	if frameScoped {
		// Track physical only (never enforces) so trace/report totals stay
		// accurate; enforcement is done by the frame stack below.
		if ctx.Budget != nil {
			ctx.Budget.TrackPhysical(name)
		}
		if err := ctx.BudgetFrames.Charge(name, position); err != nil {
			return err
		}
	} else if !ctx.DisableBudgets && ctx.Budget != nil {
		// Legacy / direct-budget path (no active frame): CheckAndConsume tracks
		// physical AND enforces on the per-context budget. Used by SetBudget-driven
		// callers and unit tests.
		if err := ctx.Budget.CheckAndConsume(name, position); err != nil {
			return err
		}
	} else if ctx.Budget != nil {
		// --no-budgets: still track physical for the report.
		ctx.Budget.TrackPhysical(name)
	}

	// M-DX25: Record usage for budget report (always, even with --no-budgets)
	if ctx.BudgetReport != nil {
		ctx.BudgetReport.RecordUsage(name, 1)
	}

	return nil
}

// SetBudget configures the budget context
//
// Parameters:
//   - budget: The budget context to use, or nil to disable budgets
func (ctx *EffContext) SetBudget(budget *BudgetContext) {
	ctx.Budget = budget
}

// WithBudget creates a copy of the context with the specified budget
//
// This is useful for function invocations that need fresh budgets.
//
// Parameters:
//   - budget: The budget context to use
//
// Returns:
//   - A new EffContext with the specified budget
func (ctx *EffContext) WithBudget(budget *BudgetContext) *EffContext {
	// Shallow copy - share all contexts except Budget
	return &EffContext{
		Caps:           ctx.Caps,
		Env:            ctx.Env,
		Clock:          ctx.Clock,
		Net:            ctx.Net,
		Debug:          ctx.Debug,
		AI:             ctx.AI,
		DOM:            ctx.DOM, // M-COG-RUNTIME (v0.21.x): preserve DOM handler across budget scopes
		Msg:            ctx.Msg, // M-COG-RUNTIME (v0.21.x): preserve Msg handler across budget scopes
		Cog:            ctx.Cog, // M-COG-RUNTIME-BROWSER (v0.21.x M4): preserve drain queue
		SharedMem:      ctx.SharedMem,
		SharedIndex:    ctx.SharedIndex,
		Contracts:      ctx.Contracts,
		Stream:         ctx.Stream,
		Process:        ctx.Process,
		Budget:         budget,
		BudgetFrames:   ctx.BudgetFrames,   // M-BUDGET-SCOPING-BUG: SHARE frame stack across budget scopes (per-execution state)
		BudgetReport:   ctx.BudgetReport,   // Preserve report across budget scopes (M-DX25)
		DisableBudgets: ctx.DisableBudgets, // Preserve --no-budgets flag
		EnvSnapshot:    ctx.EnvSnapshot,
		EnvAllowlist:   ctx.EnvAllowlist,
		Args:           ctx.Args,
		Trace:          ctx.Trace,       // Preserve trace collector across budget scopes (M-TRACE-EXPORT)
		IOWriter:       ctx.IOWriter,    // Preserve IO writer across budget scopes
		IOReader:       ctx.IOReader,    // Preserve IO reader across budget scopes
		stdinReader:    ctx.stdinReader, // Share persistent buffered reader across scopes
		FnCaller:       ctx.FnCaller,    // Preserve function caller across budget scopes (M-STREAM-BIDI)
		FnCallerN:      ctx.FnCallerN,   // Preserve multi-arg function caller across budget scopes (M-ITERATIVE-LIST)
		GoCtx:          ctx.GoCtx,       // Preserve OTEL trace context across budget scopes
		SpanWrapper:    ctx.SpanWrapper, // Preserve OTEL span wrapper across budget scopes
		randMode:       ctx.randMode,    // M-EFFECT-REPLAY-CONTRACTS: SHARE Rand-mode state across budget scopes (same execution)
		seedSet:        ctx.seedSet,     // M-EFFECT-REPLAY-CONTRACTS: preserve AILANG_SEED presence
	}
}

// PushBudgetFrame pushes a per-invocation budget frame onto the shared frame
// stack (M-BUDGET-SCOPING-BUG). Implements eval.BudgetFrameEnforcer.
//
// Called by the evaluator on entry to any function whose signature carries an
// @limit/@min annotation. The frame stack is shared across WithBudget scopes so
// ancestor frames stay active during the callee's dynamic extent (bubbling).
func (ctx *EffContext) PushBudgetFrame(fnName string, limits, mins map[string]int) {
	if ctx.BudgetFrames == nil {
		ctx.BudgetFrames = NewBudgetFrameStack()
	}
	ctx.BudgetFrames.Push(fnName, limits, mins)
}

// PopBudgetFrame pops the innermost budget frame (M-BUDGET-SCOPING-BUG).
// Implements eval.BudgetFrameEnforcer.
//
// On normal exit (bodyErr == nil) the frame's @min requirements are checked
// against the frame's own bubbled semantic count; a violation is returned as a
// *BudgetUnderrunError. On error/exceptional exit (bodyErr != nil) the frame is
// still popped but the @min check is SUPPRESSED and bodyErr is returned unchanged
// (a mid-flight failure must not be masked by a @min violation).
func (ctx *EffContext) PopBudgetFrame(fnName string, bodyErr error) error {
	if ctx.BudgetFrames == nil {
		return bodyErr
	}
	frame := ctx.BudgetFrames.Pop()
	if bodyErr != nil {
		// Error exit: suppress @min, propagate the original error unchanged.
		return bodyErr
	}
	if frame == nil {
		return nil
	}
	return frame.CheckMin("")
}

// loadEffEnv loads effect environment from OS environment variables
//
// Environment variables:
//   - AILANG_SEED: Integer seed for deterministic randomness (default: 0)
//   - TZ: Timezone string (default: "UTC")
//   - LANG: Locale string (default: "C")
//   - AILANG_FS_SANDBOX: Filesystem sandbox root (default: "" = no sandbox)
//
// Returns:
//   - Populated EffEnv with values from environment
func loadEffEnv() (EffEnv, bool) {
	seed := int64(0)
	seedSet := false
	if seedStr := os.Getenv("AILANG_SEED"); seedStr != "" {
		if s, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			seed = s
			seedSet = true // M-EFFECT-REPLAY-CONTRACTS: AILANG_SEED present → seeded mode may draw
		}
	}

	return EffEnv{
		Seed:    seed,
		TZ:      getEnv("TZ", "UTC"),
		Locale:  getEnv("LANG", "C"),
		Sandbox: os.Getenv("AILANG_FS_SANDBOX"),
	}, seedSet
}

// getEnv gets an environment variable with a default fallback
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// captureEnvSnapshot creates an immutable snapshot of environment variables
//
// Captures all environment variables from os.Environ() into a map.
// This snapshot is frozen at context creation time and never updated,
// ensuring deterministic behavior even if the OS environment changes.
//
// Returns:
//   - Map of environment variable name → value
//
// Example:
//
//	snapshot := captureEnvSnapshot()
//	// snapshot["PATH"] = "/usr/bin:/bin"
//	// snapshot["HOME"] = "/Users/mark"
func captureEnvSnapshot() map[string]string {
	snapshot := make(map[string]string)
	for _, pair := range os.Environ() {
		// Split "KEY=VALUE" into key and value
		// Handle edge case: "KEY=" (empty value) vs "KEY" (no equals)
		for i := 0; i < len(pair); i++ {
			if pair[i] == '=' {
				key := pair[:i]
				value := pair[i+1:]
				snapshot[key] = value
				break
			}
		}
	}
	return snapshot
}

// M-VERIFY-CONTRACTS: ContractChecker interface implementation for EffContext

// IsContractCheckingEnabled returns true if contract checking is enabled
//
// M-VERIFY-CONTRACTS: Contracts are enabled when a ContractContext exists
// and its Mode is not ContractModeOff.
func (ctx *EffContext) IsContractCheckingEnabled() bool {
	return ctx.Contracts != nil && ctx.Contracts.Mode != ContractModeOff
}

// CheckRequires delegates precondition checking to ContractContext
//
// M-VERIFY-CONTRACTS: Called by the evaluator when checking function preconditions.
// If no ContractContext is configured, this is a no-op.
//
// Parameters:
//   - cond: The boolean result of evaluating the precondition
//   - msg: User-provided message or auto-generated predicate string
//   - location: Source location "file.ail:42"
//
// Returns:
//   - nil if check passes or contracts disabled
//   - Error in Panic mode if check fails
func (ctx *EffContext) CheckRequires(cond bool, msg, location string) error {
	if ctx.Contracts == nil {
		return nil
	}
	// M-TRACE-EXPORT: Record contract check in semantic trace
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordContractCheck("requires", cond, msg, location, ctx.Contracts.CurrentFunction())
	}
	return ctx.Contracts.CheckRequires(cond, msg, location)
}

// CheckEnsures delegates postcondition checking to ContractContext
//
// M-VERIFY-CONTRACTS: Called by the evaluator when checking function postconditions.
// If no ContractContext is configured, this is a no-op.
//
// Parameters:
//   - cond: The boolean result of evaluating the postcondition
//   - msg: User-provided message or auto-generated predicate string
//   - location: Source location "file.ail:42"
//
// Returns:
//   - nil if check passes or contracts disabled
//   - Error in Panic mode if check fails
func (ctx *EffContext) CheckEnsures(cond bool, msg, location string) error {
	if ctx.Contracts == nil {
		return nil
	}
	// M-TRACE-EXPORT: Record contract check in semantic trace
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordContractCheck("ensures", cond, msg, location, ctx.Contracts.CurrentFunction())
	}
	return ctx.Contracts.CheckEnsures(cond, msg, location)
}

// M-TRACE-EXPORT: Trace delegate methods

// HasTraceCollector returns true if semantic trace collection is active.
// GetIOWriter returns the writer for IO effect output.
// Returns os.Stdout unless overridden (e.g., when --emit-trace redirects program output to stderr).
func (ctx *EffContext) GetIOWriter() io.Writer {
	if ctx.IOWriter != nil {
		return ctx.IOWriter
	}
	return os.Stdout
}

// FlushIO flushes any buffered IO output to the underlying writer. It is a no-op
// for unbuffered writers (e.g. piped stdout, or os.Stdout directly). Used by the
// IO.flush() effect so partial-line terminal output (progress bars, game frames)
// reaches the screen without waiting for a newline.
func (ctx *EffContext) FlushIO() error {
	if f, ok := ctx.GetIOWriter().(interface{ Flush() error }); ok {
		return f.Flush()
	}
	return nil
}

// GetIOReader returns a persistent buffered reader for IO effect input.
// Returns a bufio.Reader wrapping IOReader (or os.Stdin if not overridden).
// The reader is lazily initialized and reused across calls, so buffered data
// is preserved between readLine() invocations.
func (ctx *EffContext) GetIOReader() *bufio.Reader {
	if ctx.stdinReader == nil {
		src := ctx.IOReader
		if src == nil {
			src = os.Stdin
		}
		ctx.stdinReader = bufio.NewReader(src)
	}
	return ctx.stdinReader
}

// Clone creates a shallow copy of the EffContext for per-request isolation.
// Config fields (Caps, Env, Clock, Net, etc.) are shared by reference.
// FnCaller/FnCallerN will be re-wired by the forked evaluator.
//
// M-EFFECT-REPLAY-CONTRACTS: the Rand-mode dispatch state (randMode) is
// per-request runtime state and is NOT shared across requests — a shallow *ctx
// copy would alias the mode stack + seeded source between concurrent requests.
// We nil it so the clone lazily allocates a fresh one. seedSet and Env.Seed
// (config) are preserved so a cloned request still honours AILANG_SEED.
func (ctx *EffContext) Clone() interface{} {
	clone := *ctx // shallow copy of config + shared references
	clone.randMode = nil
	return &clone
}

// SetFnCaller sets the single-arg function caller callback.
// M-ITERATIVE-LIST: Used by embed.Engine to wire callbacks without importing effects.
func (ctx *EffContext) SetFnCaller(fn func(eval.Value, eval.Value) (eval.Value, error)) {
	ctx.FnCaller = fn
}

// SetFnCallerN sets the multi-arg function caller callback.
// M-ITERATIVE-LIST: Used by embed.Engine to wire callbacks without importing effects.
func (ctx *EffContext) SetFnCallerN(fn func(eval.Value, []eval.Value) (eval.Value, error)) {
	ctx.FnCallerN = fn
}

func (ctx *EffContext) HasTraceCollector() bool {
	return ctx.Trace != nil && ctx.Trace.Enabled()
}

// RecordFunctionEnter delegates to trace collector if present.
func (ctx *EffContext) RecordFunctionEnter(name string, args []string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordFunctionEnter(name, args)
	}
}

// RecordFunctionExit delegates to trace collector if present.
func (ctx *EffContext) RecordFunctionExit(name string, result string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordFunctionExit(name, result)
	}
}

// RecordEffect delegates to trace collector if present.
func (ctx *EffContext) RecordEffect(effectName, opName string, args []string, result string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordEffect(effectName, opName, args, result)
	}
}

// RecordModedEffect delegates to the trace collector, attaching a
// parameterised-effect mode and its replay-contract label
// (M-EFFECT-REPLAY-CONTRACTS). No-op when no trace collector is active.
func (ctx *EffContext) RecordModedEffect(effectName, opName string, args []string, result, mode, contract string) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordModedEffect(effectName, opName, args, result, mode, contract)
	}
}

// RecordAIEffect delegates to the trace collector with optional routing metadata.
// Effect name is fixed to "AI". Route may be nil for non-routed AI calls.
func (ctx *EffContext) RecordAIEffect(opName string, args []string, result string, route *trace.ResolvedRoute) {
	if ctx.Trace != nil && ctx.Trace.Enabled() {
		ctx.Trace.RecordAIEffect(opName, args, result, route)
	}
}
