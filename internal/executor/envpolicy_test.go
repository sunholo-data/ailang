package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

const stubAIPolicy = "allowed_caps = [\"IO\", \"AI\"]\nai_provider = \"stub\"\nentry = \"main\"\ntimeout_ms = 30000\n[budgets]\nAI = 2\n"

func writeTestPolicy(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "agent-policy.toml")
	if err := os.WriteFile(p, []byte(body), 0o444); err != nil {
		t.Fatal(err)
	}
	return p
}

// fleetSecrets are the secrets a cloud executor container holds (ailang-multivac
// cloud_run_jobs.tf, every lane's union) plus a few host-shaped ones.
var fleetSecrets = []string{
	"GITHUB_TOKEN", "GH_TOKEN", "AILANG_REGISTRY_API_KEY", "OPENROUTER_API_KEY", "GEMINI_API_KEY",
	"GOOGLE_API_KEY", "OLLAMA_API_KEY", "OPENAI_API_KEY", "ANTHROPIC_API_KEY", "CLAUDE_CODE_OAUTH_TOKEN",
	"AILANG_KMS_KEY", "AILANG_CODEX_AUTH_SECRET", "AILANG_SSH_KEY_SECRET", "AWS_SECRET_ACCESS_KEY",
	"SSH_AUTH_SOCK", "NPM_TOKEN", "AILANG_DISCORD_WEBHOOK_URL", "GOOGLE_APPLICATION_CREDENTIALS",
}

// AC1's name patterns: anything that looks like a credential.
var credentialPattern = regexp.MustCompile(`(?i)(_API_KEY$|TOKEN|SECRET|_KEY$|SSH_AUTH_SOCK|^AWS_|WEBHOOK|CREDENTIALS)`)

func setFleetSecrets(t *testing.T) {
	t.Helper()
	t.Setenv("AILANG_AUTH_MODE", "")
	t.Setenv("AILANG_EXECUTOR_ENV_INHERIT", "")
	for _, n := range fleetSecrets {
		t.Setenv(n, "secret-value-of-"+n)
	}
	t.Setenv("CLAUDE_CODE_MAX_OUTPUT_TOKENS", "16000")
	t.Setenv("AILANG_MODEL", "openrouter/z-ai/glm-5.3")
	t.Setenv("SOME_UNLISTED_TOOL_VAR", "x")
}

func credentialNames(t *testing.T, env []string) []string {
	t.Helper()
	var out []string
	for name := range envMap(t, env) {
		if credentialPattern.MatchString(name) && name != "CLAUDE_CODE_MAX_OUTPUT_TOKENS" {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// AC1: what CAN an injected prompt read? Per executor, the only
// credential-bearing name in the child env is that lane's ratified grant.
func TestBuildEnvironment_DefaultProfiles_OnlyTheLaneCredential(t *testing.T) {
	setFleetSecrets(t)
	cases := []struct {
		executor, model string
		apikey          bool
		want            []string
	}{
		{"claude", "sonnet", false, nil},
		{"claude", "sonnet", true, []string{"ANTHROPIC_API_KEY"}},
		{"codex", "gpt-5.5", false, []string{"OPENAI_API_KEY"}},
		{"pi", "openrouter/z-ai/glm-5.3", false, []string{"OPENROUTER_API_KEY"}},
		{"pi", "google/gemini-3-5-flash", false, []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}},
		{"pi", "ollama/qwen3.6", false, []string{"OLLAMA_API_KEY"}},
		{"pi", "openai/gpt-5.5", false, []string{"OPENAI_API_KEY"}},
		{"pi", "rig/custom-model", false, nil},
		{"opencode", "openrouter/qwen/qwen3", false, []string{"OPENROUTER_API_KEY"}},
		{"motoko", "openrouter/z-ai/glm-5.3", false, []string{"GEMINI_API_KEY", "OPENAI_API_KEY", "OPENROUTER_API_KEY"}},
		{"", "", false, nil},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s/%s/apikey=%v", tc.executor, tc.model, tc.apikey), func(t *testing.T) {
			if tc.apikey {
				t.Setenv("AILANG_AUTH_MODE", "apikey")
			}
			env, err := BuildEnvironment(EnvironmentOptions{Executor: tc.executor, Model: tc.model, Task: &Task{ID: "t"}})
			if err != nil {
				t.Fatal(err)
			}
			got := credentialNames(t, env)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Errorf("credential names in child env = %v, want %v", got, tc.want)
			}
			m := envMap(t, env)
			for _, keep := range []string{"PATH", "CLAUDE_CODE_MAX_OUTPUT_TOKENS", "AILANG_MODEL", "AILANG_RIG_LEASE"} {
				if _, ok := m[foldEnvKey(keep)]; !ok {
					t.Errorf("%s must survive the profile", keep)
				}
			}
			if _, ok := m["SOME_UNLISTED_TOOL_VAR"]; ok {
				t.Error("an unlisted host variable must not be inherited (default-deny)")
			}
		})
	}
}

