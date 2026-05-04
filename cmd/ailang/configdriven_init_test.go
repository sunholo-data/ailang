package main

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/pkg"
)

// makeSpec returns a minimal valid AIProviderSpec for tests.
func makeSpec(name, endpoint string) pkg.AIProviderSpec {
	return pkg.AIProviderSpec{
		SchemaVersion: 1,
		Name:          name,
		Endpoint:      endpoint,
		RequestShape:  "openai_chat",
		ResponsePath:  "$.choices[0].message.content",
		Auth:          pkg.AIProviderAuth{Type: "none"},
	}
}

func TestRegisterConfigDrivenProviders_SinglePackage(t *testing.T) {
	registry := ai.NewProviderRegistry()
	sources := []ManifestSource{{
		Path: "/tmp/pkg/ai_vllm/ailang.toml",
		Manifest: &pkg.PackageManifest{
			Package: pkg.PackageInfo{Name: "sunholo/ai_vllm", Version: "0.1.0"},
			AIProviders: []pkg.AIProviderSpec{
				makeSpec("vllm", "http://localhost:8000/v1/chat/completions"),
			},
		},
	}}

	if err := RegisterConfigDrivenProviders(registry, sources); err != nil {
		t.Fatalf("RegisterConfigDrivenProviders failed: %v", err)
	}
	provider, ok := registry.Lookup("vllm")
	if !ok {
		t.Fatal("provider not registered")
	}
	if provider.Name() != "vllm" {
		t.Errorf("provider.Name() = %q, want vllm", provider.Name())
	}
}

func TestRegisterConfigDrivenProviders_MultiplePackages(t *testing.T) {
	registry := ai.NewProviderRegistry()
	sources := []ManifestSource{
		{
			Path: "/tmp/pkg/ai_vllm/ailang.toml",
			Manifest: &pkg.PackageManifest{
				Package:     pkg.PackageInfo{Name: "sunholo/ai_vllm", Version: "0.1.0"},
				AIProviders: []pkg.AIProviderSpec{makeSpec("vllm", "http://localhost:8000/v1/chat/completions")},
			},
		},
		{
			Path: "/tmp/pkg/ai_llamacpp/ailang.toml",
			Manifest: &pkg.PackageManifest{
				Package:     pkg.PackageInfo{Name: "sunholo/ai_llamacpp", Version: "0.1.0"},
				AIProviders: []pkg.AIProviderSpec{makeSpec("llamacpp", "http://localhost:8080/completion")},
			},
		},
	}

	if err := RegisterConfigDrivenProviders(registry, sources); err != nil {
		t.Fatalf("RegisterConfigDrivenProviders failed: %v", err)
	}
	for _, name := range []string{"vllm", "llamacpp"} {
		if _, ok := registry.Lookup(name); !ok {
			t.Errorf("provider %q not registered", name)
		}
	}
}

func TestRegisterConfigDrivenProviders_CrossPackageDuplicate(t *testing.T) {
	registry := ai.NewProviderRegistry()
	sources := []ManifestSource{
		{
			Path: "/tmp/pkg/a/ailang.toml",
			Manifest: &pkg.PackageManifest{
				Package:     pkg.PackageInfo{Name: "vendor_a/ai_vllm", Version: "0.1.0"},
				AIProviders: []pkg.AIProviderSpec{makeSpec("vllm", "http://server-a/v1")},
			},
		},
		{
			Path: "/tmp/pkg/b/ailang.toml",
			Manifest: &pkg.PackageManifest{
				Package:     pkg.PackageInfo{Name: "vendor_b/ai_vllm", Version: "0.1.0"},
				AIProviders: []pkg.AIProviderSpec{makeSpec("vllm", "http://server-b/v1")}, // same name!
			},
		},
	}

	err := RegisterConfigDrivenProviders(registry, sources)
	if err == nil {
		t.Fatal("expected duplicate-name error across packages")
	}
	// Error must reference both source manifest paths so user can resolve
	if !strings.Contains(err.Error(), "/tmp/pkg/a/ailang.toml") ||
		!strings.Contains(err.Error(), "/tmp/pkg/b/ailang.toml") {
		t.Errorf("error must name both manifests, got: %v", err)
	}
	if !strings.Contains(err.Error(), "vllm") {
		t.Errorf("error must mention provider name, got: %v", err)
	}
}

