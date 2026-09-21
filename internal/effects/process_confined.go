package effects

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// M-EXECUTOR-POLICY-HARDENING M6 — the confined Process adapter.
//
// Restricted mode admits Process only for entries that have a confined
// schema here. Today that is read-only git — `git:status`, `git:diff`,
// `git:log` — which is what every deployed ailang_only policy asks for.
// A prefix match on the subcommand is not a boundary for git: the repo's
// own config (agent-writable through the sandbox) can name commands
// (core.fsmonitor, diff.external, diff.<driver>.textconv, core.pager,
// core.hooksPath), and read-only subcommands carry flags that reach outside
// the clone (--no-index, --output, -c, --git-dir, -C, --exec-path).
//
// So a confined invocation is built here from a per-subcommand schema, and
// three things hold around it:
//
//   - argv is BUILT: every flag must be one the schema admits, revs must look
//     like revs, pathspecs must stay inside the clone; anything else is a
//     NotAllowed denial before a process exists;
//   - the invocation is hardened: git resolved by absolute path (gitexec),
//     cwd = the sandbox root, the caller's GIT_* stripped from the
//     environment, global/system config disabled, fsmonitor/hooks/pager/
//     external diff forced off with -c, --no-optional-locks so status never
//     writes the index;
//   - `.git/` is read-only to the agent (EffEnv.ProtectGitDir, applied to
//     the FS effect and to the policy-tool), so the repo config is the
//     launcher's clone config and nothing else.
//
// Confined mode is exec-only: spawnProcess and asyncExecProcess have no
// hardened shape and are refused.

// ConfinedProcessEntry reports whether a process_allow entry (`cmd` or
// `cmd:sub`) has a confined schema — the question policy.Resolve asks for
// every entry in restricted mode.
func ConfinedProcessEntry(entry string) bool {
	cmd, sub, ok := strings.Cut(entry, ":")
	if !ok || cmd != "git" {
		return false
	}
	_, has := confinedGitSchemas[sub]
	return has
}

// ConfinedProcessEntries lists the entries restricted mode admits, for
// error messages and docs.
func ConfinedProcessEntries() []string {
	out := make([]string, 0, len(confinedGitSchemas))
	for sub := range confinedGitSchemas {
		out = append(out, "git:"+sub)
	}
	sort.Strings(out)
	return out
}

// gitFlag is one admitted flag: its spelling, whether it takes a value
// (as `--flag=v` or `--flag v` / `-nN` or `-n N`), and how to validate it.
type gitFlag struct {
	name     string
	value    bool
	validate *regexp.Regexp
}

// gitSchema is one subcommand: admitted flags, whether positionals (revs
// then `--` then pathspecs) are accepted.
type gitSchema struct {
	flags       map[string]gitFlag
	positionals bool
}

