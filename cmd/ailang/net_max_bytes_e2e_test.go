package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/testutil"
)

// --net-max-bytes / AILANG_NET_MAX_BYTES on the checkout binary, for both
// commands that grant Net from the CLI. The cap was a fixed 5 MB, so the
// hosted docparse API (serve-api) failed every user file over 5 MiB fetched by
// signed URL, whatever its plan advertised (2026-11).

const netCapProg = `module api/fetch
import std/net (httpRequest)
import std/bytes (length)

export func fetch(u: string) -> string ! {Net} = match httpRequest("GET", u, [], "") {
  Ok(resp) => "OK:${show(length(resp.bodyBytes))}",
  Err(BodyTooLarge(n)) => "TOO_LARGE:${n}",
  Err(_) => "OTHER_ERROR"
}
`

const netCapRunProg = `module prog
import std/net (httpRequest)
import std/bytes (length)

func fetch(u: string) -> string ! {Net} = match httpRequest("GET", u, [], "") {
  Ok(resp) => "OK:${show(length(resp.bodyBytes))}",
  Err(BodyTooLarge(n)) => "TOO_LARGE:${n}",
  Err(_) => "OTHER_ERROR"
}

export func main() -> () ! {IO, Net} = {
  println(fetch(SMALL));
  println(fetch(BIG))
}
`

const netCapBig = 6 * 1024 * 1024 // over the old fixed 5 MB

func netCapServer(t *testing.T) (small, big string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := 2048
		if r.URL.Path == "/big" {
			n = netCapBig
		}
		_, _ = w.Write([]byte(strings.Repeat("x", n)))
	}))
	t.Cleanup(srv.Close)
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())
	base := "http://127.0.0.1:" + port
	return base + "/small", base + "/big"
}

func TestRunNetMaxBytes_EndToEnd(t *testing.T) {
	bin := buildAilang(t)
	small, big := netCapServer(t)
	dir := t.TempDir()
	prog := strings.NewReplacer("SMALL", `"`+small+`"`, "BIG", `"`+big+`"`).Replace(netCapRunProg)
	writeAil(t, dir, "prog.ail", prog)
	base := []string{"run", "--caps", "IO,Net", "--net-allow-http", "--net-allow-localhost"}

	cases := []struct {
		name, env string
		flags     []string
		want      []string
	}{
		{"default 5MB", "", nil, []string{"OK:2048", "TOO_LARGE:5242880"}},
		{"flag", "", []string{"--net-max-bytes", "1500"}, []string{"TOO_LARGE:1500", "TOO_LARGE:1500"}},
		{"env", "50MB", nil, []string{"OK:2048", "OK:6291456"}},
		{"flag beats env", "50MB", []string{"--net-max-bytes", "1KB"}, []string{"TOO_LARGE:1024", "TOO_LARGE:1024"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(config.EnvNetMaxBytes, tc.env)
			args := append(append(append([]string{}, base...), tc.flags...), "prog.ail")
			stdout, stderr, code := testutil.RunBounded(t, dir, 60*time.Second, bin, args...)
			if code != 0 {
				t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
			}
			if got := programLines(stdout); strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("got %q, want %q\n%s", got, tc.want, stderr)
			}
		})
	}

	// A malformed variable and a flag without Net are refused, not ignored.
	t.Setenv(config.EnvNetMaxBytes, "lots")
	if _, stderr, code := testutil.RunBounded(t, dir, 60*time.Second, bin, append(base, "prog.ail")...); code == 0 || !strings.Contains(stderr, config.EnvNetMaxBytes) {
		t.Errorf("malformed %s: exit %d\n%s", config.EnvNetMaxBytes, code, stderr)
	}
	t.Setenv(config.EnvNetMaxBytes, "")
	writeAil(t, dir, "io.ail", "module io\nexport func main() -> () ! {IO} = println(\"ran\")\n")
	if stdout, stderr, code := testutil.RunBounded(t, dir, 60*time.Second, bin, "run", "--caps", "IO", "--net-max-bytes", "50MB", "io.ail"); code == 0 || !strings.Contains(stderr, "needs --caps Net") || strings.Contains(stdout, "ran") {
		t.Errorf("--net-max-bytes without Net: exit %d\n%s", code, stderr)
	}
}

