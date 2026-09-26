package streamws

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/effects"
)

// AcceptOptions configures the server side of a WebSocket upgrade.
type AcceptOptions struct {
	// Subprotocol, when non-empty, is the one subprotocol the server selects
	// (echoed in Sec-WebSocket-Protocol). Empty selects none.
	Subprotocol string
	// MaxFrameSize is the inbound read limit per message.
	MaxFrameSize int64
}

// IsUpgrade reports whether r asks for a WebSocket upgrade.
func IsUpgrade(r *http.Request) bool {
	return websocket.IsWebSocketUpgrade(r)
}

// Accept upgrades r to a WebSocket and returns it as an
// effects.StreamTransport, so the core can adopt it as an ordinary
// StreamConn (M-SERVEAPI-WS-BRIDGE M1). The caller has ALREADY checked the
// Origin, session cap and auth: CheckOrigin here accepts, because the policy
// lives in the caller and runs before any upgrade. On failure gorilla has
// written the HTTP error response.
func Accept(w http.ResponseWriter, r *http.Request, opts AcceptOptions) (effects.StreamTransport, error) {
	up := websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin:     func(*http.Request) bool { return true },
	}
	if opts.Subprotocol != "" {
		up.Subprotocols = []string{opts.Subprotocol}
	}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	if opts.MaxFrameSize > 0 {
		conn.SetReadLimit(opts.MaxFrameSize)
	}
	return &transport{conn: conn}, nil
}
