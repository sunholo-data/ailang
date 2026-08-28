package main

import (
	"encoding/json"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sunholo-data/ailang/internal/eval_harness"
)

// baseManifestParams is a representative frozen-cohort configuration. Tests
// mutate a copy so each case states exactly the one thing it varies.
func baseManifestParams() cohortManifestParams {
	return cohortManifestParams{
		baselineID:    "v1.0",
		modelSuiteTok: "agent_suite",
		models:        []string{"claude-haiku-4-5", "claude-sonnet-4-6"},
		benchmarks:    []string{"contract_leap_year", "contract_bst_validate"},
		languages:     []string{"ailang"},
		conditions:    []string{""},
		evalMode:      "agent",
		seed:          42,
		promptVersion: "",
		trials:        1,
		verify:        true,
		verifyTimeout: 5 * time.Second,
		chainID:       "chain-abc",
		startedAt:     time.Date(2026, 7, 27, 10, 0, 0, 0, time.UTC),
	}
}

func TestBuildCohortManifest_RecordsEveryPlannedField(t *testing.T) {
	m := buildCohortManifest(baseManifestParams())

	if m.BaselineID != "v1.0" {
		t.Errorf("baseline_id = %q, want v1.0", m.BaselineID)
	}
	// The manifest's prefix MUST be the exact string the reader queries with.
	if m.SourceRefPrefix != cohortSourceRefPrefix("v1.0") {
		t.Errorf("source_ref_prefix = %q, want %q", m.SourceRefPrefix, cohortSourceRefPrefix("v1.0"))
	}
	if m.EvalMode != "agent" {
		t.Errorf("eval_mode = %q, want agent", m.EvalMode)
	}
	if len(m.Languages) != 1 || m.Languages[0] != "ailang" {
		t.Errorf("languages = %v", m.Languages)
	}
	if m.ModelSuite != "agent_suite" {
		t.Errorf("model_suite = %q, want agent_suite", m.ModelSuite)
	}
	if len(m.Models) != 2 {
		t.Errorf("models = %v, want 2 entries", m.Models)
	}
	if len(m.Benchmarks) != 2 {
		t.Errorf("benchmarks = %v, want 2 entries", m.Benchmarks)
	}
	if m.Seed != 42 {
		t.Errorf("seed = %d, want 42", m.Seed)
	}
	if m.Trials != 1 {
		t.Errorf("trials = %d, want 1", m.Trials)
	}
	if !m.Verify {
		t.Error("verify = false, want true")
	}
	if m.VerifyTimeout != "5s" {
		t.Errorf("verify_timeout = %q, want 5s", m.VerifyTimeout)
	}
	if m.ChainID != "chain-abc" {
		t.Errorf("chain_id = %q, want chain-abc", m.ChainID)
	}
	if m.AILANGVersion == "" {
		t.Error("ailang_version is empty — release evidence must be self-identifying")
	}
	if m.FrozenAt.IsZero() {
		t.Error("frozen_at is zero")
	}
	if m.RunWindow.StartedAt.IsZero() {
		t.Error("run_window.started_at is zero")
	}
	if m.RunWindow.CompletedAt != nil {
		t.Error("run_window.completed_at must be nil until finalize")
	}
	if m.CohortHash == "" {
		t.Error("cohort_hash is empty")
	}
	if len(m.Executors) != len(m.Models) {
		t.Errorf("executors has %d entries, models has %d — every model needs an audit row",
			len(m.Executors), len(m.Models))
	}
}

// TestBuildCohortManifest_ModelSuiteTokenVsExplicitList: --models agent_suite
// records the suite token; an explicit comma list records "" with the same
// resolved members. Either way `models[]` is the RESOLVED list.
func TestBuildCohortManifest_ModelSuiteTokenVsExplicitList(t *testing.T) {
	suite := buildCohortManifest(baseManifestParams())

	p := baseManifestParams()
	p.modelSuiteTok = "" // explicit comma list -> no suite name
	explicit := buildCohortManifest(p)

	if suite.ModelSuite != "agent_suite" {
		t.Errorf("suite run model_suite = %q, want agent_suite", suite.ModelSuite)
	}
	if explicit.ModelSuite != "" {
		t.Errorf("explicit-list run model_suite = %q, want empty", explicit.ModelSuite)
	}
	if len(suite.Models) != len(explicit.Models) {
		t.Fatalf("resolved members differ: %v vs %v", suite.Models, explicit.Models)
	}
	// Same resolved cohort => same hash, regardless of HOW it was spelled.
	if suite.CohortHash != explicit.CohortHash {
		t.Errorf("cohort_hash must depend on the RESOLVED cohort, not its spelling: %q vs %q",
			suite.CohortHash, explicit.CohortHash)
	}
}

