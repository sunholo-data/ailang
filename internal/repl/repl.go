package repl

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/peterh/liner"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/link"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/schema"
	"github.com/sunholo/ailang/internal/test"
	"github.com/sunholo/ailang/internal/typedast"
	"github.com/sunholo/ailang/internal/types"
)


// Color functions for pretty output
var (
	green  = color.New(color.FgGreen).SprintFunc()
	red    = color.New(color.FgRed).SprintFunc()
	yellow = color.New(color.FgYellow).SprintFunc()
	cyan   = color.New(color.FgCyan).SprintFunc()
	bold   = color.New(color.Bold).SprintFunc()
	dim    = color.New(color.Faint).SprintFunc()
)

// Config holds REPL configuration
type Config struct {
	TraceDefaulting bool
	ShowCore        bool
	ShowTyped       bool
	DryLink         bool
	Verbose         bool
	ImportedModules []string
}

// REPL represents the Read-Eval-Print Loop
type REPL struct {
	config     *Config
	env        *eval.Environment
	typeEnv    *types.TypeEnv
	instEnv    *types.InstanceEnv // Type-level instances and defaults
	dictReg    *types.DictionaryRegistry
	instances  map[string]core.DictValue
	history    []string
	lastResult interface{}
	version    string // Version info from build
	buildTime  string // Build time from build
}

// New creates a new REPL instance
func New() *REPL {
	return NewWithVersion("", "")
}

// NewWithVersion creates a new REPL with version info
func NewWithVersion(version, buildTime string) *REPL {
	if version == "" {
		version = "dev"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}
	return &REPL{
		config:    &Config{},
		env:       eval.NewEnvironment(),
		typeEnv:   types.NewTypeEnv(),
		instEnv:   types.NewInstanceEnv(),
		dictReg:   types.NewDictionaryRegistry(),
		instances: make(map[string]core.DictValue),
		history:   []string{},
		version:   version,
		buildTime: buildTime,
	}
}

// EnableTrace enables execution tracing
func (r *REPL) EnableTrace() {
	r.config.Verbose = true
}

// Start begins the REPL session
func (r *REPL) Start(in io.Reader, out io.Writer) {
	// Create liner instance for readline functionality
	line := liner.NewLiner()
	defer line.Close()
	
	// Set up history file
	historyFile := filepath.Join(os.TempDir(), ".ailang_history")
	if f, err := os.Open(historyFile); err == nil {
		line.ReadHistory(f)
		f.Close()
	}
	
	// Enable multiline mode
	line.SetMultiLineMode(true)
	
	// Print welcome message with dynamic version
	versionStr := r.version
	if versionStr == "" || versionStr == "dev" {
		versionStr = "dev"
	} else {
		// Add build time if available and not unknown
		if r.buildTime != "" && r.buildTime != "unknown" {
			// Parse and format build time nicely
			if t, err := time.Parse("2006-01-02_15:04:05", r.buildTime); err == nil {
				versionStr = fmt.Sprintf("%s - %s", versionStr, t.Format("2006-01-02"))
			}
		}
	}
	fmt.Fprintf(out, "%s %s\n", bold("AILANG"), bold(versionStr))
	fmt.Fprintln(out, dim("Type :help for help, :quit to exit"))
	fmt.Fprintln(out, dim("Use ↑/↓ arrows to navigate history"))
	fmt.Fprintln(out)

	// Initialize built-in instances
	r.initBuiltins()
	
	// Auto-import prelude for convenience
	r.importModule("std/prelude", io.Discard)

	// Add command completion
	line.SetCompleter(func(line string) (c []string) {
		if strings.HasPrefix(line, ":") {
			commands := []string{":help", ":quit", ":type", ":import", ":dump-core", 
				":dump-typed", ":dry-link", ":trace-defaulting", ":instances", 
				":history", ":clear", ":reset"}
			for _, cmd := range commands {
				if strings.HasPrefix(cmd, line) {
					c = append(c, cmd)
				}
			}
		}
		return
	})

	for {
		// Use liner to get input with history support
		// Note: liner doesn't support ANSI colors in the prompt
		input, err := line.Prompt("λ> ")
		if err == io.EOF {
			fmt.Fprintln(out, green("\nGoodbye!"))
			break
		}
		if err != nil {
			fmt.Fprintf(out, "%s: %v\n", red("Error"), err)
			continue
		}

		input = strings.TrimSpace(input)

		if input == "" {
			continue
		}

		// Check if input needs continuation (ends with "in" or other indicators)
		needsContinuation := strings.HasSuffix(input, " in") || strings.HasSuffix(input, "\tin")
		
		// Multi-line input support
		if needsContinuation {
			// Continue reading lines until we get a complete expression
			var lines []string
			lines = append(lines, input)
			
			for {
				contInput, err := line.Prompt("... ")
				if err == io.EOF {
					fmt.Fprintln(out, red("\nIncomplete expression"))
					break
				}
				if err != nil {
					fmt.Fprintf(out, "%s: %v\n", red("Error"), err)
					break
				}
				
				lines = append(lines, contInput)
				
				// Check if we have a complete expression
				// For now, just check if the line is non-empty and doesn't end with certain keywords
				trimmed := strings.TrimSpace(contInput)
				if trimmed != "" && !strings.HasSuffix(trimmed, " in") && !strings.HasSuffix(trimmed, ",") {
					break
				}
			}
			
			input = strings.Join(lines, "\n")
		}

		// Add to liner history
		line.AppendHistory(input)
		
		// Add to our internal history
		r.history = append(r.history, input)

		// Handle commands
		if strings.HasPrefix(input, ":") {
			// Check if it's a quit command
			if strings.HasPrefix(input, ":quit") || strings.HasPrefix(input, ":q") || strings.HasPrefix(input, ":exit") {
				fmt.Fprintln(out, green("Goodbye!"))
				break // Exit the loop
			}
			r.handleCommand(input, out)
			continue
		}

		// Process expression through full pipeline
		r.processExpression(input, out)
	}
	
	// Save history before exiting
	if f, err := os.Create(historyFile); err == nil {
		line.WriteHistory(f)
		f.Close()
	}
}

