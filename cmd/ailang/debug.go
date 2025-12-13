package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/sunholo/ailang/internal/ast"
	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/types"
)

func runDebug() {
	debugCmd := flag.NewFlagSet("debug", flag.ExitOnError)
	showTypesFlag := debugCmd.Bool("show-types", false, "Show inferred types for expressions")
	compactFlag := debugCmd.Bool("compact", false, "Compact output (no indentation)")

	if err := debugCmd.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	if debugCmd.NArg() == 0 {
		fmt.Fprintf(os.Stderr, "%s: missing subcommand\n", red("Error"))
		printDebugHelp()
		os.Exit(1)
	}

	subcommand := debugCmd.Arg(0)

	switch subcommand {
	case "ast":
		if debugCmd.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
			fmt.Println("Usage: ailang debug ast [flags] <file.ail>")
			os.Exit(1)
		}
		runDebugAST(debugCmd.Arg(1), *showTypesFlag, *compactFlag)

	case "hash":
		if debugCmd.NArg() < 2 {
			fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
			fmt.Println("Usage: ailang debug hash <file>")
			os.Exit(1)
		}
		runDebugHash(debugCmd.Arg(1))

	case "cycles":
		// Parse cycles-specific flags
		cyclesCmd := flag.NewFlagSet("cycles", flag.ExitOnError)
		jsonFlag := cyclesCmd.Bool("json", false, "Output in JSON format")
		if err := cyclesCmd.Parse(debugCmd.Args()[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
			os.Exit(1)
		}
		if cyclesCmd.NArg() < 1 {
			fmt.Fprintf(os.Stderr, "%s: missing file argument\n", red("Error"))
			fmt.Println("Usage: ailang debug cycles [--json] <file.ail>")
			os.Exit(1)
		}
		runDebugCycles(cyclesCmd.Arg(0), *jsonFlag)

	case "types":
		// Comprehensive type debugging help (v0.5.11+)
		printTypeDebugHelp()

	default:
		fmt.Fprintf(os.Stderr, "%s: unknown debug subcommand '%s'\n", red("Error"), subcommand)
		printDebugHelp()
		os.Exit(1)
	}
}

