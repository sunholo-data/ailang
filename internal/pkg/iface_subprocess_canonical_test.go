// Package pkg_test holds the ONE assertion that cannot live in `package pkg`.
//
// Sprint acceptance criterion M3-A5 is "subprocess bytes == the in-process
// canonical builder's bytes". Satisfying it literally means importing
// internal/pipeline, and `package pkg` cannot: internal/pipeline imports
// internal/pkg (package_resolver.go:12), so an internal test file importing it
// fails with `import cycle not allowed in test`. Measured, iteration 313.
//
// An EXTERNAL test package has no such problem — measured in the same breath,
// the identical import from `package pkg_test` compiles and runs (rc=0). Go
// links the internal and external test files into ONE test binary, so the
// TestMain in iface_subprocess_test.go — which builds the child `ailang` and
// overrides resolveIfaceBinary — still applies here.
//
// The first cut of this milestone compared BuildModuleIface's output against a
// second direct invocation of the SAME `internal-dump-iface` subcommand that
// BuildModuleIface itself shells out to. That is a real assertion (it is the
// sole killer for the cmd.Dir hunk) but it compares the mechanism to itself and
// leaves the plan's actual criterion resting on a transitive argument through a
// different package's test. This file closes that gap directly.
package pkg_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/pipeline"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// TestBuildModuleIface_MatchesCanonicalBuilder is M3-A5 in its literal form:
// the subprocess wrapper's result must equal what pipeline.BuildCanonicalJSON
// produces in-process for the same package directory and module path.
//
// Kills: any divergence between the wrapper's decode/re-encode round trip and
// the canonical serialization — including a wrapper that silently returns a
// zero-valued InterfaceJSON, which `{}` would unmarshal into without error.
func TestBuildModuleIface_MatchesCanonicalBuilder(t *testing.T) {
	const modulePath = "test/pkg/canonical"
	const source = "module test/pkg/canonical\n\nexport func answer() -> int { 42 }\n"

	dir := t.TempDir()
	writeCanonicalFixture(t, dir, modulePath, source)

	ctx := context.Background()

	wantBytes, err := pipeline.BuildCanonicalJSON(ctx, dir, modulePath)
	if err != nil {
		t.Fatalf("pipeline.BuildCanonicalJSON: %v", err)
	}
	// Anti-vacuity floor: an empty or `{}` "want" would make every comparison
	// below trivially satisfiable. Fail loudly rather than pass quietly.
	if len(wantBytes) == 0 {
		t.Fatal("instrument failure: canonical builder returned zero bytes")
	}

	got, err := pkg.BuildModuleIface(ctx, dir, modulePath, pkg.DefaultPublishLimits())
	if err != nil {
		t.Fatalf("BuildModuleIface: %v", err)
	}
	if got == nil {
		t.Fatal("BuildModuleIface returned a nil interface")
	}

	// Second anti-vacuity floor, aimed at the value rather than the bytes:
	// the fixture exports exactly one function, so a wrapper returning a
	// zero-valued struct must not reach the comparison undetected.
	// NOTE: inside t.TempDir() the compiler auto-relaxes MOD010 and reports the
	// module as its full temp-qualified path, so pin the SUFFIX. Asserting
	// equality here would pin the fixture's location rather than the mechanism.
	if !strings.HasSuffix(got.Module, modulePath) {
		t.Fatalf("module = %q, want a name ending in %q", got.Module, modulePath)
	}
	if len(got.Funcs) != 1 || got.Funcs[0].Name != "answer" {
		t.Fatalf("funcs = %+v, want exactly one func named \"answer\"", got.Funcs)
	}

	gotBytes, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("marshal wrapper result: %v", err)
	}

	// Compare as normalized JSON values rather than raw bytes: the canonical
	// builder's output is the ground truth, and re-marshaling a decoded struct
	// is not required to reproduce its whitespace.
	var wantAny, gotAny any
	if err := json.Unmarshal(wantBytes, &wantAny); err != nil {
		t.Fatalf("unmarshal canonical bytes: %v", err)
	}
	if err := json.Unmarshal(gotBytes, &gotAny); err != nil {
		t.Fatalf("unmarshal wrapper bytes: %v", err)
	}
	wantNorm, _ := json.Marshal(wantAny)
	gotNorm, _ := json.Marshal(gotAny)
	if string(wantNorm) != string(gotNorm) {
		t.Fatalf("wrapper output differs from the in-process canonical builder\n got: %s\nwant: %s", gotNorm, wantNorm)
	}
}

func writeCanonicalFixture(t *testing.T, dir, modulePath, source string) {
	t.Helper()
	manifest := "[package]\nname = \"test/pkg\"\nversion = \"0.1.0\"\nedition = \"1\"\n\n[exports]\nmodules = [\"" + modulePath + "\"]\n"
	if err := os.WriteFile(filepath.Join(dir, "ailang.toml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	modFile := filepath.Join(dir, filepath.FromSlash(modulePath)+".ail")
	if err := os.MkdirAll(filepath.Dir(modFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modFile, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}