// initBuiltins initializes built-in type class instances
func (r *REPL) initBuiltins() {
	// Wrapper functions to convert Go functions to uniform eval signatures
	wrapInt2 := func(f func(int64, int64) int64) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.IntValue{Value: int(f(int64(x.Value), int64(y.Value)))}, nil
		}
	}
	
	wrapFloat2 := func(f func(float64, float64) float64) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.FloatValue{Value: f(x.Value, y.Value)}, nil
		}
	}
	
	wrapFloat1 := func(f func(float64) float64) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 1 {
				return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
			}
			x, ok := args[0].(*eval.FloatValue)
			if !ok {
				return nil, fmt.Errorf("expected float argument")
			}
			return &eval.FloatValue{Value: f(x.Value)}, nil
		}
	}
	
	wrapIntCmp2 := func(f func(int64, int64) bool) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.IntValue)
			y, ok2 := args[1].(*eval.IntValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected int arguments")
			}
			return &eval.BoolValue{Value: f(int64(x.Value), int64(y.Value))}, nil
		}
	}
	
	wrapFloatCmp2 := func(f func(float64, float64) bool) func([]eval.Value) (eval.Value, error) {
		return func(args []eval.Value) (eval.Value, error) {
			if len(args) != 2 {
				return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
			}
			x, ok1 := args[0].(*eval.FloatValue)
			y, ok2 := args[1].(*eval.FloatValue)
			if !ok1 || !ok2 {
				return nil, fmt.Errorf("expected float arguments")
			}
			return &eval.BoolValue{Value: f(x.Value, y.Value)}, nil
		}
	}
	
	// Register built-in instances with wrapped methods as BuiltinFunction
	r.instances["Num[Int]"] = core.DictValue{
		TypeClass: "Num",
		Type:      "Int",
		Methods: map[string]interface{}{
			"add": &eval.BuiltinFunction{
				Name: "add",
				Fn:   wrapInt2(func(a, b int64) int64 { 
					result := a + b
					// Integer addition
					return result
				}),
			},
			"sub": &eval.BuiltinFunction{
				Name: "sub",
				Fn:   wrapInt2(func(a, b int64) int64 { return a - b }),
			},
			"mul": &eval.BuiltinFunction{
				Name: "mul",
				Fn:   wrapInt2(func(a, b int64) int64 { 
					result := a * b
					// Integer multiplication
					return result
				}),
			},
			"div": &eval.BuiltinFunction{
				Name: "div",
				Fn: wrapInt2(func(a, b int64) int64 {
					if b == 0 {
						panic("division by zero")
					}
					return a / b
				}),
			},
		},
	}

	r.instances["Num[Float]"] = core.DictValue{
		TypeClass: "Num",
		Type:      "Float",
		Methods: map[string]interface{}{
			"add": &eval.BuiltinFunction{Name: "add", Fn: wrapFloat2(func(a, b float64) float64 { return a + b })},
			"sub": &eval.BuiltinFunction{Name: "sub", Fn: wrapFloat2(func(a, b float64) float64 { return a - b })},
			"mul": &eval.BuiltinFunction{Name: "mul", Fn: wrapFloat2(func(a, b float64) float64 { return a * b })},
			"div": &eval.BuiltinFunction{Name: "div", Fn: wrapFloat2(func(a, b float64) float64 { return a / b })},
		},
	}

	// Fractional[Float] - extends Num with fractional operations
	r.instances["Fractional[Float]"] = core.DictValue{
		TypeClass: "Fractional",
		Type:      "Float",
		Methods: map[string]interface{}{
			// Inherit all Num methods
			"add": &eval.BuiltinFunction{Name: "add", Fn: wrapFloat2(func(a, b float64) float64 { return a + b })},
			"sub": &eval.BuiltinFunction{Name: "sub", Fn: wrapFloat2(func(a, b float64) float64 { return a - b })},
			"mul": &eval.BuiltinFunction{Name: "mul", Fn: wrapFloat2(func(a, b float64) float64 { return a * b })},
			"div": &eval.BuiltinFunction{Name: "div", Fn: wrapFloat2(func(a, b float64) float64 { return a / b })},
			"neg": &eval.BuiltinFunction{Name: "neg", Fn: wrapFloat1(func(a float64) float64 { return -a })},
			"abs": &eval.BuiltinFunction{Name: "abs", Fn: wrapFloat1(func(a float64) float64 { 
				if a < 0 { return -a }; return a 
			})},
			"fromInt": &eval.BuiltinFunction{Name: "fromInt", Fn: func(args []eval.Value) (eval.Value, error) {
				if len(args) != 1 {
					return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
				}
				if iv, ok := args[0].(*eval.IntValue); ok {
					return &eval.FloatValue{Value: float64(iv.Value)}, nil
				}
				return nil, fmt.Errorf("expected int argument")
			}},
			// Fractional-specific methods
			"divide": &eval.BuiltinFunction{Name: "divide", Fn: wrapFloat2(func(a, b float64) float64 { return a / b })},
			"recip": &eval.BuiltinFunction{Name: "recip", Fn: wrapFloat1(func(a float64) float64 { return 1.0 / a })},
			"fromRational": &eval.BuiltinFunction{Name: "fromRational", Fn: func(args []eval.Value) (eval.Value, error) {
				// For now, just convert from float (simplified)
				if len(args) != 1 {
					return nil, fmt.Errorf("expected 1 argument, got %d", len(args))
				}
				if fv, ok := args[0].(*eval.FloatValue); ok {
					return fv, nil // Identity for now
				}
				return nil, fmt.Errorf("expected float argument")
			}},
		},
		Provides: []string{"Num[Float]"}, // Fractional provides Num
	}

	r.instances["Eq[Int]"] = core.DictValue{
		TypeClass: "Eq",
		Type:      "Int",
		Methods: map[string]interface{}{
			"eq":  &eval.BuiltinFunction{Name: "eq", Fn: wrapIntCmp2(func(a, b int64) bool { return a == b })},
			"neq": &eval.BuiltinFunction{Name: "neq", Fn: wrapIntCmp2(func(a, b int64) bool { return a != b })},
		},
	}

	r.instances["Eq[Float]"] = core.DictValue{
		TypeClass: "Eq",
		Type:      "Float",
		Methods: map[string]interface{}{
			"eq": &eval.BuiltinFunction{Name: "eq", Fn: wrapFloatCmp2(func(a, b float64) bool {
				// Law-compliant: reflexive for NaN
				if a != a && b != b {
					return true
				}
				return a == b
			})},
			"neq": &eval.BuiltinFunction{Name: "neq", Fn: wrapFloatCmp2(func(a, b float64) bool {
				if a != a && b != b {
					return false
				}
				return a != b
			})},
		},
	}

	r.instances["Ord[Int]"] = core.DictValue{
		TypeClass: "Ord",
		Type:      "Int",
		Methods: map[string]interface{}{
			"lt":  &eval.BuiltinFunction{Name: "lt", Fn: wrapIntCmp2(func(a, b int64) bool { return a < b })},
			"lte": &eval.BuiltinFunction{Name: "lte", Fn: wrapIntCmp2(func(a, b int64) bool { return a <= b })},
			"gt":  &eval.BuiltinFunction{Name: "gt", Fn: wrapIntCmp2(func(a, b int64) bool { return a > b })},
			"gte": &eval.BuiltinFunction{Name: "gte", Fn: wrapIntCmp2(func(a, b int64) bool { return a >= b })},
		},
		Provides: []string{"Eq[Int]"}, // Ord provides Eq
	}

	r.instances["Ord[Float]"] = core.DictValue{
		TypeClass: "Ord",
		Type:      "Float", 
		Methods: map[string]interface{}{
			"lt":  &eval.BuiltinFunction{Name: "lt", Fn: wrapFloatCmp2(func(a, b float64) bool { return a < b })},
			"lte": &eval.BuiltinFunction{Name: "lte", Fn: wrapFloatCmp2(func(a, b float64) bool { return a <= b })},
			"gt":  &eval.BuiltinFunction{Name: "gt", Fn: wrapFloatCmp2(func(a, b float64) bool { return a > b })},
			"gte": &eval.BuiltinFunction{Name: "gte", Fn: wrapFloatCmp2(func(a, b float64) bool { return a >= b })},
		},
		Provides: []string{"Eq[Float]"}, // Ord provides Eq
	}

	// Register with dictionary registry
	for key, dict := range r.instances {
		r.dictReg.RegisterInstance(key, dict)
	}
}