// The operator's AILANG_EXECUTOR_ENV_INHERIT is the explicit grant/rollback
// lever; a task cannot add one.
func TestBuildEnvironment_OperatorInheritGrant(t *testing.T) {
	setFleetSecrets(t)
	t.Setenv("AILANG_EXECUTOR_ENV_INHERIT", "GITHUB_TOKEN, SOME_UNLISTED_TOOL_VAR")
	m := mustBuild(t, EnvironmentOptions{Executor: "pi", Model: "openrouter/x/y", Task: &Task{}})
	for _, n := range []string{"GITHUB_TOKEN", "SOME_UNLISTED_TOOL_VAR", "OPENROUTER_API_KEY"} {
		if _, ok := m[n]; !ok {
			t.Errorf("%s granted by the operator but missing", n)
		}
	}
	if _, ok := m["AILANG_REGISTRY_API_KEY"]; ok {
		t.Error("an ungranted credential leaked")
	}
}

// The ailang_only lane's restricted worker selects its credentials from the
// pi child's env, so an attached policy grants exactly those.
func TestBuildEnvironment_PolicyGrantsWorkerCredentials(t *testing.T) {
	setFleetSecrets(t)
	gemini := "allowed_caps = [\"IO\", \"AI\"]\nai_provider = \"gemini-3-5-flash-lite\"\nentry = \"main\"\ntimeout_ms = 30000\n[budgets]\nAI = 2\n"
	m := mustBuild(t, EnvironmentOptions{Executor: "pi", Model: "openrouter/x/y", Task: &Task{PolicyPath: writeTestPolicy(t, gemini)}})
	for _, n := range []string{"OPENROUTER_API_KEY", "GOOGLE_API_KEY", "GEMINI_API_KEY"} {
		if _, ok := m[n]; !ok {
			t.Errorf("%s must be granted (pi model or policy-pinned provider)", n)
		}
	}
	if _, ok := m["OLLAMA_API_KEY"]; ok {
		t.Error("a key neither the model nor the policy needs leaked")
	}
	if _, err := BuildEnvironment(EnvironmentOptions{Executor: "pi", Task: &Task{PolicyPath: filepath.Join(t.TempDir(), "missing.toml")}}); err == nil {
		t.Error("an unreadable policy must fail the build, not run with guessed grants")
	}
}

func TestIsCredentialName(t *testing.T) {
	yes := []string{"GITHUB_TOKEN", "AILANG_REGISTRY_API_KEY", "AILANG_KMS_KEY", "CLAUDE_CODE_OAUTH_TOKEN", "AWS_REGION",
		"SSH_AUTH_SOCK", "AILANG_DISCORD_WEBHOOK_URL", "GOOGLE_APPLICATION_CREDENTIALS", "db_password", "SENTRY_DSN"}
	no := []string{"CLAUDE_CODE_MAX_OUTPUT_TOKENS", "AILANG_OLLAMA_MAX_TOKENS", "AILANG_SESSION_ID", "AILANG_AUTH_MODE",
		"GOPRIVATE", "PATH", "MOTOKO_AST_AUTOREAD", "AILANG_MODEL", "PI_WORKSPACE_TRUST_REMOTES"}
	for _, n := range yes {
		if !IsCredentialName(n) {
			t.Errorf("IsCredentialName(%q) = false, want true", n)
		}
	}
	for _, n := range no {
		if IsCredentialName(n) {
			t.Errorf("IsCredentialName(%q) = true, want false", n)
		}
	}
}

// D6 / AC5: the digest is over names only, stable for identical tasks.
func TestEnvNamesDigest_NamesOnly(t *testing.T) {
	a := EnvNamesDigest([]string{"B=1", "A=secret-one"})
	b := EnvNamesDigest([]string{"A=secret-two", "B=2"})
	if a != b {
		t.Error("digest must not depend on values or order")
	}
	if a == EnvNamesDigest([]string{"A=1", "B=1", "C=1"}) {
		t.Error("digest must change when the name set changes")
	}
	if len(a) != 64 || strings.Contains(a, "secret") {
		t.Errorf("digest %q is not a bare sha256 hex", a)
	}
}

// The adversarial fixture: a real child process launched with the built env
// dumps what it can see.
func TestBuildEnvironment_FixtureChildSeesNoFleetSecret(t *testing.T) {
	if os.Getenv("AILANG_ENV_FIXTURE_CHILD") == "1" {
		for _, kv := range os.Environ() {
			fmt.Println(kv)
		}
		os.Exit(0)
	}
	setFleetSecrets(t)
	env, err := BuildEnvironment(EnvironmentOptions{Executor: "pi", Model: "openrouter/x/y", Task: &Task{ID: "t"}})
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestBuildEnvironment_FixtureChildSeesNoFleetSecret$")
	cmd.Env = append(env, "AILANG_ENV_FIXTURE_CHILD=1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("fixture child: %v", err)
	}
	for _, n := range fleetSecrets {
		if n == "OPENROUTER_API_KEY" {
			continue
		}
		if strings.Contains(string(out), "secret-value-of-"+n) {
			t.Errorf("fixture child read %s", n)
		}
	}
	if !strings.Contains(string(out), "secret-value-of-OPENROUTER_API_KEY") {
		t.Error("the lane's inference credential must reach the child")
	}
}
