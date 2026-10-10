package claude

import (
	"context"
	"fmt"
	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"google.golang.org/api/idtoken"
	"math"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// validateCreditEnvironment is a second gate inside the job, before spawning
// Claude. Legacy OAuth and request-scoped API-key lanes retain their contract.
func validateCreditEnvironment(model string) error {
	account := config.ClaudeCreditAccount()
	if account == "" {
		return nil
	}
	if account != "anthropic-api-credits" || config.AuthMode() != "apikey" || model != modelreg.ClaudeCreditModel {
		return fmt.Errorf("credit budget blocked: incoherent Claude credit account/auth/model")
	}
	value, err := strconv.ParseFloat(config.MaxCostUSD(), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0.000001 || value > 2 {
		return fmt.Errorf("credit budget blocked: AILANG_MAX_COST_USD must be positive and at most $2")
	}
	u, err := url.Parse(config.AnthropicBaseURL())
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("credit budget blocked: HTTPS gateway origin required")
	}
	if config.AnthropicAPIKey() == "" || strings.HasPrefix(config.AnthropicAPIKey(), "ENC:") {
		return fmt.Errorf("credit budget blocked: gateway capability required")
	}
	return nil
}

// guardCreditChildEnvironment removes alternate routes and subscription tokens
// even if the operator's ordinary environment inheritance grants them.
func guardCreditChildEnvironment(env []string, gateway string) []string {
	// Keep MCP tools upfront and disable the separate server auto-mode
	// review channel. Any local classifier request still needs gateway admission.
	// https://code.claude.com/docs/en/mcp#configure-tool-search
	// https://code.claude.com/docs/en/env-vars#claude_code_auto_mode_server
	// The installed 2.1.295 proxy-compatibility switch disables the CLI's beta
	// extensions at their source. The gateway still rejects unverified betas and
	// body fields if a later CLI changes this behavior. See the recorded CLI
	// contract; no undocumented provider capability is forwarded or priced.
	forced := map[string]string{"ANTHROPIC_BASE_URL": gateway, "ANTHROPIC_MODEL": modelreg.ClaudeCreditModel, "ANTHROPIC_DEFAULT_HAIKU_MODEL": modelreg.ClaudeCreditModel, "ANTHROPIC_DEFAULT_SONNET_MODEL": modelreg.ClaudeCreditModel, "ANTHROPIC_DEFAULT_OPUS_MODEL": modelreg.ClaudeCreditModel, "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": "1", "ENABLE_TOOL_SEARCH": "false", "CLAUDE_CODE_AUTO_MODE_SERVER": "0", "CLAUDE_CODE_SIMULATE_PROXY_USAGE": "1"}
	remove := map[string]bool{"CLAUDE_CODE_OAUTH_TOKEN": true, "CLAUDE_CODE_OAUTH_TOKEN_HELPER": true, "ANTHROPIC_AUTH_TOKEN": true, "ANTHROPIC_CUSTOM_HEADERS": true, "CLAUDE_CODE_USE_VERTEX": true, "CLAUDE_CODE_USE_BEDROCK": true, "CLAUDE_CODE_USE_FOUNDRY": true}
	result := make([]string, 0, len(env)+len(forced))
	for _, v := range env {
		name, _, _ := strings.Cut(v, "=")
		if remove[name] {
			continue
		}
		if _, overridden := forced[name]; overridden {
			continue
		}
		result = append(result, v)
	}
	for k, v := range forced {
		result = append(result, k+"="+v)
	}
	return result
}

func creditJobIdentityHeader(ctx context.Context, gateway string, timeout time.Duration) (string, error) {
	if timeout <= 0 || timeout > 50*time.Minute {
		return "", fmt.Errorf("credit budget blocked: task timeout must be positive and at most 50m")
	}
	source, err := idtoken.NewTokenSource(ctx, gateway)
	if err != nil {
		return "", fmt.Errorf("authenticate guarded job: %w", err)
	}
	token, err := source.Token()
	if err != nil {
		return "", fmt.Errorf("authenticate guarded job: %w", err)
	}
	if token.AccessToken == "" || strings.ContainsAny(token.AccessToken, "\r\n") {
		return "", fmt.Errorf("invalid guarded job identity token")
	}
	return creditIdentityHeaders(token.AccessToken), nil
}

func prepareCreditChildEnvironment(ctx context.Context, env []string, timeout time.Duration, defaultSeconds int) ([]string, error) {
	if config.ClaudeCreditAccount() == "" {
		return env, nil
	}
	gateway := config.AnthropicBaseURL()
	env = guardCreditChildEnvironment(env, gateway)
	if timeout == 0 {
		timeout = time.Duration(defaultSeconds) * time.Second
	}
	header, err := creditJobIdentityHeader(ctx, gateway, timeout)
	if err != nil {
		return nil, err
	}
	return append(env, header), nil
}

// Both headers are intentional: Cloud Run verifies/strips the Serverless
// header, leaving Authorization intact for the gateway's cryptographic check.
// https://docs.cloud.google.com/run/docs/authenticating/service-to-service
func creditIdentityHeaders(token string) string {
	return "ANTHROPIC_CUSTOM_HEADERS=Authorization: Bearer " + token + "\nX-Serverless-Authorization: Bearer " + token
}