func printTypeDebugHelp() {
	fmt.Println(cyan("=== Type Inference Debugging (v0.5.11+) ==="))
	fmt.Println()
	fmt.Println(bold("Usage:"))
	fmt.Println("  ailang run --debug-types file.ail")
	fmt.Println("  ailang run --debug-types --node 42 file.ail  # Filter to node ID")
	fmt.Println()
	fmt.Println(bold("Output Sections:"))
	fmt.Println()
	fmt.Println("  " + green("[Substitution Map]"))
	fmt.Println("    Type variable → resolved type mappings")
	fmt.Println("    Example: α1 → int (direct), α2 → α3 → float (CHAIN)")
	fmt.Println()
	fmt.Println("  " + green("[Constraints]"))
	fmt.Println("    Type class constraints (Num, Eq, Ord, Fractional)")
	fmt.Println("    Shows: Added constraints and resolved methods")
	fmt.Println("    Example: Num int → add (resolved to 'add' builtin)")
	fmt.Println()
	fmt.Println("  " + green("[CoreTI Entries]"))
	fmt.Println("    Type information for each Core AST node")
	fmt.Println("    Shows: NodeID, type, constraints, and origins")
	fmt.Println()
	fmt.Println(bold("Understanding Origins (Provenance):"))
	fmt.Println()
	fmt.Println("  The " + cyan("Origins:") + " section answers \"why does this have this type?\"")
	fmt.Println()
	fmt.Println("  Origin Kinds:")
	fmt.Println("    " + yellow("annotation") + "   - Explicit type annotation (let x: int = ...)")
	fmt.Println("    " + yellow("literal") + "      - Inferred from literal (3.14 → float)")
	fmt.Println("    " + yellow("inferred") + "     - Fresh type variable created during inference")
	fmt.Println("    " + yellow("defaulted") + "    - Type variable defaulted (Num → int)")
	fmt.Println("    " + yellow("from_use") + "     - Inferred from call site")
	fmt.Println("    " + yellow("from_pattern") + " - Inferred from pattern match")
	fmt.Println()
	fmt.Println(bold("Common Debugging Scenarios:"))
	fmt.Println()
	fmt.Println("  " + cyan("Q: Why is my float becoming int?"))
	fmt.Println("  A: Look for Origins showing \"defaulted\" - Num defaults to int")
	fmt.Println("     Fix: Add type annotation: let add: float -> float -> float = ...")
	fmt.Println()
	fmt.Println("  " + cyan("Q: Which nodes share the same type variable?"))
	fmt.Println("  A: Search output for the same α name (e.g., α22)")
	fmt.Println("     Command: ailang run --debug-types file.ail | grep \"α22\"")
	fmt.Println()
	fmt.Println("  " + cyan("Q: Which builtin is called for + operator?"))
	fmt.Println("  A: Look for \"Constraint: Num → add\" showing method resolution")
	fmt.Println()
	fmt.Println("  " + cyan("Q: Type mismatch with α42?"))
	fmt.Println("  A: Filter to that node: ailang run --debug-types --node 42 file.ail")
	fmt.Println()
	fmt.Println(bold("Example Output:"))
	fmt.Println()
	fmt.Println("  NodeID 42: int")
	fmt.Println("    Constraint: Num → add")
	fmt.Println("    Origins:")
	fmt.Println("      - inferred: fresh type variable")
	fmt.Println("      - defaulted: defaulted to int (Num constraint)")
	fmt.Println()
	fmt.Println(bold("Demo:"))
	fmt.Println("  ailang run --debug-types --caps IO examples/debug_types_demo.ail")
	fmt.Println()
	fmt.Println(bold("Documentation:"))
	fmt.Println("  https://ailang.dev/docs/guides/debugging")
}

func printDebugHelp() {
	fmt.Println(bold("ailang debug") + " - Debug and introspection utilities")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  ast <file>     Show Core AST (ANF) with optional type information")
	fmt.Println("  cycles <file>  Detect cyclic type references in type definitions")
	fmt.Println("  types          Show type debugging help (--debug-types flag)")
	fmt.Println("  hash <file>    Compute SHA256 hash of a file (for artifacts)")
	fmt.Println()
	fmt.Println("Flags for 'debug ast':")
	fmt.Println("  --show-types        Show inferred types for expressions")
	fmt.Println("  --compact           Compact output (no indentation)")
	fmt.Println()
	fmt.Println("Flags for 'debug cycles':")
	fmt.Println("  --json              Output in JSON format (for tooling)")
	fmt.Println()
	fmt.Println("For type inference debugging, use:")
	fmt.Println("  ailang run --debug-types file.ail")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  ailang debug ast example.ail")
	fmt.Println("  ailang debug ast --show-types example.ail")
	fmt.Println("  ailang debug cycles examples/complex_types.ail")
	fmt.Println("  ailang debug cycles --json examples/complex_types.ail")
	fmt.Println("  ailang debug types")
	fmt.Println("  ailang debug hash design_docs/planned/M-FIX-123.md")
}

func runDebugAST(filename string, showTypes bool, compact bool) {
	// Read file
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Parse
	l := lexer.New(string(source), filename)
	p := parser.New(l)
	prog := p.Parse()

	// Elaborate to Core AST
	typeEnv := types.NewTypeEnv()
	elab := elaborate.NewElaborator()
	coreProg, err := elab.Elaborate(prog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Elaboration error"), err)
		os.Exit(1)
	}

	var coreTI types.CoreTypeInfo

	if showTypes {
		// Type check to populate CoreTI
		typeChecker := types.NewCoreTypeChecker()
		for _, decl := range coreProg.Decls {
			_, _, _, _, err := typeChecker.InferWithConstraints(decl, typeEnv)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s: %v\n", yellow("Type error"), err)
				// Continue anyway - show partial types
			}
		}
		coreTI = typeChecker.CoreTI
	}

	fmt.Println(cyan("=== Core AST (ANF) ==="))
	printCoreAST(coreProg, coreTI, compact, 0)
	fmt.Println()
}

