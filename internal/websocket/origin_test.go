package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/sunholo-data/ailang/internal/platform/originpolicy"
)

// dialWS opens /ws on ts with the given Origin ("" = none) and query, and
// returns the handshake status (101 on success).
func dialWS(t *testing.T, ts *httptest.Server, origin, query string) int {
	t.Helper()
	u := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws" + query
	h := http.Header{}
	if origin != "" {
		h.Set("Origin", origin)
	}
	conn, resp, err := websocket.DefaultDialer.Dial(u, h)
	if conn != nil {
		conn.Close()
	}
	if resp == nil {
		t.Fatalf("dial %s (Origin %q): no response: %v", u, origin, err)
	}
	return resp.StatusCode
}

func originServer(t *testing.T, listed []string, token string) *httptest.Server {
	t.Helper()
	server, store := setupTestServer(t)
	t.Cleanup(func() { store.Close() })
	server.SetOriginPolicy(originpolicy.New(false, listed, "GET"))
	server.SetToken(token)
	ts := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	t.Cleanup(ts.Close)
	return ts
}

// Cross-site WebSocket hijacking: a page on another origin must not be able
// to open the hub socket (M-SERVER-ORIGIN-POLICY).
func TestWSOrigin_ForeignRefused(t *testing.T) {
	ts := originServer(t, nil, "")
	for _, o := range []string{"https://evil.example", "null", "http://127.0.0.1:1"} {
		if code := dialWS(t, ts, o, ""); code != http.StatusForbidden {
			t.Errorf("Origin %q: status %d, want 403", o, code)
		}
	}
}

func TestWSOrigin_SameOriginListedAndCLIAdmitted(t *testing.T) {
	ts := originServer(t, []string{"http://localhost:3000"}, "")
	for _, o := range []string{ts.URL, "http://localhost:3000", ""} {
		if code := dialWS(t, ts, o, ""); code != http.StatusSwitchingProtocols {
			t.Errorf("Origin %q: status %d, want 101", o, code)
		}
	}
}

// The default policy (no SetOriginPolicy call) is same-origin only, never
// "accept everything".
func TestWSOrigin_DefaultPolicyIsSameOrigin(t *testing.T) {
	server, store := setupTestServer(t)
	defer store.Close()
	ts := httptest.NewServer(http.HandlerFunc(server.HandleWebSocket))
	defer ts.Close()
	if code := dialWS(t, ts, "https://evil.example", ""); code != http.StatusForbidden {
		t.Fatalf("default policy, foreign Origin: status %d, want 403", code)
	}
	if code := dialWS(t, ts, ts.URL, ""); code != http.StatusSwitchingProtocols {
		t.Fatalf("default policy, same origin: status %d, want 101", code)
	}
}

// With COORDINATOR_API_KEY set, the existing external-client contract holds:
// a valid ?token= admits any origin (the website-builder sidecar and the
// documented browser integration), a missing/wrong token does not.
func TestWSOrigin_TokenContract(t *testing.T) {
	ts := originServer(t, nil, "s3cret")
	if code := dialWS(t, ts, "https://portal.example", "?token=s3cret"); code != http.StatusSwitchingProtocols {
		t.Errorf("foreign origin + valid token: status %d, want 101", code)
	}
	if code := dialWS(t, ts, "https://portal.example", "?token=nope"); code != http.StatusForbidden {
		t.Errorf("foreign origin + wrong token: status %d, want 403", code)
	}
	if code := dialWS(t, ts, "", ""); code != http.StatusUnauthorized {
		t.Errorf("no Origin, no token: status %d, want 401", code)
	}
	if code := dialWS(t, ts, "", "?token=s3cret"); code != http.StatusSwitchingProtocols {
		t.Errorf("no Origin + valid token: status %d, want 101", code)
	}
	if code := dialWS(t, ts, ts.URL, ""); code != http.StatusSwitchingProtocols {
		t.Errorf("same origin, no token: status %d, want 101", code)
	}
}
