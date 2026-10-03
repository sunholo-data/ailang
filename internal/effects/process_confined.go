package effects

import (
	"fmt"
	"os"
	"path/filepath"
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
//     external diff/textconv forced off, --no-optional-locks so status never
//     writes the index;
//   - repository discovery is pinned to the sandbox (security audit
//     2026-10-01 F-A1): GIT_CEILING_DIRECTORIES is the sandbox's PARENT, so
//     a `.git` at the sandbox root (the clone-root case) is found but git
//     never walks above the sandbox into an ancestor repository, and
//     safe.bareRepository=explicit stops the sandbox itself (agent-writable
//     when it is not a clone root) from being taken as a planted bare repo;
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
// then `--` then pathspecs) are accepted, and the flags always emitted
// right after the subcommand.
type gitSchema struct {
	flags       map[string]gitFlag
	positionals bool
	forced      []string
}

// noDiffDrivers is forced on every subcommand that can render a diff: no
// external diff program and no textconv filter, whatever the config or
// .gitattributes say. (`-c diff.external=` alone is not enough: git treats
// the empty value as a program named "" and every patch diff dies with
// "external diff died".)
var noDiffDrivers = []string{"--no-ext-diff", "--no-textconv"}

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
	), positionals: true, forced: noDiffDrivers},
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
	), positionals: true, forced: noDiffDrivers},
}

// gitHooksDisabledPath is the core.hooksPath of every confined git: a path
// UNDER a character device. Git looks hooks up as <hooksPath>/<name>, and
// any lookup below /dev/null fails with ENOTDIR on every POSIX system —
// nobody, root included, can create a directory there, so no hook can ever
// be planted at this path (unlike a merely-nonexistent path such as
// /nonexistent, which a root-running container could mkdir). An empty value
// is not used: what `core.hooksPath=` means has varied across git versions.
// Restricted mode, the only user of this adapter, is refused on Windows.
const gitHooksDisabledPath = "/dev/null/ailang-hooks-disabled"

// gitHardening is prepended to every confined argv.
var gitHardening = []string{
	"--no-optional-locks",
	"-c", "core.fsmonitor=false",
	"-c", "core.hooksPath=" + gitHooksDisabledPath,
	"-c", "core.pager=cat",
	"-c", "diff.external=",
	"-c", "diff.noprefix=false",
	"-c", "protocol.allow=never",
	"-c", "safe.bareRepository=explicit",
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
	out = append(out, sc.forced...)
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
			if !sc.positionals || !reRev.MatchString(tok) || hasDotDotSegment(tok) {
				return nil, fmt.Errorf("%q is not a revision git %s accepts", tok, sub)
			}
			out = append(out, tok)
		}
	}
	return out, nil
}

// hasDotDotSegment reports whether a revision token has a `..` path
// segment (`d/../../etc/hosts`). A rev range (`A..B`) has none; a path that
// climbs does. Outside a repository `git diff <a> <b>` is an implicit
// --no-index diff of two PATHS, so a rev-shaped token must never be able to
// name a file above the sandbox.
func hasDotDotSegment(tok string) bool {
	for _, seg := range strings.Split(tok, "/") {
		if seg == ".." {
			return true
		}
	}
	return false
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

// gitCeiling returns the GIT_CEILING_DIRECTORIES value that pins repository
// discovery to sandbox (security audit 2026-10-01 F-A1).
//
// Git never chdirs up INTO a ceiling directory, and a ceiling equal to the
// cwd is ignored (verified against git 2.54: ceiling == cwd still walks up
// and leaks the parent repository). So the ceiling is the sandbox's PARENT:
// git checks the sandbox itself (finding `.git` in the clone-root case) and
// stops there; from a sandbox that is a repo SUBDIRECTORY discovery fails
// with "not a git repository".
//
// The parent is computed from the symlink-resolved sandbox — the path git
// sees as its cwd (macOS: /tmp → /private/tmp) — rather than relying on
// git's own resolution of the ceiling. GIT_CEILING_DIRECTORIES is a
// colon-separated list, so a path containing ':' cannot be expressed and is
// refused rather than split into ceilings that do not bind.
func gitCeiling(sandbox string) (string, error) {
	if sandbox == "" {
		return "", fmt.Errorf("confined git needs an fs_sandbox: without one repository discovery starts at the executor's cwd")
	}
	abs, err := filepath.Abs(sandbox)
	if err != nil {
		return "", fmt.Errorf("fs_sandbox %q: %w", sandbox, err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("fs_sandbox %q: %w", sandbox, err)
	}
	parent := filepath.Dir(real)
	if strings.ContainsRune(parent, filepath.ListSeparator) {
		return "", fmt.Errorf("fs_sandbox %q: its parent %q contains %q, which GIT_CEILING_DIRECTORIES cannot express", sandbox, parent, filepath.ListSeparator)
	}
	return parent, nil
}

// confinedEnv is the child's environment: the current process environment
// (already an allowlist in a restricted worker) minus every GIT_* variable,
// plus the hardening: no global/system config, no prompts, no pager, and
// discovery bounded at the sandbox (gitCeiling).
func confinedEnv(sandbox string) ([]string, error) {
	ceiling, err := gitCeiling(sandbox)
	if err != nil {
		return nil, err
	}
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
		"GIT_CEILING_DIRECTORIES="+ceiling,
	), nil
}
