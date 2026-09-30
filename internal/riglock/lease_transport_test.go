package riglock

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

type captureRT struct{ got http.Header }

func (c *captureRT) RoundTrip(r *http.Request) (*http.Response, error) {
	c.got = r.Header.Clone()
	return httptest.NewRecorder().Result(), nil
}

func leaseHeaderFor(t *testing.T, lease, url string) string {
	t.Helper()
	t.Setenv(config.EnvRigLease, lease)
	rt := &captureRT{}
	c := WithLease(&http.Client{Transport: rt})
	req, _ := http.NewRequest("POST", url, nil)
	if _, err := c.Do(req); err != nil {
		t.Fatal(err)
	}
	return rt.got.Get(LeaseHeader)
}

func TestWithLease(t *testing.T) {
	const tok = "0123456789abcdef0123456789abcdef"
	cases := []struct {
		name, lease, url, want string
	}{
		{"loopback v4 carries the lease", tok, "http://127.0.0.1:11434/v1/chat/completions", tok},
		{"localhost carries the lease", tok, "http://localhost:11434/api/generate", tok},
		{"loopback v6 carries the lease", tok, "http://[::1]:11434/api/chat", tok},
		{"remote host never sees the token", tok, "https://openrouter.ai/api/v1/chat/completions", ""},
		{"no lease held sends nothing", config.RigLeaseNone, "http://127.0.0.1:11434/api/generate", ""},
		{"unset lease sends nothing", "", "http://127.0.0.1:11434/api/generate", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := leaseHeaderFor(t, c.lease, c.url); got != c.want {
				t.Fatalf("X-Rig-Lease = %q, want %q", got, c.want)
			}
		})
	}
}

// The lease is read per request: eval-suite takes the lock (and exports the
// token) after its AI clients already exist.
func TestWithLease_ReadsLeasePerRequest(t *testing.T) {
	t.Setenv(config.EnvRigLease, config.RigLeaseNone)
	rt := &captureRT{}
	c := WithLease(&http.Client{Transport: rt})
	t.Setenv(config.EnvRigLease, "ffffffffffffffffffffffffffffffff")
	req, _ := http.NewRequest("POST", "http://127.0.0.1:11434/api/chat", nil)
	if _, err := c.Do(req); err != nil {
		t.Fatal(err)
	}
	if got := rt.got.Get(LeaseHeader); got != "ffffffffffffffffffffffffffffffff" {
		t.Fatalf("X-Rig-Lease = %q, want the lease exported after the client was built", got)
	}
}
