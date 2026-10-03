package policytool

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/policy"
	"github.com/sunholo-data/ailang/internal/proctree"
)

// The CLI operations. Each op has ONE schema: the ailang argv prefix it
// expands to, the kind of each positional it accepts, and the flags it
// admits. The argv is built from validated fields — the caller never
// supplies an argv, so there is nothing to guess about which token might be
// a path.

// argKind is what a positional or flag value must be.
type argKind int

const (
	kindPath   argKind = iota // relative, inside the sandbox root
	kindModule                // module path: std/fs, lib/x — no traversal, not absolute
	kindWord                  // one token: no leading dash, no separators
	kindQuery                 // free text, no leading dash
)

// cliSchema is one op's grammar.
type cliSchema struct {
	cmd, sub string
	// field says which Request field supplies the positional, and its kind.
	field string
	kind  argKind
	// required: the positional must be present.
	required bool
	// flags admitted, by name: true = takes a value (kindWord), false = boolean.
	flags map[string]bool
	// cwd is where the command runs: the sandbox root when the op touches
	// files, else nowhere in particular ("" = inherit).
	needsRoot bool
	// writes lists the sandbox-relative paths the op will write for this
	// request (nil = writes nothing in the sandbox). Each passes the same
	// protection as the write/edit ops BEFORE the child runs (#1554): the
	// child is an unconfined process, so the gate is here or nowhere. The
	// compile cache is not listed — runAilang points it outside the sandbox.
	writes func(req Request) []string
}

// quorumArtifactDir is where `ailang design-quorum` writes its verdict
// artifacts, relative to its cwd (internal/mission/quorum.ArtifactDir; a
// test pins the two together).
const quorumArtifactDir = ".ailang/state/mission-quorum"

// fmtWrites: `fmt --write` rewrites its path in place (atomically, via a
// temp file in the same directory). Without --write it only prints.
func fmtWrites(req Request) []string {
	if _, ok := req.Flags["write"]; ok {
		return []string{req.Path}
	}
	return nil
}

func lockWrites(Request) []string         { return []string{"ailang.lock"} }
func designQuorumWrites(Request) []string { return []string{quorumArtifactDir} }

// gateOnlyCommands execute programs: the only execution route is ailang_run.
var gateOnlyCommands = map[string]bool{"run": true, "exec": true, "repl": true, "replay": true, "watch": true, "select-best": true}

// defaultCLIAllow mirrors the pi extension's CLI_DEFAULT_ALLOW: what an
// ailang_only agent may run when the policy names no cli_allow.
var defaultCLIAllow = []string{
	"check", "ai-check", "iface", "fmt", "test", "docs:search", "examples", "builtins",
	"pkg-docs", "tree", "prompt", "agent-prompt", "devtools-prompt", "policy-check", "axioms", "version",
}

