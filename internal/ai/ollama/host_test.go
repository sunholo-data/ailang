package ollama

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ollama/ollama/envconfig"
	"github.com/sunholo-data/ailang/internal/config"
)

// hostCases is every OLLAMA_HOST shape the rig, the plists and the factory
// have produced, plus the quoting and port edge cases.
var hostCases = []string{
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
	"http://10.0.0.5:-1",
	"http://10.0.0.5:11434/v1",
	"  http://127.0.0.1:11434  ",
	"http://host:11434/some/path",
	`"http://127.0.0.1:11435"`,
	`'127.0.0.1:11436'`,
	` "quoted-with-space:1" `,
	"ftp://weird:21",
	"::1",
	"[::1]",
}

// TestHostFromEnvironmentGolden pins OLLAMA_HOST resolution against
// testdata/wire/host.json, captured from the SDK's envconfig.Host() (which
// ClientFromEnvironment used) before the SDK was removed.
func TestHostFromEnvironmentGolden(t *testing.T) {
	want := loadHostGolden(t)
	for _, in := range hostCases {
		t.Setenv(config.EnvOllamaHost, in)
		if got := envconfig.Host().String(); got != want[in] {
			t.Errorf("OLLAMA_HOST=%q resolves to %q, golden %q", in, got, want[in])
		}
	}
}

// TestParseHostGolden pins parseHost — the resolver for the provider client,
// which takes OLLAMA_HOST verbatim (no quote trimming) — against the same
// golden for every unquoted shape.
func TestParseHostGolden(t *testing.T) {
	want := loadHostGolden(t)
	for _, in := range hostCases {
		if strings.ContainsAny(in, `"'`) {
			continue
		}
		if got := parseHost(in).String(); got != want[in] {
			t.Errorf("parseHost(%q) = %q, golden %q", in, got, want[in])
		}
	}
}

func loadHostGolden(t *testing.T) map[string]string {
	t.Helper()
	data, err := os.ReadFile("testdata/wire/host.json")
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	for _, in := range hostCases {
		if _, ok := m[in]; !ok {
			t.Fatalf("host golden has no entry for %q", in)
		}
	}
	return m
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