// processExpression runs an expression through the full pipeline
func (r *REPL) processExpression(input string, out io.Writer) {
	// Step 1: Parse
	l := lexer.New(input, "<repl>")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		r.printParserErrors(p.Errors(), out)
		return
	}

	// Step 2: Elaborate to Core (with dictionary-passing)
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Elaboration error"), err)
		return
	}
	
	// Extract the first declaration as an expression
	if len(coreProg.Decls) == 0 {
		fmt.Fprintln(out, yellow("Empty expression"))
		return
	}
	coreExpr := coreProg.Decls[0]

	if r.config.ShowCore {
		fmt.Fprintf(out, "%s\n", dim("Core AST:"))
		fmt.Fprintln(out, formatCore(coreExpr, "  "))
	}

	// Step 3: Type check with constraints
	typeChecker := types.NewCoreTypeCheckerWithInstances(r.instEnv)
	typeChecker.EnableTraceDefaulting(r.config.TraceDefaulting)
	
	typedNode, qualType, constraints, err := typeChecker.InferWithConstraints(coreExpr, r.typeEnv)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Type error"), err)
		if r.config.TraceDefaulting {
			r.printDefaultingFailure(constraints, out)
		}
		return
	}

	// Step 4: Dictionary elaboration (resolve constraints to dictionaries)
	// Get resolved constraints from the type checker - this also triggers defaulting
	resolved := typeChecker.GetResolvedConstraints()
	
	// CRITICAL FIX: Manually call fillOperatorMethods to set correct method names
	// The REPL's InferWithConstraints doesn't call this automatically
	// Fill operator methods manually for dictionary elaboration
	typeChecker.FillOperatorMethods(coreExpr)
	
	// Get the final type after defaulting - prefer concrete types from post-defaulting
	typeToDisplay := r.getFinalTypeAfterDefaulting(typedNode, qualType, resolved)
	
	// Pretty print the final type
	prettyType := r.normalizeTypeName(typeToDisplay)
	
	if r.config.ShowTyped {
		fmt.Fprintf(out, "%s\n", dim("Typed AST:"))
		fmt.Fprintln(out, formatTyped(typedNode, "  "))
	}
	
	// Create a temporary program for elaboration
	tempProg := &core.Program{Decls: []core.CoreExpr{coreExpr}}
	elaboratedProg, err := elaborate.ElaborateWithDictionaries(tempProg, resolved)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Dictionary elaboration error"), err)
		r.suggestMissingInstances(constraints, out)
		return
	}
	
	// Extract the elaborated expression
	if len(elaboratedProg.Decls) == 0 {
		fmt.Fprintln(out, yellow("Empty result after elaboration"))
		return
	}
	elaboratedCore := elaboratedProg.Decls[0]

	// Step 5: Verify ANF
	if err := elaborate.VerifyANF(elaboratedProg); err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("ANF verification error"), err)
		return
	}

	// Step 6: Link dictionaries
	linker := link.NewLinker()
	
	// Add instances to linker with canonical keys
	r.registerDictionariesForLinker(linker)

	if r.config.DryLink {
		// Dry run to show required instances
		required := linker.DryRun(elaboratedCore)
		if len(required) > 0 {
			fmt.Fprintf(out, "%s\n", yellow("Required instances:"))
			for _, key := range required {
				fmt.Fprintf(out, "  • %s\n", key)
			}
		}
		return
	}

	linkedCore, err := linker.Link(elaboratedCore)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Linking error"), err)
		return
	}

	// Step 7: Evaluate
	evaluator := eval.NewCoreEvaluator()
	
	// Add dictionaries to evaluator with canonical keys
	r.registerDictionariesForEvaluator(evaluator)

	result, err := evaluator.Eval(linkedCore)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Runtime error"), err)
		return
	}

	// Store result
	r.lastResult = result

	// Pretty print result with type on the same line
	fmt.Fprintf(out, "%s :: %s\n", formatValue(result), cyan(prettyType))
}


