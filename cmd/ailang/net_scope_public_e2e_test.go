package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// M-NET-SCOPE-PUBLIC (#1522) end to end on the checkout binary: with
// serve-api's permissive flags (http + localhost + metadata allowed), a
// function declaring ! {Net[scope=public]} cannot reach a loopback listener,
// while the bare-Net twin in the same program can. Also proves both
// subsumption directions type-check: the public function calls a bare-Net
// helper, and the bare-Net main calls the public function.
const netScopeProg = `module prog
import std/net (httpRequest)
func fetch(u: string) -> string ! {Net} = match httpRequest("GET", u, [], "") {
  Ok(resp) => "BODY:${resp.body}",
  Err(_) => "DENIED"
}
export func publicFetch(u: string) -> string ! {Net[scope=public]} = fetch(u)
export func main() -> () ! {IO, Net} = {
  println(publicFetch(URL));
  println(fetch(URL))
}
`

func TestNetScopePublic_EndToEnd(t *testing.T) {
	bin := buildAilang(t)
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	_, port, _ := net.SplitHostPort(srv.Listener.Addr().String())

	dir := t.TempDir()
	writeAil(t, dir, "prog.ail", strings.ReplaceAll(netScopeProg, "URL", `"http://127.0.0.1:`+port+`/x"`))
	stdout, stderr, code := testutil.RunBounded(t, dir, 60*time.Second, bin, "run",
		"--caps", "IO,Net", "--net-allow-http", "--net-allow-localhost", "--net-allow-metadata", "prog.ail")
	if code != 0 {
		t.Fatalf("exit %d\n%s%s", code, stdout, stderr)
	}
	lines := programLines(stdout)
	if len(lines) < 2 || lines[0] != "DENIED" {
		t.Fatalf("SSRF: Net[scope=public] reached loopback, got %q\n%s", stdout, stderr)
	}
	if lines[1] != "BODY:ok" {
		t.Fatalf("bare-Net control must reach the listener, got %q\n%s", lines[1], stderr)
	}
	if hits.Load() != 1 {
		t.Fatalf("listener hit %d times, want exactly 1 (the bare-Net call)", hits.Load())
	}
}
