package repl

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/elaborate"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/lexer"
	"github.com/sunholo/ailang/internal/parser"
	"github.com/sunholo/ailang/internal/schema"
	"github.com/sunholo/ailang/internal/test"
	"github.com/sunholo/ailang/internal/types"
)

// HandleCommand processes REPL commands (exported for WASM)
func (r *REPL) HandleCommand(cmd string, out io.Writer) {
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

	case ":propose":
		filename, err := ParseProposeCommand(cmd)
		if err != nil {
			fmt.Fprintf(out, red("Error: %v\n"), err)
			return
		}
		if err := ProposePlanCommand(filename); err != nil {
			fmt.Fprintf(out, red("Error: %v\n"), err)
		}

	case ":scaffold":
		planFile, outputDir, overwrite, err := ParseScaffoldCommand(cmd)
		if err != nil {
			fmt.Fprintf(out, red("Error: %v\n"), err)
			return
		}
		if err := ScaffoldCommand(planFile, outputDir, overwrite); err != nil {
			fmt.Fprintf(out, red("Error: %v\n"), err)
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

	typedNode, _, qualType, constraints, err := typeChecker.InferWithConstraints(coreExpr, r.typeEnv)
	if err != nil {
		fmt.Fprintf(out, "%s: %v\n", red("Type error"), err)
		return
	}

	// Note: We don't update r.typeEnv here since :type is read-only (updatedEnv ignored)

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
		_ = r.instEnv.Add(&types.ClassInstance{
			ClassName: "Num",
			TypeHead:  &types.TCon{Name: "int"},
			Dict:      types.Dict{"add": "", "sub": "", "mul": "", "div": ""},
		})
		_ = r.instEnv.Add(&types.ClassInstance{
			ClassName: "Num",
			TypeHead:  &types.TCon{Name: "float"},
			Dict:      types.Dict{"add": "", "sub": "", "mul": "", "div": ""},
		})

		// Fractional instances (extends Num)
		_ = r.instEnv.Add(&types.ClassInstance{
			ClassName: "Fractional",
			TypeHead:  &types.TCon{Name: "float"},
			Dict:      types.Dict{"add": "", "sub": "", "mul": "", "div": ""},
			Super:     []string{"Num"},
		})

		// Eq instances
		_ = r.instEnv.Add(&types.ClassInstance{
			ClassName: "Eq",
			TypeHead:  &types.TCon{Name: "int"},
			Dict:      types.Dict{"eq": "", "neq": ""},
		})
		_ = r.instEnv.Add(&types.ClassInstance{
			ClassName: "Eq",
			TypeHead:  &types.TCon{Name: "float"},
			Dict:      types.Dict{"eq": "", "neq": ""},
		})

		// Ord instances (with superclass Eq)
		_ = r.instEnv.Add(&types.ClassInstance{
			ClassName: "Ord",
			TypeHead:  &types.TCon{Name: "int"},
			Dict:      types.Dict{"lt": "", "lte": "", "gt": "", "gte": ""},
			Super:     []string{"Eq"},
		})
		_ = r.instEnv.Add(&types.ClassInstance{
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
	fmt.Fprintln(out, "  :propose <plan.json>    Validate an architecture plan")
	fmt.Fprintln(out, "  :scaffold --from-plan <plan.json> [--output <dir>] [--overwrite]")
	fmt.Fprintln(out, "                          Generate module stubs from plan")
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