// prettyPrintQualifiedType formats a type with its constraints
func (r *REPL) prettyPrintQualifiedType(typ types.Type, constraints []types.Constraint) string {
	var parts []string
	
	// Collect type variables
	typeVars := collectTypeVars(typ)
	
	if len(typeVars) > 0 || len(constraints) > 0 {
		// Add quantifier
		if len(typeVars) > 0 {
			varList := strings.Join(typeVars, " ")
			parts = append(parts, fmt.Sprintf("∀%s.", varList))
		}
		
		// Add constraints
		if len(constraints) > 0 {
			var constraintStrs []string
			for _, c := range constraints {
				constraintStrs = append(constraintStrs, formatConstraint(c))
			}
			parts = append(parts, fmt.Sprintf("%s ⇒", strings.Join(constraintStrs, ", ")))
		}
	}
	
	// Add the type
	parts = append(parts, formatType(typ))
	
	return strings.Join(parts, " ")
}

// handleCommand processes REPL commands
func (r *REPL) handleCommand(cmd string, out io.Writer) {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case ":help", ":h":
		r.printHelp(out)

	case ":quit", ":q", ":exit":
		fmt.Fprintln(out, green("Goodbye!"))
		// Exit is handled by caller

	case ":type", ":t":
		if len(parts) < 2 {
			fmt.Fprintln(out, "Usage: :type <expression>")
			return
		}
		input := strings.Join(parts[1:], " ")
		r.showType(input, out)

	case ":import", ":i":
		if len(parts) < 2 {
			fmt.Fprintln(out, "Usage: :import <module>")
			return
		}
		r.importModule(parts[1], out)

	case ":dump-core":
		r.config.ShowCore = !r.config.ShowCore
		status := "disabled"
		if r.config.ShowCore {
			status = "enabled"
		}
		fmt.Fprintf(out, "Core AST dumping %s\n", yellow(status))

	case ":dump-typed":
		r.config.ShowTyped = !r.config.ShowTyped
		status := "disabled"
		if r.config.ShowTyped {
			status = "enabled"
		}
		fmt.Fprintf(out, "Typed AST dumping %s\n", yellow(status))

	case ":dry-link":
		r.config.DryLink = !r.config.DryLink
		status := "disabled"
		if r.config.DryLink {
			status = "enabled"
		}
		fmt.Fprintf(out, "Dry linking %s\n", yellow(status))

	case ":trace-defaulting":
		if len(parts) < 2 {
			fmt.Fprintln(out, "Usage: :trace-defaulting on|off")
			return
		}
		r.config.TraceDefaulting = parts[1] == "on"
		fmt.Fprintf(out, "Defaulting trace %s\n", yellow(parts[1]))

	case ":instances":
		r.showInstances(out)

	case ":history":
		r.showHistory(out)

	case ":clear":
		fmt.Print("\033[H\033[2J")

	case ":reset":
		r.env = eval.NewEnvironment()
		r.typeEnv = types.NewTypeEnv()
		r.instEnv = types.NewInstanceEnv()
		// Re-import prelude after reset
		r.importModule("std/prelude", io.Discard)
		fmt.Fprintln(out, green("Environment reset (prelude auto-imported)"))

	case ":effects":
		if len(parts) < 2 {
			fmt.Fprintln(out, "Usage: :effects <expression>")
			return
		}
		input := strings.Join(parts[1:], " ")
		if err := EffectsCommand(input); err != nil {
			fmt.Fprintf(out, red("Error: %v\n"), err)
		}

	case ":compact":
		if len(parts) < 2 {
			fmt.Fprintln(out, "Usage: :compact on|off")
			return
		}
		enabled := parts[1] == "on"
		schema.SetCompactMode(enabled)
		fmt.Fprintf(out, "Compact JSON mode %s\n", yellow(parts[1]))

	case ":test":
		if len(parts) >= 2 && parts[1] == "--json" {
			r.runTestsJSON(out)
		} else {
			r.runTests(out)
		}

	default:
		fmt.Fprintf(out, "Unknown command: %s\n", cmd)
		fmt.Fprintln(out, "Type :help for help")
	}
}

