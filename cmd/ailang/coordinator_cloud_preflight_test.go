package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

func TestResolveContainerProvider(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		image     string
		want      string
		wantErr   string
	}{
		{"agree", "codex", "codex", "codex", ""},
		{"image answers when dispatcher silent", "", "codex", "codex", ""},
		{"dispatcher answers when image silent", "pi", "", "pi", ""},
		{"whitespace is not a declaration", "  ", "codex", "codex", ""},
		{"multi-CLI image defers to the request", "opencode", multiCLIImage, "opencode", ""},

		// The regression this file exists for.
		{"codex image asked for opencode", "opencode", "codex", "", "cannot run opencode"},
		{"pi image asked for claude", "claude", "pi", "", "cannot run claude"},

		// No silent default: guessing "claude" is what shipped the bug.
		{"neither side knows", "", "", "", "AILANG_PROVIDER is unset"},
		{"multi-CLI image with no request", "", multiCLIImage, "", "must name one"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveContainerProvider(tc.requested, tc.image)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("resolveContainerProvider(%q, %q) = %q, want error containing %q",
						tc.requested, tc.image, got, tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("resolveContainerProvider(%q, %q) = %q, want %q",
					tc.requested, tc.image, got, tc.want)
			}
		})
	}
}

// TestResolveContainerProviderNeverGuesses pins the property, not the cases: the
// container must never return a provider that neither side named. The old code
// returned "claude" out of nowhere, and a codex image ran opencode.
func TestResolveContainerProviderNeverGuesses(t *testing.T) {
	inputs := []string{"", "claude", "codex", "pi", multiCLIImage, "nonsense"}
	for _, req := range inputs {
		for _, img := range inputs {
			got, err := resolveContainerProvider(req, img)
			if err != nil {
				continue
			}
			if got != req && got != img {
				t.Errorf("resolveContainerProvider(%q, %q) = %q — invented a provider neither side named",
					req, img, got)
			}
		}
	}
}

var (
	envProviderRe = regexp.MustCompile(`(?m)^ENV\s+AILANG_IMAGE_PROVIDER=(\S+)`)
	fromAilangRe  = regexp.MustCompile(`(?m)^FROM\s+\S*/ailang/(\S+?):`)
)

// imageProviderFor resolves what an image declares, following FROM up the ailang
// image chain exactly as Docker does — so the "-go" variants are proven to
// inherit rather than assumed to.
func imageProviderFor(t *testing.T, dockerDir, image string) (string, error) {
	t.Helper()
	for depth := 0; depth < 8; depth++ {
		path := filepath.Join(dockerDir, "Dockerfile."+image)
		body, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("no Dockerfile for image %q: %w", image, err)
		}
		if m := envProviderRe.FindSubmatch(body); m != nil {
			return string(m[1]), nil
		}
		m := fromAilangRe.FindSubmatch(body)
		if m == nil {
			return "", fmt.Errorf("%s declares no AILANG_IMAGE_PROVIDER and no ailang parent", path)
		}
		image = string(m[1])
	}
	return "", fmt.Errorf("FROM chain too deep for %q", image)
}

// TestDockerfilesDeclareTheDispatchersProvider is the drift guard.
//
// The dispatcher derives the CLI from the variant (coordinator.VariantProviders);
// the image declares the CLI it installs. Those are two statements of one fact,
// in two repos' worth of files, and them disagreeing IS the bug. This test fails
// if they ever do.
func TestDockerfilesDeclareTheDispatchersProvider(t *testing.T) {
	dockerDir := filepath.Join("..", "..", "docker")
	if _, err := os.Stat(dockerDir); err != nil {
		t.Skipf("docker/ not present: %v", err)
	}

	// variant -> image name. Empty/"default" are the plain agent image.
	variantImage := map[string]string{
		"":          "agent",
		"default":   "agent",
		"go":        "agent-go",
		"codex":     "agent-codex",
		"codex-go":  "agent-codex-go",
		"gemini":    "agent-gemini",
		"gemini-go": "agent-gemini-go",
		"opencode":  "agent-opencode",
		"pi":        "agent-pi",
		"pi-go":     "agent-pi-go",
		"motoko":    "agent-motoko",
		"eval":      "agent-eval",
		"eval-go":   "agent-eval-go",
	}

	table := coordinator.VariantProviders()
	if len(table) == 0 {
		t.Fatal("coordinator.VariantProviders() is empty")
	}

	for variant, providers := range table {
		image, ok := variantImage[variant]
		if !ok {
			t.Errorf("variant %q has no image mapping in this test — a new variant needs one", variant)
			continue
		}
		declared, err := imageProviderFor(t, dockerDir, image)
		if err != nil {
			t.Errorf("variant %q (image %s): %v", variant, image, err)
			continue
		}

		// nil means the image carries several CLIs, so provider is a choice.
		if providers == nil {
			if declared != multiCLIImage {
				t.Errorf("variant %q is multi-CLI in the dispatcher but image %s declares %q, want %q",
					variant, image, declared, multiCLIImage)
			}
			continue
		}
		if len(providers) != 1 {
			t.Errorf("variant %q maps to %v — this test assumes single-CLI variants have exactly one",
				variant, providers)
			continue
		}
		if declared != providers[0] {
			t.Errorf("DRIFT: dispatcher runs %q on variant %q, but image %s installs %q",
				providers[0], variant, image, declared)
		}
	}
}

// TestEveryAgentDockerfileDeclaresAProvider catches a NEW agent image added
// without a declaration, which would silently fall back to trusting whatever the
// dispatcher sent — the pre-2026-09-05 behaviour.
func TestEveryAgentDockerfileDeclaresAProvider(t *testing.T) {
	dockerDir := filepath.Join("..", "..", "docker")
	entries, err := os.ReadDir(dockerDir)
	if err != nil {
		t.Skipf("docker/ not present: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, "Dockerfile.agent") {
			continue
		}
		// agent-base installs no executor CLI; it is the floor others build on.
		if name == "Dockerfile.agent-base" {
			continue
		}
		image := strings.TrimPrefix(name, "Dockerfile.")
		if _, err := imageProviderFor(t, dockerDir, image); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// TestAgentBaseDeclaresNothing pins the floor: agent-base has no executor CLI,
// so it must not claim one.
func TestAgentBaseDeclaresNothing(t *testing.T) {
	path := filepath.Join("..", "..", "docker", "Dockerfile.agent-base")
	f, err := os.Open(path)
	if err != nil {
		t.Skipf("agent-base not present: %v", err)
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if strings.Contains(sc.Text(), "AILANG_IMAGE_PROVIDER") {
			t.Errorf("agent-base declares an executor provider (%q) but installs no executor CLI",
				strings.TrimSpace(sc.Text()))
		}
	}
}
