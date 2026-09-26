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

	// Group is where the command is FILED: "" puts it in the visible top
	// level, groupDev / groupOps / groupEval put it in that group. S5 M2
	// made this the real grouping attribute; M1 had overloaded it with the
	// language-vs-platform split, which is a different question with a
	// different consumer (see Language below) and now has its own field.
	//
	// Filing a command in a group does NOT move its spelling. D1 is the
	// binding constraint of this sprint: 593 fleet references to `ailang
	// messages`, 332 to `ailang coordinator`, 216 to `ailang chains` live in
	// launchd drivers, make targets, skills and workflows, and not one of
	// them may have to change. So every grouped command keeps its top-level
	// route AND gains `ailang <group> <name>`; the group only decides which
	// help list it appears in.
	Group string

	// Language marks the commands a user or agent runs to work WITH AILANG,
	// as opposed to operating the fleet. They must stay quiet and
	// dependency-free at startup: no observatory DB stat, no git probe
	// (M-V1-SIMPLIFY-S1 M6). This is a CONTRACT, not a presentation choice —
	// which is why it is no longer spelled as a Group value. A language
	// command can be filed anywhere (`disasm` is a dev command and a
	// language command); the two facts are independent.
	// commands_table_test.go pins the membership against the pre-S5
	// isLanguageCommand switch, route by route.
	Language bool

	// Hidden removes the row from EVERY help list, including its own group's.
	// It is for routes that exist only so a caller keeps working:
	// `internal-dump-iface` (which internal/pkg/iface_subprocess.go execs)
	// and the bare pkg verbs that `ailang pkg <verb>` supersedes.
	Hidden  bool
	Summary string
	Run     func(args []string) error
}

// The groups. A group is a hidden drawer, not a rename: `ailang ops messages`
// and `ailang messages` are the same route, and `ailang <group> --help` is how
// a name that is no longer in the one-screen top level stays discoverable.
const (
	groupDev  = "dev"
	groupOps  = "ops"
	groupEval = "eval"
	// groupPkg files the bare pkg verbs (`ailang add`, `ailang publish`, ...)
	// that `ailang pkg <verb>` now supersedes. Its rows are all Hidden: the
	// canonical listing is printPkgHelp, generated from pkgSubcommands().
	groupPkg = "pkg"
)

// dispatchGroups are the groups that get an `ailang <group> ...` entry point
// and a generated `ailang <group> --help`. groupPkg is absent because `pkg`
// predates the group machinery and keeps its own handler and help.
var dispatchGroups = []string{groupDev, groupOps, groupEval}

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
	if len(args) == 0 || invocationIsLanguage(args) {
		return func() {}
	}
	probeObservatoryHealth()
	return probeStaleBinary
}

// invocationIsLanguage reports whether this argv reaches a language command,
// following ONE group hop.
//
// The hop matters because S5 M2 gave every language command a second spelling.
// `ailang builtins` and `ailang dev builtins` run the same code, so they must
// make the same startup decision — and without the hop the second one stats
// the observatory DB and shells out to git, because it sees the word "dev".
// Measured: `ailang dev builtins` emitted two Observatory lines that `ailang
// builtins` did not.
//
// One hop only. Groups do not nest, and an unknown name stays platform, which
// keeps a typo probing exactly as it did before the table.
func invocationIsLanguage(args []string) bool {
	if len(args) == 0 {
		return false
	}
	if isLanguageCommand(args[0]) {
		return true
	}
	if isGroupName(args[0]) && len(args) > 1 {
		if c := lookupGroupMember(args[0], args[1]); c != nil {
			return c.Language
		}
	}
	return false
}

// allCommands is the whole table, assembled once from the per-area files plus
// the two generated sets: the group entry points (`dev` and `ops` — `eval` is
// a real command too, so evalCommands owns its row) and the legacy bare pkg
// verbs.
//
// commandIndex maps every name AND alias to its row. A duplicate route is a
// programming error and panics at startup rather than silently shadowing one.
var (
	allCommands  []Command
	commandIndex map[string]*Command
)

