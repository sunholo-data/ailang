package runner

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/trace"
)

// M-IFC-CROSS-MODULE M3: the gap M-TRACE-LABEL-AWARE documented in std/secret.
// The IFC check used to be per-module, so a secret() value returned by a
// function in ANOTHER USER module reached the importer as an unlabelled
// result, its calls were never stamped for redaction, and a deep trace
// carried the value verbatim. The importer now reads the library's exported
// IFC summary (fetchKey's intrinsic <secret>), so its calls are redaction sites.

const xmodTraceLib = `module lib

import std/secret (secret)

export func fetchKey() -> string ! {Secret} = secret("op://Fake/canary/credential", "cross-module trace test")
`

const xmodTraceApp = `module app

import lib (fetchKey)
import std/string (substring)
import std/fs (writeFile)

func bearer(token: string) -> string ! {} = "Bearer ${token}"

export func main() -> () ! {IO, FS, Secret} {
  let key = fetchKey()
  let header = bearer(key)
  writeFile("OUT_PATH", substring(header, 7, 39));
  println("prefix: ${substring(key, 0, 6)}")
}
`

// runXmodTraceFixture writes the two-module fixture into dir (both modules
// resolve relative to it) and runs app.main with a collector at tier. The
// compile cache lives in dir, so a second call in the same dir is warm.
func runXmodTraceFixture(t *testing.T, root, dir string, tier trace.Tier) (string, []trace.TraceEvent) {
	t.Helper()
	t.Setenv("AILANG_STDLIB_PATH", filepath.Join(root, "std"))
	t.Setenv("AILANG_CACHE_DIR", "")
	t.Chdir(dir)
	// ToSlash: a raw Windows path (C:\Users\...) inside an AILANG string literal is an illegal \U escape.
	app := strings.Replace(xmodTraceApp, "OUT_PATH", filepath.ToSlash(filepath.Join(dir, "out.txt")), 1)
	for name, src := range map[string]string{"lib.ail": xmodTraceLib, "app.ail": app} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
	}
	file := filepath.Join(dir, "app.ail")
	var out bytes.Buffer
	rt, res := terminalFixtureRuntime(t, file, "", strings.NewReader(""), &out)
	ctx := rt.GetEvaluator().GetEffContext().(*effects.EffContext)
	ctx.Grant(effects.Capability{Name: "FS"})
	ctx.Grant(effects.Capability{Name: "Secret"})
	ctx.Secret = &effects.SecretContext{Resolver: fakeSecretResolver{}}
	collector := trace.NewCollectorWithTier(tier)
	ctx.Trace = collector
	if err := ExecuteModuleEntrypoint(rt, ModuleExecParams{
		Filename: file, Iface: res.Interface, Modules: res.Modules, Entry: "main", ArgsJSON: "null",
		NoPrint: true, Quiet: true, PipelineResult: &res,
	}); err != nil {
		t.Fatalf("run: %v", err)
	}
	written, _ := os.ReadFile(filepath.Join(dir, "out.txt"))
	return string(written), collector.Events()
}

func TestTraceRedactsSecretReturnedByAnotherUserModule(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, tier := range []trace.Tier{trace.TierStandard, trace.TierDeep} {
		t.Run(tier.String(), func(t *testing.T) {
			dir := t.TempDir()
			for _, run := range []string{"cold", "warm"} {
				if run == "warm" {
					// The cached caller is served without re-running the IFC
					// check, so its TraceRedact stamps must come from the cache.
					if _, err := os.Stat(filepath.Join(dir, ".ailang", "cache", "compile", "manifest.json")); err != nil {
						t.Fatalf("warm run has no compile cache to serve from: %v", err)
					}
				}
				written, events := runXmodTraceFixture(t, root, dir, tier)
				// Controls: the program still sees the value (redaction is
				// trace-only; main also prints its 6-character prefix), and
				// the trace has the effect surface.
				if written != traceLabelCanary {
					t.Fatalf("%s: program wrote %q, want the resolved value itself", run, written)
				}
				raw, err := json.Marshal(events)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(raw), `"writeFile"`) {
					t.Fatalf("%s: control: no writeFile effect in the %s trace:\n%s", run, tier, raw)
				}
				if n := strings.Count(string(raw), traceLabelCanary); n > 0 {
					t.Fatalf("%s: %d verbatim copies of a secret returned by another user module in the %s trace:\n%s", run, n, tier, raw)
				}
			}
		})
	}
}
