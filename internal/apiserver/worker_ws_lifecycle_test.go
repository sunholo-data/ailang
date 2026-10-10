package apiserver

import (
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
)

func assertWorkerWSInternalClose(t *testing.T, f *wsFixture, path string) {
	t.Helper()
	client := f.mustDial(t, path)
	_ = client.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, _, err := client.ReadMessage()
	var closeErr *websocket.CloseError
	if !errors.As(err, &closeErr) || closeErr.Code != websocket.CloseInternalServerErr {
		t.Fatalf("handler failure should retain route-owned 1011 close, got %v", err)
	}
	waitFor(t, func() bool { f.srv.ws.mu.Lock(); defer f.srv.ws.mu.Unlock(); return len(f.srv.ws.sessions) == 0 })
}

func TestWorkerWS_HandlerFailurePreservesInternalErrorClose(t *testing.T) {
	f := newWSFixture(t, Config{}, "", func(ctx *effects.EffContext) {
		limit := 0
		ctx.SetBudget(effects.NewBudgetContext(map[string]*int{"Stream": &limit}))
	}, "echo")
	assertWorkerWSInternalClose(t, f, "/echo")
}

func TestWorkerWS_BorrowedSourceCleanupPreservesRouteClose(t *testing.T) {
	// Reuse the fixture's real server/engine policy and native transport. The
	// source adapter is created successfully, then a second effect exceeds its
	// budget. Request cleanup must leave the connection for the route's 1011.
	f := newWSFixture(t, Config{}, "", func(ctx *effects.EffContext) {
		limit := 1
		ctx.SetBudget(effects.NewBudgetContext(map[string]*int{"Stream": &limit}))
	}, "echo")
	fixture := filepath.Join(f.srv.basePath, "ws", "worker_failure.ail")
	program := `module ws/worker_failure
import std/stream (StreamConn, sourceOfConn, transmit)
@route("WS", "/worker-failure")
export func fail(client: StreamConn) -> unit ! {Stream} {
 let borrowed = sourceOfConn(client, "client", 1);
 let result = transmit(client, "must not be sent");
 ()
}
`
	if err := os.WriteFile(fixture, []byte(program), 0600); err != nil {
		t.Fatal(err)
	}
	if err := f.srv.LoadModules([]string{fixture}); err != nil {
		t.Fatal(err)
	}
	if err := f.srv.ValidateWSRoutes(); err != nil {
		t.Fatal(err)
	}
	// A new route projection includes the newly loaded fixture; the original
	// server remains valid and idle, so no handler/router mutation races occur.
	server := httptest.NewServer(f.srv.buildRoutes())
	defer server.Close()
	projected := &wsFixture{srv: f.srv, http: server, eff: f.eff}
	assertWorkerWSInternalClose(t, projected, "/worker-failure")
}
