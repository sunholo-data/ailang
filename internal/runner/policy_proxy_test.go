package runner

import (
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/policy"
)

// TestApplyPolicy_RestrictedRefusesProxyForNetAndStream: restricted mode
// cannot pin a destination behind a proxy, so it refuses one for BOTH HTTP
// effects. Stream joined Net here when its SSE/NDJSON transports started
// honoring HTTP(S)_PROXY (m-net-effect-proxy-boundary M2); without this a
// restricted run with a proxy configured would send Stream traffic unpinned.
func TestApplyPolicy_RestrictedRefusesProxyForNetAndStream(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want bool
	}{
		{policy.ModeRestricted, true},
		{policy.ModeTrustedHost, false},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			ctx := effects.NewEffContext(nil)
			ctx.Net = effects.NewNetContext()
			ctx.Stream = effects.NewStreamContext()
			ApplyPolicy(ctx, &policy.Resolved{Mode: tc.mode})
			if ctx.Net.RefuseProxy != tc.want {
				t.Errorf("Net.RefuseProxy = %v, want %v", ctx.Net.RefuseProxy, tc.want)
			}
			if ctx.Stream.RefuseProxy != tc.want {
				t.Errorf("Stream.RefuseProxy = %v, want %v", ctx.Stream.RefuseProxy, tc.want)
			}
		})
	}
}
