package config

import "testing"

// DefaultBindHost is the one bind rule for AILANG HTTP listeners
// (M-SERVEAPI-BIND-HOST-CORS F2): loopback unless PORT (the Cloud Run
// convention) is set.
func TestDefaultBindHost(t *testing.T) {
	t.Setenv(EnvPort, "")
	if got := DefaultBindHost(); got != "127.0.0.1" {
		t.Errorf("PORT unset: DefaultBindHost() = %q, want 127.0.0.1", got)
	}
	t.Setenv(EnvPort, "8080")
	if got := DefaultBindHost(); got != "0.0.0.0" {
		t.Errorf("PORT=8080: DefaultBindHost() = %q, want 0.0.0.0", got)
	}
}
