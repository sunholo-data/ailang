package observatory

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/modelreg"
)

// M-V1-SIMPLIFY-S3 M2: the observatory prices through modelreg. The alias and
// date-suffix tests that used to live here moved to internal/modelreg/resolve_test.go
// with the behaviour.

// loadShippedRegistry prices against the embedded registry regardless of cwd —
// the property the old cwd-relative loader lacked.
func loadShippedRegistry(t *testing.T) {
	t.Helper()
	ResetPricingConfig()
	if GetPricingConfig() == nil {
		t.Fatal("the embedded model registry must load from any cwd")
	}
}

func TestPriceTokens_UnknownModelIsErrUnpricedNotZero(t *testing.T) {
	loadShippedRegistry(t)
	cost, err := PriceTokens("unknown-nonexistent-model", 1000, 1000, 0, 0)
	if !errors.Is(err, ErrUnpriced) {
		t.Fatalf("err = %v, want ErrUnpriced", err)
	}
	if !errors.Is(err, modelreg.ErrUnknownModel) {
		t.Errorf("ErrUnpriced must wrap modelreg.ErrUnknownModel so callers can branch on either")
	}
	if cost != 0 {
		t.Errorf("cost = %v on an error; want 0 with the error as the signal", cost)
	}
	if !strings.Contains(err.Error(), "unknown-nonexistent-model") {
		t.Errorf("error must name the model; got %v", err)
	}
}

func TestResolveCostFromTokens_DistinguishesUnpricedFromFree(t *testing.T) {
	loadShippedRegistry(t)
	if _, ok := ResolveCostFromTokens("unknown-nonexistent-model", 1000, 1000); ok {
		t.Error("an unknown model must resolve=false, never a fabricated $0")
	}
	if _, ok := ResolveCostFromTokens("", 1000, 1000); ok {
		t.Error("an empty model must resolve=false")
	}
	cost, ok := ResolveCostFromTokens("motoko-local-qwen3-5-35b-a3b-mxfp8", 1000, 1000)
	if !ok || cost != 0 {
		t.Errorf("a free local row must resolve=true at $0; got (%v, %v)", cost, ok)
	}
}

// The float entry points the OTLP receiver calls cannot return an error, so an
// unpriced model must be LOUD on stderr (once per model) rather than a silent 0.
func TestCalculateCostFromTokens_UnpricedIsLoudOncePerModel(t *testing.T) {
	loadShippedRegistry(t)
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stderr
	os.Stderr = w
	CalculateCostFromTokens("unknown-nonexistent-model", 1000, 1000)
	CalculateCostFromTokens("unknown-nonexistent-model", 1000, 1000)
	os.Stderr = old
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "UNPRICED") || !strings.Contains(out, "unknown-nonexistent-model") {
		t.Errorf("expected an UNPRICED line naming the model on stderr; got %q", out)
	}
	if strings.Count(out, "UNPRICED") != 1 {
		t.Errorf("expected exactly one UNPRICED line for two calls on the same model; got %q", out)
	}
	if got := CalculateCostFromTokens("unknown-nonexistent-model", 1000, 1000); got != 0 {
		t.Errorf("unpriced float path must return 0, got %v", got)
	}
}

func TestCalculateCostFromTokens_ZeroTokensNeedNoLookup(t *testing.T) {
	loadShippedRegistry(t)
	if cost := CalculateCostFromTokens("claude-sonnet-4-5", 0, 0); cost != 0 {
		t.Errorf("zero tokens = %f, want 0", cost)
	}
	if cost := CalculateCostFromTokens("", 1000, 1000); cost != 0 {
		t.Errorf("empty model = %f, want 0", cost)
	}
}

func TestCalculateCostFromTokens_ResolvesKeyAndAPIName(t *testing.T) {
	loadShippedRegistry(t)
	want := 0.003 + 0.015 // claude-sonnet-4-5: 1000 in + 1000 out
	for _, name := range []string{"claude-sonnet-4-5", "claude-sonnet-4-5-20250929"} {
		got := CalculateCostFromTokens(name, 1000, 1000)
		if got < want*0.99 || got > want*1.01 {
			t.Errorf("CalculateCostFromTokens(%q) = %f, want ~%f", name, got, want)
		}
	}
}

