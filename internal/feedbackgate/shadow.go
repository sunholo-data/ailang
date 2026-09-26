package feedbackgate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// M-AI-DECIDE-SYSTEM-ONE, audit site #1 (design_docs/planned/m-ai-decide-system-one-audit.md):
// ask a System One decision model (TypeSafe Jev, via the sunholo/decisions
// package) the classifier's four questions BESIDE the Haiku classifier, and
// record both answers in the audit row. The shadow never changes a Verdict —
// it exists to measure agreement on live traffic before any switch is designed.
//
// Off by default. Enabling it (config `shadow: openrouter|direct` or
// AILANG_FEEDBACK_GATE_SHADOW) sends the submission body to TypeSafe (via
// OpenRouter or directly) — that is the data-boundary ruling, made by whoever
// enables it, not by this code.

// Shadow modes. ShadowOff disables the stage; the other two name the wire the
// System One call takes (see sunholo/decisions Transport).
const (
	ShadowOff        = "off"
	ShadowOpenRouter = "openrouter"
	ShadowDirect     = "direct"
)

// ShadowRunner produces a System One answer for one Input. Implementations
// must be safe to call concurrently. The production runner is a subprocess of
// the ailang binary running internal/feedbackgate/shadow/feedback_shadow.ail;
// tests inject a fake.
type ShadowRunner interface {
	Run(ctx context.Context, in Input) (ShadowResult, error)
}

// ShadowResult is the parsed JSON line the shadow program prints. Field names
// match the program's output; Error is non-empty when the model call failed
// (the row is still recorded — an unavailable oracle is data).
type ShadowResult struct {
	Transport    string  `json:"transport"`
	Model        string  `json:"model"`
	ID           string  `json:"id"`
	Degraded     bool    `json:"degraded"`
	DegradedWhy  string  `json:"degraded_why"`
	LatencyMs    int     `json:"latency_ms"`
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	CostUSD      float64 `json:"cost_usd"`
	ListPriceUSD float64 `json:"list_price_usd"`
	Genuine      struct {
		P float64 `json:"p"`
	} `json:"genuine"`
	Injection struct {
		P float64 `json:"p"`
	} `json:"injection"`
	Category struct {
		Choice        string             `json:"choice"`
		Confidence    float64            `json:"confidence"`
		Probabilities map[string]float64 `json:"probabilities"`
	} `json:"category"`
	Value struct {
		Score         float64            `json:"score"`
		Label         string             `json:"label"`
		Confidence    float64            `json:"confidence"`
		Probabilities map[string]float64 `json:"probabilities"`
	} `json:"value"`
	Error string `json:"error"`
}

// ShadowVerdict is what rides on a Verdict when the shadow ran: the raw
// result plus what classifierVerdict WOULD have returned on the System One
// answers, and whether that agrees with the Haiku verdict actually applied.
type ShadowVerdict struct {
	Result      ShadowResult
	WouldAction string
	WouldReason string
	Agrees      bool
	// Err is set when the runner itself failed (subprocess, parse). Distinct
	// from Result.Error, which is the model call failing inside the program.
	Err string
}

// shadowThreshold is the boolean threshold: the Haiku classifier returns
// bools; a System One Noul returns P(true). p >= 0.5 is the faithful
// translation of "true", and the SAME rule is applied to both bool fields so
// the comparison measures the oracle, not a tuned threshold. Tuning is a
// design decision for the switch, not for the shadow.
const shadowThreshold = 0.5

// shadowVerdict mirrors classifierVerdict branch for branch on a System One
// answer. Keep the order identical: injection → reject; value none → file;
// not genuine → file; category mismatch → file; else dispatch.
func shadowVerdict(in Input, r ShadowResult) (action, reason string) {
	if r.Error != "" {
		return ActionFile, ReasonClassifierError
	}
	if r.Injection.P >= shadowThreshold {
		return ActionReject, ReasonClassifierInjection
	}
	if r.Value.Label == "none" {
		return ActionFile, ReasonClassifierNoValue
	}
	if r.Genuine.P < shadowThreshold {
		return ActionFile, ReasonClassifierNotGenuine
	}
	if r.Category.Choice != strippedCategory(in.Category) {
		return ActionFile, ReasonClassifierMismatch
	}
	return ActionDispatch, ReasonPassed
}