func TestRegisterConfigDrivenProviders_MultipleProvidersInOneManifest(t *testing.T) {
	registry := ai.NewProviderRegistry()
	sources := []ManifestSource{{
		Path: "/tmp/pkg/multi/ailang.toml",
		Manifest: &pkg.PackageManifest{
			Package: pkg.PackageInfo{Name: "sunholo/ai_multi", Version: "0.1.0"},
			AIProviders: []pkg.AIProviderSpec{
				makeSpec("vllm", "http://localhost:8000"),
				makeSpec("llamacpp", "http://localhost:8080"),
			},
		},
	}}

	if err := RegisterConfigDrivenProviders(registry, sources); err != nil {
		t.Fatalf("RegisterConfigDrivenProviders failed: %v", err)
	}
	if names := registry.Names(); len(names) != 2 {
		t.Errorf("expected 2 providers, got %d: %v", len(names), names)
	}
}

func TestRegisterConfigDrivenProviders_EmptyManifestList(t *testing.T) {
	registry := ai.NewProviderRegistry()
	if err := RegisterConfigDrivenProviders(registry, nil); err != nil {
		t.Errorf("empty manifest list should not error, got: %v", err)
	}
	if names := registry.Names(); len(names) != 0 {
		t.Errorf("expected empty registry, got: %v", names)
	}
}

func TestRegisterConfigDrivenProviders_ManifestWithoutProviders(t *testing.T) {
	// Most packages won't declare [[ai_provider]] — the harvest should
	// quietly skip them, not error.
	registry := ai.NewProviderRegistry()
	sources := []ManifestSource{{
		Path: "/tmp/pkg/normal/ailang.toml",
		Manifest: &pkg.PackageManifest{
			Package: pkg.PackageInfo{Name: "sunholo/json", Version: "0.3.1"},
			// AIProviders omitted
		},
	}}
	if err := RegisterConfigDrivenProviders(registry, sources); err != nil {
		t.Errorf("manifest without providers should not error, got: %v", err)
	}
	if names := registry.Names(); len(names) != 0 {
		t.Errorf("expected empty registry, got: %v", names)
	}
}

func TestRegisterConfigDrivenProviders_NilManifestSkipped(t *testing.T) {
	registry := ai.NewProviderRegistry()
	sources := []ManifestSource{
		{Path: "/tmp/nil.toml", Manifest: nil}, // skipped
		{
			Path: "/tmp/pkg/real/ailang.toml",
			Manifest: &pkg.PackageManifest{
				Package:     pkg.PackageInfo{Name: "real/pkg", Version: "0.1.0"},
				AIProviders: []pkg.AIProviderSpec{makeSpec("vllm", "http://localhost")},
			},
		},
	}
	if err := RegisterConfigDrivenProviders(registry, sources); err != nil {
		t.Fatalf("error: %v", err)
	}
	if _, ok := registry.Lookup("vllm"); !ok {
		t.Error("real provider should still be registered when nil manifests are present")
	}
}

func TestRegisterConfigDrivenProviders_NilRegistryDefaultsToGlobal(t *testing.T) {
	// Reset global to keep this test isolated from other tests using it.
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	sources := []ManifestSource{{
		Path: "/tmp/pkg/global-test/ailang.toml",
		Manifest: &pkg.PackageManifest{
			Package:     pkg.PackageInfo{Name: "test/global", Version: "0.1.0"},
			AIProviders: []pkg.AIProviderSpec{makeSpec("global-test-provider", "http://localhost")},
		},
	}}

	if err := RegisterConfigDrivenProviders(nil, sources); err != nil {
		t.Fatalf("error: %v", err)
	}
	if _, ok := ai.GlobalProviderRegistry.Lookup("global-test-provider"); !ok {
		t.Error("nil registry should default to GlobalProviderRegistry")
	}
}

func TestLookupConfigDrivenProvider(t *testing.T) {
	ai.GlobalProviderRegistry.Reset()
	defer ai.GlobalProviderRegistry.Reset()

	sources := []ManifestSource{{
		Path: "/tmp/pkg/lookup-test/ailang.toml",
		Manifest: &pkg.PackageManifest{
			Package:     pkg.PackageInfo{Name: "test/lookup", Version: "0.1.0"},
			AIProviders: []pkg.AIProviderSpec{makeSpec("vllm", "http://localhost")},
		},
	}}
	if err := RegisterConfigDrivenProviders(nil, sources); err != nil {
		t.Fatal(err)
	}
	if p := LookupConfigDrivenProvider("vllm"); p == nil {
		t.Error("LookupConfigDrivenProvider returned nil for registered provider")
	} else if p.Name() != "vllm" {
		t.Errorf("provider name = %q, want vllm", p.Name())
	}
	if p := LookupConfigDrivenProvider("nonexistent"); p != nil {
		t.Error("LookupConfigDrivenProvider returned non-nil for unregistered provider")
	}
}
