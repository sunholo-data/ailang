package opencode

import (
	"os"
	"strings"
	"testing"
)

func TestOllamaRigConfigCarriesLeaseWithoutChangingEverydayProvider(t *testing.T) {
	b, err := os.ReadFile("testdata/opencode_ollama_config.jsonc")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"ollama-rig"`, `"apiKey": "{env:AILANG_RIG_LEASE}"`, `"baseURL": "http://127.0.0.1:11434/v1"`} {
		if !strings.Contains(s, want) {
			t.Errorf("config missing %s", want)
		}
	}
	start := strings.Index(s, `"ollama":`)
	end := strings.Index(s, `"ollama-rig":`)
	if start < 0 || end <= start {
		t.Fatal("provider blocks missing or out of order")
	}
	beforeRig := s[start:end]
	if strings.Contains(beforeRig, "AILANG_RIG_LEASE") {
		t.Fatal("everyday ollama provider unexpectedly requires the rig lease")
	}
}