// runTests runs tests in normal mode
func (r *REPL) runTests(out io.Writer) {
	fmt.Fprintln(out, yellow("Running tests..."))
	// TODO: Implement test discovery and execution
	fmt.Fprintln(out, "No tests found (test discovery not yet implemented)")
}

// runTestsJSON runs tests and outputs JSON report
func (r *REPL) runTestsJSON(out io.Writer) {
	// Create a new test runner with current time
	startTime := time.Now()
	report := test.NewReport()
	
	// TODO: Discover and run actual tests
	// For now, create empty report
	report.Finalize(startTime)
	
	jsonData, err := report.ToJSON()
	if err != nil {
		fmt.Fprintf(out, red("Error generating test report: %v\n"), err)
		return
	}
	
	fmt.Fprintln(out, string(jsonData))
}

// showType shows just the type of an expression without evaluating
func (r *REPL) showType(input string, out io.Writer) {
	// Parse
	l := lexer.New(input, "<repl>")
	p := parser.New(l)
	program := p.Parse()

	if len(p.Errors()) > 0 {
		r.printParserErrors(p.Errors(), out)
		return
	}

	// Elaborate
	elaborator := elaborate.NewElaborator()
	coreProg, err := elaborator.Elaborate(program)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Elaboration error"), err)
		return
	}
	
	if len(coreProg.Decls) == 0 {
		fmt.Fprintln(out, yellow("Invalid expression"))
		return
	}
	coreExpr := coreProg.Decls[0]

	// Type check with instance environment for defaulting
	typeChecker := types.NewCoreTypeCheckerWithInstances(r.instEnv)
	typeChecker.EnableTraceDefaulting(r.config.TraceDefaulting)
	
	typedNode, qualType, constraints, err := typeChecker.InferWithConstraints(coreExpr, r.typeEnv)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Type error"), err)
		return
	}

	// Get resolved constraints to trigger defaulting
	resolved := typeChecker.GetResolvedConstraints()
	
	// Get the final type after defaulting
	finalType := r.getFinalTypeAfterDefaulting(typedNode, qualType, resolved)
	
	// Pretty print the final type
	prettyType := r.prettyPrintFinalType(finalType, constraints)
	fmt.Fprintf(out, "%s :: %s\n", input, cyan(prettyType))
}

// importModule loads type class instances from a module
func (r *REPL) importModule(module string, out io.Writer) {
	switch module {
	case "std/prelude":
		// Add standard prelude instances
		fmt.Fprintf(out, "Importing %s...\n", module)
		
		// Set type-level defaults for numeric literals
		r.instEnv.SetDefault("Num", &types.TCon{Name: "int"})
		r.instEnv.SetDefault("Fractional", &types.TCon{Name: "float"})
		
		// Add type-level instances
		// Num instances
		r.instEnv.Add(&types.ClassInstance{
			ClassName: "Num",
			TypeHead:  &types.TCon{Name: "int"},
			Dict:      types.Dict{"add": "", "sub": "", "mul": "", "div": ""},
		})
		r.instEnv.Add(&types.ClassInstance{
			ClassName: "Num",
			TypeHead:  &types.TCon{Name: "float"},
			Dict:      types.Dict{"add": "", "sub": "", "mul": "", "div": ""},
		})
		
		// Fractional instances (extends Num)
		r.instEnv.Add(&types.ClassInstance{
			ClassName: "Fractional",
			TypeHead:  &types.TCon{Name: "float"},
			Dict:      types.Dict{"add": "", "sub": "", "mul": "", "div": ""},
			Super:     []string{"Num"},
		})
		
		// Eq instances
		r.instEnv.Add(&types.ClassInstance{
			ClassName: "Eq",
			TypeHead:  &types.TCon{Name: "int"},
			Dict:      types.Dict{"eq": "", "neq": ""},
		})
		r.instEnv.Add(&types.ClassInstance{
			ClassName: "Eq",
			TypeHead:  &types.TCon{Name: "float"},
			Dict:      types.Dict{"eq": "", "neq": ""},
		})
		
		// Ord instances (with superclass Eq)
		r.instEnv.Add(&types.ClassInstance{
			ClassName: "Ord",
			TypeHead:  &types.TCon{Name: "int"},
			Dict:      types.Dict{"lt": "", "lte": "", "gt": "", "gte": ""},
			Super:     []string{"Eq"},
		})
		r.instEnv.Add(&types.ClassInstance{
			ClassName: "Ord",
			TypeHead:  &types.TCon{Name: "float"},
			Dict:      types.Dict{"lt": "", "lte": "", "gt": "", "gte": ""},
			Super:     []string{"Eq"},
		})
		
		// Re-initialize runtime dictionaries to ensure they're loaded
		r.initBuiltins()
		
		// Show instances (already using normalized names)
		r.instances["Show[Int]"] = core.DictValue{
			TypeClass: "Show",
			Type:      "Int",
			Methods: map[string]interface{}{
				"show": func(a int64) string { return fmt.Sprintf("%d", a) },
			},
		}
		
		r.instances["Show[Float]"] = core.DictValue{
			TypeClass: "Show",
			Type:      "Float",
			Methods: map[string]interface{}{
				"show": func(a float64) string { return fmt.Sprintf("%g", a) },
			},
		}
		
		r.instances["Show[String]"] = core.DictValue{
			TypeClass: "Show", 
			Type:      "String",
			Methods: map[string]interface{}{
				"show": func(s string) string { return fmt.Sprintf("%q", s) },
			},
		}
		
		r.instances["Show[Bool]"] = core.DictValue{
			TypeClass: "Show",
			Type:      "Bool",
			Methods: map[string]interface{}{
				"show": func(b bool) string { 
					if b {
						return "true" 
					}
					return "false"
				},
			},
		}

		r.config.ImportedModules = append(r.config.ImportedModules, module)
		fmt.Fprintf(out, "%s Imported %s\n", green("✓"), module)
		
	default:
		fmt.Fprintf(out, "%s: Unknown module %s\n", red("Error"), module)
	}
}

