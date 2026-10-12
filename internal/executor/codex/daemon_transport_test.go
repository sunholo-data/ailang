package codex

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
)

func TestDaemonTransportValidatesOneTextMessagePerFrame(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		kind          int
		wantErr       bool
	}{
		{"text object", `{"id":2,"result":{"ok":true}}`, websocket.TextMessage, false},
		{"surrounding newline", "\n{\"id\":2,\"result\":{}}\n", websocket.TextMessage, false},
		{"binary rejected", `{"id":2,"result":{}}`, websocket.BinaryMessage, true},
		{"trailing second object rejected", `{"id":2,"result":{}}` + "\n" + `{"id":3,"result":{}}`, websocket.TextMessage, true},
		{"malformed rejected", `{"id":`, websocket.TextMessage, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.WriteMessage(tc.kind, []byte(tc.payload))
			}))
			defer server.Close()
			conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			conn.SetReadLimit(16 << 20)
			message, err := (&daemonWebSocketTransport{conn: conn}).receiveMessage()
			if (err != nil) != tc.wantErr {
				t.Fatalf("message=%+v error=%v wantError=%v", message, err, tc.wantErr)
			}
			if !tc.wantErr && (message.ID == nil || *message.ID != 2) {
				t.Fatalf("message=%+v", message)
			}
		})
	}
}
