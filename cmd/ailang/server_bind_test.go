package main

import (
	"net"
	"testing"
)

// isPortInUse must probe the exact address the server binds: a holder on
// 127.0.0.1 is invisible to a wildcard ":port" probe on macOS
// (M-SERVEAPI-BIND-HOST-CORS M1, serve-api-port-collision triage doc).
func TestIsPortInUse_ExactAddress(t *testing.T) {
	holder, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := holder.Addr().String()
	if !isPortInUse(addr) {
		t.Fatalf("isPortInUse(%s) = false while it is held", addr)
	}
	holder.Close()
	if isPortInUse(addr) {
		t.Fatalf("isPortInUse(%s) = true after release", addr)
	}
}