// Prod OpenRouter spans carry the slug as gen_ai.request.model. The old generic
// path priced these at 0 because it never tried api_name (sampled 2026-09-01..13:
// 72/72 token-bearing LLM Generation spans stored cost_usd=null).
func TestCalculateCostFromTokens_PricesOpenRouterSlugs(t *testing.T) {
	loadShippedRegistry(t)
	if got := CalculateCostFromTokens("z-ai/glm-5.3-flash", 10_000, 1_000); got <= 0 {
		t.Errorf("OpenRouter slug priced at %v; the registry has this api_name", got)
	}
}

// Cache tokens are billed at the model's OWN declared rates. Until 2026-09-15
// this file hardcoded cache reads at 10% and writes at 125% of input for every
// provider — Anthropic's schedule applied to models that publish other rates.
func TestCalculateCostFromTokensWithCache_UsesTheRowsOwnCacheRates(t *testing.T) {
	loadShippedRegistry(t)
	cfg := GetPricingConfig()

	// Anthropic rows declare exactly the multipliers that used to be hardcoded,
	// so this row is unchanged by the migration (within float rounding).
	{
		m := cfg.Models["claude-sonnet-4-5"]
		got := CalculateCostFromTokensWithCache("claude-sonnet-4-5", 1000, 500, 10_000, 2_000)
		old := 1.0*m.Pricing.InputPer1K + 0.5*m.Pricing.OutputPer1K +
			10.0*m.Pricing.InputPer1K*0.1 + 2.0*m.Pricing.InputPer1K*1.25
		if diff := got - old; diff > 1e-12 || diff < -1e-12 {
			t.Errorf("anthropic: new %.10f != old hardcode %.10f", got, old)
		}
	}

	// A non-Anthropic row with a declared cache-read rate must NOT price at the
	// Anthropic multipliers.
	{
		key, m, err := cfg.Resolve("deepseek/deepseek-v4-flash-0731")
		if err != nil {
			t.Fatal(err)
		}
		if m.Pricing.CacheReadPer1K == 0 {
			t.Fatalf("%s: fixture expects a declared cache_read_per_1k", key)
		}
		const read, write = 100_000, 100_000
		got := CalculateCostFromTokensWithCache(key, 0, 0, read, write)
		anthropicStyle := float64(read)/1000*m.Pricing.InputPer1K*0.1 + float64(write)/1000*m.Pricing.InputPer1K*1.25
		writeRate := m.Pricing.CacheWritePer1K
		if writeRate == 0 {
			writeRate = m.Pricing.InputPer1K
		}
		want := float64(read)/1000*m.Pricing.CacheReadPer1K + float64(write)/1000*writeRate
		if diff := got - want; diff > 1e-12 || diff < -1e-12 {
			t.Errorf("%s: cost = %.10f, want %.10f from the row's own rates", key, got, want)
		}
		if diff := got - anthropicStyle; diff < 1e-12 && diff > -1e-12 {
			t.Errorf("%s: cost equals the 10%%/125%% figure %.10f — the hardcode is back", key, anthropicStyle)
		}
	}
}

func TestCalculateCacheSavings_IsInputMinusDeclaredReadRate(t *testing.T) {
	loadShippedRegistry(t)
	cfg := GetPricingConfig()
	m := cfg.Models["claude-sonnet-4-5"]
	got := CalculateCacheSavings("claude-sonnet-4-5", 10_000)
	want := 10.0 * (m.Pricing.InputPer1K - m.Pricing.CacheReadPer1K)
	if diff := got - want; diff > 1e-12 || diff < -1e-12 {
		t.Errorf("savings = %.10f, want %.10f", got, want)
	}
	if CalculateCacheSavings("", 10_000) != 0 || CalculateCacheSavings("unknown-nonexistent-model", 10_000) != 0 {
		t.Error("no model / unknown model must attribute no savings")
	}
	// A row without a declared read rate bills reads at input rate: nothing saved.
	for k, row := range cfg.Models {
		if row.Pricing.InputPer1K > 0 && row.Pricing.CacheReadPer1K == 0 {
			if s := CalculateCacheSavings(k, 10_000); s != 0 {
				t.Errorf("%s: no declared read rate but savings = %v", k, s)
			}
			break
		}
	}
}