func printCoreAST(prog *core.Program, coreTI types.CoreTypeInfo, compact bool, depth int) {
	indent := ""
	if !compact {
		indent = strings.Repeat("  ", depth)
	}

	fmt.Printf("%sProgram:\n", indent)
	for i, decl := range prog.Decls {
		fmt.Printf("%s  [%d] ", indent, i)
		printCoreExpr(decl, coreTI, compact, depth+1)
	}
}

func printCoreExpr(expr core.CoreExpr, coreTI types.CoreTypeInfo, compact bool, depth int) {
	indent := ""
	if !compact {
		indent = strings.Repeat("  ", depth)
	}

	// Node ID annotation
	nodeID := ""
	if expr != nil {
		nodeID = fmt.Sprintf(" [#%d]", expr.ID())
	}

	// Type annotation
	typeStr := ""
	if coreTI != nil && expr != nil {
		if t, ok := coreTI.Get(expr.ID()); ok {
			typeStr = fmt.Sprintf(" :: %s", green(t.String()))
		}
	}

	switch e := expr.(type) {
	case *core.Lit:
		fmt.Printf("Lit(%v)%s%s\n", e.Value, nodeID, typeStr)

	case *core.Var:
		fmt.Printf("Var(%s)%s%s\n", e.Name, nodeID, typeStr)

	case *core.VarGlobal:
		fmt.Printf("VarGlobal(%s.%s)%s%s\n", e.Ref.Module, e.Ref.Name, nodeID, typeStr)

	case *core.Let:
		fmt.Printf("Let(%s)%s%s:\n", e.Name, nodeID, typeStr)
		fmt.Printf("%s  Value: ", indent)
		printCoreExpr(e.Value, coreTI, compact, depth+1)
		fmt.Printf("%s  Body:  ", indent)
		printCoreExpr(e.Body, coreTI, compact, depth+1)

	case *core.Lambda:
		fmt.Printf("Lambda([%s])%s%s:\n", strings.Join(e.Params, ", "), nodeID, typeStr)
		fmt.Printf("%s  Body: ", indent)
		printCoreExpr(e.Body, coreTI, compact, depth+1)

	case *core.App:
		fmt.Printf("App%s%s:\n", nodeID, typeStr)
		fmt.Printf("%s  Func: ", indent)
		printCoreExpr(e.Func, coreTI, compact, depth+1)
		for i, arg := range e.Args {
			fmt.Printf("%s  Arg[%d]: ", indent, i)
			printCoreExpr(arg, coreTI, compact, depth+1)
		}

	case *core.If:
		fmt.Printf("If%s%s:\n", nodeID, typeStr)
		fmt.Printf("%s  Cond: ", indent)
		printCoreExpr(e.Cond, coreTI, compact, depth+1)
		fmt.Printf("%s  Then: ", indent)
		printCoreExpr(e.Then, coreTI, compact, depth+1)
		fmt.Printf("%s  Else: ", indent)
		printCoreExpr(e.Else, coreTI, compact, depth+1)

	case *core.List:
		fmt.Printf("List[%d]%s%s:\n", len(e.Elements), nodeID, typeStr)
		for i, elem := range e.Elements {
			fmt.Printf("%s  [%d]: ", indent, i)
			printCoreExpr(elem, coreTI, compact, depth+1)
		}

	case *core.Intrinsic:
		fmt.Printf("Intrinsic(%v)%s%s:\n", e.Op, nodeID, typeStr)
		for i, arg := range e.Args {
			fmt.Printf("%s  Arg[%d]: ", indent, i)
			printCoreExpr(arg, coreTI, compact, depth+1)
		}

	default:
		fmt.Printf("%T%s%s\n", expr, nodeID, typeStr)
	}
}