var (
	reDigits   = regexp.MustCompile(`^[0-9]{1,6}$`)
	reWord     = regexp.MustCompile(`^[A-Za-z0-9_@.:+/ -]{1,200}$`)
	reFormat   = regexp.MustCompile(`^[%A-Za-z0-9()<>:,._\[\] |-]{0,300}$`)
	reUntrack  = regexp.MustCompile(`^(all|normal|no)$`)
	rePorcel   = regexp.MustCompile(`^(v1|v2|1|2)$`)
	reRev      = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./~^@{}-]{0,200}$`)
	rePathspec = regexp.MustCompile(`^[^\x00]{1,1000}$`)
)

func flagsOf(fs ...gitFlag) map[string]gitFlag {
	m := make(map[string]gitFlag, len(fs))
	for _, f := range fs {
		m[f.name] = f
	}
	return m
}

var confinedGitSchemas = map[string]gitSchema{
	"status": {flags: flagsOf(
		gitFlag{name: "--porcelain", value: false},
		gitFlag{name: "--porcelain=", value: true, validate: rePorcel},
		gitFlag{name: "--short"}, gitFlag{name: "-s"},
		gitFlag{name: "--branch"}, gitFlag{name: "-b"},
		gitFlag{name: "--untracked-files=", value: true, validate: reUntrack},
		gitFlag{name: "--no-color"},
	), positionals: true},
	"diff": {flags: flagsOf(
		gitFlag{name: "--stat"}, gitFlag{name: "--name-only"}, gitFlag{name: "--name-status"},
		gitFlag{name: "--cached"}, gitFlag{name: "--staged"}, gitFlag{name: "--no-color"},
		gitFlag{name: "--unified=", value: true, validate: reDigits},
		gitFlag{name: "-U", value: true, validate: reDigits},
	), positionals: true},
	"log": {flags: flagsOf(
		gitFlag{name: "--oneline"}, gitFlag{name: "--stat"}, gitFlag{name: "--name-only"},
		gitFlag{name: "--name-status"}, gitFlag{name: "--no-color"}, gitFlag{name: "--no-merges"},
		gitFlag{name: "-n", value: true, validate: reDigits},
		gitFlag{name: "--max-count=", value: true, validate: reDigits},
		gitFlag{name: "--format=", value: true, validate: reFormat},
		gitFlag{name: "--pretty=", value: true, validate: reFormat},
		gitFlag{name: "--since=", value: true, validate: reWord},
		gitFlag{name: "--until=", value: true, validate: reWord},
		gitFlag{name: "--author=", value: true, validate: reWord},
	), positionals: true},
}

// gitHardening is prepended to every confined argv.
var gitHardening = []string{
	"--no-optional-locks",
	"-c", "core.fsmonitor=false",
	"-c", "core.hooksPath=/dev/null",
	"-c", "core.pager=cat",
	"-c", "diff.external=",
	"-c", "diff.noprefix=false",
	"-c", "protocol.allow=never",
}

// confineGit validates args for one git subcommand and returns the argv to
// run (hardening + subcommand + validated tokens).
func confineGit(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("confined git needs a subcommand (%s)", strings.Join(ConfinedProcessEntries(), ", "))
	}
	sub := args[0]
	sc, ok := confinedGitSchemas[sub]
	if !ok {
		return nil, fmt.Errorf("git %s has no confined schema (restricted mode admits %s)", sub, strings.Join(ConfinedProcessEntries(), ", "))
	}
	out := append(append([]string{}, gitHardening...), sub)
	rest := args[1:]
	afterDashDash := false
	for i := 0; i < len(rest); i++ {
		tok := rest[i]
		switch {
		case afterDashDash:
			if !rePathspec.MatchString(tok) || strings.HasPrefix(tok, "-") || strings.HasPrefix(tok, "/") || tok == ".." || strings.HasPrefix(tok, "../") || strings.Contains(tok, "/../") || strings.HasSuffix(tok, "/..") {
				return nil, fmt.Errorf("pathspec %q must stay inside the clone", tok)
			}
			out = append(out, tok)
		case tok == "--":
			if !sc.positionals {
				return nil, fmt.Errorf("git %s takes no pathspecs", sub)
			}
			afterDashDash = true
			out = append(out, tok)
		case strings.HasPrefix(tok, "-"):
			val, err := matchFlag(sc, rest, &i)
			if err != nil {
				return nil, err
			}
			out = append(out, val...)
		default:
			if !sc.positionals || !reRev.MatchString(tok) {
				return nil, fmt.Errorf("%q is not a revision git %s accepts", tok, sub)
			}
			out = append(out, tok)
		}
	}
	return out, nil
}

// matchFlag validates rest[*i] (advancing *i for a separate value) against
// the schema and returns the token(s) to emit.
func matchFlag(sc gitSchema, rest []string, i *int) ([]string, error) {
	tok := rest[*i]
	// Exact boolean flag.
	if f, ok := sc.flags[tok]; ok && !f.value {
		return []string{tok}, nil
	}
	// `--flag=value`.
	if eq := strings.IndexByte(tok, '='); eq > 0 {
		if f, ok := sc.flags[tok[:eq+1]]; ok && f.value {
			v := tok[eq+1:]
			if !f.validate.MatchString(v) {
				return nil, fmt.Errorf("flag %s: value %q is not admitted", tok[:eq], v)
			}
			return []string{tok}, nil
		}
	}
	// `-nN` / `-UN` attached, or `-n N` / `-U N` separate.
	for _, short := range []string{"-n", "-U"} {
		f, ok := sc.flags[short]
		if !ok || !f.value {
			continue
		}
		if tok == short {
			if *i+1 >= len(rest) {
				return nil, fmt.Errorf("flag %s needs a value", short)
			}
			v := rest[*i+1]
			if !f.validate.MatchString(v) {
				return nil, fmt.Errorf("flag %s: value %q is not admitted", short, v)
			}
			*i++
			return []string{short, v}, nil
		}
		if strings.HasPrefix(tok, short) {
			v := tok[len(short):]
			if !f.validate.MatchString(v) {
				return nil, fmt.Errorf("flag %s: value %q is not admitted", short, v)
			}
			return []string{tok}, nil
		}
	}
	return nil, fmt.Errorf("flag %q is not admitted for this subcommand (admitted: %s)", tok, admittedFlags(sc))
}

func admittedFlags(sc gitSchema) string {
	names := make([]string, 0, len(sc.flags))
	for n := range sc.flags {
		names = append(names, strings.TrimSuffix(n, "="))
	}
	sort.Strings(names)
	return strings.Join(names, " ")
}

// confinedEnv is the child's environment: the current process environment
// (already an allowlist in a restricted worker) minus every GIT_* variable,
// plus the hardening: no global/system config, no prompts, no pager.
func confinedEnv() []string {
	var out []string
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "GIT_") {
			continue
		}
		out = append(out, kv)
	}
	return append(out,
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_TERMINAL_PROMPT=0",
		"GIT_PAGER=cat",
		"GIT_OPTIONAL_LOCKS=0",
	)
}
