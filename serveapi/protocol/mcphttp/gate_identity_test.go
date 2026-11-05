package mcphttp

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/serveapi/protocol"
)

// M-MCP-OAUTH-CLIENT-MODELS M2 scope (quorum round 1): the embedded mcphttp
// path is unchanged. It calls only Gate.Admit, so a bool gate answers exactly
// as TestGateParity pins, and an embedder that builds NewIdentityGate gets
// the same 401/200 contract, with an empty subject refused.
func TestMcphttpGateUnchanged(t *testing.T) {
	meta := func(r *http.Request) string {
		return protocol.PublicBaseURL(r) + "/.well-known/oauth-protected-resource/mcp/connect/"
	}
	identity := func(_ context.Context, tok string) (*protocol.Identity, error) {
		switch tok {
		case "good":
			return &protocol.Identity{Subject: "u1"}, nil
		case "empty":
			return &protocol.Identity{}, nil
		}
		return nil, nil
	}
	h := gatedTestHandler(t, protocol.NewIdentityGate(identity, meta, time.Second, 4))
	for _, c := range []struct {
		tool, token string
		want        int
	}{
		{"echo", "", 200},
		{"gated", "", 401},
		{"gated", "bad", 401},
		{"gated", "empty", 401},
		{"gated", "good", 200},
	} {
		if w := postCall(h, c.tool, c.token); w.Code != c.want {
			t.Errorf("%s token=%q: %d, want %d (%s)", c.tool, c.token, w.Code, c.want, w.Body)
		}
	}
}
