package eval_analysis

import "testing"

// TestRatingsForMode_ByLang constructs trials across two languages where one
// model is strong on AILANG but weak on Python (and vice versa), then asserts
// ratingsForMode returns a per-language "byLang" block with distinct per-language
// model ELOs — i.e. the fits are not blended across languages.
func TestRatingsForMode_ByLang(t *testing.T) {
	pass := func(id, lang, model string, ok bool) *BenchmarkResult {
		return &BenchmarkResult{ID: id, Lang: lang, Model: model, CompileOk: ok, RuntimeOk: ok, StdoutOk: ok}
	}

	var results []*BenchmarkResult
	benches := []string{"b1", "b2", "b3"}
	// modelA: passes everything in AILANG, fails everything in Python.
	// modelB: fails everything in AILANG, passes everything in Python.
	for _, b := range benches {
		results = append(results,
			pass(b, "ailang", "modelA", true),
			pass(b, "ailang", "modelB", false),
			pass(b, "python", "modelA", false),
			pass(b, "python", "modelB", true),
		)
	}

	block := ratingsForMode(results)
	if block == nil {
		t.Fatal("ratingsForMode returned nil")
	}

	// Backward-compat: combined keys still present.
	for _, k := range []string{"models", "benchmarks", "saturation"} {
		if _, ok := block[k]; !ok {
			t.Errorf("combined block missing key %q", k)
		}
	}

	byLangAny, ok := block["byLang"]
	if !ok {
		t.Fatal("block missing byLang sub-map")
	}
	byLang, ok := byLangAny.(map[string]interface{})
	if !ok {
		t.Fatalf("byLang is %T, want map[string]interface{}", byLangAny)
	}

	elo := func(lang, model string) float64 {
		langBlockAny, ok := byLang[lang]
		if !ok {
			t.Fatalf("byLang missing language %q", lang)
		}
		langBlock := langBlockAny.(map[string]interface{})
		for _, m := range langBlock["models"].([]map[string]interface{}) {
			if m["id"].(string) == model {
				return m["elo"].(float64)
			}
		}
		t.Fatalf("model %q not found in byLang[%q]", model, lang)
		return 0
	}

	aAilang := elo("ailang", "modelA")
	aPython := elo("python", "modelA")
	bAilang := elo("ailang", "modelB")
	bPython := elo("python", "modelB")

	// modelA should dominate AILANG; modelB should dominate Python. If the fits
	// were blended, these per-language ELOs would collapse toward each other.
	if aAilang <= aPython {
		t.Errorf("modelA AILANG-ELO (%.1f) should exceed Python-ELO (%.1f)", aAilang, aPython)
	}
	if bPython <= bAilang {
		t.Errorf("modelB Python-ELO (%.1f) should exceed AILANG-ELO (%.1f)", bPython, bAilang)
	}
	// Cross-language: modelA strong on AILANG, modelB strong on Python.
	if aAilang <= bAilang {
		t.Errorf("modelA AILANG-ELO (%.1f) should exceed modelB AILANG-ELO (%.1f)", aAilang, bAilang)
	}
	if bPython <= aPython {
		t.Errorf("modelB Python-ELO (%.1f) should exceed modelA Python-ELO (%.1f)", bPython, aPython)
	}
}

// TestRatingsForMode_Coverage verifies per-model benchmark coverage + maxCoverage
// are surfaced so consumers can gate under-covered models out of the ranking
// (M-EVAL-VALIDITY-DISCIPLINE): an ELO over 1 benchmark must not rank next to one
// over 3.
func TestRatingsForMode_Coverage(t *testing.T) {
	mk := func(id, model string, ok bool) *BenchmarkResult {
		return &BenchmarkResult{ID: id, Lang: "ailang", Model: model, CompileOk: ok, RuntimeOk: ok, StdoutOk: ok}
	}
	var results []*BenchmarkResult
	// full: runs b1,b2,b3 ; sparse: runs only b1
	for _, b := range []string{"b1", "b2", "b3"} {
		results = append(results, mk(b, "full", true))
	}
	results = append(results, mk("b1", "sparse", true))

	block := ratingsForMode(results)
	if got, ok := block["maxCoverage"].(int); !ok || got != 3 {
		t.Fatalf("maxCoverage = %v, want 3", block["maxCoverage"])
	}
	cov := map[string]int{}
	for _, m := range block["models"].([]map[string]interface{}) {
		cov[m["id"].(string)] = m["benchmarks"].(int)
	}
	if cov["full"] != 3 {
		t.Errorf("full coverage = %d, want 3", cov["full"])
	}
	if cov["sparse"] != 1 {
		t.Errorf("sparse coverage = %d, want 1", cov["sparse"])
	}
}
