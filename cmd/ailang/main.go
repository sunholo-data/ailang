package main

import (
	"embed"
	"flag"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/sunholo-data/ailang/internal/loader"
	"github.com/sunholo-data/ailang/internal/prompt"
	"github.com/sunholo-data/ailang/internal/schema"
	"github.com/sunholo-data/ailang/internal/version"
)

//go:embed all:prompts
var embeddedPrompts embed.FS

// Version, Commit, and BuildTime are aliased from internal/version for
// backward compatibility with existing callers in this package.
var (
	Version   = version.Version
	Commit    = version.Commit
	BuildTime = version.BuildTime

	// Color output
	green   = color.New(color.FgGreen).SprintFunc()
	red     = color.New(color.FgRed).SprintFunc()
	yellow  = color.New(color.FgYellow).SprintFunc()
	cyan    = color.New(color.FgCyan).SprintFunc()
	blue    = color.New(color.FgBlue).SprintFunc()
	magenta = color.New(color.FgMagenta).SprintFunc()
	bold    = color.New(color.Bold).SprintFunc()

	// Global flags
	_ = false // quietMode placeholder for future use
)

func main() {
	// Platform backends behind the core's registration seams (platform_init.go).
	registerPlatform()

	// Set embedded filesystem for prompts (bundled in binary) — one FS carries
	// every kind, so `ailang prompt`, `agent-prompt` and `devtools-prompt`
	// all work from anywhere.
	prompt.SetEmbeddedFS(embeddedPrompts)

	var (
		versionFlag             = flag.Bool("version", false, "Print version information")
		helpFlag                = flag.Bool("help", false, "Show help")
		learnFlag               = flag.Bool("learn", false, "Enable learning mode (collect training data)")
		traceFlag               = flag.Bool("trace", false, "Enable execution tracing")
		compactFlag             = flag.Bool("compact", false, "Use compact JSON output")
		quietFlag               = flag.Bool("quiet", false, "Suppress progress messages (only show program output)")
		binopShimFlag           = flag.Bool("experimental-binop-shim", false, "Enable experimental operator shim")
		failOnShimFlag          = flag.Bool("fail-on-shim", false, "Fail if operator shim would be used (CI mode)")
		requireLoweringFlag     = flag.Bool("require-lowering", false, "Require operator lowering pass")
		trackInstantiationsFlag = flag.Bool("track-instantiations", false, "Track and dump polymorphic type instantiations")
		maxRecursionDepthFlag   = flag.Int("max-recursion-depth", 10000, "Maximum recursion depth (default: 10000)")
		noMonoFlag              = flag.Bool("no-mono", false, "Disable monomorphization (emergency escape hatch)")
		debugCompileFlag        = flag.Bool("debug-compile", false, "Show compilation statistics (specialization counts, etc.)")
		strictSyntaxFlag        = flag.Bool("strict-syntax", false, "Disable syntactic sugar (require canonical syntax)")
	)

	flag.Parse()

	// The global flags the dispatch table's closures need (repl, watch).
	globals = globalFlags{
		learn:               *learnFlag,
		trace:               *traceFlag,
		strictSyntax:        *strictSyntaxFlag,
		binopShim:           *binopShimFlag,
		failOnShim:          *failOnShimFlag,
		requireLowering:     *requireLoweringFlag,
		trackInstantiations: *trackInstantiationsFlag,
		noMono:              *noMonoFlag,
		debugCompile:        *debugCompileFlag,
		maxRecursionDepth:   *maxRecursionDepthFlag,
	}

	// Language commands (run/check/fmt/...) open no state database and print
	// nothing to stderr that the program itself did not print. The observatory
	// health check and the stale-binary probe run only for platform commands
	// (M-V1-SIMPLIFY-S1 M6). The membership list is the table's Group attribute
	// since S5 M1; platformStartup holds the whole decision so a test can count
	// the probes instead of reading this file.
	runStaleProbe := platformStartup(flag.Args())

	// Set binary version for stdlib compatibility check
	// Version is set by ldflags at build time (e.g., "v0.4.8")
	loader.BinaryVersion = Version

	// Set compact mode globally if flag is provided
	if *compactFlag {
		schema.SetCompactMode(true)
	}

	// Set quiet mode globally (placeholder for future use)
	_ = *quietFlag

	if *versionFlag {
		printVersion()
		return
	}

	if *helpFlag || flag.NArg() == 0 {
		printHelp()
		return
	}

	// Check for stale binary (DX: prevents confusion when testing changes).
	// After the --version/--help short circuits, exactly as before.
	runStaleProbe()

	command := flag.Arg(0)
	if err := guardEvalRemoteRead(command, flag.Args()[1:], os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	dispatchCommand(command, flag.Args()[1:])
}

// dim renders text in the terminal's dim attribute.
func dim(s string) string {
	return "\033[2m" + s + "\033[0m"
}
