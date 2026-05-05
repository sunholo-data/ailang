package ai

import (
	"context"
	"strings"
	"testing"
)

// fakeProvider is a minimal Provider used to populate the registry in tests.
type fakeProvider struct{ name string }

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Generate(ctx context.Context, req *Request) (*Response, error) {
	return &Response{Text: "ok from " + f.name}, nil
}
func (f *fakeProvider) Step(ctx context.Context, req *Request) (*Response, error) {
	return f.Generate(ctx, req)
}

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewProviderRegistry()
	p := &fakeProvider{name: "vllm"}
	if err := r.Register("vllm", p, "/tmp/pkg/vllm/ailang.toml"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok := r.Lookup("vllm")
	if !ok {
		t.Fatal("Lookup: not found after Register")
	}
	if got.Name() != "vllm" {
		t.Errorf("Lookup returned name %q, want vllm", got.Name())
	}
	if src := r.SourceOf("vllm"); src != "/tmp/pkg/vllm/ailang.toml" {
		t.Errorf("SourceOf = %q", src)
	}
}

func TestRegistry_LookupMiss(t *testing.T) {
	r := NewProviderRegistry()
	if _, ok := r.Lookup("nonexistent"); ok {
		t.Error("Lookup found provider that was never registered")
	}
}

func TestRegistry_DuplicateNameError(t *testing.T) {
	r := NewProviderRegistry()
	p1 := &fakeProvider{name: "vllm"}
	p2 := &fakeProvider{name: "vllm"}

	if err := r.Register("vllm", p1, "/tmp/pkg/a/ailang.toml"); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	err := r.Register("vllm", p2, "/tmp/pkg/b/ailang.toml")
	if err == nil {
		t.Fatal("expected duplicate-name error")
	}
	// Error message should name BOTH source paths so the user can resolve
	if !strings.Contains(err.Error(), "/tmp/pkg/a/ailang.toml") ||
		!strings.Contains(err.Error(), "/tmp/pkg/b/ailang.toml") {
		t.Errorf("error should name both source paths, got: %v", err)
	}
	// And mention the conflicting name
	if !strings.Contains(err.Error(), "vllm") {
		t.Errorf("error should mention provider name, got: %v", err)
	}
}

func TestRegistry_IdempotentRegister(t *testing.T) {
	// Re-registering the EXACT same provider+source should be a no-op.
	// Useful when manifests are loaded twice in tests or under autoreload.
	r := NewProviderRegistry()
	p := &fakeProvider{name: "vllm"}
	src := "/tmp/pkg/vllm/ailang.toml"
	if err := r.Register("vllm", p, src); err != nil {
		t.Fatalf("first Register: %v", err)
	}
	if err := r.Register("vllm", p, src); err != nil {
		t.Errorf("idempotent re-Register should not error: %v", err)
	}
}

func TestRegistry_EmptyName(t *testing.T) {
	r := NewProviderRegistry()
	err := r.Register("", &fakeProvider{name: "x"}, "/tmp/x.toml")
	if err == nil || !strings.Contains(err.Error(), "name is empty") {
		t.Errorf("expected empty-name error, got %v", err)
	}
}

func TestRegistry_NilProvider(t *testing.T) {
	r := NewProviderRegistry()
	err := r.Register("vllm", nil, "/tmp/x.toml")
	if err == nil || !strings.Contains(err.Error(), "nil provider") {
		t.Errorf("expected nil-provider error, got %v", err)
	}
}

func TestRegistry_NamesAlphabetical(t *testing.T) {
	r := NewProviderRegistry()
	for _, n := range []string{"vllm", "anthropic_proxy", "llamacpp"} {
		if err := r.Register(n, &fakeProvider{name: n}, "/tmp/"+n+".toml"); err != nil {
			t.Fatal(err)
		}
	}
	names := r.Names()
	want := []string{"anthropic_proxy", "llamacpp", "vllm"}
	if len(names) != len(want) {
		t.Fatalf("len = %d, want %d", len(names), len(want))
	}
	for i, n := range want {
		if names[i] != n {
			t.Errorf("names[%d] = %q, want %q", i, names[i], n)
		}
	}
}

func TestRegistry_Reset(t *testing.T) {
	r := NewProviderRegistry()
	if err := r.Register("vllm", &fakeProvider{name: "vllm"}, "/tmp/x.toml"); err != nil {
		t.Fatal(err)
	}
	r.Reset()
	if _, ok := r.Lookup("vllm"); ok {
		t.Error("Reset did not clear the registry")
	}
	if names := r.Names(); len(names) != 0 {
		t.Errorf("Names after Reset = %v, want empty", names)
	}
}

func TestRegistry_DiagnosticsForBuiltinShadow(t *testing.T) {
	r := NewProviderRegistry()
	// Register a provider with a built-in name — allowed but flagged.
	if err := r.Register("openai", &fakeProvider{name: "openai"}, "/tmp/shadow/ailang.toml"); err != nil {
		t.Fatal(err)
	}
	diags := r.Diagnostics()
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d: %v", len(diags), diags)
	}
	if !strings.Contains(diags[0], "shadows a built-in") || !strings.Contains(diags[0], "openai") {
		t.Errorf("diagnostic message = %q", diags[0])
	}
}

func TestRegistry_NoDiagnosticsForNonBuiltinNames(t *testing.T) {
	r := NewProviderRegistry()
	if err := r.Register("vllm", &fakeProvider{name: "vllm"}, "/tmp/x.toml"); err != nil {
		t.Fatal(err)
	}
	if diags := r.Diagnostics(); len(diags) != 0 {
		t.Errorf("expected no diagnostics for non-builtin name, got: %v", diags)
	}
}

func TestIsBuiltinName(t *testing.T) {
	cases := map[string]bool{
		"openai":     true,
		"anthropic":  true,
		"gemini":     true,
		"ollama":     true,
		"openrouter": true,
		"OpenAI":     true, // case-insensitive
		"vllm":       false,
		"":           false,
	}
	for name, want := range cases {
		if got := IsBuiltinName(name); got != want {
			t.Errorf("IsBuiltinName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	// Registry is documented as concurrency-safe via sync.RWMutex.
	// Race detector must not flag this test.
	r := NewProviderRegistry()
	done := make(chan bool, 100)
	for i := 0; i < 50; i++ {
		go func() {
			_ = r.Register("vllm", &fakeProvider{name: "vllm"}, "/tmp/x.toml")
			done <- true
		}()
		go func() {
			_, _ = r.Lookup("vllm")
			r.Names()
			done <- true
		}()
	}
	for i := 0; i < 100; i++ {
		<-done
	}
}
