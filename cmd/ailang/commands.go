package main

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/observatory"
)

// Command is one route through the CLI. The table below is the ONLY place a
// command name is written down: main.go looks a name up here and calls Run,
// and `ailang --help` is rendered from the same rows, so the help and the
// dispatcher can no longer disagree (they did for 15 commands before
// M-V1-SIMPLIFY-S5 M1).
//
// Run takes the argument tail (everything after the command name) and returns
// an error. A command that must control its own exit code — `mission`, which
// maps error classes onto exit codes — calls os.Exit inside its closure; every
// other non-nil error is reported as "Error: <err>" on stderr with exit 1,
// which is what each `if err := xCommand(args); err != nil` arm of the old
// switch did.
type Command struct {
	Name    string
	Aliases []string
	// Group is "" for a platform command and groupLanguage for a language
	// one. In M1 that is the whole of its meaning: it carries the
	// language-vs-platform split that isLanguageCommand() held before, and
	// every command is still visible at the top level. S5 M2 turns it into
	// the real grouping attribute (dev/ops/eval) — do not add group names
	// here before then.
	Group   string
	Hidden  bool
	Summary string
	Run     func(args []string) error
}

// groupLanguage marks the commands a user or agent runs to work WITH AILANG,
// as opposed to operating the fleet. They must stay quiet and dependency-free
// at startup: no observatory DB stat, no git probe (M-V1-SIMPLIFY-S1 M6). The
// membership list is a contract — commands_language.go holds it, and
// commands_language_test.go pins it against the pre-S5 isLanguageCommand.
const groupLanguage = "lang"

// globalFlags carries the values of the global flags main parses, for the few
// commands whose behaviour depends on them. main assigns it once after
// flag.Parse, before dispatch; the table's closures read it at call time.
type globalFlags struct {
	learn               bool
	trace               bool
	strictSyntax        bool
	binopShim           bool
	failOnShim          bool
	requireLowering     bool
	trackInstantiations bool
	noMono              bool
	debugCompile        bool
	maxRecursionDepth   int
}

var globals globalFlags

// probeObservatoryHealth and probeStaleBinary are the two startup side effects
// a PLATFORM command runs at startup and a LANGUAGE command must not. They are
// variables so a test can count the calls rather than read main.go
// (commands_probe_test.go).
var (
	// Observatory health check — detect bloated DB early (M-OBS-RETENTION).
	// Fast path: just os.Stat, no DB open unless cleanup is needed.
	probeObservatoryHealth = func() { observatory.CheckHealth(observatory.DefaultDatabasePath()) }
	probeStaleBinary       = func() { checkStaleBinary() }
)

// platformStartup runs the startup probes this invocation is entitled to and
// returns the stale-binary probe for main to run at its own point in the
// sequence (after the --version and --help short circuits, where it has always
// been). args is the positional tail: an empty one, or a language command,
// gets neither probe.
func platformStartup(args []string) (runStaleProbe func()) {
	if len(args) == 0 || isLanguageCommand(args[0]) {
		return func() {}
	}
	probeObservatoryHealth()
	return probeStaleBinary
}

// allCommands is assembled once, from the three per-group files.
var allCommands = func() []Command {
	var out []Command
	out = append(out, languageCommands()...)
	out = append(out, evalCommands()...)
	out = append(out, platformCommands()...)
	return out
}()

// commandIndex maps every name AND alias to its row. Built once; a duplicate
// name is a programming error and panics at startup rather than silently
// shadowing a route.
var commandIndex = func() map[string]*Command {
	idx := make(map[string]*Command, len(allCommands)*2)
	for i := range allCommands {
		c := &allCommands[i]
		for _, n := range append([]string{c.Name}, c.Aliases...) {
			if _, dup := idx[n]; dup {
				panic("ailang: duplicate command route " + n)
			}
			idx[n] = c
		}
	}
	return idx
}()

// helpFallbackCommands are the commands whose own argument parsing REJECTS
// --help: each treats it as a subcommand, a filename or an unknown flag and
// exits non-zero. Measured against bin/ailang-before with
// tools/cli_surface_snapshot.sh, not assumed. For these the dispatcher answers
// --help from the table instead of handing it to the command, so `--help`
// exits 0 at every level. Everything NOT listed here already answers --help
// itself (usually through flag.ExitOnError, which exits 0 on flag.ErrHelp) and
// keeps its own, richer help text.
//
// `daemon` and `pkg` are deliberately absent: each answers --help inside its
// own closure, because the same help block also has to serve its no-argument
// path (and, for pkg, list the verbs the group accepts).
var helpFallbackCommands = map[string]bool{
	"ast-edit":            true,
	"builtins":            true,
	"chains":              true,
	"dashboard":           true,
	"doctor":              true,
	"eval-chains":         true,
	"eval-compare":        true,
	"eval-elo":            true,
	"eval-matrix":         true,
	"eval-report":         true,
	"eval-summary":        true,
	"eval-sweet-spot":     true,
	"eval-trend":          true,
	"internal-dump-iface": true,
	"observatory":         true,
	"pi":                  true,
	"trace":               true,
	"watch":               true,
}

