package effects

import (
	"bufio"
	"context"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/trace"
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
	SharedMem      *SharedMemContext     // SharedMem effect state (v0.5.11 M-DX15)
	SharedIndex    *SharedIndexContext   // SharedIndex effect state (v0.5.11 M-DX16)
	Contracts      *ContractContext      // Contract effect state (M-VERIFY)
	Stream         *StreamContext        // Stream effect state (M-STREAM-BIDI)
	Process        *ProcessContext       // Process effect state (M-PROCESS)
	Budget         *BudgetContext        // Budget tracking for effect limits (v0.7.0 M-CAPABILITY-BUDGETS)
	BudgetReport   *BudgetReport         // Budget usage report (--budget-report flag, M-DX25)
	DisableBudgets bool                  // Bypass budget enforcement (--no-budgets flag)
	EnvSnapshot    map[string]string     // Env effect: immutable snapshot of environment variables
	EnvAllowlist   []string              // Env effect: allowed variable names (nil = allow all)
	Args           []string              // CLI arguments passed to the program (excluding program name)
	Trace          *trace.Collector      // Semantic trace collector (--emit-trace flag, M-TRACE-EXPORT)
	IOWriter       io.Writer             // Override for IO effect output (nil = os.Stdout)
	IOReader       io.Reader             // Override for IO effect input (nil = os.Stdin)
	stdinReader    *bufio.Reader         // Persistent buffered reader for readLine (lazily initialized)

	// M-DX25: Scoped budget charging
	DeclaredBudgets map[string]int // Callee's declared @limit values (for charging caller on return)
	CallerContext   *EffContext    // Reference to caller's context (for charging on scope exit)

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
	return &EffContext{
		Caps:         make(map[string]Capability),
		Env:          loadEffEnv(),
		Clock:        NewClockContext(), // Initialize monotonic time anchor
		Net:          NewNetContext(),   // Initialize secure network defaults
		EnvSnapshot:  captureEnvSnapshot(),
		EnvAllowlist: nil, // nil = allow all (no restrictions by default)
		Args:         args,
	}
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

	// Then check and consume budget (if configured)
	// Skip budget check if DisableBudgets is set (--no-budgets flag)
	if ctx.Budget != nil && !ctx.DisableBudgets {
		if err := ctx.Budget.CheckAndConsume(name, position); err != nil {
			return err
		}
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
		Caps:            ctx.Caps,
		Env:             ctx.Env,
		Clock:           ctx.Clock,
		Net:             ctx.Net,
		Debug:           ctx.Debug,
		AI:              ctx.AI,
		SharedMem:       ctx.SharedMem,
		SharedIndex:     ctx.SharedIndex,
		Contracts:       ctx.Contracts,
		Stream:          ctx.Stream,
		Process:         ctx.Process,
		Budget:          budget,
		BudgetReport:    ctx.BudgetReport,   // Preserve report across budget scopes (M-DX25)
		DisableBudgets:  ctx.DisableBudgets, // Preserve --no-budgets flag
		EnvSnapshot:     ctx.EnvSnapshot,
		EnvAllowlist:    ctx.EnvAllowlist,
		Args:            ctx.Args,
		Trace:           ctx.Trace,       // Preserve trace collector across budget scopes (M-TRACE-EXPORT)
		IOWriter:        ctx.IOWriter,    // Preserve IO writer across budget scopes
		IOReader:        ctx.IOReader,    // Preserve IO reader across budget scopes
		stdinReader:     ctx.stdinReader, // Share persistent buffered reader across scopes
		DeclaredBudgets: nil,             // Reset for new scope (will be set by WithBudgetLimits)
		CallerContext:   nil,             // Reset for new scope (will be set by WithBudgetLimits)
		FnCaller:        ctx.FnCaller,    // Preserve function caller across budget scopes (M-STREAM-BIDI)
		FnCallerN:       ctx.FnCallerN,   // Preserve multi-arg function caller across budget scopes (M-ITERATIVE-LIST)
		GoCtx:           ctx.GoCtx,       // Preserve OTEL trace context across budget scopes
		SpanWrapper:     ctx.SpanWrapper, // Preserve OTEL span wrapper across budget scopes
	}
}

