package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// --net-timeout must reach the wire. The 30s default was a hard ceiling with
// no flag, so a local model answering in 45s looked like a dead one (Daneel,
// 2026-09-11). Proved both ways against a server that sleeps 300ms: a 50ms
// timeout fails, a 2s timeout succeeds.
func TestSetupNetHandler_TimeoutReachesTheWire(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		_, _ = w.Write([]byte("late"))
	}))
	defer server.Close()

	fetch := func(timeout string) (eval.Value, error) {
		effCtx := effects.NewEffContext(nil)
		effCtx.Grant(effects.NewCapability("Net"))
		if err := setupNetHandler(effCtx, true, "", true, false, timeout); err != nil {
			t.Fatal(err)
		}
		return effects.Call(effCtx, "Net", "httpGet", []eval.Value{&eval.StringValue{Value: server.URL}})
	}

	if _, err := fetch("50ms"); err == nil || !strings.Contains(strings.ToLower(err.Error()), "deadline") && !strings.Contains(strings.ToLower(err.Error()), "timeout") {
		t.Errorf("50ms timeout against a 300ms server: err=%v, want a deadline/timeout error", err)
	}
	v, err := fetch("2s")
	if err != nil {
		t.Fatalf("2s timeout against a 300ms server failed: %v", err)
	}
	if s, ok := v.(*eval.StringValue); !ok || s.Value != "late" {
		t.Errorf("body = %#v, want \"late\"", v)
	}
}

func TestSetupNetHandler_TimeoutRejectsGarbage(t *testing.T) {
	for _, bad := range []string{"soon", "-1s", "0", "30"} {
		effCtx := effects.NewEffContext(nil)
		effCtx.Grant(effects.NewCapability("Net"))
		if err := setupNetHandler(effCtx, false, "", false, false, bad); err == nil {
			t.Errorf("--net-timeout %q accepted", bad)
		}
	}
	// Without the Net capability the flag is inert, not an error.
	if err := setupNetHandler(effects.NewEffContext(nil), false, "", false, false, "soon"); err != nil {
		t.Errorf("flag validated without Net cap: %v", err)
	}
}