// TestBuildCohortManifest_ModelsAreDataDrivenFromModelsYML is the
// re-freezability guarantee (Mark's 2026-07-27 ratification: "assume current
// cohort but this may have light changes depending on release date"). NOTHING in
// Go names a model: the manifest's members come from expandModelSuite ->
// GetAgentSuite() reading benchmarks/models.yml. This test READS models.yml
// rather than pinning a literal name list, so a suite edit does NOT fail the
// test but DOES change the manifest (and its hash) visibly.
func TestBuildCohortManifest_ModelsAreDataDrivenFromModelsYML(t *testing.T) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		t.Skipf("models.yml unavailable: %v", err)
	}
	cfg := modelreg.GlobalModelsConfig
	if cfg == nil {
		t.Skip("models.yml not loaded")
	}
	want := cfg.GetAgentSuite()
	if len(want) == 0 {
		t.Skip("agent_suite is empty in models.yml")
	}

	resolved := expandModelSuite("agent_suite", cfg)
	if len(resolved) != len(want) {
		t.Fatalf("expandModelSuite(agent_suite) = %v, models.yml GetAgentSuite() = %v", resolved, want)
	}
	for i := range want {
		if resolved[i] != want[i] {
			t.Fatalf("expandModelSuite(agent_suite)[%d] = %q, models.yml has %q", i, resolved[i], want[i])
		}
	}

	p := baseManifestParams()
	p.models = resolved
	m := buildCohortManifest(p)
	if len(m.Models) != len(want) {
		t.Errorf("manifest models = %v, want the %d models.yml agent_suite members", m.Models, len(want))
	}
	// Every model gets an executor audit row, even if resolution fails.
	if len(m.Executors) != len(want) {
		t.Errorf("executors has %d rows for %d models", len(m.Executors), len(want))
	}
	for _, e := range m.Executors {
		if e.Model == "" {
			t.Error("executor row with empty model")
		}
		if e.Executor == "" && e.Error == "" {
			t.Errorf("model %q has neither an executor nor a reason — silent gap in the provenance audit hook", e.Model)
		}
	}
}

// TestCohortHash_OrderIndependentForModelsAndBenchmarks — the hash identifies a
// SET of models/benchmarks, so listing them in a different order is the same
// cohort.
func TestCohortHash_OrderIndependentForModelsAndBenchmarks(t *testing.T) {
	a := baseManifestParams()
	b := baseManifestParams()
	b.models = []string{"claude-sonnet-4-6", "claude-haiku-4-5"}
	b.benchmarks = []string{"contract_bst_validate", "contract_leap_year"}

	if got, want := buildCohortManifest(b).CohortHash, buildCohortManifest(a).CohortHash; got != want {
		t.Errorf("cohort_hash changed under reordering: %q vs %q", got, want)
	}
}

// TestCohortHash_StableAcrossBuilds — two independent builds of identical inputs
// agree, so a reviewer can recompute the published hash.
func TestCohortHash_StableAcrossBuilds(t *testing.T) {
	first := buildCohortManifest(baseManifestParams()).CohortHash
	for i := 0; i < 20; i++ {
		if got := buildCohortManifest(baseManifestParams()).CohortHash; got != first {
			t.Fatalf("cohort_hash unstable on build %d: %q vs %q", i, got, first)
		}
	}
}

