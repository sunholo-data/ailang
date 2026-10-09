package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/config"
)

// proxy is a raw UDS relay, not a JSONL transport: the daemon requires a
// WebSocket handshake even over Unix. Connect to precisely the same socket.
func dialDaemon(ctx context.Context, env []string) (*websocket.Conn, error) {
	home, err := daemonHome(env)
	if err != nil {
		return nil, err
	}
	sock := filepath.Join(home, "app-server-control", "app-server-control.sock")
	sock, err = filepath.EvalSymlinks(sock)
	if err != nil {
		return nil, fmt.Errorf("resolve managed Codex daemon socket: %w", err)
	}
	d := websocket.Dialer{NetDialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", sock)
	}}
	c, _, err := d.DialContext(ctx, "ws://localhost/", nil)
	if err != nil {
		return nil, fmt.Errorf("connect to managed Codex daemon: %w", err)
	}
	c.SetReadLimit(16 << 20)
	return c, nil
}

type daemonWebSocketStream struct {
	conn    *websocket.Conn
	pending *bytes.Reader
}

func (s *daemonWebSocketStream) Read(p []byte) (int, error) {
	for s.pending == nil || s.pending.Len() == 0 {
		kind, b, err := s.conn.ReadMessage()
		if err != nil {
			return 0, err
		}
		if kind != websocket.TextMessage {
			return 0, fmt.Errorf("unexpected Codex daemon frame type")
		}
		s.pending = bytes.NewReader(append(b, '\n'))
	}
	return s.pending.Read(p)
}
func (s *daemonWebSocketStream) Write(p []byte) (int, error) {
	if err := s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return 0, err
	}
	if err := s.conn.WriteMessage(websocket.TextMessage, bytes.TrimSpace(p)); err != nil {
		return 0, err
	}
	return len(p), nil
}

// CallDaemonRPC makes quota/attended RPC calls on the same credential owner.
func CallDaemonRPC(ctx context.Context, home, method string, params json.RawMessage) (json.RawMessage, error) {
	c, err := dialDaemon(ctx, []string{"CODEX_HOME=" + home})
	if err != nil {
		return nil, err
	}
	defer c.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = c.SetReadDeadline(deadline)
		_ = c.SetWriteDeadline(deadline)
	}
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = c.Close()
		case <-done:
		}
	}()
	stream := &daemonWebSocketStream{conn: c}
	enc := json.NewEncoder(stream)
	dec := json.NewDecoder(stream)
	if err := enc.Encode(map[string]any{"id": 1, "method": "initialize", "params": map[string]any{"clientInfo": map[string]any{"name": "ailang-mission-quota", "version": "1"}}}); err != nil {
		return nil, err
	}
	for {
		var m daemonMessage
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		if m.ID != nil && *m.ID == 1 {
			if m.Error != nil {
				return nil, fmt.Errorf("Codex daemon initialize failed: %s", m.Error.Message)
			}
			break
		}
	}
	if err := enc.Encode(map[string]any{"method": "initialized"}); err != nil {
		return nil, err
	}
	request := map[string]any{"id": 2, "method": method}
	if len(params) > 0 {
		request["params"] = params
	}
	if err := enc.Encode(request); err != nil {
		return nil, err
	}
	for {
		var m daemonMessage
		if err := dec.Decode(&m); err != nil {
			return nil, err
		}
		if m.ID != nil && *m.ID == 2 {
			if m.Error != nil {
				return nil, fmt.Errorf("Codex daemon %s: %s", method, m.Error.Message)
			}
			if len(m.Result) == 0 {
				return nil, io.ErrUnexpectedEOF
			}
			return m.Result, nil
		}
	}
}

func daemonHome(env []string) (string, error) {
	home := config.CodexHome()
	if env != nil {
		home = ""
		for _, v := range env {
			if strings.HasPrefix(v, "CODEX_HOME=") {
				home = strings.TrimPrefix(v, "CODEX_HOME=")
			}
		}
	}
	if home == "" {
		return "", fmt.Errorf("Codex daemon requires explicit CODEX_HOME")
	}
	if !filepath.IsAbs(home) {
		return "", fmt.Errorf("Codex daemon CODEX_HOME must be absolute")
	}

	return home, nil
}