// cliSchemas is the whole restricted CLI surface. A subcommand absent here
// is not reachable through the tool even when cli_allow names it.
var cliSchemas = map[string]cliSchema{
	"check":           {cmd: "check", field: "Path", kind: kindPath, required: true, flags: map[string]bool{"json": false, "quiet": false, "strict-syntax": false}, needsRoot: true},
	"ai_check":        {cmd: "ai-check", field: "Path", kind: kindPath, required: true, flags: map[string]bool{"timeout": true}, needsRoot: true},
	"fmt":             {cmd: "fmt", field: "Path", kind: kindPath, required: true, flags: map[string]bool{"write": false, "check": false}, needsRoot: true, writes: fmtWrites},
	"iface":           {cmd: "iface", field: "Module", kind: kindModule, required: true, flags: map[string]bool{"compact": false}, needsRoot: true},
	"test":            {cmd: "test", field: "Path", kind: kindPath, flags: map[string]bool{"json": false, "allow-skips": false, "no-color": false, "package": true}, needsRoot: true},
	"docs_search":     {cmd: "docs", sub: "search", field: "Query", kind: kindQuery, required: true, flags: map[string]bool{"json": false, "limit": true}},
	"examples_search": {cmd: "examples", sub: "search", field: "Query", kind: kindQuery, required: true, flags: map[string]bool{}},
	"examples_list":   {cmd: "examples", sub: "list", flags: map[string]bool{"tags": true, "status": true}},
	"examples_show":   {cmd: "examples", sub: "show", field: "Module", kind: kindWord, required: true, flags: map[string]bool{}},
	"examples_tags":   {cmd: "examples", sub: "tags", flags: map[string]bool{}},
	"builtins_list":   {cmd: "builtins", sub: "list", flags: map[string]bool{"json": false, "module": true, "query": true, "by-module": false, "by-effect": false, "verbose": false}},
	"builtins_show":   {cmd: "builtins", sub: "show", field: "Module", kind: kindWord, required: true, flags: map[string]bool{}},
	"pkg_docs":        {cmd: "pkg-docs", field: "Module", kind: kindModule, required: true, flags: map[string]bool{}, needsRoot: true},
	"tree":            {cmd: "tree", field: "Path", kind: kindPath, flags: map[string]bool{}, needsRoot: true},
	"prompt":          {cmd: "prompt", flags: map[string]bool{}},
	"agent_prompt":    {cmd: "agent-prompt", flags: map[string]bool{}},
	"devtools_prompt": {cmd: "devtools-prompt", flags: map[string]bool{}},
	"policy_check":    {cmd: "policy-check", field: "Path", kind: kindPath, required: true, flags: map[string]bool{}, needsRoot: true},
	"axioms":          {cmd: "axioms", flags: map[string]bool{}},
	"version":         {cmd: "version", flags: map[string]bool{}},
	"lock":            {cmd: "lock", flags: map[string]bool{}, needsRoot: true, writes: lockWrites},
	"design_quorum":   {cmd: "design-quorum", field: "Path", kind: kindPath, required: true, flags: map[string]bool{"max-cost-usd": true, "controller-verdict": true, "controller-note": true}, needsRoot: true, writes: designQuorumWrites},
}

// flagValueKinds names the valued flags whose value is not a plain word:
// --package is a sandbox path, --module a module path (std/fs), --query free
// text. Every other valued flag takes a kindWord.
var flagValueKinds = map[string]argKind{"package": kindPath, "module": kindModule, "query": kindQuery}

// cliAllowed applies the policy's cli_allow (or the default set) to an op:
// an entry `cmd` or `cmd:sub` admits it. Gate-only commands never are.
func (h *Host) cliAllowed(op string) bool {
	sc, ok := cliSchemas[op]
	if !ok || gateOnlyCommands[sc.cmd] {
		return false
	}
	list := h.res.CLIAllow
	if list == nil {
		list = defaultCLIAllow
	}
	for _, e := range list {
		if e == sc.cmd || (sc.sub != "" && e == sc.cmd+":"+sc.sub) {
			return true
		}
	}
	return false
}

var wordRE = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.@+:-]*$`)
var moduleRE = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_./-]*$`)

// validate checks one value against a kind and returns the token to place
// in argv (for a path: the in-root relative name).
func (h *Host) validate(kind argKind, v string) (string, error) {
	if strings.HasPrefix(v, "-") {
		return "", fmt.Errorf("%q looks like a flag; flags go in the flags field", v)
	}
	switch kind {
	case kindPath:
		if h.root == nil {
			return "", fmt.Errorf("path %q: the policy admits no FS", v)
		}
		rel, err := h.root.Rel(v)
		if err != nil {
			return "", err
		}
		if rel == ".." || strings.HasPrefix(rel, "../") || strings.Contains(rel, "/../") {
			return "", fmt.Errorf("path %q leaves the sandbox", v)
		}
		// The CLI child is a separate, UNCONFINED process: the path handed to
		// it must resolve inside the root NOW (Stat through the handle refuses
		// any component that leaves it) and must not be a symlink at all (a
		// link the child follows later is a link this check cannot vouch for).
		if _, err := h.root.Stat(rel); err != nil {
			return "", fmt.Errorf("path %q: %v", v, err)
		}
		if fi, err := h.root.Lstat(rel); err != nil {
			return "", fmt.Errorf("path %q: %v", v, err)
		} else if fi.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("path %q is a symlink; CLI operations take real files inside the sandbox", v)
		}
		return rel, nil
	case kindModule:
		if !moduleRE.MatchString(v) || strings.Contains(v, "..") {
			return "", fmt.Errorf("module %q is not a module path", v)
		}
		return v, nil
	case kindWord:
		if !wordRE.MatchString(v) || strings.Contains(v, "..") {
			return "", fmt.Errorf("%q is not a simple token", v)
		}
		return v, nil
	case kindQuery:
		if strings.ContainsAny(v, "\x00\n") || len(v) > 4096 {
			return "", fmt.Errorf("query is malformed")
		}
		return v, nil
	}
	return "", fmt.Errorf("unknown kind")
}

