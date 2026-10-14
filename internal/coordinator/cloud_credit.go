package coordinator

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/creditbudget"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"google.golang.org/api/idtoken"
)

// CloudCreditCapability contains only a task-scoped gateway credential.
// The Anthropic provider credential is never available to the dispatcher/job.
type CloudCreditCapability struct {
	Capability string `json:"capability"`
	GatewayURL string `json:"gateway_url"`
	AccountID  string `json:"account_id"`
}

// applyCloudCreditAgentConfig runs after legacy provider budget resolution.
// A missing guarded limit stays zero so admission fails closed, rather than
// inheriting a provider budget that belongs to a different spending lane.
func applyCloudCreditAgentConfig(p *DispatchParams, agent *AgentConfig) {
	if agent == nil {
		return
	}
	p.CreditAccount = agent.CreditAccount
	p.CreditGatewayURL = agent.CreditGatewayURL
	p.CreditJobIdentity = agent.CreditJobIdentity
	if agent.CreditAccount != "" {
		p.MaxCostUSD = agent.CreditMaxCostUSD
	}
}

func ValidateCloudCreditLane(p DispatchParams) error {
	if p.CreditAccount == "" {
		return nil
	}
	if p.CreditAccount != "anthropic-api-credits" || p.Provider != "claude" || p.AuthMode != "apikey" || p.Model != modelreg.ClaudeCreditModel || p.APIKey != "" {
		return fmt.Errorf("credit lane requires Claude apikey Haiku 5.5 with no provider key override")
	}
	if math.IsNaN(p.MaxCostUSD) || math.IsInf(p.MaxCostUSD, 0) || p.MaxCostUSD < 0.000001 || p.MaxCostUSD > 2 {
		return fmt.Errorf("credit lane requires a positive task budget at most $2")
	}
	u, e := url.Parse(p.CreditGatewayURL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("credit lane requires an HTTPS gateway origin")
	}
	if p.CreditJobIdentity == "" || p.TaskID == "" {
		return fmt.Errorf("credit lane requires task and expected job service-account identity")
	}
	return nil
}

// ErrCreditGatewayFromTest refuses a real gateway identity inside a test binary.
//
// A cloud agent's environment carries its live task capability and gateway
// URL, and agents run `go test` there. On 2026-10-12 the evaluator ran
// `go test ./cmd/ailang/...` in its own job; a test calling
// coordinatorExecuteJob reached the deferred lease release with that live
// capability, the gateway released the task mid-run, and the evaluator's next
// request was refused (402, task-f257f24d). Tests pass a stub token instead.
var ErrCreditGatewayFromTest = errors.New("credit gateway: refusing a real identity token from a test binary")

func CloudCreditIdentityToken(ctx context.Context, audience string) (string, error) {
	if testing.Testing() {
		return "", ErrCreditGatewayFromTest
	}
	s, e := idtoken.NewTokenSource(ctx, audience)
	if e != nil {
		return "", e
	}
	t, e := s.Token()
	if e != nil {
		return "", e
	}
	return t.AccessToken, nil
}
func AdmitCloudCreditTask(ctx context.Context, p DispatchParams, client *http.Client, token func(context.Context, string) (string, error)) (CloudCreditCapability, error) {
	var result CloudCreditCapability
	if err := ValidateCloudCreditLane(p); err != nil {
		return result, err
	}
	if p.CreditAccount == "" {
		return result, fmt.Errorf("no credit account")
	}
	timeout := 30 * time.Minute
	if p.Timeout != "" {
		parsed, e := time.ParseDuration(p.Timeout)
		if e != nil || parsed <= 0 {
			return result, fmt.Errorf("invalid guarded task timeout")
		}
		timeout = parsed
	}
	if timeout > 50*time.Minute {
		return result, fmt.Errorf("guarded task timeout exceeds 50m identity-token lifetime allowance")
	}
	task := creditbudget.Task{AccountID: p.CreditAccount, ID: p.TaskID, AttemptID: uuid.NewString(), JobIdentity: p.CreditJobIdentity, Models: []string{p.Model}, Ceiling: creditbudget.MicroUSD(math.Floor(p.MaxCostUSD * 1e6)), LeaseUntil: time.Now().UTC().Add(timeout + 5*time.Minute)}
	payload, e := json.Marshal(task)
	if e != nil {
		return result, e
	}
	if token == nil {
		token = CloudCreditIdentityToken
	}
	identity, e := token(ctx, p.CreditGatewayURL)
	if e != nil {
		return result, e
	}
	if identity == "" {
		return result, fmt.Errorf("empty coordinator identity token")
	}
	req, e := http.NewRequestWithContext(ctx, http.MethodPost, p.CreditGatewayURL+"/admin/tasks", bytes.NewReader(payload))
	if e != nil {
		return result, e
	}
	req.Header.Set("Authorization", "Bearer "+identity)
	req.Header.Set("X-Serverless-Authorization", "Bearer "+identity)
	req.Header.Set("Content-Type", "application/json")
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, e := copyClient.Do(req)
	if e != nil {
		return result, e
	}
	defer response.Body.Close()
	data, e := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if e != nil {
		return result, e
	}
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("credit task blocked (%d): %s", response.StatusCode, data)
	}
	if e = json.Unmarshal(data, &result); e != nil {
		return result, e
	}
	if result.Capability == "" || result.AccountID != p.CreditAccount || result.GatewayURL != p.CreditGatewayURL {
		return CloudCreditCapability{}, fmt.Errorf("credit authority returned inconsistent task capability")
	}
	return result, nil
}

// ReleaseCloudCreditTask releases a signed attempt's concurrency lease only.
// Existing reservations and unresolved provider exposure remain fully charged.
func ReleaseCloudCreditTask(ctx context.Context, cap CloudCreditCapability, client *http.Client, token func(context.Context, string) (string, error)) error {
	u, err := url.Parse(cap.GatewayURL)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || cap.Capability == "" || cap.AccountID != "anthropic-api-credits" {
		return fmt.Errorf("invalid scoped credit release contract")
	}
	if token == nil {
		token = CloudCreditIdentityToken
	}
	identity, err := token(ctx, cap.GatewayURL)
	if err != nil {
		return err
	}
	if identity == "" {
		return fmt.Errorf("empty release identity token")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cap.GatewayURL+"/v1/tasks/release", bytes.NewBufferString("{}"))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+identity)
	req.Header.Set("X-Serverless-Authorization", "Bearer "+identity)
	req.Header.Set("x-api-key", cap.Capability)
	req.Header.Set("Content-Type", "application/json")
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	response, err := copyClient.Do(req)
	if err != nil {
		return fmt.Errorf("credit lease release unavailable: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNoContent {
		return fmt.Errorf("credit lease release rejected (%d); inspect canonical credit status", response.StatusCode)
	}
	return nil
}
