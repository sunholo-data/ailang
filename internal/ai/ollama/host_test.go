package ollama

import (
	"os"
	"testing"

	"github.com/ollama/ollama/envconfig"
	"github.com/sunholo-data/ailang/internal/config"
)

// TestParseHostMatchesSDK pins parseHost against the SDK's own OLLAMA_HOST
// parser: for every endpoint shape the rig, the plists and the factory have
// produced, the URL we hand ollamaapi.NewClient must equal the one
// ClientFromEnvironment would have derived from the exported variable
// (which NewClient no longer exports — M-V1-SIMPLIFY-S4 M4).
func TestParseHostMatchesSDK(t *testing.T) {
	cases := []string{
		"",
		defaultEndpoint,
		"http://127.0.0.1:11434",
		"http://127.0.0.1:11434/",
		"127.0.0.1:11434",
		"localhost",
		"localhost:11434",
		"http://localhost",
		"https://ollama.example.com",
		"ollama.com",
		"http://ollama.com",
		"0.0.0.0:11434",
		"[::1]:11434",
		"http://[::1]:11434",
		"http://10.0.0.5:99999",
		"http://10.0.0.5:11434/v1",
		"  http://127.0.0.1:11434  ",
		"http://host:11434/some/path",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			t.Setenv(config.EnvOllamaHost, in)
			want := envconfig.Host()
			got := parseHost(in)
			if got.String() != want.String() {
				t.Fatalf("parseHost(%q) = %q, SDK envconfig.Host() = %q", in, got, want)
			}
		})
	}
}

// TestNewClientDoesNotExportOllamaHost is the seam the migration closed:
// constructing a client with an explicit endpoint must not change what a
// child process (or the next client) sees in OLLAMA_HOST.
func TestNewClientDoesNotExportOllamaHost(t *testing.T) {
	t.Setenv(config.EnvOllamaHost, "")
	os.Unsetenv(config.EnvOllamaHost)
	c, err := NewClient(WithEndpoint("http://10.1.2.3:11434"))
	if err != nil {
		t.Fatal(err)
	}
	if c.endpoint != "http://10.1.2.3:11434" {
		t.Fatalf("endpoint = %q", c.endpoint)
	}
	if v, set := os.LookupEnv(config.EnvOllamaHost); set {
		t.Fatalf("NewClient exported %s=%q; the endpoint must reach the SDK client explicitly", config.EnvOllamaHost, v)
	}
}

// TestNewClientOllamaHostWinsOverOption pins the documented precedence: the
// rig's domain global must route every process to the capped server even
// when the factory passes an endpoint.
func TestNewClientOllamaHostWinsOverOption(t *testing.T) {
	t.Setenv(config.EnvOllamaHost, "http://127.0.0.1:11435")
	c, err := NewClient(WithEndpoint("http://10.1.2.3:11434"))
	if err != nil {
		t.Fatal(err)
	}
	if c.endpoint != "http://127.0.0.1:11435" {
		t.Fatalf("endpoint = %q, want OLLAMA_HOST to win", c.endpoint)
	}
}