// TestCohortHash_ChangesWhenCohortMembershipChanges — a models.yml edit must be
// VISIBLE in the artifact, never silent.
func TestCohortHash_ChangesWhenCohortMembershipChanges(t *testing.T) {
	base := buildCohortManifest(baseManifestParams()).CohortHash

	mutations := []struct {
		name  string
		apply func(*cohortManifestParams)
	}{
		{"model added", func(p *cohortManifestParams) { p.models = append(p.models, "gpt5-2-codex") }},
		{"model removed", func(p *cohortManifestParams) { p.models = p.models[:1] }},
		{"model renamed", func(p *cohortManifestParams) { p.models = []string{"claude-haiku-4-5", "claude-opus-5"} }},
		{"benchmark added", func(p *cohortManifestParams) { p.benchmarks = append(p.benchmarks, "prompt_injection") }},
		{"benchmark removed", func(p *cohortManifestParams) { p.benchmarks = p.benchmarks[:1] }},
		{"language added", func(p *cohortManifestParams) { p.languages = []string{"ailang", "python"} }},
		{"eval mode", func(p *cohortManifestParams) { p.evalMode = "standard" }},
		{"seed", func(p *cohortManifestParams) { p.seed = 43 }},
		{"prompt version", func(p *cohortManifestParams) { p.promptVersion = "v0.3.22" }},
		{"conditions", func(p *cohortManifestParams) { p.conditions = []string{"full"} }},
		{"trials", func(p *cohortManifestParams) { p.trials = 3 }},
	}
	for _, mu := range mutations {
		t.Run(mu.name, func(t *testing.T) {
			p := baseManifestParams()
			mu.apply(&p)
			if got := buildCohortManifest(p).CohortHash; got == base {
				t.Errorf("cohort_hash did NOT change after %s — cohort drift would be silent", mu.name)
			}
		})
	}
}

// TestCohortHash_IgnoresRunIdentity — the hash identifies the COHORT, not the
// RUN, so a re-freeze of the same cohort is recognisable as the same cohort.
func TestCohortHash_IgnoresRunIdentity(t *testing.T) {
	base := buildCohortManifest(baseManifestParams()).CohortHash

	invariants := []struct {
		name  string
		apply func(*cohortManifestParams)
	}{
		{"frozen_at / started_at", func(p *cohortManifestParams) {
			p.startedAt = time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
		}},
		{"chain_id", func(p *cohortManifestParams) { p.chainID = "chain-totally-different" }},
		{"baseline_id", func(p *cohortManifestParams) { p.baselineID = "v1.0-rc2" }},
		{"model_suite spelling", func(p *cohortManifestParams) { p.modelSuiteTok = "" }},
	}
	for _, inv := range invariants {
		t.Run(inv.name, func(t *testing.T) {
			p := baseManifestParams()
			inv.apply(&p)
			if got := buildCohortManifest(p).CohortHash; got != base {
				t.Errorf("cohort_hash changed with %s — it must identify the cohort, not the run", inv.name)
			}
		})
	}
}

// TestWriteCohortManifest_RoundTrips — the artifact on disk must parse back into
// the same values a reviewer would recompute.
func TestWriteCohortManifest_RoundTrips(t *testing.T) {
	dir := t.TempDir()
	m := buildCohortManifest(baseManifestParams())

	path, err := writeCohortManifest(dir, m)
	if err != nil {
		t.Fatalf("writeCohortManifest: %v", err)
	}
	if filepath.Base(path) != cohortManifestFilename {
		t.Errorf("wrote %q, want basename %q", path, cohortManifestFilename)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("manifest path %q is not absolute — the run's stdout must be copy-pasteable", path)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var got CohortManifest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("manifest is not valid JSON: %v\n%s", err, raw)
	}
	if got.CohortHash != m.CohortHash {
		t.Errorf("round-trip cohort_hash = %q, want %q", got.CohortHash, m.CohortHash)
	}
	if got.BaselineID != m.BaselineID || got.SourceRefPrefix != m.SourceRefPrefix {
		t.Errorf("round-trip identity fields differ: %+v", got)
	}
	if len(got.Models) != len(m.Models) || len(got.Benchmarks) != len(m.Benchmarks) {
		t.Errorf("round-trip lists differ: %v / %v", got.Models, got.Benchmarks)
	}
}

