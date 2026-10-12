package claudegateway

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sunholo-data/ailang/internal/creditbudget"
	"github.com/sunholo-data/ailang/internal/modelreg"
)

const AccountID = "anthropic-api-credits"
const upstreamURL = "https://api.anthropic.com/v1/messages"
const requestTimeout = 10 * time.Minute

// Gateway's provider key is loaded only in the isolated service. Authenticate
// verifies a Google ID token for this service audience and returns its identity.
type Gateway struct {
	Authority    creditbudget.Authority
	ProviderKey  string
	SigningKey   []byte
	PublicURL    string
	Client       *http.Client
	Authenticate func(context.Context, string) (string, error)
	Operators    map[string]bool
	Coordinators map[string]string // authenticated coordinator -> permitted job SA
	Now          func() time.Time
	Diagnostic   func(UsageDiagnostic) // optional test/structured logging sink
	Receipt      func(UsageReceipt)    // verified billing receipt after durable settlement
}

func (g *Gateway) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}
func fail(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"type": "error", "error": map[string]string{"type": "permission_error", "message": msg}})
}
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/admin/") {
		g.admin(w, r)
		return
	}
	isRelease := r.URL.Path == "/v1/tasks/release"
	if (r.URL.Path != "/v1/messages" && !isRelease) || r.Method != http.MethodPost {
		fail(w, 404, "unsupported route")
		return
	}
	cap, err := VerifyCapability(g.SigningKey, r.Header.Get("x-api-key"), g.now())
	if err != nil || cap.AccountID != AccountID {
		fail(w, 401, "invalid task capability")
		return
	}
	if g.Authenticate == nil {
		fail(w, 503, "identity verifier unavailable")
		return
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		fail(w, 401, "job identity required")
		return
	}
	identity, err := g.Authenticate(r.Context(), strings.TrimPrefix(auth, "Bearer "))
	if err != nil || (identity != cap.JobIdentity && (!isRelease || g.Coordinators[identity] != cap.JobIdentity)) {
		fail(w, 403, "job identity does not match capability")
		return
	}
	if isRelease {
		if g.Authority == nil {
			fail(w, 503, "credit authority unavailable")
			return
		}
		if err = g.Authority.ReleaseAttempt(r.Context(), cap.AccountID, cap.TaskID, cap.AttemptID, cap.JobIdentity); err != nil {
			fail(w, 409, "task release refused")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if g.Authority == nil || g.ProviderKey == "" {
		fail(w, 503, "credit authority unavailable")
		return
	}
	// Experimental betas can enable different billing categories. Admission is an
	// allowlist, not a promise to proxy every future Claude Code feature.
	for _, beta := range strings.Split(r.Header.Get("anthropic-beta"), ",") {
		switch strings.TrimSpace(beta) {
		case "", "prompt-caching-2024-07-31", "extended-cache-ttl-2025-04-11":
		default:
			fail(w, 400, "unsupported beta feature")
			return
		}
	}
	if r.Header.Get("anthropic-version") != "2023-06-01" {
		fail(w, 400, "unsupported API version")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 8<<20))
	if err != nil {
		fail(w, 400, "request too large")
		return
	}
	model, maxTokens, stream, err := validateMessage(raw)
	if err != nil {
		fail(w, 400, err.Error())
		return
	}
	amount, revision, err := modelreg.ReserveClaudeRequest(model, maxTokens)
	if err != nil {
		fail(w, 503, "pricing contract unavailable")
		return
	}
	deadline := g.now().Add(requestTimeout)
	if cap.ExpiresAt.Before(deadline) {
		fail(w, 402, "task capability expires before request deadline")
		return
	}
	requestID := uuid.NewString()
	reservation := creditbudget.Reservation{AccountID: cap.AccountID, TaskID: cap.TaskID, RequestID: requestID, Amount: creditbudget.MicroUSD(amount), PricingRevision: revision, Deadline: deadline, AttemptID: cap.AttemptID, JobIdentity: cap.JobIdentity, Model: model}
	if _, err = g.Authority.Reserve(r.Context(), reservation); err != nil {
		fail(w, 402, "credit budget blocked; inspect coordinator credits status")
		return
	}
	// The forwarding marker is durable BEFORE opening the upstream connection.
	// A crash from this point onward must retain the full reservation.
	if err = g.Authority.MarkForwarded(r.Context(), cap.AccountID, requestID); err != nil {
		fail(w, 503, "cannot durably mark request; reservation retained")
		return
	}
	settleCtx, settleCancel := context.WithTimeout(context.WithoutCancel(r.Context()), requestTimeout+30*time.Second)
	defer settleCancel()
	var trace usageTrace
	var upstreamID, providerRequestID string
	stage := "upstream_connect"
	unresolved := func() {
		markErr := g.Authority.MarkUnresolved(settleCtx, cap.AccountID, requestID)
		g.reportUnresolved(UsageDiagnostic{
			Event: "credit_usage_unresolved", AccountID: cap.AccountID,
			TaskID: diagnosticID(cap.TaskID), RequestID: requestID,
			ProviderRequestID: diagnosticID(providerRequestID), MessageID: diagnosticID(upstreamID),
			Stage: stage, Reason: diagnosticReason(err), UnresolvedMarked: markErr == nil, Stream: trace,
		})
	}
	ctx, cancel := context.WithDeadline(r.Context(), deadline)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(raw))
	if err != nil {
		unresolved()
		fail(w, 502, "upstream request failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", g.ProviderKey)
	req.Header.Set("anthropic-version", r.Header.Get("anthropic-version"))
	if b := r.Header.Get("anthropic-beta"); b != "" {
		req.Header.Set("anthropic-beta", b)
	}
	client := g.Client
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	// Never follow an upstream redirect with a credential, or admit another send
	// under the first reservation. Copy rather than mutating a shared client.
	copyClient := *client
	copyClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := copyClient.Do(req)
	if err != nil {
		unresolved()
		fail(w, 502, "upstream outcome unknown; reservation retained")
		return
	}
	defer resp.Body.Close()
	providerRequestID = resp.Header.Get("request-id")
	stage = "upstream_status"
	if resp.StatusCode != 200 {
		unresolved()
		fail(w, 502, "upstream did not return verifiable usage; reservation retained")
		return
	}
	w.Header().Set("X-Ailang-Credit-Request", requestID)
	if id := resp.Header.Get("request-id"); id != "" {
		w.Header().Set("request-id", id)
	}
	var usage modelreg.ClaudeUsage
	if stream {
		stage = "stream_usage"
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
			unresolved()
			fail(w, 502, "invalid upstream stream")
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		usage, upstreamID, err = relayStreamWithTrace(w, resp.Body, model, maxTokens, &trace)
	} else {
		stage = "message_usage"
		var body []byte
		body, err = io.ReadAll(io.LimitReader(resp.Body, 32<<20+1))
		if err == nil && len(body) > 32<<20 {
			err = errors.New("response too large")
		}
		if err == nil {
			if obj, decodeErr := decodeObject(body); decodeErr == nil {
				trace.Final = usageObservation(obj["usage"])
				trace.FinalSeen = true
				json.Unmarshal(obj["id"], &upstreamID)
			}
			var verifiedID string
			usage, verifiedID, err = messageUsage(body, model, maxTokens)
			if err == nil {
				upstreamID = verifiedID
				trace.observeVerified(usage)
			}
		}
		if err == nil {
			w.Header().Set("Content-Type", "application/json")
			_, err = w.Write(body)
		}
	}
	if err != nil {
		unresolved()
		return
	}
	stage = "pricing"
	cost, err := modelreg.ClaudeUsageCostAtRevision(model, usage, revision)
	if err != nil {
		unresolved()
		return
	}
	stage = "settlement"
	if err = g.Authority.Settle(settleCtx, cap.AccountID, requestID, creditbudget.MicroUSD(cost), upstreamID); err != nil {
		unresolved()
		return
	}
	g.reportSettled(UsageReceipt{Event: "credit_usage_settled", AccountID: cap.AccountID,
		TaskID: cap.TaskID, RequestID: requestID, ProviderRequestID: providerRequestID,
		MessageID: upstreamID, Model: model, PricingRevision: revision, CostMicroUSD: cost, Usage: usage})
}
func (g *Gateway) admin(w http.ResponseWriter, r *http.Request) {
	if g.Authenticate == nil || g.Authority == nil {
		fail(w, 503, "admin authority unavailable")
		return
	}
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		fail(w, 401, "Google ID token required")
		return
	}
	identity, err := g.Authenticate(r.Context(), strings.TrimPrefix(auth, "Bearer "))
	if err != nil {
		fail(w, 401, "invalid identity")
		return
	}
	if r.URL.Path == "/admin/tasks" && r.Method == http.MethodPost {
		jobIdentity, ok := g.Coordinators[identity]
		if !ok {
			fail(w, 403, "coordinator identity required")
			return
		}
		var task creditbudget.Task
		if err = decodeAdmin(w, r, &task); err != nil {
			fail(w, 400, err.Error())
			return
		}
		if task.AccountID != AccountID || len(task.Models) != 1 || task.Models[0] != "claude-haiku-5-5" || task.JobIdentity != jobIdentity || task.LeaseUntil.After(g.now().Add(time.Hour)) || !task.LeaseUntil.After(g.now().Add(requestTimeout)) {
			fail(w, 400, "invalid task contract")
			return
		}
		cap := Capability{AccountID: task.AccountID, TaskID: task.ID, AttemptID: task.AttemptID, JobIdentity: jobIdentity, ExpiresAt: task.LeaseUntil}
		token, err := SignCapability(g.SigningKey, cap)
		if err != nil {
			fail(w, 503, "capability signing unavailable")
			return
		}
		if err = g.Authority.AdmitTask(r.Context(), task); err != nil {
			fail(w, 402, "credit budget blocked")
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"capability": token, "gateway_url": g.PublicURL, "account_id": task.AccountID})
		return
	}
	if !g.Operators[identity] {
		fail(w, 403, "credit operator identity required")
		return
	}
	prefix := "/admin/credits/" + AccountID + "/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		fail(w, 404, "unknown credit account")
		return
	}
	action := strings.TrimPrefix(r.URL.Path, prefix)
	if g.recoveryAdmin(w, r, identity, action) {
		return
	}
	var status creditbudget.Status
	switch {
	case action == "status" && r.Method == http.MethodGet:
		status, err = g.Authority.Status(r.Context(), AccountID)
	case action == "confirm" && r.Method == http.MethodPost:
		var c creditbudget.Confirmation
		if err = decodeAdmin(w, r, &c); err != nil {
			fail(w, 400, err.Error())
			return
		}
		if c.AccountID != AccountID || c.Operator != "" {
			fail(w, 400, "invalid account or caller-supplied operator")
			return
		}
		c.Operator = identity
		status, err = g.Authority.Confirm(r.Context(), c)
	case action == "promote" && r.Method == http.MethodPost:
		var c struct{ Evidence string }
		if err = decodeAdmin(w, r, &c); err != nil {
			fail(w, 400, err.Error())
			return
		}
		err = g.Authority.CompleteCanary(r.Context(), AccountID, identity, c.Evidence)
		if err == nil {
			status, err = g.Authority.Status(r.Context(), AccountID)
		}
	case action == "enabled" && r.Method == http.MethodPost:
		var c struct {
			Enabled  bool
			Evidence string
		}
		if err = decodeAdmin(w, r, &c); err != nil {
			fail(w, 400, err.Error())
			return
		}
		err = g.Authority.SetEnabled(r.Context(), AccountID, c.Enabled, identity, c.Evidence)
		if err == nil {
			status, err = g.Authority.Status(r.Context(), AccountID)
		}
	default:
		fail(w, 404, "unsupported admin route")
		return
	}
	if err != nil {
		fail(w, 409, fmt.Sprintf("credit operation refused: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
func decodeAdmin(w http.ResponseWriter, r *http.Request, v any) error {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16384))
	if err != nil {
		return errors.New("admin request too large")
	}
	if _, err = decodeObject(raw); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	return d.Decode(v)
}
