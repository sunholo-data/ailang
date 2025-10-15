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
	"github.com/sunholo/ailang/internal/effects"
	"github.com/sunholo/ailang/internal/eval"
	"github.com/sunholo/ailang/internal/runtime"
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

	// Persistent evaluator (v0.3.3 fix - resolves builtins properly)
	evaluator       *eval.CoreEvaluator
	builtinRegistry *runtime.BuiltinRegistry
	effContext      *effects.EffContext
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

	// Create persistent evaluator with builtin resolver (v0.3.3 fix)
	// This mirrors what `ailang run` does and fixes the "no resolver" error
	evaluator := eval.NewCoreEvaluator()
	builtinRegistry := runtime.NewBuiltinRegistry(evaluator)
	builtinResolver := runtime.NewBuiltinOnlyResolver(builtinRegistry)
	evaluator.SetGlobalResolver(builtinResolver)

	// Create effect context (grant IO by default for REPL convenience)
	effContext := effects.NewEffContext()
	effContext.Grant(effects.NewCapability("IO")) // Allow println, readLine, etc. in REPL
	evaluator.SetEffContext(effContext)

	// Enable experimental binop shim for REPL (handles float equality until OpLowering is complete)
	evaluator.SetExperimentalBinopShim(true)

	r := &REPL{
		config:          &Config{},
		env:             evaluator.Env(), // Share the evaluator's environment (for persistent let bindings)
		typeEnv:         types.NewTypeEnv(),
		instEnv:         types.NewInstanceEnv(),
		dictReg:         types.NewDictionaryRegistry(),
		instances:       make(map[string]core.DictValue),
		history:         []string{},
		version:         version,
		buildTime:       buildTime,
		evaluator:       evaluator,
		builtinRegistry: builtinRegistry,
		effContext:      effContext,
	}

	// Register dictionaries with the persistent evaluator
	r.registerDictionariesForEvaluator(r.evaluator)

	return r
}

// EnableTrace enables execution tracing
func (r *REPL) EnableTrace() {
	r.config.Verbose = true
}

// getPrompt returns the REPL prompt with active capabilities
func (r *REPL) getPrompt() string {
	if len(r.effContext.Caps) == 0 {
		return "λ> "
	}

	// Collect and sort capability names for consistent display
	caps := make([]string, 0, len(r.effContext.Caps))
	for name := range r.effContext.Caps {
		caps = append(caps, name)
	}
	sort.Strings(caps)

	// Format as λ[IO,FS]>
	return fmt.Sprintf("λ[%s]> ", strings.Join(caps, ","))
}

// Start begins the REPL session
func (r *REPL) Start(in io.Reader, out io.Writer) {
	// Create liner instance for readline functionality
	line := liner.NewLiner()
	defer line.Close()

	// Set up history file
	historyFile := filepath.Join(os.TempDir(), ".ailang_history")
	if f, err := os.Open(historyFile); err == nil {
		_, _ = line.ReadHistory(f) // Ignore error as history is optional
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

	// Auto-import prelude for type class instances
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
		prompt := r.getPrompt()
		input, err := line.Prompt(prompt)
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
			r.HandleCommand(input, out)
			continue
		}

		// Process expression through full pipeline
		r.ProcessExpression(input, out)
	}

	// Save history before exiting
	if f, err := os.Create(historyFile); err == nil {
		_, _ = line.WriteHistory(f) // Ignore error as history is optional
		f.Close()
	}
}