// lookupCommand resolves a name or alias.
func lookupCommand(name string) (*Command, bool) {
	c, ok := commandIndex[name]
	return c, ok
}

// isLanguageCommand reports whether cmd is a language command. Unknown names
// are platform, which keeps the startup probes running for a typo exactly as
// they did before the table.
func isLanguageCommand(cmd string) bool {
	c, ok := lookupCommand(cmd)
	return ok && c.Group == groupLanguage
}

// wantsHelp reports whether the argument tail is a bare help request.
func wantsHelp(args []string) bool {
	return len(args) > 0 && (args[0] == "--help" || args[0] == "-h")
}

// dispatchCommand routes one invocation. It returns only when the command returned
// nil; every other path exits.
func dispatchCommand(name string, args []string) {
	c, ok := lookupCommand(name)
	if !ok {
		unknownCommand(name)
		return // unreachable: unknownCommand exits
	}
	if wantsHelp(args) && helpFallbackCommands[c.Name] {
		printCommandHelp(os.Stdout, c)
		return
	}
	if err := c.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", red("Error"), err)
		os.Exit(1)
	}
}

// unknownCommand prints one line and a suggestion on STDERR and exits 1.
// Nothing goes to stdout: the old default arm dumped 16 KB of printHelp there,
// which buried the error and polluted every `ailang <typo> | jq` pipeline.
func unknownCommand(name string) {
	fmt.Fprintf(os.Stderr, "%s: unknown command '%s'\n", red("Error"), name)
	if s := suggestCommand(name); s != "" {
		fmt.Fprintf(os.Stderr, "Did you mean '%s'? Run 'ailang --help' for the command list.\n", s)
	} else {
		fmt.Fprintln(os.Stderr, "Run 'ailang --help' for the command list.")
	}
	os.Exit(1)
}

// suggestCommand returns the closest route to name, or "" when nothing is
// close enough. Ties break lexicographically so the suggestion is stable.
func suggestCommand(name string) string {
	best, bestDist := "", 1<<30
	for route := range commandIndex {
		d := editDistance(name, route)
		if d < bestDist || (d == bestDist && route < best) {
			best, bestDist = route, d
		}
	}
	// A suggestion has to be closer than "throw the word away": three edits,
	// and fewer edits than the typo has characters.
	if bestDist <= 3 && bestDist < len(name) {
		return best
	}
	return ""
}

// editDistance is the Levenshtein distance between a and b, over bytes.
func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := 0; j <= len(b); j++ {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

// printCommandHelp renders the table's own help for one command. It is what
// `--help` answers with for a command that cannot answer it itself.
func printCommandHelp(w io.Writer, c *Command) {
	fmt.Fprintf(w, "%s - %s\n\n", bold("ailang "+c.Name), c.Summary)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintf(w, "  ailang %s [arguments]\n", c.Name)
	if len(c.Aliases) > 0 {
		fmt.Fprintf(w, "\nAliases: %s\n", strings.Join(c.Aliases, ", "))
	}
	fmt.Fprintf(w, "\nRun '%s' with no arguments for its subcommands, or '%s' for the full command list.\n",
		cyan("ailang "+c.Name), cyan("ailang --help"))
}

// renderCommandList writes the generated command sections of `ailang --help`.
// Hidden rows are omitted; within a section the order is alphabetical, because
// the table has no hand-maintained ordering to preserve.
func renderCommandList(w io.Writer) {
	sections := []struct {
		title string
		group string
	}{
		{"Language commands:", groupLanguage},
		{"Platform commands:", ""},
	}
	for _, s := range sections {
		var rows []*Command
		for i := range allCommands {
			c := &allCommands[i]
			if c.Hidden || c.Group != s.group {
				continue
			}
			rows = append(rows, c)
		}
		if len(rows) == 0 {
			continue
		}
		sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
		fmt.Fprintln(w, s.title)
		for _, c := range rows {
			name := c.Name
			if len(c.Aliases) > 0 {
				name += " (" + strings.Join(c.Aliases, ", ") + ")"
			}
			fmt.Fprintf(w, "  %s%s%s\n", cyan(name), helpPad(name, 30), c.Summary)
		}
		fmt.Fprintln(w)
	}
}

// helpPad returns the spaces that take s to width, with a single space minimum
// so a long name never runs into its summary.
func helpPad(s string, width int) string {
	if len(s) >= width {
		return " "
	}
	return strings.Repeat(" ", width-len(s))
}

// noArgs adapts the many commands that parse flag.Args() themselves and take
// no parameters. The tail is already available to them through the global
// flag package, exactly as it was from the switch.
func noArgs(run func()) func([]string) error {
	return func([]string) error {
		run()
		return nil
	}
}
