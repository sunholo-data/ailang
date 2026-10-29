package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/pkg"
	"github.com/sunholo-data/ailang/internal/testutil"
)

func buildPublishProbe(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "publish-env-probe"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	_, stderr, code := testutil.RunBounded(t, ".", 120*time.Second, "go", "build", "-o", path, "./testdata/publish_env_probe")
	if code != 0 {
		t.Fatalf("probe build: %s", stderr)
	}
	return path
}

func writePublishProbePackage(t *testing.T, root, probe string) {
	t.Helper()
	files := map[string]string{
		"ailang.toml": "[package]\nname=\"test/keyproof\"\nversion=\"0.1.0\"\nedition=\"1\"\n[exports]\nmodules=[\"test/keyproof/model\"]\n[effects]\nmax=[\"IO\",\"Process\"]\n",
		"model.ail":   "module test/keyproof/model\nexport func value() -> int = 42\n",
		"_smoke.ail": fmt.Sprintf(`module _smoke
import std/process (exec)
import std/io (exit)
export func main() -> () ! {IO, Process} {
 match exec(%q, []) {
  Ok(out) => if out.exitCode == 0 then () else exit(44),
  Err(_) => exit(45)
 }
}
`, filepath.ToSlash(probe)),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(root, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPublishChild_RawEnvironmentAndFrozenUpload(t *testing.T) {
	binary, probe := buildAilang(t), buildPublishProbe(t)
	isolatePublishCLI(t)
	std, err := filepath.Abs("../../std")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("AILANG_STDLIB_PATH", std)
	t.Setenv("AILANG_NO_CACHE", "1")
	t.Setenv("AILANG_Z3_PATH", filepath.Join(t.TempDir(), "absent-z3.exe"))
	for _, lane := range []string{"env", "file"} {
		t.Run(lane, func(t *testing.T) { runPublishChildArm(t, binary, probe, lane) })
	}
}

func runPublishChildArm(t *testing.T, binary, probe, lane string) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	writePublishProbePackage(t, root, probe)
	before, err := pkg.ContentHash(root)
	if err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(home, "presence")
	expected, key, args := preparePublishLane(t, home, lane)
	server, uploads, reads := publishProofServer(t, key)
	defer server.Close()
	t.Setenv(config.EnvRegistry, server.URL)
	t.Setenv(config.EnvRegistryValidator, server.URL)
	result := runCLIIn(t, binary, home, root, args...)
	if lane == "file" && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		if !strings.Contains(result.stderr, "unsupported") || uploads.Load() != 0 {
			t.Fatalf("unsupported file lane: %+v", result)
		}
		return
	}
	if result.timedOut || result.exitCode == 0 || uploads.Load() != 1 || !strings.Contains(result.stderr, "synthetic validator refusal") || strings.Contains(result.stdout+result.stderr, key) {
		t.Fatalf("raw child/upload proof failed uploads=%d reads=%d: %+v", uploads.Load(), reads.Load(), result)
	}
	got, err := os.ReadFile(marker)
	if err != nil || string(got) != expected {
		t.Fatalf("raw child was not measured: %s %v", got, err)
	}
	assertPublishAuthorUnchanged(t, root, home, before)
	wantWarnings := 0
	if lane == "env" {
		wantWarnings = 1
	}
	if strings.Count(result.stderr, "is inherited") != wantWarnings {
		t.Fatal("wrong legacy warning count")
	}
}

func preparePublishLane(t *testing.T, home, lane string) (expected, key string, args []string) {
	t.Helper()
	t.Setenv("PUBLISH_PROBE_MARKER", filepath.Join(home, "presence"))
	t.Setenv("PUBLISH_PROBE_REPLACE", "")
	t.Setenv(config.EnvRegistryAPIKey, "")
	expected, key = "KEY_PRESENT", "env-synthetic-publish-key"
	args = []string{"publish"}
	if lane == "env" {
		t.Setenv(config.EnvRegistryAPIKey, key)
	} else {
		expected, key = "KEY_ABSENT", "file-synthetic-publish-key"
		if err := os.Unsetenv(config.EnvRegistryAPIKey); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(home, "publisher-key")
		if err := os.WriteFile(path, []byte(key), 0600); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--key-file", path)
		t.Setenv("PUBLISH_PROBE_REPLACE", path)
	}
	t.Setenv("PUBLISH_PROBE_EXPECT", expected)
	return expected, key, args
}

func publishProofServer(t *testing.T, key string) (*httptest.Server, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	var uploads, reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			reads.Add(1)
			_, _ = io.WriteString(w, `{"packages":[]}`)
			return
		}
		uploads.Add(1)
		if r.Header.Get("X-API-Key") != key {
			t.Error("upload reread or changed selected key")
		}
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), key) {
			t.Error("key leaked into upload artifacts")
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, "synthetic validator refusal: "+key)
	}))
	return server, &uploads, &reads
}

func assertPublishAuthorUnchanged(t *testing.T, root, home, before string) {
	t.Helper()
	after, err := pkg.ContentHash(root)
	if err != nil || before != after {
		t.Fatal("author package changed")
	}
	if _, err := os.Stat(filepath.Join(root, pkg.LockFileName)); !os.IsNotExist(err) {
		t.Fatalf("author lock changed: %v", err)
	}
	for _, db := range []string{"collaboration.db", "coordinator.db"} {
		if _, err := os.Stat(filepath.Join(home, ".ailang", "state", db)); !os.IsNotExist(err) {
			t.Fatalf("failed publish opened store %s", db)
		}
	}
}
