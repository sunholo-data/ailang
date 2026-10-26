package pi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"strings"
	"sync"

	"github.com/sunholo-data/ailang/internal/executor"
)

type piUsageCapture struct {
	mu       sync.Mutex
	digest   hash.Hash
	evidence executor.UsageEvidence
}

func newPiUsageCapture() *piUsageCapture {
	return &piUsageCapture{digest: sha256.New(), evidence: executor.UsageEvidence{
		Source: "pi-ndjson", Version: "settled-message-usage/1",
		Observations:       []executor.UsageObservation{},
		UnavailableReasons: []string{"pi stream does not attest exhaustive provider request coverage"},
	}}
}

func (c *piUsageCapture) gap(reason string) {
	for _, old := range c.evidence.UnavailableReasons {
		if old == reason {
			return
		}
	}
	c.evidence.UnavailableReasons = append(c.evidence.UnavailableReasons, reason)
}

func (c *piUsageCapture) observe(line []byte, ev *piEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.digest.Write(line)
	c.digest.Write([]byte{'\n'})
	if ev == nil {
		c.gap("unparsed stream line")
		return
	}
	switch ev.Type {
	case "agent_settled":
		c.evidence.Settled = true
	case "auto_retry_start":
		c.evidence.RetryEvents++
		c.gap("retry request coverage unreported")
	case "message_end":
		if ev.Message == nil || ev.Message.Role != "assistant" {
			return
		}
		if ev.Message.Usage == nil {
			c.gap("assistant message without usage")
			return
		}
		c.addUsage(line, ev.Message)
	default:
		if strings.Contains(ev.Type, "compaction") {
			c.evidence.CompactionEvents++
			c.gap("compaction request coverage unreported")
		}
	}
}

func (c *piUsageCapture) addUsage(line []byte, message *piMessage) {
	var raw struct {
		Input     *int `json:"input"`
		Output    *int `json:"output"`
		Read      *int `json:"cacheRead"`
		Write     *int `json:"cacheWrite"`
		Reasoning *int `json:"reasoning"`
		Total     *int `json:"totalTokens"`
		Cost      struct {
			Total json.RawMessage `json:"total"`
		} `json:"cost"`
	}
	if err := json.Unmarshal(message.Usage.Raw, &raw); err != nil {
		c.gap("invalid usage fields")
		return
	}
	digest := sha256.Sum256(line)
	hexDigest := hex.EncodeToString(digest[:])
	u := executor.UsageObservation{
		ImportID: fmt.Sprintf("message-%d-%s", len(c.evidence.Observations)+1, hexDigest), EventDigest: hexDigest,
		RequestID: message.RequestID, Provider: message.Provider, Model: message.Model,
		Input: raw.Input, Output: raw.Output, CacheRead: raw.Read, CacheWrite: raw.Write,
		Reasoning: raw.Reasoning, ReportedTotal: raw.Total, Contract: "exclusive/1", OutputScheme: "combined",
		CostBasis: "unknown", Failed: message.StopReason == "error" || message.StopReason == "aborted",
		RawUsage: append([]byte(nil), message.Usage.Raw...),
	}
	if len(raw.Cost.Total) > 0 && string(raw.Cost.Total) != "null" {
		// The original number lexeme survives. Pi prices are a local table,
		// never a provider invoice, even when the decimal is exactly representable.
		amount := string(raw.Cost.Total)
		if raw.Cost.Total[0] != '"' {
			u.CostAmount = &amount
			u.CostBasis = "list_price"
		}
	}
	c.evidence.Observations = append(c.evidence.Observations, u)
}

func (c *piUsageCapture) snapshot() *executor.UsageEvidence {
	c.mu.Lock()
	defer c.mu.Unlock()
	x := c.evidence
	x.StreamDigest = hex.EncodeToString(c.digest.Sum(nil))
	x.Observations = append([]executor.UsageObservation{}, x.Observations...)
	x.UnavailableReasons = append([]string{}, x.UnavailableReasons...)
	if !x.Settled {
		x.UnavailableReasons = append(x.UnavailableReasons, "stream did not settle")
	}
	return &x
}
