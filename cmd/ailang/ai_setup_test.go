package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ai"
	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// #1499: --ai-key-file is read at setup and becomes the provider credential;
// the flag beats AILANG_AI_KEY_FILE; a bad file fails before the program runs.
func TestAISetup_ResolveAuth_KeyFile(t *testing.T) {
	t.Setenv("AILANG_AI_KEY_FILE", "")
	dir := t.TempDir()
	flagFile := filepath.Join(dir, "flag.key")
	envFile := filepath.Join(dir, "env.key")
	if err := os.WriteFile(flagFile, []byte("flag-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(envFile, []byte("env-secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if a, err := (aiSetup{}).resolveAuth(); err != nil || a.fromKeyFile || len(a.options) != 0 {
		t.Fatalf("no flags: %+v %v", a, err)
	}
	if a, err := (aiSetup{KeyFile: flagFile}).resolveAuth(); err != nil || !a.fromKeyFile {
		t.Fatalf("flag: %+v %v", a, err)
	}
	t.Setenv("AILANG_AI_KEY_FILE", envFile)
	if a, err := (aiSetup{}).resolveAuth(); err != nil || !a.fromKeyFile {
		t.Fatalf("env: %+v %v", a, err)
	}
	if _, err := (aiSetup{KeyFile: filepath.Join(dir, "missing")}).resolveAuth(); err == nil {
		t.Fatal("missing key file must fail at setup")
	}
}

// #1499 end to end through the CLI's handler setup: the key-file credential
// reaches a Google client built on the AI Studio lane, and with --ai-no-adc
// and no key the setup fails with AuthFailed instead of picking Vertex ADC.
func TestSetupAIHandler_GoogleKeyFileAndNoADC(t *testing.T) {
	for _, v := range []string{"GOOGLE_API_KEY", "GEMINI_API_KEY", "AILANG_AI_KEY_FILE", "AILANG_AI_NO_ADC"} {
		t.Setenv(v, "")
	}
	t.Setenv("GOOGLE_CLOUD_PROJECT", "dev-project") // ADC would otherwise resolve
	t.Setenv("GOOGLE_CLOUD_LOCATION", "global")
	model := &eval_harness.ModelConfig{Provider: "google", APIName: "gemini-2.5-flash"}

	effCtx := effects.NewEffContext(nil)
	err := setupAIHandlerFromConfig(effCtx, model, "gemini-2-5-flash", nil, nil, aiAuth{})
	if err != nil {
		t.Fatalf("control (ADC lane): %v", err)
	}

	noADC, err := (aiSetup{NoADC: true}).resolveAuth()
	if err != nil {
		t.Fatal(err)
	}
	err = setupAIHandlerFromConfig(effects.NewEffContext(nil), model, "gemini-2-5-flash", nil, nil, noADC)
	var aiErr *ai.AIError
	if !errors.As(err, &aiErr) || aiErr.Code != ai.CodeAuthFailed {
		t.Fatalf("--ai-no-adc without a key: want AuthFailed, got %v", err)
	}

	keyFile := filepath.Join(t.TempDir(), "gemini.key")
	if err := os.WriteFile(keyFile, []byte("file-secret-1499\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	withKey, err := (aiSetup{NoADC: true, KeyFile: keyFile}).resolveAuth()
	if err != nil {
		t.Fatal(err)
	}
	if err := setupAIHandlerFromConfig(effects.NewEffContext(nil), model, "gemini-2-5-flash", nil, nil, withKey); err != nil {
		t.Fatalf("--ai-no-adc + --ai-key-file: %v", err)
	}
}

// A key file for a lane that takes no key is refused, not silently ignored.
func TestSetupAIHandler_KeyFileRefusedForKeylessLane(t *testing.T) {
	t.Setenv("AILANG_AI_KEY_FILE", "")
	keyFile := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(keyFile, []byte("k\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := (aiSetup{KeyFile: keyFile}).resolveAuth()
	if err != nil {
		t.Fatal(err)
	}
	// Ollama is the local lane: the check fires before any daemon probe.
	model := &eval_harness.ModelConfig{Provider: "ollama", APIName: "m"}
	err = setupAIHandlerFromConfig(effects.NewEffContext(nil), model, "m", nil, nil, a)
	if err == nil || !strings.Contains(err.Error(), "--ai-key-file") {
		t.Fatalf("want a --ai-key-file refusal for the local lane, got %v", err)
	}
}

// #1497: --ai-stub-fixtures swaps the stub for the fixture replayer, needs
// --ai-stub, and a broken file fails setup.
func TestSetupAIHandler_StubFixtures(t *testing.T) {
	good := filepath.Join(t.TempDir(), "fx.json")
	if err := os.WriteFile(good, []byte(`{"version":1,"fixtures":[{"kind":"text","prompt":"hi","response":"yo"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	effCtx := effects.NewEffContext(nil)
	if err := setupAIHandler(effCtx, aiSetup{Stub: true, StubFixtures: good}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if got, err := effCtx.AI.Call("hi"); err != nil || got != "yo" {
		t.Fatalf("fixture replay: %q %v", got, err)
	}
	if _, err := effCtx.AI.Call("unrecorded"); err == nil {
		t.Fatal("a fixture miss must not fall back to the default stub")
	}
	if err := setupAIHandler(effects.NewEffContext(nil), aiSetup{StubFixtures: good}, nil, nil); err == nil {
		t.Fatal("--ai-stub-fixtures without --ai-stub must be refused")
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte(`{"version":9}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := setupAIHandler(effects.NewEffContext(nil), aiSetup{Stub: true, StubFixtures: bad}, nil, nil); err == nil {
		t.Fatal("an invalid fixture file must fail setup")
	}
}
