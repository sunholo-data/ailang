package executor

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// EvalContainmentAdmissionError reports an unqualified backend before model setup.
type EvalContainmentAdmissionError struct {
	Backend string
	Reason  string
}

func (e *EvalContainmentAdmissionError) Error() string {
	return "eval containment admission closed for " + e.Backend + ": " + e.Reason
}

// ErrEvalContainmentAdmissionClosed is deliberately unconditional for pi until
// the outer kernel boundary and no-network tool execution can coexist. Direct
// host-side candidate confinement is a separately measured primitive.
var ErrEvalContainmentAdmissionClosed = &EvalContainmentAdmissionError{Backend: "pi", Reason: "macOS nested sandbox_apply is refused; host-side confined tool bridge requires separate qualification"}

// EvalContainment is trusted controller-owned policy, never benchmark agent_env.
// A digest identifies a declaration, not a sandbox qualification. Dispatch must
// separately require actual platform/tool canaries for its admitted boundary.
type EvalContainment struct {
	PolicyPath   string
	PolicyDigest string
}
type EvalContainmentPolicy struct {
	Schema            string   `json:"schema"`
	WorkspaceRoot     string   `json:"workspaceRoot"`
	ProtectedRoots    []string `json:"protectedRoots"`
	ReadOnlyRoots     []string `json:"readOnlyRoots"`
	RuntimeModule     string   `json:"runtimeModule"`
	RuntimeVersion    string   `json:"runtimeVersion"`
	RuntimeLock       string   `json:"runtimeLock"`
	RuntimeLockDigest string   `json:"runtimeLockDigest"`
	InferenceDomains  []string `json:"inferenceDomains"`
}

const EvalContainmentPolicyEnv = "AILANG_MAINTENANCE_POLICY"
const EvalContainmentDigestEnv = "AILANG_MAINTENANCE_POLICY_DIGEST"
const EvalContainmentWrapperEnv = "AILANG_MAINTENANCE_WRAPPER"

func LoadEvalContainment(c *EvalContainment, workspace string) (*EvalContainmentPolicy, error) {
	if c == nil {
		return nil, fmt.Errorf("containment policy is absent")
	}
	if err := containmentPath(c.PolicyPath, false); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(c.PolicyPath)
	if err != nil {
		return nil, err
	}
	if !containmentDigest(raw, c.PolicyDigest) {
		return nil, fmt.Errorf("containment policy digest differs")
	}
	// Canonical controller JSON also rules out duplicate/case-folded field names.
	// Refuse other encodings rather than letting independent parsers disagree.
	var p EvalContainmentPolicy
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&p); err != nil {
		return nil, err
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		return nil, fmt.Errorf("trailing containment policy JSON")
	}
	canonical, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	var compact bytes.Buffer
	if err = json.Compact(&compact, raw); err != nil {
		return nil, err
	}
	if !bytes.Equal(compact.Bytes(), canonical) {
		return nil, fmt.Errorf("containment policy must use canonical field order and exact unique keys")
	}
	if p.Schema != "maintenance.containment/1" || p.RuntimeVersion != "0.0.71" || len(p.ProtectedRoots) == 0 || p.ReadOnlyRoots == nil || p.InferenceDomains == nil {
		return nil, fmt.Errorf("invalid containment schema/runtime/roots")
	}
	if p.WorkspaceRoot != workspace {
		return nil, fmt.Errorf("containment workspace does not match task workspace")
	}
	if err = containmentPath(workspace, true); err != nil {
		return nil, err
	}
	for _, root := range p.ProtectedRoots {
		if err = containmentPath(root, true); err != nil {
			return nil, err
		}
		if containmentWithin(root, workspace) || containmentWithin(workspace, root) {
			return nil, fmt.Errorf("protected source and writable workspace overlap")
		}
	}
	for _, root := range p.ReadOnlyRoots {
		if err = containmentPath(root, false); err != nil {
			if err = containmentPath(root, true); err != nil {
				return nil, err
			}
		}
		if containmentWithin(workspace, root) || containmentWithin(root, workspace) {
			return nil, fmt.Errorf("trusted runtime and writable workspace overlap")
		}
	}
	for _, path := range []string{p.RuntimeModule, p.RuntimeLock, c.PolicyPath} {
		if err = containmentPath(path, false); err != nil {
			return nil, err
		}
		if containmentWithin(workspace, path) {
			return nil, fmt.Errorf("trusted policy/runtime file is writable by candidate")
		}
	}
	lock, err := os.ReadFile(p.RuntimeLock)
	if err != nil {
		return nil, err
	}
	if !containmentDigest(lock, p.RuntimeLockDigest) {
		return nil, fmt.Errorf("sandbox runtime lock identity differs")
	}
	for _, domain := range p.InferenceDomains {
		if domain == "" || strings.ContainsAny(domain, "/*\\:@ \t\n") {
			return nil, fmt.Errorf("inference domain must be an exact hostname")
		}
	}
	return &p, nil
}
func containmentDigest(raw []byte, want string) bool {
	sum := sha256.Sum256(raw)
	return len(want) == 64 && hex.EncodeToString(sum[:]) == want
}
func containmentWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func containmentPath(path string, directory bool) error {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return fmt.Errorf("containment path must be canonical absolute: %q", path)
	}
	for current := path; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("containment symlink rejected: %s", current)
		}
		if current == path && ((directory && !info.IsDir()) || (!directory && !info.Mode().IsRegular())) {
			return fmt.Errorf("unexpected containment path type: %s", path)
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}