// WithBudgetLimits creates a new context with budget limits from a map[string]int
// This implements the eval.BudgetEnforcer interface to avoid import cycles.
//
// M-DX25: This creates a scoped budget context that tracks:
// - DeclaredBudgets: the callee's declared limits (for charging caller on return)
// - CallerContext: reference to caller for semantic charging on scope exit
//
// Parameters:
//   - limits: Map of effect name to budget limit (e.g., {"IO": 5, "Rand": 10})
//
// Returns:
//   - A new EffContext with the specified budget limits (as interface{})
func (ctx *EffContext) WithBudgetLimits(limits map[string]int) interface{} {
	// Convert map[string]int to map[string]*int for NewBudgetContext
	ptrLimits := make(map[string]*int, len(limits))
	for effect, limit := range limits {
		l := limit // capture for pointer
		ptrLimits[effect] = &l
	}
	budget := NewBudgetContext(ptrLimits)
	newCtx := ctx.WithBudget(budget)

	// M-DX25: Store declared budgets and caller reference for scoped charging
	newCtx.DeclaredBudgets = make(map[string]int, len(limits))
	for effect, limit := range limits {
		newCtx.DeclaredBudgets[effect] = limit
	}
	newCtx.CallerContext = ctx

	return newCtx
}

// PopScopeAndChargeCaller charges the caller context with declared semantic budgets
//
// M-DX25: When a scoped function returns, the caller is charged the callee's
// declared budget (not the actual physical usage). This implements the
// "charge declared amount" semantic charging model.
//
// This method should be called when restoring the old effect context after
// evaluating a function with declared budgets.
//
// If this context has no CallerContext (pass-through mode), this is a no-op.
func (ctx *EffContext) PopScopeAndChargeCaller() {
	if ctx.CallerContext == nil || len(ctx.DeclaredBudgets) == 0 {
		return
	}

	// Charge caller's semantic budget with declared amounts
	caller := ctx.CallerContext
	if caller.Budget != nil {
		for effect, declared := range ctx.DeclaredBudgets {
			// Increment caller's semantic usage by declared amount
			caller.Budget.ChargeSemanticOnly(effect, declared)
		}
	}

	// Also record in budget report if active
	if caller.BudgetReport != nil {
		for effect, declared := range ctx.DeclaredBudgets {
			// Record semantic charge to caller (not physical - that's tracked separately)
			// For now we record as function attribution when available
			if caller.BudgetReport.CurrentFunction != "" {
				if caller.BudgetReport.FunctionUsage[caller.BudgetReport.CurrentFunction] == nil {
					caller.BudgetReport.FunctionUsage[caller.BudgetReport.CurrentFunction] = make(map[string]int)
				}
				caller.BudgetReport.FunctionUsage[caller.BudgetReport.CurrentFunction][effect] += declared
				caller.BudgetReport.TotalUsage[effect] += declared
			}
		}
	}
}

// SetMinBudgets sets minimum usage requirements on the context's budget
//
// M-DX25 M4: Implements eval.MinBudgetEnforcer interface.
// Called by evaluator after WithBudgetLimits to set minimum constraints.
//
// Parameters:
//   - minLimits: Map of effect name to minimum required usage
func (ctx *EffContext) SetMinBudgets(minLimits map[string]int) {
	if ctx.Budget == nil || len(minLimits) == 0 {
		return
	}
	for effect, min := range minLimits {
		ctx.Budget.minLimits[effect] = min
	}
}

// CheckMinimums verifies all minimum requirements are met
//
// M-DX25 M4: Implements eval.MinimumChecker interface.
// Called by evaluator on scope exit to ensure effects were exercised.
//
// Parameters:
//   - position: Source position for error reporting
//
// Returns:
//   - nil if all minimums satisfied
//   - BudgetUnderrunError if any minimum is not met
func (ctx *EffContext) CheckMinimums(position string) error {
	if ctx.Budget == nil {
		return nil
	}
	return ctx.Budget.CheckMinimum(position)
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
func loadEffEnv() EffEnv {
	seed := int64(0)
	if seedStr := os.Getenv("AILANG_SEED"); seedStr != "" {
		if s, err := strconv.ParseInt(seedStr, 10, 64); err == nil {
			seed = s
		}
	}

	return EffEnv{
		Seed:    seed,
		TZ:      getEnv("TZ", "UTC"),
		Locale:  getEnv("LANG", "C"),
		Sandbox: os.Getenv("AILANG_FS_SANDBOX"),
	}
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
