package modelreg

import (
	"math"
	"testing"
)

func creditRegistry(t *testing.T) *ModelsConfig {
	t.Helper()
	c, err := LoadModelsConfigBytes(embeddedModelsYAML)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestClaudeCreditReservationUsesFullWindow(t *testing.T) {
	c := creditRegistry(t)
	for _, test := range []struct {
		output int
		want   int64
	}{{128000, 1320000}, {1, 1000003}, {64000, 1160000}} {
		got, revision, err := c.ReserveClaudeRequest("claude-haiku-5-5", test.output)
		if err != nil || got != test.want || revision == "" {
			t.Fatalf("reserve %d = (%d, %q, %v), want %d", test.output, got, revision, err, test.want)
		}
	}
}

func TestClaudeCreditUsageTierIncludesCacheAndRoundsUp(t *testing.T) {
	c := creditRegistry(t)
	for _, test := range []struct {
		name  string
		usage ClaudeUsage
		want  int64
	}{
		{"boundary", ClaudeUsage{InputTokens: 100000, OutputTokens: 2}, 10001},
		{"above boundary", ClaudeUsage{InputTokens: 100001, OutputTokens: 2}, 50006},
		{"cache drives tier", ClaudeUsage{InputTokens: 1, CacheReadTokens: 100000, OutputTokens: 1}, 5003},
		{"mixed TTL", ClaudeUsage{InputTokens: 1000, CacheReadTokens: 2000, CacheWrite5mTokens: 3000, CacheWrite1hTokens: 4000, OutputTokens: 5}, 1298},
		{"single read token", ClaudeUsage{CacheReadTokens: 1}, 1},
		{"long mixed TTL", ClaudeUsage{InputTokens: 1000, CacheReadTokens: 100000, CacheWrite5mTokens: 3000, CacheWrite1hTokens: 4000, OutputTokens: 5}, 11388},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.ClaudeUsageCost("claude-haiku-5-5", test.usage)
			if err != nil || got != test.want {
				t.Fatalf("cost = %d, %v; want %d", got, err, test.want)
			}
		})
	}
}

func TestClaudeCreditRejectsUnknownAndImpossibleUsage(t *testing.T) {
	c := creditRegistry(t)
	for _, model := range []string{"haiku", "claude-haiku-4-5", "unknown", "claude-haiku-5-5-20261007"} {
		if _, _, err := c.ReserveClaudeRequest(model, 10); err == nil {
			t.Fatalf("accepted non-allowlisted %q", model)
		}
	}
	for _, output := range []int{0, -1, 128001} {
		if _, _, err := c.ReserveClaudeRequest("claude-haiku-5-5", output); err == nil {
			t.Fatalf("accepted output %d", output)
		}
	}
	for _, usage := range []ClaudeUsage{{InputTokens: -1}, {OutputTokens: -1}, {CacheReadTokens: -1}, {CacheWrite5mTokens: -1}, {CacheWrite1hTokens: -1}, {InputTokens: 1000000, CacheReadTokens: 1}, {OutputTokens: 128001}, {InputTokens: math.MaxInt64, CacheWrite1hTokens: math.MaxInt64}} {
		if _, err := c.ClaudeUsageCost("claude-haiku-5-5", usage); err == nil {
			t.Fatalf("accepted impossible usage %+v", usage)
		}
	}
}

func TestClaudeCreditFailsClosedWithoutCompleteVerifiedCard(t *testing.T) {
	for _, mutate := range []func(*ClaudeCreditPricing){
		func(p *ClaudeCreditPricing) { p.BoundsVerified = false },
		func(p *ClaudeCreditPricing) { p.Revision = "" },
		func(p *ClaudeCreditPricing) { p.SourceURL = "" },
		func(p *ClaudeCreditPricing) { p.MaxInputTokens = 0 },
		func(p *ClaudeCreditPricing) { p.LongContext.CacheWrite1h = 0 },
		func(p *ClaudeCreditPricing) { p.ShortContext.Input = -1 },
		func(p *ClaudeCreditPricing) { p.LongContext.Output = math.MaxInt64 },
	} {
		c := creditRegistry(t)
		mutate(c.Models["claude-haiku-5-5"].Pricing.ClaudeCredit)
		if _, _, err := c.ReserveClaudeRequest("claude-haiku-5-5", 10); err == nil {
			t.Fatal("incomplete or unverified card authorized a request")
		}
	}
}

func TestClaudeCreditRevisionChangesWithRates(t *testing.T) {
	c := creditRegistry(t)
	_, before, err := c.ReserveClaudeRequest("claude-haiku-5-5", 10)
	if err != nil {
		t.Fatal(err)
	}
	c.Models["claude-haiku-5-5"].Pricing.ClaudeCredit.LongContext.Output++
	_, after, err := c.ReserveClaudeRequest("claude-haiku-5-5", 10)
	if err != nil || before == after {
		t.Fatalf("pricing revision did not change: %v", err)
	}
	if _, err := c.ClaudeUsageCostAtRevision("claude-haiku-5-5", ClaudeUsage{InputTokens: 1}, before); err == nil {
		t.Fatal("settled at changed rates")
	}
}

func TestClaudeCreditBoundCoversEverySupportedCategory(t *testing.T) {
	c := creditRegistry(t)
	reservation, revision, err := c.ReserveClaudeRequest("claude-haiku-5-5", 128000)
	if err != nil {
		t.Fatal(err)
	}
	// Hidden thinking is billed inside OutputTokens, so the entire 128K cap is
	// reserved even if no visible text is returned. Each input class may occupy
	// the whole window, including the most expensive one-hour cache write.
	for _, usage := range []ClaudeUsage{
		{InputTokens: 1000000, OutputTokens: 128000},
		{CacheReadTokens: 1000000, OutputTokens: 128000},
		{CacheWrite5mTokens: 1000000, OutputTokens: 128000},
		{CacheWrite1hTokens: 1000000, OutputTokens: 128000},
		{InputTokens: 250000, CacheReadTokens: 250000, CacheWrite5mTokens: 250000, CacheWrite1hTokens: 250000, OutputTokens: 128000},
	} {
		cost, err := c.ClaudeUsageCostAtRevision("claude-haiku-5-5", usage, revision)
		if err != nil || cost > reservation {
			t.Fatalf("usage %+v exceeds reservation: %d > %d, %v", usage, cost, reservation, err)
		}
	}
}

func TestClaudeCreditRequiresLoadedRegistryAndCard(t *testing.T) {
	var unloaded *ModelsConfig
	if _, _, err := unloaded.ReserveClaudeRequest("claude-haiku-5-5", 10); err == nil {
		t.Fatal("unloaded registry authorized request")
	}
	c := creditRegistry(t)
	m := c.Models["claude-haiku-5-5"]
	m.Pricing.ClaudeCredit = nil
	c.Models["claude-haiku-5-5"] = m
	if _, _, err := c.ReserveClaudeRequest("claude-haiku-5-5", 10); err == nil {
		t.Fatal("legacy scalar rates authorized request")
	}
}