// runShadow runs the shadow beside a Haiku verdict and attaches the outcome.
// It never alters v.Action/Reason/Cost. A runner error is recorded, not
// returned: the shadow cannot fail the gate.
func runShadow(ctx context.Context, in Input, cfg FeedbackGateConfig, v Verdict) Verdict {
	if cfg.Shadow == nil || cfg.ShadowMode == "" || cfg.ShadowMode == ShadowOff {
		return v
	}
	sv := &ShadowVerdict{}
	res, err := cfg.Shadow.Run(ctx, in)
	if err != nil {
		sv.Err = err.Error()
		sv.WouldAction, sv.WouldReason = ActionFile, ReasonClassifierError
	} else {
		sv.Result = res
		sv.WouldAction, sv.WouldReason = shadowVerdict(in, res)
	}
	sv.Agrees = sv.WouldAction == v.Action
	v.Shadow = sv
	return v
}

// SubprocessShadow runs feedback_shadow.ail with the ailang binary. Dir must be
// the directory holding the program and its ailang.toml (the package resolves
// from there). Transport is ShadowOpenRouter or ShadowDirect; FallbackModel ""
// disables the chat-LLM fallback ("-" on the wire).
type SubprocessShadow struct {
	Binary        string
	Dir           string
	Transport     string
	FallbackModel string
	Timeout       time.Duration // 0 → 45s (the Net deadline is 30s; startup + parse on top)
}

// NewSubprocessShadow builds the production runner.
func NewSubprocessShadow(binary, dir, transport, fallbackModel string) *SubprocessShadow {
	return &SubprocessShadow{Binary: binary, Dir: dir, Transport: transport, FallbackModel: fallbackModel}
}

// Run executes the program and parses its single JSON line.
func (s *SubprocessShadow) Run(ctx context.Context, in Input) (ShadowResult, error) {
	if s == nil || s.Binary == "" || s.Dir == "" {
		return ShadowResult{}, fmt.Errorf("shadow runner not configured (binary=%q dir=%q)", s.Binary, s.Dir)
	}
	timeout := s.Timeout
	if timeout == 0 {
		timeout = 45 * time.Second
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	input, err := json.Marshal(map[string]string{
		"category": strippedCategory(in.Category), "from": in.From, "inbox": in.Inbox, "body": in.Body,
	})
	if err != nil {
		return ShadowResult{}, err
	}
	fallback := s.FallbackModel
	if fallback == "" {
		fallback = "-"
	}
	cmd := exec.CommandContext(cctx, s.Binary, "run", "--caps", "Net,Env,IO,Clock", "--entry", "main",
		"feedback_shadow.ail", s.Transport, fallback, string(input))
	cmd.Dir = s.Dir
	cmd.Env = append(cmd.Environ(), "AILANG_RELAX_MODULES=1")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return ShadowResult{}, fmt.Errorf("shadow subprocess: %w (stderr: %s)", err, lastLine(stderr.String()))
	}
	return ParseShadowOutput(stdout.String())
}

// ParseShadowOutput takes the program's stdout (which may carry pipeline
// chatter before the row) and returns the last JSON line.
func ParseShadowOutput(out string) (ShadowResult, error) {
	line := lastJSONLine(out)
	if line == "" {
		return ShadowResult{}, fmt.Errorf("shadow output has no JSON line: %s", lastLine(out))
	}
	var r ShadowResult
	if err := json.Unmarshal([]byte(line), &r); err != nil {
		return ShadowResult{}, fmt.Errorf("shadow output not parseable: %w", err)
	}
	return r, nil
}

func lastJSONLine(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if strings.HasPrefix(l, "{") && strings.HasSuffix(l, "}") {
			return l
		}
	}
	return ""
}

func lastLine(s string) string {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	if len(lines) == 0 {
		return ""
	}
	return strings.TrimSpace(lines[len(lines)-1])
}