// Both are built in init() rather than in their own initialiser expressions,
// and that is load-bearing, not style. S5 M2 gave the table group entry points
// whose Run closures call lookupGroupMember, which reads allCommands — a
// reference cycle Go rejects at compile time ("initialization cycle for
// allCommands"). init() bodies run after every package variable is
// initialised and are not part of that dependency graph, so the cycle is
// broken without making either symbol a function and rewriting every use.
func init() {
	allCommands = append(allCommands, languageCommands()...)
	allCommands = append(allCommands, evalCommands()...)
	allCommands = append(allCommands, platformCommands()...)
	allCommands = append(allCommands, groupEntryCommands()...)
	allCommands = append(allCommands, pkgLegacyCommands()...)

	commandIndex = make(map[string]*Command, len(allCommands)*2)
	for i := range allCommands {
		c := &allCommands[i]
		for _, n := range append([]string{c.Name}, c.Aliases...) {
			if _, dup := commandIndex[n]; dup {
				panic("ailang: duplicate command route " + n)
			}
			commandIndex[n] = c
		}
	}
}

// helpFallbackCommands are the commands whose own argument parsing REJECTS
// --help: each treats it as a subcommand, a filename or an unknown flag and
// exits non-zero. Measured against bin/ailang-before with
// tools/cli_surface_snapshot.sh, not assumed. For these the dispatcher answers
// --help from the table instead of handing it to the command, so `--help`
// exits 0 at every level. Everything NOT listed here already answers --help
// itself (usually through flag.ExitOnError, which exits 0 on flag.ErrHelp) and
// keeps its own, richer help text.
//
// `daemon`, `pkg` and `chains` are deliberately absent: each answers --help
// inside its own closure, because the same help block also has to serve its
// no-argument path (and, for pkg, list the verbs the group accepts; for
// chains, the four namespaces M-V1-SIMPLIFY-S5 M3 folded in, which the
// table's generic block cannot name).
var helpFallbackCommands = map[string]bool{
	"ast-edit":            true,
	"builtins":            true,
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
	return ok && c.Language
}

// invokedAs is the spelling the user actually typed, before alias resolution.
// dispatchCommand sets it once; runAsTopLevel resets it for a group route.
//
// Exactly one command needs it. S5 M2 absorbed `ai-check` into
// `check --verify`, keeping `ai-check` as an ALIAS of `check` — so the two
// routes share a row and the row's Run has to tell them apart. Reading the
// invoked name from the dispatcher is better than reading os.Args[1], which is
// a global flag rather than the command name whenever one precedes it.
var invokedAs string

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
	invokedAs = name
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

// editDistance is the optimal-string-alignment distance between a and b, over
// bytes: Levenshtein plus an adjacent transposition at cost 1. Swapped letters
// are the commonest typo, and pure Levenshtein charges them 2, which is what
// let "chian" (a swap away from "chain") tie with "bin" and "check" at 3 the
// day `bin` joined the table.
func editDistance(a, b string) int {
	prev2 := make([]int, len(b)+1)
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
			if i > 1 && j > 1 && a[i-1] == b[j-2] && a[i-2] == b[j-1] && prev2[j-2]+1 < cur[j] {
				cur[j] = prev2[j-2] + 1
			}
		}
		prev2, prev, cur = prev, cur, prev2
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

// visibleTopLevel returns the rows `ailang --help` lists: filed in no group and
// not hidden, ordered by name.
//
// The count is the gate Phase 3 exists to close — `commands_top_level` 89 -> at
// most 20 — and commands_groups_test.go asserts both the number and the exact
// membership, so a command cannot drift back into the top level unnoticed.
func visibleTopLevel() []*Command {
	var rows []*Command
	for i := range allCommands {
		c := &allCommands[i]
		if c.Hidden || c.Group != "" {
			continue
		}
		rows = append(rows, c)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	return rows
}

// renderCommandList writes the generated command section of `ailang --help`,
// plus the footer that makes the hidden groups discoverable.
//
// Before S5 this list was hand-written and had drifted: 15 commands the switch
// accepted never appeared in it. M1 generated it from the table; M2 shrank it
// to the visible top level and put the rest behind `ailang dev --help` and
// `ailang ops --help`. Nothing was removed — every grouped command still
// answers to its own name, which is D1.
func renderCommandList(w io.Writer) {
	fmt.Fprintln(w, "Commands:")
	for _, c := range visibleTopLevel() {
		name := c.Name
		if len(c.Aliases) > 0 {
			name += " (" + strings.Join(c.Aliases, ", ") + ")"
		}
		fmt.Fprintf(w, "  %s%s%s\n", cyan(name), helpPad(name, 30), c.Summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "More commands (grouped; each also keeps its own top-level name):")
	for _, g := range dispatchGroups {
		if g == groupEval {
			continue // eval is in the visible list above
		}
		label := "ailang " + g + " --help"
		fmt.Fprintf(w, "  %s%s%s\n", cyan(label), helpPad(label, 30), groupSummaries[g])
	}
	fmt.Fprintln(w)
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