// TestWriteCohortManifest_CreatesMissingOutputDir — the manifest is written
// before the run loop, which is also before the output dir necessarily exists.
func TestWriteCohortManifest_CreatesMissingOutputDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "v1.0.0")
	if _, err := writeCohortManifest(dir, buildCohortManifest(baseManifestParams())); err != nil {
		t.Fatalf("writeCohortManifest into a missing dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, cohortManifestFilename)); err != nil {
		t.Errorf("manifest not present: %v", err)
	}
}

// TestFinalizeCohortManifest_PopulatesCompletedAt — the run window is only
// closed out at finalize; the hash must NOT move (it excludes timestamps).
func TestFinalizeCohortManifest_PopulatesCompletedAt(t *testing.T) {
	dir := t.TempDir()
	m := buildCohortManifest(baseManifestParams())
	if _, err := writeCohortManifest(dir, m); err != nil {
		t.Fatalf("initial write: %v", err)
	}
	hashBefore := m.CohortHash

	completed := time.Date(2026, 7, 27, 12, 34, 56, 0, time.UTC)
	path, err := finalizeCohortManifest(dir, m, completed)
	if err != nil {
		t.Fatalf("finalizeCohortManifest: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var got CohortManifest
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("bad JSON: %v", err)
	}
	if got.RunWindow.CompletedAt == nil {
		t.Fatal("run_window.completed_at is still nil after finalize")
	}
	if !got.RunWindow.CompletedAt.Equal(completed) {
		t.Errorf("completed_at = %v, want %v", got.RunWindow.CompletedAt, completed)
	}
	if got.RunWindow.StartedAt.IsZero() {
		t.Error("finalize clobbered started_at")
	}
	if got.CohortHash != hashBefore {
		t.Errorf("finalize changed cohort_hash (%q -> %q); the hash must exclude timestamps",
			hashBefore, got.CohortHash)
	}
}

// TestFinalizeCohortManifest_NilManifestIsANoOp — the default (no --baseline)
// run has no manifest, and finalize must not invent one.
func TestFinalizeCohortManifest_NilManifestIsANoOp(t *testing.T) {
	dir := t.TempDir()
	path, err := finalizeCohortManifest(dir, nil, time.Now())
	if err != nil {
		t.Errorf("nil manifest should be a silent no-op, got %v", err)
	}
	if path != "" {
		t.Errorf("nil manifest returned path %q", path)
	}
	if _, err := os.Stat(filepath.Join(dir, cohortManifestFilename)); !os.IsNotExist(err) {
		t.Error("a manifest was written for a nil manifest")
	}
}

// TestWriteCohortManifest_NilWritesNothing is the "no --baseline -> no manifest"
// contract at the function boundary: the default run must leave the output dir
// exactly as it is today.
func TestWriteCohortManifest_NilWritesNothing(t *testing.T) {
	dir := t.TempDir()
	path, err := writeCohortManifest(dir, nil)
	if err != nil {
		t.Errorf("writeCohortManifest(dir, nil) = %v, want nil", err)
	}
	if path != "" {
		t.Errorf("path = %q, want empty", path)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("default path polluted the output dir: %v", entries)
	}
}

// TestCleanResults_PreservesCohortManifest is a REGRESSION test.
//
// cleanResults globs "<outputDir>/*.json" and runs a few lines AFTER the M4a
// freeze-time manifest write, so it used to delete cohort_manifest.json on every
// run that was not --skip-existing — leaving the freeze command printing a path
// to a file that no longer existed. Per-benchmark results must still be cleaned.
func TestCleanResults_PreservesCohortManifest(t *testing.T) {
	dir := t.TempDir()

	if _, err := writeCohortManifest(dir, buildCohortManifest(baseManifestParams())); err != nil {
		t.Fatalf("writeCohortManifest: %v", err)
	}
	results := []string{
		"fizzbuzz_ailang_claude-haiku-4-5_20260727_120000.json",
		"contract_leap_year_ailang_gpt5-mini_20260727_120001.json",
	}
	for _, name := range results {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "summary.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("seed summary.json: %v", err)
	}

	cleanResults(dir)

	if _, err := os.Stat(filepath.Join(dir, cohortManifestFilename)); err != nil {
		t.Errorf("cleanResults deleted %s: %v", cohortManifestFilename, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "summary.json")); err != nil {
		t.Errorf("cleanResults deleted summary.json: %v", err)
	}
	for _, name := range results {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("cleanResults did NOT delete the stale result %s (err=%v)", name, err)
		}
	}
}