// showInstances displays available type class instances
func (r *REPL) showInstances(out io.Writer) {
	fmt.Fprintln(out, bold("Available instances:"))
	
	// Group by type class
	byClass := make(map[string][]string)
	for key := range r.instances {
		parts := strings.Split(key, "[")
		if len(parts) >= 1 {
			className := parts[0]
			byClass[className] = append(byClass[className], key)
		}
	}
	
	for className, instances := range byClass {
		fmt.Fprintf(out, "  %s:\n", yellow(className))
		for _, inst := range instances {
			dict := r.instances[inst]
			fmt.Fprintf(out, "    • %s", inst)
			if len(dict.Provides) > 0 {
				fmt.Fprintf(out, " %s", dim(fmt.Sprintf("(provides %s)", strings.Join(dict.Provides, ", "))))
			}
			fmt.Fprintln(out)
		}
	}
}

// showHistory displays command history
func (r *REPL) showHistory(out io.Writer) {
	for i, cmd := range r.history {
		fmt.Fprintf(out, "%3d  %s\n", i+1, cmd)
	}
}

// printHelp shows available commands
func (r *REPL) printHelp(out io.Writer) {
	fmt.Fprintln(out, bold("REPL Commands:"))
	fmt.Fprintln(out, "  :help, :h                Show this help")
	fmt.Fprintln(out, "  :quit, :q                Exit the REPL")
	fmt.Fprintln(out, "  :type <expr>             Show type of expression")
	fmt.Fprintln(out, "  :effects <expr>          Show type and effects without evaluating")
	fmt.Fprintln(out, "  :import <module>         Load module instances")
	fmt.Fprintln(out, "  :dump-core              Toggle Core AST display")
	fmt.Fprintln(out, "  :dump-typed             Toggle Typed AST display")
	fmt.Fprintln(out, "  :dry-link               Show required instances without evaluating")
	fmt.Fprintln(out, "  :trace-defaulting on|off Enable/disable defaulting trace")
	fmt.Fprintln(out, "  :instances              Show available type class instances")
	fmt.Fprintln(out, "  :test [--json]          Run tests (with optional JSON output)")
	fmt.Fprintln(out, "  :compact on|off         Enable/disable compact JSON mode")
	fmt.Fprintln(out, "  :history                Show command history")
	fmt.Fprintln(out, "  :clear                  Clear the screen")
	fmt.Fprintln(out, "  :reset                  Reset the environment")
	fmt.Fprintln(out)
	fmt.Fprintln(out, bold("Examples:"))
	fmt.Fprintln(out, "  let add = \\x y. x + y in add(1)(2)")
	fmt.Fprintln(out, "  :type \\x. x + x")
	fmt.Fprintln(out, "  :effects 1 + 2")
	fmt.Fprintln(out, "  :test --json")
	fmt.Fprintln(out, "  :import std/prelude")
}

// printParserErrors displays parser errors nicely
func (r *REPL) printParserErrors(errors []error, out io.Writer) {
	fmt.Fprintf(out, "%s:\n", red("Parser errors"))
	for _, err := range errors {
		fmt.Fprintf(out, "  • %v\n", err)
	}
}

// printDefaultingFailure shows why defaulting failed
func (r *REPL) printDefaultingFailure(constraints []types.Constraint, out io.Writer) {
	fmt.Fprintf(out, "%s\n", yellow("Defaulting failure details:"))
	fmt.Fprintln(out, "  Ambiguous constraints:")
	for _, c := range constraints {
		if isAmbiguous(c) {
			fmt.Fprintf(out, "    • %s\n", formatConstraint(c))
		}
	}
	fmt.Fprintln(out, "  Current defaults:")
	fmt.Fprintln(out, "    • Num → Int")
	fmt.Fprintln(out, "    • Fractional → Float")
}

// suggestMissingInstances provides helpful suggestions for missing instances
func (r *REPL) suggestMissingInstances(constraints []types.Constraint, out io.Writer) {
	fmt.Fprintf(out, "%s\n", yellow("Missing instances:"))
	for _, c := range constraints {
		key := constraintToKey(c)
		if _, exists := r.instances[key]; !exists {
			fmt.Fprintf(out, "  • %s\n", key)
			
			// Suggest import if in prelude
			if isInPrelude(key) {
				fmt.Fprintf(out, "    %s\n", dim("Try: :import std/prelude"))
			}
		}
	}
}

// Helper functions

