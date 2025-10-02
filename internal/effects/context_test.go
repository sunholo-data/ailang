package effects

import (
	"os"
	"testing"
)

func TestNewCapability(t *testing.T) {
	cap := NewCapability("IO")

	if cap.Name != "IO" {
		t.Errorf("expected Name='IO', got %q", cap.Name)
	}

	if cap.Meta == nil {
		t.Error("expected Meta map to be initialized")
	}

	if len(cap.Meta) != 0 {
		t.Errorf("expected empty Meta map, got %d entries", len(cap.Meta))
	}
}

func TestNewEffContext(t *testing.T) {
	ctx := NewEffContext()

	if ctx.Caps == nil {
		t.Error("expected Caps map to be initialized")
	}

	if len(ctx.Caps) != 0 {
		t.Errorf("expected no capabilities granted by default, got %d", len(ctx.Caps))
	}

	// Environment should be loaded
	if ctx.Env.TZ == "" {
		t.Error("expected TZ to have default value")
	}
}

func TestGrantCapability(t *testing.T) {
	ctx := NewEffContext()

	ioCap := NewCapability("IO")
	ctx.Grant(ioCap)

	if !ctx.HasCap("IO") {
		t.Error("expected IO capability to be granted")
	}

	if ctx.HasCap("FS") {
		t.Error("expected FS capability to not be granted")
	}
}

func TestGrantMultipleCapabilities(t *testing.T) {
	ctx := NewEffContext()

	ctx.Grant(NewCapability("IO"))
	ctx.Grant(NewCapability("FS"))
	ctx.Grant(NewCapability("Net"))

	caps := []string{"IO", "FS", "Net"}
	for _, name := range caps {
		if !ctx.HasCap(name) {
			t.Errorf("expected %s capability to be granted", name)
		}
	}

	if len(ctx.Caps) != 3 {
		t.Errorf("expected 3 capabilities, got %d", len(ctx.Caps))
	}
}

func TestGrantIdempotent(t *testing.T) {
	ctx := NewEffContext()

	ctx.Grant(NewCapability("IO"))
	ctx.Grant(NewCapability("IO")) // Grant same cap twice

	if len(ctx.Caps) != 1 {
		t.Errorf("expected 1 capability after duplicate grant, got %d", len(ctx.Caps))
	}
}

func TestRequireCap_Success(t *testing.T) {
	ctx := NewEffContext()
	ctx.Grant(NewCapability("IO"))

	err := ctx.RequireCap("IO")
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestRequireCap_Missing(t *testing.T) {
	ctx := NewEffContext()

	err := ctx.RequireCap("IO")
	if err == nil {
		t.Error("expected error for missing capability")
	}

	capErr, ok := err.(*CapabilityError)
	if !ok {
		t.Errorf("expected *CapabilityError, got %T", err)
	}

	if capErr.Effect != "IO" {
		t.Errorf("expected Effect='IO', got %q", capErr.Effect)
	}
}

func TestCapabilityError_Message(t *testing.T) {
	err := NewCapabilityError("FS")
	msg := err.Error()

	expectedSubstrings := []string{
		"effect 'FS'",
		"requires capability",
		"Hint: Run with --caps FS",
	}

	for _, substr := range expectedSubstrings {
		if !contains(msg, substr) {
			t.Errorf("expected error message to contain %q, got: %s", substr, msg)
		}
	}
}

func TestLoadEffEnv_Defaults(t *testing.T) {
	// Clear environment variables
	os.Unsetenv("AILANG_SEED")
	os.Unsetenv("TZ")
	os.Unsetenv("LANG")
	os.Unsetenv("AILANG_FS_SANDBOX")

	env := loadEffEnv()

	if env.Seed != 0 {
		t.Errorf("expected default Seed=0, got %d", env.Seed)
	}

	if env.TZ != "UTC" {
		t.Errorf("expected default TZ='UTC', got %q", env.TZ)
	}

	if env.Locale != "C" {
		t.Errorf("expected default Locale='C', got %q", env.Locale)
	}

	if env.Sandbox != "" {
		t.Errorf("expected default Sandbox='', got %q", env.Sandbox)
	}
}

func TestLoadEffEnv_FromEnvironment(t *testing.T) {
	// Set environment variables
	os.Setenv("AILANG_SEED", "42")
	os.Setenv("TZ", "America/New_York")
	os.Setenv("LANG", "en_US.UTF-8")
	os.Setenv("AILANG_FS_SANDBOX", "/tmp/sandbox")

	defer func() {
		os.Unsetenv("AILANG_SEED")
		os.Unsetenv("TZ")
		os.Unsetenv("LANG")
		os.Unsetenv("AILANG_FS_SANDBOX")
	}()

	env := loadEffEnv()

	if env.Seed != 42 {
		t.Errorf("expected Seed=42, got %d", env.Seed)
	}

	if env.TZ != "America/New_York" {
		t.Errorf("expected TZ='America/New_York', got %q", env.TZ)
	}

	if env.Locale != "en_US.UTF-8" {
		t.Errorf("expected Locale='en_US.UTF-8', got %q", env.Locale)
	}

	if env.Sandbox != "/tmp/sandbox" {
		t.Errorf("expected Sandbox='/tmp/sandbox', got %q", env.Sandbox)
	}
}

func TestLoadEffEnv_InvalidSeed(t *testing.T) {
	os.Setenv("AILANG_SEED", "invalid")
	defer os.Unsetenv("AILANG_SEED")

	env := loadEffEnv()

	// Should default to 0 on parse error
	if env.Seed != 0 {
		t.Errorf("expected Seed=0 for invalid input, got %d", env.Seed)
	}
}

func TestCapabilityMetadata(t *testing.T) {
	cap := NewCapability("FS")
	cap.Meta["sandbox"] = "/tmp"
	cap.Meta["max_bytes"] = 1048576

	if cap.Meta["sandbox"] != "/tmp" {
		t.Errorf("expected sandbox='/tmp', got %v", cap.Meta["sandbox"])
	}

	if cap.Meta["max_bytes"] != 1048576 {
		t.Errorf("expected max_bytes=1048576, got %v", cap.Meta["max_bytes"])
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