// TestCohortModelSuiteToken records HOW the cohort was selected. The suite-name
// set is read from modelSuiteResolvers, so this test cannot drift from
// expandModelSuite.
func TestCohortModelSuiteToken(t *testing.T) {
	tests := []struct {
		name       string
		modelsFlag string
		fullSuite  bool
		want       string
	}{
		{"suite token", "agent_suite", false, "agent_suite"},
		{"suite token with whitespace", "  agent_suite  ", false, "agent_suite"},
		{"other suite token", "ollama_suite", false, "ollama_suite"},
		{"explicit comma list", "claude-haiku-4-5,gpt5-2-codex", false, ""},
		{"single explicit model", "claude-haiku-4-5", false, ""},
		{"no --models defaults to dev_models", "", false, "dev_models"},
		{"no --models with --full", "", true, "extended_suite"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cohortModelSuiteToken(tt.modelsFlag, tt.fullSuite); got != tt.want {
				t.Errorf("cohortModelSuiteToken(%q, %v) = %q, want %q",
					tt.modelsFlag, tt.fullSuite, got, tt.want)
			}
		})
	}
}

// TestCohortModelSuiteToken_CoversEveryResolvableSuite: a suite that
// expandModelSuite can resolve but the manifest cannot NAME would leave a frozen
// cohort's provenance silently incomplete. Both read modelSuiteResolvers, so this
// test passes automatically for any suite added to that one table — and fails if
// someone reintroduces a second hardcoded list.
func TestCohortModelSuiteToken_CoversEveryResolvableSuite(t *testing.T) {
	if len(modelSuiteResolvers) == 0 {
		t.Fatal("modelSuiteResolvers is empty")
	}
	for token := range modelSuiteResolvers {
		if !isModelSuiteToken(token) {
			t.Errorf("isModelSuiteToken(%q) = false for a resolvable suite", token)
		}
		if got := cohortModelSuiteToken(token, false); got != token {
			t.Errorf("cohortModelSuiteToken(%q) = %q — resolvable suite not recorded in the manifest", token, got)
		}
	}
	if isModelSuiteToken("definitely_not_a_suite") {
		t.Error("isModelSuiteToken accepted an unknown token")
	}
}

// TestCohortManifest_ExecutorsAreTheProvenanceAuditHook.
//
// This field is load-bearing for the out_of_scope_provenance limitation:
// ClassifyStageCost treats ANY stage.Cost > 0 as authoritative
// CostStatusReported, but the SUBSCRIPTION claude CLI reports a non-zero
// total_cost_usd while nothing is billed. M4a does not fix that (open product
// decision) — its obligation is not to HIDE it. Recording the resolved executor
// per model is what lets a reviewer see which rows are subscription lanes.
func TestCohortManifest_ExecutorsAreTheProvenanceAuditHook(t *testing.T) {
	if err := eval_harness.InitModelsConfig(); err != nil {
		t.Skipf("models.yml unavailable: %v", err)
	}
	cfg := modelreg.GlobalModelsConfig
	if cfg == nil {
		t.Skip("models.yml not loaded")
	}

	p := baseManifestParams()
	p.models = expandModelSuite("agent_suite", cfg)
	if len(p.models) == 0 {
		t.Skip("agent_suite empty")
	}
	m := buildCohortManifest(p)

	// Executor rows must be sorted by model so the artifact is byte-deterministic.
	for i := 1; i < len(m.Executors); i++ {
		if m.Executors[i-1].Model > m.Executors[i].Model {
			t.Errorf("executors not sorted by model: %q before %q",
				m.Executors[i-1].Model, m.Executors[i].Model)
		}
	}

	// At least one distinct executor name must be resolved, otherwise the hook
	// carries no information at all.
	distinct := map[string]bool{}
	for _, e := range m.Executors {
		if e.Executor != "" {
			distinct[e.Executor] = true
		}
	}
	if len(distinct) == 0 {
		t.Errorf("no executor resolved for any agent_suite model — the provenance audit hook is empty: %+v", m.Executors)
	}
}
