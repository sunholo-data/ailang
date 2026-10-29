package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/sunholo-data/ailang/internal/pkg"
)

type smokeRegistryRequests struct{ reads, writes atomic.Int32 }

func smokeTestRegistry(t *testing.T) (string, *smokeRegistryRequests) {
	t.Helper()
	requests := &smokeRegistryRequests{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			requests.writes.Add(1)
			http.Error(w, "no publish authorized", http.StatusMethodNotAllowed)
			return
		}
		requests.reads.Add(1)
		if r.URL.Path == "/index.json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"packages":[]}`))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return server.URL, requests
}

type smokeCLIProof struct {
	binary, root, author, home string
	inputs                     map[string]string
	content                    string
	requests                   *smokeRegistryRequests
}

func newSmokeCLIProof(t *testing.T) *smokeCLIProof {
	t.Helper()
	proof := &smokeCLIProof{binary: buildAilang(t), root: t.TempDir(), home: t.TempDir(), inputs: make(map[string]string)}
	fixture, err := filepath.Abs("../../internal/pkg/testdata/smoke_staged_lock")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(proof.root, os.DirFS(fixture)); err != nil {
		t.Fatal(err)
	}
	registry, requests := smokeTestRegistry(t)
	proof.requests = requests
	t.Setenv("AILANG_REGISTRY", registry)
	t.Setenv("AILANG_PACKAGE_ROOT", filepath.Join(proof.root, "registry"))
	t.Setenv("AILANG_NO_CACHE", "1")
	t.Setenv("AILANG_STORAGE_MESSAGING", "local")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	// These fixtures have no contracts and must work on Windows without Z3.
	t.Setenv("AILANG_Z3_PATH", filepath.Join(proof.root, "absent-z3.exe"))
	std, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", std)
	proof.author = filepath.Join(proof.root, "a")
	smokePath := filepath.Join(proof.author, pkg.SmokeFile)
	smoke, err := os.ReadFile(smokePath)
	if err != nil {
		t.Fatal(err)
	}
	strict := strings.Replace(string(smoke), `if data == "${source}_ASSET" && own == "STAGED_ROOT"`, `if source == "REGISTRY_SOURCE" && data == "REGISTRY_SOURCE_ASSET" && own == "STAGED_ROOT"`, 1)
	if strict == string(smoke) {
		t.Fatal("CLI registry-byte discriminator was not installed")
	}
	if err := os.WriteFile(smokePath, []byte(strict), 0600); err != nil {
		t.Fatal(err)
	}
	lock := runCLIIn(t, proof.binary, proof.home, proof.author, "lock")
	if lock.exitCode != 0 {
		t.Fatalf("lock: %+v", lock)
	}
	// Prove this invocation rejects the matching same-version local pair.
	// Only the staged registry code AND assets can make its smoke pass.
	local := runCLIIn(t, proof.binary, proof.home, proof.author, "run", "--quiet", "--relax-modules", "--caps", "IO,FS", "--entry", "main", pkg.SmokeFile)
	if local.exitCode != 23 {
		t.Fatalf("local bytes did not hit the discriminator: %+v", local)
	}
	for _, name := range []string{pkg.ManifestFile, pkg.LockFileName} {
		data, err := os.ReadFile(filepath.Join(proof.author, name))
		if err != nil {
			t.Fatal(err)
		}
		proof.inputs[name] = string(data)
	}
	proof.content, err = pkg.ContentHash(proof.author)
	if err != nil {
		t.Fatal(err)
	}
	return proof
}

func (proof *smokeCLIProof) assertUnchanged(t *testing.T) {
	t.Helper()
	for name, want := range proof.inputs {
		after, err := os.ReadFile(filepath.Join(proof.author, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != want {
			t.Fatalf("changed author %s", name)
		}
	}
	content, err := pkg.ContentHash(proof.author)
	if err != nil {
		t.Fatal(err)
	}
	if content != proof.content {
		t.Fatal("changed author source/assets")
	}
	for _, database := range []string{"collaboration.db", "coordinator.db"} {
		if _, err := os.Stat(filepath.Join(proof.home, ".ailang", "state", database)); !os.IsNotExist(err) {
			t.Fatalf("dry-run opened message/coordinator store %s: %v", database, err)
		}
	}
	if proof.requests.writes.Load() != 0 {
		t.Fatalf("unexpected registry publish/write requests: %d", proof.requests.writes.Load())
	}
}

func TestPkgSmokeStaging_QualityAndDryRun(t *testing.T) {
	for _, available := range []bool{true, false} {
		name := "published"
		if !available {
			name = "missing_exact_version_with_newer_cached"
		}
		t.Run(name, func(t *testing.T) {
			proof := newSmokeCLIProof(t)
			if !available {
				base := filepath.Join(proof.root, "registry", "test", "b")
				if err := os.Rename(filepath.Join(base, "0.1.0"), filepath.Join(base, "0.2.0")); err != nil {
					t.Fatal(err)
				}
			}
			for _, mode := range []string{"quality", "dry-run"} {
				t.Run(mode, func(t *testing.T) {
					args := []string{"publish", "--dry-run"}
					if mode == "quality" {
						args = []string{"pkg", "quality", "--json", "."}
					}
					result := runCLIIn(t, proof.binary, proof.home, proof.author, args...)
					if available && result.exitCode != 0 || !available && result.exitCode == 0 {
						t.Fatalf("available=%v code=%d stdout=%s stderr=%s", available, result.exitCode, result.stdout, result.stderr)
					}
					proof.assertUnchanged(t)
					if mode != "quality" {
						return
					}
					var report pkg.QualityReport
					if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
						t.Fatalf("quality stdout must be only JSON: %v\n%s", err, result.stdout)
					}
					if !report.Compile.OK || report.Smoke == nil || report.Smoke.Passed != available {
						t.Fatalf("wrong quality outcome: compile=%+v smoke=%+v", report.Compile, report.Smoke)
					}
					if !available && (!strings.Contains(report.Smoke.Notes, "test/b@0.1.0") || !strings.Contains(report.Smoke.Notes, "AILANG_PACKAGE_ROOT") || len(report.Smoke.Notes) > 2048) {
						t.Fatalf("missing canonical provisioning cause: %+v", report.Smoke)
					}
				})
			}
			if proof.requests.reads.Load() == 0 {
				t.Fatal("registry instrument was blind")
			}
		})
	}
}
