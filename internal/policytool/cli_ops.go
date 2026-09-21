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
}

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
	"fmt":             {cmd: "fmt", field: "Path", kind: kindPath, required: true, flags: map[string]bool{"write": false, "check": false}, needsRoot: true},
	"iface":           {cmd: "iface", field: "Module", kind: kindModule, required: true, flags: map[string]bool{"compact": false}, needsRoot: true},
	"test":            {cmd: "test", field: "Path", kind: kindPath, flags: map[string]bool{"json": false, "allow-skips": false, "no-color": false, "package": true}, needsRoot: true},
	"docs_search":     {cmd: "docs", sub: "search", field: "Query", kind: kindQuery, required: true, flags: map[string]bool{"json": false, "limit": true}},
	"examples_search": {cmd: "examples", sub: "search", field: "Query", kind: kindQuery, required: true, flags: map[string]bool{}},
	"examples_list":   {cmd: "examples", sub: "list", flags: map[string]bool{"tag": true, "status": true}},
	"examples_show":   {cmd: "examples", sub: "show", field: "Module", kind: kindWord, required: true, flags: map[string]bool{}},
	"examples_tags":   {cmd: "examples", sub: "tags", flags: map[string]bool{}},
	"builtins_list":   {cmd: "builtins", sub: "list", flags: map[string]bool{"json": false}},
	"builtins_show":   {cmd: "builtins", sub: "show", field: "Module", kind: kindWord, required: true, flags: map[string]bool{}},
	"pkg_docs":        {cmd: "pkg-docs", field: "Module", kind: kindModule, required: true, flags: map[string]bool{}, needsRoot: true},
	"tree":            {cmd: "tree", field: "Path", kind: kindPath, flags: map[string]bool{}, needsRoot: true},
	"prompt":          {cmd: "prompt", flags: map[string]bool{}},
	"agent_prompt":    {cmd: "agent-prompt", flags: map[string]bool{}},
	"devtools_prompt": {cmd: "devtools-prompt", flags: map[string]bool{}},
	"policy_check":    {cmd: "policy-check", field: "Path", kind: kindPath, required: true, flags: map[string]bool{}, needsRoot: true},
	"axioms":          {cmd: "axioms", flags: map[string]bool{}},
	"version":         {cmd: "version", flags: map[string]bool{}},
	"lock":            {cmd: "lock", flags: map[string]bool{}, needsRoot: true},
	"design_quorum":   {cmd: "design-quorum", field: "Path", kind: kindPath, required: true, flags: map[string]bool{"max-cost-usd": true, "controller-verdict": true, "controller-note": true}, needsRoot: true},
}

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
			argv = append(argv, "--"+name)
			continue
		}
		val := req.Flags[name]
		kind := kindWord
		if name == "package" {
			kind = kindPath
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
	stdout, stderr, code := h.run(dir, argv)
	const capBytes = 64 * 1024
	trim := func(s string) string {
		if len(s) > capBytes {
			return s[:capBytes] + "\n…[truncated]"
		}
		return s
	}
	return Response{OK: code == 0, Argv: argv, ExitCode: code, Stdout: trim(stdout), Stderr: trim(stderr)}
}

// runAilang executes the named ailang binary in its own process group under
// a bounded timeout.
func runAilang(exe, dir string, argv []string) (string, string, int) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, argv...)
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
