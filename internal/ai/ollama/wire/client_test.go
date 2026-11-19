package wire

import (
	"net/http"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/config"
)

// The one behaviour the SDK had and this client drops: it signed requests to
// ollama.com with the `ollama signin` device key. An unsigned request there
// would 401 far from its cause, so the client refuses the host up front.
func TestNewClientRefusesOllamaCom(t *testing.T) {
	for _, host := range []string{"ollama.com", "https://ollama.com", "http://ollama.com:8080", "  'ollama.com'  "} {
		t.Run(host, func(t *testing.T) {
			t.Setenv(config.EnvOllamaHost, host)
			c, err := ClientFromEnvironment()
			if err == nil || c != nil {
				t.Fatalf("ClientFromEnvironment() with OLLAMA_HOST=%q = (%v, %v), want a refusal", host, c, err)
			}
			if !strings.Contains(err.Error(), "OLLAMA_HOST") || !strings.Contains(err.Error(), "ollama.com") {
				t.Errorf("refusal should name the host and the fix, got %q", err)
			}
		})
	}
}

func TestNewClientAcceptsOtherHosts(t *testing.T) {
	for _, host := range []string{"", "127.0.0.1:11434", "https://ollama.example.com", "http://api.ollama.com.internal:1"} {
		if _, err := NewClient(ParseHost(host), http.DefaultClient); err != nil {
			t.Errorf("NewClient(%q) = %v, want ok", host, err)
		}
	}
}

// Zero-value and null arguments marshal the way the SDK's did: {} when the
// arguments were never set, null when they were decoded from JSON null.
func TestToolCallFunctionArgumentsZeroAndNull(t *testing.T) {
	var zero ToolCallFunctionArguments
	if got := zero.String(); got != "{}" {
		t.Errorf("zero String() = %q, want {}", got)
	}
	var fromNull ToolCallFunctionArguments
	if err := fromNull.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatal(err)
	}
	if got := fromNull.String(); got != "null" {
		t.Errorf("null String() = %q, want null", got)
	}
}
