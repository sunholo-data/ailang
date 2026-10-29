package pkg

// authority.go is the package surface's ONE chokepoint for the two
// authorities a confined agent must not have (M-PKG-AUTHORITY-CHOKEPOINT,
// #1720): network egress, and writes under the user's ~/.ailang.
//
// It is the only place in internal/pkg and the package verbs of cmd/ailang
// allowed to build an HTTP client, run git, or resolve the user's home
// directory. authority_boundary_test.go fails CI when any other file does,
// so a new I/O site cannot be unguarded by default — the way #1704's three
// per-function guards left 22 sites open in the same files.

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
)

// Authority is a capability the package surface may need from the host.
type Authority int

const (
	// AuthNetwork is any outbound connection: HTTP, git transport, Pub/Sub.
	AuthNetwork Authority = iota
	// AuthHomeWrite is creating, modifying or deleting anything under ~/.ailang.
	AuthHomeWrite
)

func (a Authority) String() string {
	switch a {
	case AuthNetwork:
		return "network"
	case AuthHomeWrite:
		return "home-write"
	}
	return fmt.Sprintf("authority(%d)", int(a))
}

// Remedies name what the operator does instead. RegistryRemedy is the #1704
// text, pinned by TestRegistryCacheConfinementWrites.
const (
	RegistryRemedy = "provision AILANG_PACKAGE_ROOT or run 'ailang install' outside the sandbox"
	GitRemedy      = "run 'ailang lock' outside the sandbox or provision AILANG_PACKAGE_ROOT"
	OperatorRemedy = "run it outside the sandbox"
)

// ConfinedError is the refusal every gate returns. It is a distinct type so
// a best-effort caller can tell "skipped because confined" from a real
// failure (D7) without matching text.
type ConfinedError struct {
	Authority Authority
	What      string
	Remedy    string
}

func (e *ConfinedError) Error() string {
	return fmt.Sprintf("confined by AILANG_AGENT_POLICY: %s is operator authority — %s", e.What, e.Remedy)
}

// IsConfinedError reports whether err is (or wraps) a confinement refusal.
func IsConfinedError(err error) bool {
	var ce *ConfinedError
	return errors.As(err, &ce)
}

// Confined reports whether the host has installed an agent policy.
func Confined() bool { return config.AgentPolicy() != "" }

// Require returns the canonical refusal when confined, nil otherwise. An
// empty remedy means RegistryRemedy. This is the single source of the text.
func Require(a Authority, what, remedy string) error {
	if !Confined() {
		return nil
	}
	if remedy == "" {
		remedy = RegistryRemedy
	}
	return &ConfinedError{Authority: a, What: what, Remedy: remedy}
}

// RefuseIfConfined is the #1704 spelling of Require(AuthNetwork, what, "").
func RefuseIfConfined(what string) error { return Require(AuthNetwork, what, "") }

// --- network: HTTP ---------------------------------------------------------

// gatedTransport refuses every request under confinement, so a request built
// at a call site that forgot its own pre-check is still gated.
type gatedTransport struct {
	what string
	base http.RoundTripper
}

func (g *gatedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := Require(AuthNetwork, g.what, ""); err != nil {
		if req.Body != nil {
			req.Body.Close() // RoundTripper contract: always close the body
		}
		return nil, err
	}
	return g.base.RoundTrip(req)
}

// HTTPClient is the ONLY *http.Client package code may hold. what names the
// operation in the refusal ("querying the registry API").
func HTTPClient(what string, timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: &gatedTransport{what: what, base: http.DefaultTransport}}
}

// --- network + home-write: git ----------------------------------------------

// gitNetworkSubcommands open a transport (or can: `remote update`).
var gitNetworkSubcommands = map[string]bool{
	"clone": true, "fetch": true, "ls-remote": true, "pull": true,
	"push": true, "submodule": true, "remote": true, "archive": true,
}

// gitReadOnlySubcommands never write the repository they run in.
var gitReadOnlySubcommands = map[string]bool{"rev-parse": true, "cat-file": true}

func gitSubcommand(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

func gitNeedsNetwork(args []string) bool { return gitNetworkSubcommands[gitSubcommand(args)] }
func gitReadOnly(args []string) bool     { return gitReadOnlySubcommands[gitSubcommand(args)] }

// gitCommand builds the hardened invocation: ext:: transports disabled and
// no credential prompt. Callers put `--` before any positional URL.
func gitCommand(dir string, args ...string) *exec.Cmd {
	full := append([]string{"-c", "protocol.ext.allow=never"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	return cmd
}

// Git runs git for the package surface. A network subcommand requires
// AuthNetwork; any other invocation in a directory under ~/.ailang that is
// not read-only requires AuthHomeWrite. Stdout is returned; stderr streams
// to the user, as git's progress always has.
func Git(dir string, args ...string) ([]byte, error) {
	what := "running 'git " + gitSubcommand(args) + "'"
	if gitNeedsNetwork(args) {
		if err := Require(AuthNetwork, what, GitRemedy); err != nil {
			return nil, err
		}
	}
	if !gitReadOnly(args) && underUserDir(dir) {
		if err := Require(AuthHomeWrite, what+" in the git cache", GitRemedy); err != nil {
			return nil, err
		}
	}
	cmd := gitCommand(dir, args...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

// --- home-write: ~/.ailang ---------------------------------------------------

// UserDir computes ~/.ailang/<sub> without creating anything.
func UserDir(sub string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".ailang", filepath.FromSlash(sub)), nil
}

// EnsureUserDir requires AuthHomeWrite, then creates ~/.ailang/<sub>.
func EnsureUserDir(sub string) (string, error) {
	if err := Require(AuthHomeWrite, "writing ~/.ailang/"+sub, ""); err != nil {
		return "", err
	}
	dir, err := UserDir(sub)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// underUserDir reports whether dir is ~/.ailang or inside it. An empty dir
// (the process cwd) is not.
func underUserDir(dir string) bool {
	if dir == "" {
		return false
	}
	root, err := UserDir("")
	if err != nil {
		return true // cannot prove it is outside: treat as inside
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return true
	}
	rel, err := filepath.Rel(root, abs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ensureDir creates dir, requiring AuthHomeWrite when it lies under ~/.ailang.
// A directory elsewhere (a test's temp cache, an operator's --bin-dir) is not
// home authority.
func ensureDir(dir, what, remedy string) error {
	if underUserDir(dir) {
		if err := Require(AuthHomeWrite, what, remedy); err != nil {
			return err
		}
	}
	return os.MkdirAll(dir, 0755)
}

// EnsureRegistryCacheDir creates the writable HOME registry cache for operators.
// Package reads must use RegistryCacheDir, which does not create anything.
func EnsureRegistryCacheDir() (string, error) {
	if err := Require(AuthHomeWrite, "writing the registry cache", ""); err != nil {
		return "", err
	}
	return EnsureUserDir("cache/registry")
}