func formatCore(expr core.CoreExpr, indent string) string {
	// Format Core AST for display
	switch e := expr.(type) {
	case *core.Var:
		return fmt.Sprintf("%sVar(%s)", indent, e.Name)
	case *core.Lit:
		return fmt.Sprintf("%sLit(%v)", indent, e.Value)
	case *core.Lambda:
		return fmt.Sprintf("%sLam(%v) ->\n%s", indent, e.Params, formatCore(e.Body, indent+"  "))
	case *core.App:
		args := ""
		for i, arg := range e.Args {
			if i > 0 {
				args += ",\n"
			}
			args += formatCore(arg, indent+"  ")
		}
		return fmt.Sprintf("%sApp(\n%s,\n%s)", indent, 
			formatCore(e.Func, indent+"  "), args)
	case *core.Let:
		return fmt.Sprintf("%sLet(%s) =\n%s\n%sin\n%s", indent, e.Name,
			formatCore(e.Value, indent+"  "), indent,
			formatCore(e.Body, indent+"  "))
	case *core.DictApp:
		return fmt.Sprintf("%sDictApp(%s, %s, [...])", indent, e.Dict, e.Method)
	default:
		return fmt.Sprintf("%s%T", indent, e)
	}
}

func formatTyped(expr typedast.TypedNode, indent string) string {
	// Format TypedAST for display
	typ := expr.GetType()
	
	// Convert interface{} to string for display
	typeStr := fmt.Sprintf("%v", typ)
	
	switch e := expr.(type) {
	case *typedast.TypedVar:
		return fmt.Sprintf("%sVar(%s : %s)", indent, e.Name, typeStr)
	case *typedast.TypedLit:
		return fmt.Sprintf("%sLit(%v : %s)", indent, e.Value, typeStr)
	case *typedast.TypedLambda:
		paramStr := fmt.Sprintf("%v", e.Params)
		return fmt.Sprintf("%sLam(%s) ->\n%s", indent, paramStr,
			formatTyped(e.Body, indent+"  "))
	case *typedast.TypedApp:
		argsStr := ""
		for i, arg := range e.Args {
			if i > 0 {
				argsStr += "\n"
			}
			argsStr += formatTyped(arg, indent+"  ")
		}
		return fmt.Sprintf("%sApp : %s\n%s\n%s", indent, typeStr,
			formatTyped(e.Func, indent+"  "), argsStr)
	default:
		return fmt.Sprintf("%s%T : %s", indent, e, typeStr)
	}
}

