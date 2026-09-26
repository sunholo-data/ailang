package apiserver

import (
	"bytes"
	"errors"
	"log"
	"net"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// newListenServer builds a Server with no modules; listen() needs nothing else.
func newListenServer(t *testing.T, cfg Config) *Server {
	t.Helper()
	srv := New(t.TempDir(), cfg)
	t.Cleanup(func() { _ = srv.Close() })
	return srv
}

func TestListenAddress_DefaultLoopback(t *testing.T) {
	t.Setenv("PORT", "")
	srv := newListenServer(t, Config{Port: "0"})
	ln, err := srv.listen()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	ip := ln.Addr().(*net.TCPAddr).IP
	if !ip.IsLoopback() {
		t.Fatalf("default bind with PORT unset = %s, want a loopback address", ip)
	}
}

func TestListenAddress_PortEnvSelectsWildcard(t *testing.T) {
	t.Setenv("PORT", "0")
	srv := newListenServer(t, Config{Port: "0"})
	if got := srv.bindHost(); got != "0.0.0.0" {
		t.Fatalf("bind host with PORT set = %q, want 0.0.0.0", got)
	}
	ln, err := srv.listen()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	if ip := ln.Addr().(*net.TCPAddr).IP; !ip.IsUnspecified() {
		t.Fatalf("listener with PORT set = %s, want the unspecified (wildcard) address", ip)
	}
}

func TestListenAddress_ExplicitBindWins(t *testing.T) {
	probe, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	probe.Close()

	t.Setenv("PORT", "0")
	srv := newListenServer(t, Config{Port: "0", Bind: "::1"})
	ln, err := srv.listen()
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()
	if ip := ln.Addr().(*net.TCPAddr).IP; !ip.Equal(net.IPv6loopback) {
		t.Fatalf("--bind ::1 with PORT set bound %s, want ::1", ip)
	}
}

// A taken port must fail Start before the banner claims a normal launch
// (serve-api-port-collision triage doc).
func TestListen_PortInUseFailsBeforeBanner(t *testing.T) {
	t.Setenv("PORT", "")
	holder, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("pre-bind: %v", err)
	}
	defer holder.Close()
	port := holder.Addr().(*net.TCPAddr).Port

	var buf bytes.Buffer
	prevOut, prevFlags := log.Writer(), log.Flags()
	log.SetOutput(&buf)
	defer func() { log.SetOutput(prevOut); log.SetFlags(prevFlags) }()

	srv := newListenServer(t, Config{Port: strconv.Itoa(port)})
	errc := make(chan error, 1)
	go func() { errc <- srv.Start() }()

	select {
	case err := <-errc:
		if err == nil {
			t.Fatal("Start returned nil on a taken port")
		}
		// Windows reports WSAEADDRINUSE, which is not syscall.EADDRINUSE.
		if runtime.GOOS != "windows" && !errors.Is(err, syscall.EADDRINUSE) {
			t.Fatalf("Start error = %v, want EADDRINUSE", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Start did not fail on a taken port (it is serving on another address)")
	}
	if strings.Contains(buf.String(), "AILANG API Server") {
		t.Fatalf("startup banner printed before the bind failed:\n%s", buf.String())
	}
}