// fieldValue reads the positional from the request by schema field name.
func fieldValue(req Request, field string) string {
	switch field {
	case "Path":
		return req.Path
	case "Module":
		return req.Module
	case "Query":
		return req.Query
	}
	return ""
}

// buildArgv turns a validated request into the ailang argv.
func (h *Host) buildArgv(req Request) ([]string, cliSchema, error) {
	sc := cliSchemas[req.Op]
	argv := []string{sc.cmd}
	if sc.sub != "" {
		argv = append(argv, sc.sub)
	}
	if req.Op == "policy_check" {
		// The policy is the launcher's: never the request's.
		argv = append(argv, "--policy", h.policyPath)
	}
	// Flags: only the schema's, values validated as words.
	names := make([]string, 0, len(req.Flags))
	for name := range req.Flags {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		takesValue, ok := sc.flags[name]
		if !ok {
			return nil, sc, fmt.Errorf("op %s does not admit flag --%s (admitted: %s)", req.Op, name, flagNames(sc))
		}
		if !takesValue {
			// A boolean flag is on when present with "" or "true". Any other
			// value is refused rather than ignored: "false" used to mean
			// --write (#1554).
			if v := req.Flags[name]; v != "" && v != "true" {
				return nil, sc, fmt.Errorf("flag --%s is boolean: pass \"\" or \"true\" to set it, omit it to leave it off (got %q)", name, v)
			}
			argv = append(argv, "--"+name)
			continue
		}
		val := req.Flags[name]
		kind, ok := flagValueKinds[name]
		if !ok {
			kind = kindWord
		}
		tok, err := h.validate(kind, val)
		if err != nil {
			return nil, sc, fmt.Errorf("flag --%s: %v", name, err)
		}
		argv = append(argv, "--"+name, tok)
	}
	// Positional.
	if sc.field != "" {
		v := fieldValue(req, sc.field)
		if v == "" {
			if sc.required {
				return nil, sc, fmt.Errorf("op %s requires %s", req.Op, strings.ToLower(sc.field))
			}
		} else {
			tok, err := h.validate(sc.kind, v)
			if err != nil {
				return nil, sc, err
			}
			argv = append(argv, tok)
		}
	}
	// A request carrying fields the op has no use for is a malformed call,
	// not something to silently drop.
	for name, present := range map[string]bool{"path": req.Path != "", "module": req.Module != "", "query": req.Query != "", "content": req.Content != "", "old_text": req.OldText != "", "new_text": req.NewText != "", "package": req.Package != ""} {
		if present && !strings.EqualFold(name, sc.field) && !(name == "package" && req.Op == "test") {
			return nil, sc, fmt.Errorf("op %s does not take %s", req.Op, name)
		}
	}
	if req.Op == "test" && req.Package != "" {
		tok, err := h.validate(kindPath, req.Package)
		if err != nil {
			return nil, sc, fmt.Errorf("package: %v", err)
		}
		argv = append(argv, "--package", tok)
	}
	return argv, sc, nil
}

func flagNames(sc cliSchema) string {
	names := make([]string, 0, len(sc.flags))
	for n := range sc.flags {
		names = append(names, "--"+n)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "none"
	}
	return strings.Join(names, " ")
}

// cli validates, builds and runs one CLI op.
func (h *Host) cli(req Request) Response {
	if gateOnlyCommands[cliSchemas[req.Op].cmd] {
		return refuse("`ailang %s` executes programs; the only execution route is ailang_run", cliSchemas[req.Op].cmd)
	}
	if !h.cliAllowed(req.Op) {
		return refuse("op %s is not in the policy's cli_allow (%s)", req.Op, joinStrings(h.summary().CLI))
	}
	argv, sc, err := h.buildArgv(req)
	if err != nil {
		return refuse("%v", err)
	}
	dir := ""
	if sc.needsRoot {
		if h.root == nil {
			return refuse("op %s needs the sandbox root, and the policy admits no FS", req.Op)
		}
		dir = h.root.Dir()
	}
	if sc.writes != nil {
		for _, p := range sc.writes(req) {
			if why := h.protected(req.Op, p); why != "" {
				return refuse("%s", why)
			}
		}
	}
	stdout, stderr, code := h.run(dir, argv)
	return h.capOutput(req, argv, stdout, stderr, code)
}

