package executor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func containmentTestPolicy(t *testing.T) (*EvalContainment, EvalContainmentPolicy, string) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	app := filepath.Join(root, "app")
	oracle := filepath.Join(root, "oracle")
	sdk := filepath.Join(root, "sdk")
	for _, p := range []string{app, oracle, sdk} {
		if err = os.Mkdir(p, 0700); err != nil {
			t.Fatal(err)
		}
	}
	module := filepath.Join(sdk, "index.js")
	lock := filepath.Join(sdk, "package-lock.json")
	_ = os.WriteFile(module, []byte("test-only"), 0600)
	_ = os.WriteFile(lock, []byte("locked test identity"), 0600)
	h := sha256.Sum256([]byte("locked test identity"))
	p := EvalContainmentPolicy{Schema: "maintenance.containment/1", WorkspaceRoot: app, ProtectedRoots: []string{oracle}, ReadOnlyRoots: []string{sdk}, RuntimeModule: module, RuntimeVersion: "0.0.71", RuntimeLock: lock, RuntimeLockDigest: hex.EncodeToString(h[:]), InferenceDomains: []string{}}
	raw, _ := json.Marshal(p)
	path := filepath.Join(root, "policy.json")
	_ = os.WriteFile(path, raw, 0600)
	sum := sha256.Sum256(raw)
	return &EvalContainment{PolicyPath: path, PolicyDigest: hex.EncodeToString(sum[:])}, p, app
}
func TestEvalContainmentLoadsAndRejectsAlterations(t *testing.T) {
	c, p, app := containmentTestPolicy(t)
	if _, err := LoadEvalContainment(c, app); err != nil {
		t.Fatal(err)
	}
	c.PolicyDigest = "bad"
	if _, err := LoadEvalContainment(c, app); err == nil {
		t.Fatal("accepted changed policy identity")
	}
	c, p, app = containmentTestPolicy(t)
	p.WorkspaceRoot = filepath.Dir(app)
	raw, _ := json.Marshal(p)
	_ = os.WriteFile(c.PolicyPath, raw, 0600)
	h := sha256.Sum256(raw)
	c.PolicyDigest = hex.EncodeToString(h[:])
	if _, err := LoadEvalContainment(c, app); err == nil {
		t.Fatal("accepted wrong workspace")
	}
	c, _, app = containmentTestPolicy(t)
	if _, err := LoadEvalContainment(c, filepath.Dir(app)); err == nil {
		t.Fatal("accepted broader workspace")
	}
}
func TestEvalContainmentCannotUseExtraEnv(t *testing.T) {
	for _, name := range []string{"AILANG_MAINTENANCE_POLICY", "AILANG_MAINTENANCE_POLICY_DIGEST", "AILANG_MAINTENANCE_WRAPPER"} {
		if err := ValidateExtraEnv(map[string]string{name: "override"}); err == nil {
			t.Fatalf("accepted harness-owned %s", name)
		}
	}
}
