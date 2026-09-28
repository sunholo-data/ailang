// lease_transport.go — the client half of the lease: carry AILANG_RIG_LEASE
// to the rig gateway on every request ailang itself sends to a local model.
//
// pi and opencode read the token from their own provider configs. Everything
// that reaches ollama through ailang's own HTTP clients — `--ai ollama:…`, the
// OpenAI-compatible client that motoko uses (OPENAI_BASE_URL=…:11434/v1), the
// lang nightly — gets it here, so one change covers them all.
package riglock

import (
	"net"
	"net/http"

	"github.com/sunholo-data/ailang/internal/config"
)

// LeaseHeader is the header the rig gateway reads first (internal/riggate).
const LeaseHeader = "X-Rig-Lease"

// WithLease returns a client that adds X-Rig-Lease to requests for loopback
// hosts while the process carries a live lease. The token is read per request,
// because a process that takes the lock itself (eval-suite) exports it after its
// clients were built. It never leaves the machine: remote hosts get no header.
func WithLease(c *http.Client) *http.Client {
	if c == nil {
		c = http.DefaultClient
	}
	if _, ok := c.Transport.(leaseTransport); ok {
		return c
	}
	cp := *c
	cp.Transport = leaseTransport{base: c.Transport}
	return &cp
}

type leaseTransport struct{ base http.RoundTripper }

func (t leaseTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	tok := config.RigLease()
	if tok == "" || tok == config.RigLeaseNone || r.Header.Get(LeaseHeader) != "" || !loopbackHost(r.URL.Hostname()) {
		return base.RoundTrip(r)
	}
	r2 := r.Clone(r.Context())
	r2.Header.Set(LeaseHeader, tok)
	return base.RoundTrip(r2)
}

func loopbackHost(h string) bool {
	if h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}