// --net-max-bytes is refused without --caps Net rather than ignored, and a
// malformed value (flag or variable) stops serve-api before it listens.
func TestServeAPINetMaxBytes_Refusals(t *testing.T) {
	t.Setenv(config.EnvNetMaxBytes, "")
	if err := serveAPICommand([]string{"--caps", "IO", "--net-max-bytes", "50MB", t.TempDir()}); err == nil || !strings.Contains(err.Error(), "needs --caps Net") {
		t.Errorf("without --caps Net: err = %v", err)
	}
	if err := serveAPICommand([]string{"--caps", "Net", "--net-max-bytes", "0", t.TempDir()}); err == nil || !strings.Contains(err.Error(), "--net-max-bytes") {
		t.Errorf("--net-max-bytes 0: err = %v", err)
	}
	t.Setenv(config.EnvNetMaxBytes, "50 parsecs")
	if err := serveAPICommand([]string{"--caps", "Net", t.TempDir()}); err == nil || !strings.Contains(err.Error(), config.EnvNetMaxBytes) {
		t.Errorf("malformed %s: err = %v", config.EnvNetMaxBytes, err)
	}
}

// The configured cap reaches a serve-api route: the env variable raises it
// past the old 5 MB (the docparse case), and the flag overrides the variable.
func TestServeAPINetMaxBytes_EndToEnd(t *testing.T) {
	binary := buildAilang(t)
	small, big := netCapServer(t)

	moduleRoot := t.TempDir()
	modulePath := filepath.Join(moduleRoot, "api", "fetch.ail")
	if err := os.MkdirAll(filepath.Dir(modulePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modulePath, []byte(netCapProg), 0644); err != nil {
		t.Fatal(err)
	}

	serve := func(env string, flags ...string) func(url string) string {
		port := freePort(t)
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		args := append(append([]string{"serve-api", "--port", port, "--caps", "Net"}, flags...), moduleRoot)
		cmd := exec.CommandContext(ctx, binary, args...)
		cmd.Env = append(os.Environ(), config.EnvNetMaxBytes+"="+env)
		var stderr serveAPIDebugBuffer
		cmd.Stderr = &stderr
		if err := cmd.Start(); err != nil {
			cancel()
			t.Fatal(err)
		}
		t.Cleanup(func() {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			cancel()
		})
		base := "http://127.0.0.1:" + port
		waitForServer(t, base+"/api/_health", &stderr)
		return func(url string) string {
			t.Helper()
			payload, _ := json.Marshal(map[string]any{"args": []string{url}})
			resp, err := http.Post(base+"/api/api/fetch/fetch", "application/json", strings.NewReader(string(payload)))
			if err != nil {
				t.Fatalf("POST: %v; stderr:\n%s", err, stderr.String())
			}
			defer resp.Body.Close()
			var body map[string]any
			if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
				t.Fatalf("decode: %v; stderr:\n%s", err, stderr.String())
			}
			s, _ := body["result"].(string)
			if s == "" {
				t.Fatalf("no string result: %v; stderr:\n%s", body, stderr.String())
			}
			return s
		}
	}

	fetchEnv := serve("50MB")
	if got := fetchEnv(big); got != "OK:6291456" {
		t.Errorf("AILANG_NET_MAX_BYTES=50MB, 6MiB body: %s, want OK:6291456", got)
	}
	fetchFlag := serve("50MB", "--net-max-bytes", "1500")
	if got := fetchFlag(small); got != "TOO_LARGE:1500" {
		t.Errorf("--net-max-bytes 1500 over env 50MB, 2KB body: %s, want TOO_LARGE:1500", got)
	}
	fetchDefault := serve("")
	if got := fetchDefault(big); got != "TOO_LARGE:5242880" {
		t.Errorf("default, 6MiB body: %s, want TOO_LARGE:5242880", got)
	}
}
