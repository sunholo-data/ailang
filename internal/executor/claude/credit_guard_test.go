package claude

import (
	"strings"
	"testing"
)

func TestGuardedClaudeEnvironmentRejectsUnboundedBudget(t *testing.T) {
	t.Setenv("AILANG_CLAUDE_CREDIT_ACCOUNT", "anthropic-api-credits")
	t.Setenv("AILANG_AUTH_MODE", "apikey")
	t.Setenv("ANTHROPIC_API_KEY", "scoped-capability")
	t.Setenv("ANTHROPIC_BASE_URL", "https://gateway")
	for _, raw := range []string{"", "NaN", "Inf", "-1", "0", "2junk", "0.0000001"} {
		t.Setenv("AILANG_MAX_COST_USD", raw)
		if err := validateCreditEnvironment("claude-haiku-5-5"); err == nil {
			t.Fatalf("accepted budget %q", raw)
		}
	}
	t.Setenv("AILANG_MAX_COST_USD", "2")
	if err := validateCreditEnvironment("claude-haiku-5-5"); err != nil {
		t.Fatal(err)
	}
	if err := validateCreditEnvironment("opus"); err == nil {
		t.Fatal("wrong model accepted")
	}
}
func TestGuardedClaudeEnvironmentStripsAlternativeCredentials(t *testing.T) {
	env := guardCreditChildEnvironment([]string{"ANTHROPIC_API_KEY=capability", "CLAUDE_CODE_OAUTH_TOKEN=secret", "ANTHROPIC_AUTH_TOKEN=secret", "ANTHROPIC_BASE_URL=https://wrong", "CLAUDE_CODE_USE_VERTEX=1", "ANTHROPIC_MODEL=opus", "HOME=/tmp"}, "https://gateway")
	values := map[string]string{}
	for _, v := range env {
		for i := range v {
			if v[i] == '=' {
				values[v[:i]] = v[i+1:]
				break
			}
		}
	}
	if values["CLAUDE_CODE_OAUTH_TOKEN"] != "" || values["ANTHROPIC_AUTH_TOKEN"] != "" || values["CLAUDE_CODE_USE_VERTEX"] != "" {
		t.Fatal("alternative credentials survived")
	}
	if values["ANTHROPIC_API_KEY"] != "capability" || values["ANTHROPIC_BASE_URL"] != "https://gateway" || values["ANTHROPIC_MODEL"] != "claude-haiku-5-5" {
		t.Fatal(values)
	}
}

func TestCreditHeadersPreserveSignatureAcrossCloudRunIAM(t *testing.T) {
	headers := creditIdentityHeaders("complete.jwt.signature")
	if !strings.Contains(headers, "Authorization: Bearer complete.jwt.signature\nX-Serverless-Authorization: Bearer complete.jwt.signature") {
		t.Fatal("both Google ID-token headers required")
	}
}

func TestGuardedClaudeEnvironmentRejectsInheritedProviderRoutes(t *testing.T) {
	blocked := []string{
		"CLAUDE_CODE_USE_ANTHROPIC_AWS", "CLAUDE_CODE_USE_ANTHROPIC_GOOGLE_CLOUD", "CLAUDE_CODE_USE_MANTLE", "CLAUDE_CODE_USE_GATEWAY", "CLAUDE_CODE_USE_FUTURE_PROVIDER",
		"ANTHROPIC_AWS_BASE_URL", "ANTHROPIC_GOOGLE_CLOUD_BASE_URL", "ANTHROPIC_BEDROCK_MANTLE_BASE_URL", "ANTHROPIC_UNIX_SOCKET", "ANTHROPIC_FUTURE_TRANSPORT",
		"CLAUDE_CODE_SKIP_AWS_AUTH", "_CLAUDE_CODE_ASSUME_FIRST_PARTY_BASE_URL", "CLAUDE_CODE_API_BASE_URL",
		"CLAUDE_CODE_PROVIDER_MANAGED_BY_HOST", "CLAUDE_CODE_HOST_GATEWAY_LINEAGE", "CLAUDE_CODE_HOST_CREDS_FILE", "CLAUDE_CODE_HOST_AUTH_ENV_VAR", "CLAUDE_CODE_SDK_HAS_HOST_AUTH_REFRESH", "CLAUDE_CODE_GATEWAY_TOKEN_FILE_DESCRIPTOR", "CLAUDE_CODE_GATEWAY_HINT_HEADERS",
		"CLAUDE_CONFIG_DIR", "CLAUDE_CODE_SETTINGS_PATH", "CLAUDE_CODE_OAUTH_TOKEN_HELPER", "ANTHROPIC_CUSTOM_HEADERS",
	}
	inherited := []string{"ANTHROPIC_API_KEY=capability", "HOME=/tmp/task", "PATH=/usr/bin", "CLAUDE_CODE_ENABLE_TELEMETRY=1"}
	for _, name := range blocked {
		inherited = append(inherited, name+"=inherited-route")
	}
	values := map[string]string{}
	for _, entry := range guardCreditChildEnvironment(inherited, "https://gateway") {
		name, value, _ := strings.Cut(entry, "=")
		values[name] = value
	}
	for _, name := range blocked {
		if _, present := values[name]; present {
			t.Errorf("inherited route/config selector survived: %s", name)
		}
	}
	if values["ANTHROPIC_API_KEY"] != "capability" || values["ANTHROPIC_BASE_URL"] != "https://gateway" || values["CLAUDE_CODE_SIMULATE_PROXY_USAGE"] != "1" || values["HOME"] != "/tmp/task" || values["CLAUDE_CODE_ENABLE_TELEMETRY"] != "1" {
		t.Fatal("guard lost scoped gateway, compatibility, or harness environment")
	}
}

func TestGuardedClaudeDisablesUnreviewedBillableDiscovery(t *testing.T) {
	env := guardCreditChildEnvironment([]string{"ENABLE_TOOL_SEARCH=true", "CLAUDE_CODE_AUTO_MODE_SERVER=1", "CLAUDE_CODE_SIMULATE_PROXY_USAGE=0"}, "https://gateway")
	values := map[string]string{}
	for _, variable := range env {
		key, value, _ := strings.Cut(variable, "=")
		values[key] = value
	}
	if values["ENABLE_TOOL_SEARCH"] != "false" || values["CLAUDE_CODE_AUTO_MODE_SERVER"] != "0" || values["CLAUDE_CODE_SIMULATE_PROXY_USAGE"] != "1" {
		t.Fatal("billable discovery/server-classifier configuration not disabled")
	}
}