// outputLimit is the byte budget for one CLI op's stdout+stderr: the policy's
// max_output_bytes (restricted mode always sets one), else the restricted
// default — the response lands in a model's context, so the tool keeps a
// ceiling even under a trusted_host policy that leaves runs unbounded.
func (h *Host) outputLimit() int64 {
	if h.res.MaxOutputBytes > 0 {
		return h.res.MaxOutputBytes
	}
	return policy.DefaultRestrictedMaxOutputBytes
}

// capOutput applies the output limit to a finished op. A --json op is all or
// nothing: a cut document is unparseable, so over the limit it is a named
// refusal (#1551). Text output is cut — stdout first, then stderr from what
// is left — with a marker naming the limit and the size it came from.
func (h *Host) capOutput(req Request, argv []string, stdout, stderr string, code int) Response {
	limit := h.outputLimit()
	total := int64(len(stdout) + len(stderr))
	if _, isJSON := req.Flags["json"]; isJSON && total > limit {
		r := refuse("op %s produced %d bytes of --json output, over the policy's max_output_bytes (%d); a truncated document would not parse, so nothing is returned — narrow the request%s or raise max_output_bytes in the policy",
			req.Op, total, limit, narrowHint(req.Op))
		r.Argv, r.ExitCode = argv, code
		return r
	}
	budget := limit
	cut := func(s string) string {
		n := int64(len(s))
		if n <= budget {
			budget -= n
			return s
		}
		kept := s[:budget]
		budget = 0
		return kept + fmt.Sprintf("\n…[truncated: %d bytes total, over the policy's max_output_bytes (%d)]", total, limit)
	}
	out := cut(stdout)
	errOut := cut(stderr)
	return Response{OK: code == 0, Argv: argv, ExitCode: code, Stdout: out, Stderr: errOut}
}

// narrowHint names the flags that shrink an op's output, where it has any.
func narrowHint(op string) string {
	switch op {
	case "builtins_list":
		return " (--module std/fs, --query <text>)"
	case "docs_search":
		return " (--limit N)"
	}
	return ""
}

// childEnv is the environment every CLI child runs with: the caller's, with
// AILANG_AGENT_POLICY pinned to this host's policy. A child that sees it
// knows it is confined — `ailang examples` then never resolves a
// cwd-relative corpus an agent could have planted (#1552).
func (h *Host) childEnv() []string {
	prefix := config.EnvAgentPolicy + "="
	env := make([]string, 0, len(os.Environ())+1)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, prefix) {
			env = append(env, kv)
		}
	}
	return append(env, prefix+h.policyPath)
}

// runAilang executes the named ailang binary in its own process group under
// a bounded timeout.
//
// The child's compile and prompt caches go to a private temp dir OUTSIDE the
// sandbox (AILANG_CACHE_DIR), removed when the child exits: by default the
// compiler writes <module dir>/.ailang/cache/compile, so a plain `check`
// or `test` was a write into the sandbox that no fs_deny_write entry
// (`.ailang/**`) could stop — the child is unconfined (#1554).
func runAilang(exe, dir string, argv, env []string) (string, string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cacheDir, err := os.MkdirTemp("", "ailang-policy-tool-cache-*")
	if err != nil {
		return "", "policy-tool: creating the child's private cache dir: " + err.Error(), 1
	}
	defer os.RemoveAll(cacheDir)
	cmd := exec.CommandContext(ctx, exe, argv...)
	cmd.Env = append(append([]string{}, env...), config.EnvCacheDir+"="+cacheDir)
	if dir != "" {
		cmd.Dir = filepath.Clean(dir)
	}
	proctree.Configure(cmd)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if ok := asExitError(err, &ee); ok {
			return out.String(), errb.String(), ee.ExitCode()
		}
		return out.String(), errb.String() + "\n" + err.Error(), 1
	}
	return out.String(), errb.String(), 0
}

func asExitError(err error, target **exec.ExitError) bool {
	ee, ok := err.(*exec.ExitError)
	if ok {
		*target = ee
	}
	return ok
}
