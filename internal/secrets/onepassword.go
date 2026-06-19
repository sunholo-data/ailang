// Package secrets resolves secret references (e.g. "op://Vault/Item/field") to
// their plaintext values via a pluggable backend. The only backend implemented
// today is the 1Password CLI (`op`).
//
// Design constraints (see design_docs/.../m-secret-effect-remote-approval.md):
//   - References are safe to log/store/diff; resolved VALUES are not. Nothing in
//     this package logs a resolved value, and error paths never embed one.
//   - There is NO silent fallback (CLAUDE.md §2): a failed resolution returns
//     ErrSecretUnavailable so callers treat it as fatal rather than proceeding
//     with an empty/placeholder credential.
package secrets

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Resolver resolves a secret reference to its plaintext value. Implementations
// MUST NOT log or otherwise expose the resolved value, and MUST return a
// structured error (never a silent fallback) when resolution fails.
type Resolver interface {
	Read(ctx context.Context, ref string) (string, error)
}

// Sentinel errors. Callers should test with errors.Is.
var (
	// ErrInvalidRef indicates the reference is not a well-formed op:// URI.
	ErrInvalidRef = errors.New("invalid secret reference")
	// ErrSecretUnavailable indicates the backend could not resolve the ref
	// (CLI missing, auth failure, item not found, etc.). Per AILANG policy
	// there is NO fallback value — callers must treat this as fatal.
	ErrSecretUnavailable = errors.New("secret unavailable")
)

// refScheme is the required prefix for 1Password secret references.
const refScheme = "op://"

// OnePasswordResolver resolves references via the 1Password CLI (`op`). In
// trust model A (v0.26.0) the CLI authenticates with a service-account token
// (OP_SERVICE_ACCOUNT_TOKEN) present in the process environment.
type OnePasswordResolver struct {
	// runner runs the op CLI and returns its stdout. It is injectable so tests
	// can exercise the resolver without a real op binary or vault.
	runner func(ctx context.Context, args ...string) ([]byte, error)
}

// NewOnePasswordResolver returns a resolver backed by the real `op` binary.
func NewOnePasswordResolver() *OnePasswordResolver {
	return &OnePasswordResolver{runner: execOp}
}

// Read resolves ref to its plaintext value. The ref is validated before any CLI
// call so a malformed reference never reaches the backend.
func (r *OnePasswordResolver) Read(ctx context.Context, ref string) (string, error) {
	if err := ValidateRef(ref); err != nil {
		return "", err
	}
	// --no-newline keeps single-line secrets clean; we also trim defensively.
	out, err := r.runner(ctx, "read", "--no-newline", ref)
	if err != nil {
		// op's error on a failed read does not contain the value (it could not
		// be resolved). We surface the ref (safe) but never any stdout.
		return "", fmt.Errorf("%w: %s: %v", ErrSecretUnavailable, ref, err)
	}
	return strings.TrimRight(string(out), "\r\n"), nil
}

// ValidateRef checks that ref is a well-formed 1Password secret reference:
//
//	op://<vault>/<item>/[section/]<field>
//
// At minimum vault, item, and field must be present and non-empty.
func ValidateRef(ref string) error {
	if !strings.HasPrefix(ref, refScheme) {
		return fmt.Errorf("%w: must start with %q, got %q", ErrInvalidRef, refScheme, ref)
	}
	rest := strings.TrimPrefix(ref, refScheme)
	parts := strings.Split(rest, "/")
	if len(parts) < 3 {
		return fmt.Errorf("%w: expected %svault/item/field, got %q", ErrInvalidRef, refScheme, ref)
	}
	for _, p := range parts {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("%w: empty path segment in %q", ErrInvalidRef, ref)
		}
	}
	return nil
}

// execOp runs the real `op` binary, capturing stdout and stderr separately so a
// resolved value (stdout) is never folded into an error message (stderr only).
func execOp(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "op", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("op %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}
