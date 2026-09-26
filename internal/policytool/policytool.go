// Package policytool is the narrow, typed tool endpoint an ailang_only agent
// reaches the filesystem and the ailang CLI through (M-EXECUTOR-POLICY-
// HARDENING M4, D4).
//
// The pi adapter (.pi/extensions/ailang-exec.ts) forwards STRUCTURED requests
// — never raw shell text, never an argv it composed itself — to
// `ailang policy-tool`, which resolves the operator policy once (from the
// launcher-owned path, not from the request) and dispatches on an explicit
// table:
//
//   - read / write / edit are root-anchored through internal/fileguard: an
//     outside path, a symlink out, or a traversal fails inside the syscall.
//   - the CLI operations (check, fmt, iface, test, …) have a per-command
//     schema: which positional kinds and which flags are legal. The argv is
//     BUILT here from validated fields; a flag or path the schema does not
//     name is a refusal, and a subcommand with no schema is refused even when
//     the policy's cli_allow lists it — adding a CLI command never adds a
//     restricted tool by itself.
//
// Every refusal is a named reason in the response; nothing falls back to a
// looser path.
package policytool

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/fileguard"
	"github.com/sunholo-data/ailang/internal/policy"
)

// Request is one tool call. Op selects the handler; the other fields are
// that handler's typed inputs — unknown or missing fields are refusals.
type Request struct {
	Op string `json:"op"`
	// read / write / edit / check / fmt / tree / policy_check / design_quorum
	Path string `json:"path,omitempty"`
	// write
	Content string `json:"content,omitempty"`
	// edit: OldText must occur exactly once
	OldText string `json:"old_text,omitempty"`
	NewText string `json:"new_text,omitempty"`
	// iface / pkg_docs
	Module string `json:"module,omitempty"`
	// docs_search / examples / builtins
	Query string `json:"query,omitempty"`
	// test: an optional package directory
	Package string `json:"package,omitempty"`
	// Flags are the boolean/valued flags the op's schema admits, by name
	// without dashes ("json": "", "limit": "5").
	Flags map[string]string `json:"flags,omitempty"`
}

// Response is the one shape every op answers with.
type Response struct {
	OK      bool   `json:"ok"`
	Refused string `json:"refused,omitempty"`
	// read
	Content string `json:"content,omitempty"`
	// CLI ops
	Argv     []string `json:"argv,omitempty"`
	ExitCode int      `json:"exit_code,omitempty"`
	Stdout   string   `json:"stdout,omitempty"`
	Stderr   string   `json:"stderr,omitempty"`
	// summary
	Summary *Summary `json:"summary,omitempty"`
}

// Summary is what the adapter shows the model and uses to decide what it
// may ask for: produced in Go from the resolved policy, never parsed from
// TOML by the adapter.
type Summary struct {
	Mode         string   `json:"security_mode"`
	PolicyDigest string   `json:"policy_digest"`
	Sandbox      string   `json:"fs_sandbox"`
	Caps         []string `json:"caps"`
	NetAllow     []string `json:"net_allow"`
	ProcessAllow []string `json:"process_allow"`
	// CLI is the effective subcommand allowlist (the default set when the
	// policy names none) intersected with the ops that have a schema.
	CLI []string `json:"cli"`
	// Ops is every op this endpoint will answer, for the adapter's table.
	Ops       []string `json:"ops"`
	TimeoutMs int64    `json:"timeout_ms"`
}

// Host is the resolved endpoint for one policy.
type Host struct {
	policyPath string
	res        *policy.Resolved
	root       *fileguard.Root
	// run executes an ailang CLI argv in dir; injected for tests.
	run func(dir string, argv []string) (stdout, stderr string, code int)
}

// Open resolves the policy at policyPath and opens the sandbox root. A policy
// without FS has no root: file ops refuse, CLI ops run with no cwd grant.
func Open(policyPath string) (*Host, error) {
	res, _, err := policy.LoadResolved(policyPath)
	if err != nil {
		return nil, err
	}
	h := &Host{policyPath: policyPath, res: res, run: noBinary}
	if res.Root != "" {
		root, err := fileguard.Open(res.Root)
		if err != nil {
			return nil, fmt.Errorf("policy-tool: %w", err)
		}
		h.root = root
	}
	return h, nil
}

// SetBinary names the ailang binary CLI ops execute. The CLI passes its own
// path; the library default refuses so that no caller (a test binary, an
// embedder) ends up executing ITSELF.
func (h *Host) SetBinary(path string) {
	h.run = func(dir string, argv []string) (string, string, int) { return runAilang(path, dir, argv) }
}

// noBinary is the default runner: a named refusal.
func noBinary(string, []string) (string, string, int) {
	return "", "policy-tool: no ailang binary configured for CLI operations (SetBinary)", 1
}

// Close releases the root handle.
func (h *Host) Close() error {
	if h.root != nil {
		return h.root.Close()
	}
	return nil
}

// Resolved is the policy this host serves.
func (h *Host) Resolved() *policy.Resolved { return h.res }

// Dispatch answers one request. It never panics on a malformed request and
// never returns a Go error for a refusal — refusals are data.
func (h *Host) Dispatch(req Request) Response {
	switch req.Op {
	case "summary":
		return Response{OK: true, Summary: h.summary()}
	case "read":
		return h.read(req)
	case "write":
		return h.write(req)
	case "edit":
		return h.edit(req)
	}
	if _, ok := cliSchemas[req.Op]; ok {
		return h.cli(req)
	}
	return refuse("unknown op %q (ops: %s)", req.Op, strings.Join(h.ops(), ", "))
}

func refuse(format string, a ...any) Response {
	return Response{OK: false, Refused: fmt.Sprintf(format, a...)}
}

// ops is every op the host answers.
func (h *Host) ops() []string {
	out := []string{"summary", "read", "write", "edit"}
	for op := range cliSchemas {
		if h.cliAllowed(op) {
			out = append(out, op)
		}
	}
	sort.Strings(out)
	return out
}

func (h *Host) summary() *Summary {
	var cli []string
	for op := range cliSchemas {
		if h.cliAllowed(op) {
			cli = append(cli, op)
		}
	}
	sort.Strings(cli)
	return &Summary{
		Mode:         h.res.Mode,
		PolicyDigest: h.res.Digest,
		Sandbox:      h.res.Root,
		Caps:         h.res.Effects,
		NetAllow:     h.res.NetAllow,
		ProcessAllow: h.res.ProcessAllow,
		CLI:          cli,
		Ops:          h.ops(),
		TimeoutMs:    h.res.Timeout.Milliseconds(),
	}
}