func formatValue(val interface{}) string {
	// Format evaluation result
	switch v := val.(type) {
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%g", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case string:
		return v
	case eval.Value:
		return v.String()
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatType(t types.Type) string {
	switch typ := t.(type) {
	case *types.TVar:
		return typ.Name
	case *types.TVar2:
		// Handle TVar2 type variables (used during type checking)
		return typ.Name
	case *types.TCon:
		// Normalize type constructor names for display
		return types.NormalizeTypeName(typ)
	case *types.TApp:
		// Check if it's a function type (-> constructor)
		if con, ok := typ.Constructor.(*types.TCon); ok && con.Name == "->" {
			if len(typ.Args) == 2 {
				return fmt.Sprintf("%s → %s", formatType(typ.Args[0]), formatType(typ.Args[1]))
			}
		}
		// Generic application
		args := make([]string, len(typ.Args))
		for i, arg := range typ.Args {
			args[i] = formatType(arg)
		}
		return fmt.Sprintf("%s %s", formatType(typ.Constructor), strings.Join(args, " "))
	case *types.TList:
		return fmt.Sprintf("[%s]", formatType(typ.Element))
	case *types.TRecord:
		// Sort field names for deterministic output
		keys := make([]string, 0, len(typ.Fields))
		for k := range typ.Fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		
		fields := make([]string, len(keys))
		for i, k := range keys {
			fields[i] = fmt.Sprintf("%s: %s", k, formatType(typ.Fields[k]))
		}
		return fmt.Sprintf("{%s}", strings.Join(fields, ", "))
	default:
		return fmt.Sprintf("%v", t)
	}
}

func formatConstraint(c types.Constraint) string {
	return fmt.Sprintf("%s %s", c.Class, formatType(c.Type))
}

func collectTypeVars(t types.Type) []string {
	vars := make(map[string]bool)
	collectVarsHelper(t, vars)
	
	var result []string
	for v := range vars {
		result = append(result, v)
	}
	return result
}

func collectVarsHelper(t types.Type, vars map[string]bool) {
	switch typ := t.(type) {
	case *types.TVar:
		vars[typ.Name] = true
	case *types.TApp:
		collectVarsHelper(typ.Constructor, vars)
		for _, arg := range typ.Args {
			collectVarsHelper(arg, vars)
		}
	}
}

func isAmbiguous(c types.Constraint) bool {
	// A constraint is ambiguous if its type variable doesn't appear in the result type
	if _, ok := c.Type.(*types.TVar); ok {
		// In a complete implementation, would check if var appears in the result type
		return true
	}
	return false
}

func constraintToKey(c types.Constraint) string {
	typeStr := formatType(c.Type)
	// Normalize type string for key
	typeStr = strings.ReplaceAll(typeStr, " ", "")
	return fmt.Sprintf("%s[%s]", c.Class, typeStr)
}

func isInPrelude(key string) bool {
	preludeInstances := []string{
		"Show[Int]", "Show[Float]", "Show[String]", "Show[Bool]",
		"Read[Int]", "Read[Float]", "Read[String]",
		"Enum[Int]", "Bounded[Int]", "Bounded[Bool]",
	}
	
	for _, inst := range preludeInstances {
		if inst == key {
			return true
		}
	}
	return false
}

// registerDictionariesForLinker registers all dictionaries with canonical keys for the linker
func (r *REPL) registerDictionariesForLinker(linker *link.Linker) {
	for _, dict := range r.instances {
		// Convert "Num[Int]" to canonical keys like "prelude::Num::Int::add"
		className := dict.TypeClass
		
		// Create a proper Type for key generation
		typeForKey := &types.TCon{Name: dict.Type}
		
		// Register each method with its canonical key
		for methodName := range dict.Methods {
			canonicalKey := types.MakeDictionaryKey("prelude", className, typeForKey, methodName)
			linker.AddDictionary(canonicalKey, dict)
		}
	}
}

// registerDictionariesForEvaluator registers all dictionaries with canonical keys for the evaluator
func (r *REPL) registerDictionariesForEvaluator(evaluator *eval.CoreEvaluator) {
	for _, dict := range r.instances {
		// Convert "Num[Int]" to canonical keys like "prelude::Num::Int::add"
		className := dict.TypeClass
		
		// Create a proper Type for key generation
		typeForKey := &types.TCon{Name: dict.Type}
		
		// Register each method with its canonical key
		for methodName := range dict.Methods {
			canonicalKey := types.MakeDictionaryKey("prelude", className, typeForKey, methodName)
			evaluator.AddDictionary(canonicalKey, dict)
		}
		
		// Also register the base dictionary for lookups (no method name)
		baseKey := types.MakeDictionaryKey("prelude", className, typeForKey, "")
		evaluator.AddDictionary(baseKey, dict)
	}
}

// getFinalTypeAfterDefaulting gets the final type after defaulting has been applied
func (r *REPL) getFinalTypeAfterDefaulting(typedNode typedast.TypedNode, qualType types.Type, resolved map[uint64]*types.ResolvedConstraint) types.Type {
	// Debug: print what we're getting (only if trace is enabled)
	if r.config.TraceDefaulting {
		// Getting final type after constraint resolution
	}
	
	// Strategy: Prefer concrete types over type variables, in this order:
	// 1. Concrete TCon from typedNode.GetType() (if not a TVar)
	// 2. Concrete type from resolved constraints for this node ID
	// 3. Any concrete type from resolved constraints (from defaulting)
	// 4. Fallback to qualType
	
	// First check if the typed node already has a concrete type
	nodeType := typedNode.GetType()
	if t, ok := nodeType.(types.Type); ok {
		switch typ := t.(type) {
		case *types.TCon:
			// Already concrete - use it
			return typ
		}
	}
	
	// If we have resolved constraints, look for concrete types
	if resolved != nil && len(resolved) > 0 {
		// Check if the root node has a defaulted type
		if rc, ok := resolved[typedNode.GetNodeID()]; ok && rc.Type != nil {
			if con, ok := rc.Type.(*types.TCon); ok {
				return con
			}
		}
		
		// Look for any resolved constraint with a concrete type from defaulting
		for _, rc := range resolved {
			if rc.Type != nil {
				if con, ok := rc.Type.(*types.TCon); ok {
					// Found a concrete type from defaulting
					return con
				}
			}
		}
	}
	
	// Fall back to the original qualified type
	return qualType
}

// prettyPrintFinalType formats the final type after defaulting
func (r *REPL) prettyPrintFinalType(typ types.Type, constraints []types.Constraint) string {
	// First normalize the type name
	normalizedType := r.normalizeTypeName(typ)
	
	// If there are no remaining constraints, just return the type
	remainingConstraints := r.filterResolvedConstraints(constraints, typ)
	if len(remainingConstraints) == 0 {
		return normalizedType
	}
	
	// Format with remaining constraints
	var parts []string
	for _, c := range remainingConstraints {
		parts = append(parts, formatConstraint(c))
	}
	parts = append(parts, normalizedType)
	return strings.Join(parts, " => ")
}

// normalizeTypeName converts internal type representations to user-friendly names
func (r *REPL) normalizeTypeName(typ types.Type) string {
	switch t := typ.(type) {
	case *types.TCon:
		// Normalize common type constructor names
		switch t.Name {
		case "int":
			return "Int"
		case "float":
			return "Float"
		case "bool":
			return "Bool"
		case "string":
			return "String"
		default:
			return t.Name
		}
	case *types.TVar:
		// Format type variables nicely
		return t.Name
	case *types.TVar2:
		// Check if it was defaulted to a concrete type
		if t.Name == "int" || t.Name == "Int" {
			return "Int"
		} else if t.Name == "float" || t.Name == "Float" {
			return "Float"
		}
		// Otherwise show as a type variable
		return t.Name
	default:
		return formatType(typ)
	}
}

// filterResolvedConstraints removes constraints that have been resolved via defaulting
func (r *REPL) filterResolvedConstraints(constraints []types.Constraint, finalType types.Type) []types.Constraint {
	var remaining []types.Constraint
	
	// If the final type is concrete, all constraints have been resolved
	switch finalType.(type) {
	case *types.TCon:
		// Concrete type - all constraints resolved
		return remaining
	}
	
	// Otherwise keep constraints on remaining type variables
	for _, c := range constraints {
		if _, ok := c.Type.(*types.TVar); ok {
			remaining = append(remaining, c)
		} else if _, ok := c.Type.(*types.TVar2); ok {
			remaining = append(remaining, c)
		}
	}
	
	return remaining
}