func runDebugHash(filename string) {
	// Read file
	content, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	// Compute hash (SHA-256, first 16 hex chars)
	h := sha256.Sum256(content)
	hash := hex.EncodeToString(h[:])[:16]

	// Print hash (just the hash, for easy script usage)
	fmt.Println(hash)
}

// CycleKind classifies whether a cycle is expected or suspicious
type CycleKind string

const (
	CycleExpected   CycleKind = "expected"   // stdlib types, recursive ADTs
	CycleSuspicious CycleKind = "suspicious" // user-defined, may cause issues
)

// CycleInfo holds information about a detected cycle
type CycleInfo struct {
	Kind     CycleKind `json:"kind"`
	TypeName string    `json:"type_name"`
	Path     []string  `json:"path"`
	Depth    int       `json:"depth"`
	Note     string    `json:"note,omitempty"`
}

// CyclesReport is the JSON output format
type CyclesReport struct {
	File    string      `json:"file"`
	Cycles  []CycleInfo `json:"cycles"`
	Summary struct {
		Suspicious int `json:"suspicious"`
		Expected   int `json:"expected"`
		Total      int `json:"total"`
	} `json:"summary"`
}

func runDebugCycles(filename string, outputJSON bool) {
	// Read and parse the file
	source, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}

	l := lexer.New(string(source), filename)
	p := parser.New(l)
	prog := p.Parse()

	if len(p.Errors()) > 0 {
		fmt.Fprintf(os.Stderr, "%s: parse errors\n", red("Error"))
		for _, e := range p.Errors() {
			fmt.Fprintf(os.Stderr, "  %s\n", e)
		}
		os.Exit(1)
	}

	// Get declarations from the Module
	var decls []ast.Node
	if prog.Module != nil {
		decls = prog.Module.Decls
	}

	// Use types.DetectCycles for improved cycle detection
	// This properly handles generic types like List[a] and Tree[a]
	typeCycles := types.DetectCycles(decls, filename)

	// Convert to CLI format
	var cycles []CycleInfo
	for _, tc := range typeCycles {
		cycles = append(cycles, CycleInfo{
			Kind:     CycleKind(tc.Kind),
			TypeName: tc.TypeName,
			Path:     tc.Path,
			Depth:    tc.Depth,
			Note:     tc.Note,
		})
	}

	// Build report
	report := CyclesReport{
		File:   filename,
		Cycles: cycles,
	}
	for _, c := range cycles {
		if c.Kind == CycleSuspicious {
			report.Summary.Suspicious++
		} else {
			report.Summary.Expected++
		}
		report.Summary.Total++
	}

	if outputJSON {
		outputCyclesJSON(report)
	} else {
		outputCyclesHuman(report)
	}
}

func outputCyclesJSON(report CyclesReport) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
}

func outputCyclesHuman(report CyclesReport) {
	fmt.Printf("Analyzing type graph for %s...\n\n", report.File)

	if len(report.Cycles) == 0 {
		fmt.Println(green("No cyclic type references found."))
		return
	}

	fmt.Printf("Found %d cyclic type reference(s):\n\n", len(report.Cycles))

	for i, cycle := range report.Cycles {
		kindStr := yellow("[SUSPICIOUS]")
		if cycle.Kind == CycleExpected {
			kindStr = green("[EXPECTED]")
		}

		fmt.Printf("Cycle %d %s: %s\n", i+1, kindStr, cycle.TypeName)
		fmt.Printf("  Path: %s\n", strings.Join(cycle.Path, " → "))
		fmt.Printf("  Depth: %d node(s)\n", cycle.Depth)
		if cycle.Note != "" {
			fmt.Printf("  Note: %s\n", cycle.Note)
		}
		fmt.Println()
	}

	fmt.Println("Summary:")
	fmt.Printf("  - %d suspicious cycle(s) (may cause hangs without cycle-safe traversal)\n", report.Summary.Suspicious)
	fmt.Printf("  - %d expected cycle(s) (standard recursive patterns)\n", report.Summary.Expected)
}
