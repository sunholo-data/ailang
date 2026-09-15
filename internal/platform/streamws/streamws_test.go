package streamws

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// echoServer upgrades and echoes every frame; when closeAfter > 0 it closes
// the socket with that code after the first frame instead of echoing.
func echoServer(t *testing.T, closeAfter int) *httptest.Server {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			msgType, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if closeAfter > 0 {
				_ = conn.WriteControl(websocket.CloseMessage,
					websocket.FormatCloseMessage(closeAfter, "bye"), time.Now().Add(time.Second))
				return
			}
			if err := conn.WriteMessage(msgType, data); err != nil {
				return
			}
		}
	}))
}

func dial(t *testing.T, srv *httptest.Server) effects.StreamTransport {
	t.Helper()
	tr, err := Open(effects.StreamDialConfig{
		URL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		HandshakeTimeout: 2 * time.Second,
		MaxFrameSize:     1 << 16,
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return tr
}

func TestTransport_EchoTextAndBinary(t *testing.T) {
	srv := echoServer(t, 0)
	defer srv.Close()
	tr := dial(t, srv)
	defer tr.Close()

	if err := tr.Send(effects.StreamFrame{Kind: effects.StreamFrameText, Data: []byte("hi")}); err != nil {
		t.Fatalf("Send text: %v", err)
	}
	f, err := tr.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if f.Kind != effects.StreamFrameText || string(f.Data) != "hi" {
		t.Errorf("text echo: got kind=%d data=%q", f.Kind, f.Data)
	}

	if err := tr.Send(effects.StreamFrame{Kind: effects.StreamFrameBinary, Data: []byte{1, 2, 3}}); err != nil {
		t.Fatalf("Send binary: %v", err)
	}
	f, err = tr.Recv()
	if err != nil {
		t.Fatalf("Recv: %v", err)
	}
	if f.Kind != effects.StreamFrameBinary || len(f.Data) != 3 {
		t.Errorf("binary echo: got kind=%d data=%v", f.Kind, f.Data)
	}
}

func TestTransport_CleanCloseIsStreamCloseError(t *testing.T) {
	srv := echoServer(t, websocket.CloseGoingAway)
	defer srv.Close()
	tr := dial(t, srv)
	defer tr.Close()

	_ = tr.Send(effects.StreamFrame{Kind: effects.StreamFrameText, Data: []byte("x")})
	_, err := tr.Recv()
	var ce *effects.StreamCloseError
	if !errors.As(err, &ce) {
		t.Fatalf("want *effects.StreamCloseError, got %T: %v", err, err)
	}
	if ce.Code != websocket.CloseGoingAway || ce.Reason != "bye" {
		t.Errorf("close error: got code=%d reason=%q", ce.Code, ce.Reason)
	}
}

func TestTransport_AbnormalCloseIsPlainError(t *testing.T) {
	srv := echoServer(t, websocket.CloseInternalServerErr)
	defer srv.Close()
	tr := dial(t, srv)
	defer tr.Close()

	_ = tr.Send(effects.StreamFrame{Kind: effects.StreamFrameText, Data: []byte("x")})
	_, err := tr.Recv()
	if err == nil {
		t.Fatal("expected an error after a 1011 close")
	}
	var ce *effects.StreamCloseError
	if errors.As(err, &ce) {
		t.Errorf("1011 must not be reported as a clean close: %v", err)
	}
}

func TestOpen_HandshakeFailureCarriesHTTPStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := Open(effects.StreamDialConfig{
		URL:              "ws" + strings.TrimPrefix(srv.URL, "http"),
		HandshakeTimeout: 2 * time.Second,
		MaxFrameSize:     1 << 16,
	})
	if err == nil {
		t.Fatal("expected handshake failure")
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("error should carry the HTTP status: %v", err)
	}
}

func TestRegister_InstallsWSTransport(t *testing.T) {
	Register()
	t.Cleanup(func() { effects.RegisterStreamTransport("ws", nil) })

	srv := echoServer(t, 0)
	defer srv.Close()

	// Prove the registration is what the core resolves for a ws:// URL.
	ctx := effects.NewEffContext(nil)
	ctx.Grant(effects.NewCapability("Stream"))
	ctx.Stream = effects.NewStreamContext()
	ctx.Stream.AllowHTTP = true
	ctx.Stream.AllowLocalhost = true
	result, err := effects.StreamConnect(ctx, []eval.Value{
		&eval.StringValue{Value: "ws" + strings.TrimPrefix(srv.URL, "http")},
		&eval.RecordValue{Fields: map[string]eval.Value{}},
	})
	if err != nil {
		t.Fatalf("core could not open through the registered transport: %v", err)
	}
	tagged, ok := result.(*eval.TaggedValue)
	if !ok || tagged.CtorName != "Ok" {
		t.Fatalf("expected Ok(StreamConn), got %v", result)
	}
	_, _ = effects.StreamClose(ctx, []eval.Value{tagged.Fields[0]})
}
