package main

// cohort_manifest.json — the reproducibility artifact for a frozen baseline
// cohort (M-COST-PER-SUCCESS-KPI M4a-2).
//
// The design doc's final acceptance criterion requires "the exact frozen cohort
// manifest and command output; a reviewer can independently recompute it without
// dashboard JavaScript". This file is that artifact.
//
// DESIGN RULE — nothing in this file may NAME a model or a benchmark. Every
// identity field is DERIVED from the resolved run configuration (`--models
// agent_suite` -> expandModelSuite -> ModelsConfig.GetAgentSuite() reading
// benchmarks/models.yml). That is what makes the cohort re-freezable: if
// `agent_suite` changes before the release, re-running the freeze command
// produces a manifest with the new members AND a different cohort_hash, so the
// drift is VISIBLE in the artifact instead of silent. A re-freeze publishes as a
// NEW baseline id (v1.0-rc2, …) — never a rewrite of a published one.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/sunholo-data/ailang/internal/modelreg"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/version"
)

// cohortManifestFilename is the fixed basename inside the run's output dir
// (which already respects --bank-by-version).
const cohortManifestFilename = "cohort_manifest.json"

// CohortRunWindow bounds the wall-clock window the cohort was measured in.
// CompletedAt is nil until finalizeSuiteRun closes the run out.
type CohortRunWindow struct {
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
}

// CohortModelExecutor is one row of the per-model executor audit.
//
// This is the out_of_scope_provenance AUDIT HOOK. internal/observatory/
// cost_classify.go rule 1 treats ANY stage.Cost > 0 as authoritative
// CostStatusReported — but the SUBSCRIPTION claude CLI reports a non-zero
// total_cost_usd even when nothing is billed (live-probed 2026-07-27 with both
// Anthropic API keys stripped: 10 in / 46 out returned total_cost_usd
// 0.0108355). So on a key-less rig, `claude`-executor rows are LIST-PRICE
// EQUIVALENTS, not metered dollars, while the OpenRouter lanes are real spend —
// and the rollup blends both under one "reported" label.
//
// M4a designs NO fix for that (it is an open product decision). Its only
// obligation is not to HIDE it: recording the resolved executor per model is
// what lets a reviewer see which rows are subscription lanes.
type CohortModelExecutor struct {
	Model    string `json:"model"`
	Executor string `json:"executor"`        // resolved agent_cli ("claude", "codex", "opencode", "managed_agents"); "" if unresolved
	Error    string `json:"error,omitempty"` // why resolution failed (no agent_cli, retired CLI, unknown model). Never silently omitted.
}

// CohortManifest is the recorded, versioned cohort identity + run provenance.
type CohortManifest struct {
	// Cohort identity (what was measured).
	BaselineID      string   `json:"baseline_id"`
	SourceRefPrefix string   `json:"source_ref_prefix"`
	EvalMode        string   `json:"eval_mode"`
	Languages       []string `json:"languages"`
	Conditions      []string `json:"conditions"`
	ModelSuite      string   `json:"model_suite"` // the literal --models suite token, "" for an explicit comma list
	Models          []string `json:"models"`      // RESOLVED members (from models.yml)
	Benchmarks      []string `json:"benchmarks"`  // RESOLVED ids (post-discovery, post---tier)
	Seed            int64    `json:"seed"`
	PromptVersion   string   `json:"prompt_version"`
	Trials          int      `json:"trials"`

	// Verification configuration (a frozen cohort without it cannot yield the KPI).
	Verify        bool   `json:"verify"`
	VerifyTimeout string `json:"verify_timeout"`

	// Provenance audit.
	Executors []CohortModelExecutor `json:"executors"`

	// Run provenance (deliberately EXCLUDED from cohort_hash).
	FrozenAt      time.Time       `json:"frozen_at"`
	RunWindow     CohortRunWindow `json:"run_window"`
	AILANGVersion string          `json:"ailang_version"`
	GitCommit     string          `json:"git_commit"`
	ChainID       string          `json:"chain_id"`

	// CohortHash identifies the COHORT, not the RUN.
	CohortHash string `json:"cohort_hash"`
}

// cohortManifestParams is the resolved run configuration the manifest is derived
// from. It is populated at freeze time — AFTER model/benchmark resolution and
// the agent-mode model filter, so no field can record an unresolved list.
type cohortManifestParams struct {
	baselineID    string
	modelSuiteTok string // the literal --models token when it named a suite, else ""
	models        []string
	benchmarks    []string
	languages     []string
	conditions    []string
	evalMode      string
	seed          int64
	promptVersion string
	trials        int
	verify        bool
	verifyTimeout time.Duration
	chainID       string
	startedAt     time.Time
}

// cohortIdentity is the canonical, sorted pre-image of cohort_hash.
//
// Field set is deliberate: it covers everything that changes WHAT was measured
// and excludes everything that only changes WHICH RUN measured it. So
// frozen_at / git_commit / chain_id / ailang_version / baseline_id are all
// absent — a re-freeze of an unchanged cohort under a new baseline id is
// recognisably the SAME cohort, while a models.yml edit is recognisably a
// DIFFERENT one.
type cohortIdentity struct {
	EvalMode      string   `json:"eval_mode"`
	Languages     []string `json:"languages"`
	Conditions    []string `json:"conditions"`
	Models        []string `json:"models"`
	Benchmarks    []string `json:"benchmarks"`
	Seed          int64    `json:"seed"`
	PromptVersion string   `json:"prompt_version"`
	Trials        int      `json:"trials"`
}

// sortedCopy returns a sorted copy, never mutating the caller's slice (the same
// slices back the live run's job loop).
func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// cohortHash is SHA-256 over the canonical JSON of cohortIdentity. All list
// fields are sorted, so the hash is order-independent: a cohort is a SET of
// models × benchmarks × languages × conditions, not an ordering of them.
func cohortHash(id cohortIdentity) string {
	id.Languages = sortedCopy(id.Languages)
	id.Conditions = sortedCopy(id.Conditions)
	id.Models = sortedCopy(id.Models)
	id.Benchmarks = sortedCopy(id.Benchmarks)

	// Struct field order is fixed by the type, so encoding/json is deterministic.
	canonical, err := json.Marshal(id)
	if err != nil {
		// Unreachable: cohortIdentity is plain strings/ints. Panic rather than
		// return a bogus hash — a wrong cohort_hash on release evidence is worse
		// than a crash (CLAUDE.md §2: no silent fallbacks on published data).
		panic(fmt.Sprintf("cohort identity is not marshalable: %v", err))
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:])
}

// resolveCohortExecutors builds the per-model executor audit rows, sorted by
// model so the artifact is byte-deterministic.
//
// A model whose executor cannot be resolved records the REASON rather than being
// dropped: a missing row would make the cohort look smaller than it was.
func resolveCohortExecutors(models []string) []CohortModelExecutor {
	out := make([]CohortModelExecutor, 0, len(models))
	for _, m := range sortedCopy(models) {
		row := CohortModelExecutor{Model: m}
		if modelreg.GlobalModelsConfig == nil {
			row.Error = "models.yml not loaded"
		} else if execName, _, err := modelreg.GlobalModelsConfig.GetExecutorForModel(m); err != nil {
			row.Error = err.Error()
		} else {
			row.Executor = execName
		}
		out = append(out, row)
	}
	return out
}

// buildCohortManifest derives the manifest from the resolved run configuration.
func buildCohortManifest(p cohortManifestParams) *CohortManifest {
	now := time.Now().UTC()
	started := p.startedAt
	if started.IsZero() {
		started = now
	}

	return &CohortManifest{
		BaselineID:      p.baselineID,
		SourceRefPrefix: cohortSourceRefPrefix(p.baselineID),
		EvalMode:        p.evalMode,
		Languages:       append([]string(nil), p.languages...),
		Conditions:      append([]string(nil), p.conditions...),
		ModelSuite:      p.modelSuiteTok,
		Models:          append([]string(nil), p.models...),
		Benchmarks:      append([]string(nil), p.benchmarks...),
		Seed:            p.seed,
		PromptVersion:   p.promptVersion,
		Trials:          p.trials,
		Verify:          p.verify,
		VerifyTimeout:   p.verifyTimeout.String(),
		Executors:       resolveCohortExecutors(p.models),
		FrozenAt:        now,
		RunWindow:       CohortRunWindow{StartedAt: started.UTC()},
		AILANGVersion:   version.Version,
		GitCommit:       version.Commit,
		ChainID:         p.chainID,
		CohortHash: cohortHash(cohortIdentity{
			EvalMode:      p.evalMode,
			Languages:     p.languages,
			Conditions:    p.conditions,
			Models:        p.models,
			Benchmarks:    p.benchmarks,
			Seed:          p.seed,
			PromptVersion: p.promptVersion,
			Trials:        p.trials,
		}),
	}
}

// writeCohortManifest writes the manifest into dir and returns its ABSOLUTE
// path, so the run's own stdout is copy-pasteable release evidence. The dir is
// created if absent (the manifest is written before the run loop, hence possibly
// before the output dir exists).
func writeCohortManifest(dir string, m *CohortManifest) (string, error) {
	if m == nil {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("could not create output dir %s: %w", dir, err)
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return "", fmt.Errorf("could not encode cohort manifest: %w", err)
	}
	path := filepath.Join(dir, cohortManifestFilename)
	if err := os.WriteFile(path, append(raw, '\n'), 0644); err != nil {
		return "", fmt.Errorf("could not write %s: %w", path, err)
	}
	if abs, absErr := filepath.Abs(path); absErr == nil {
		path = abs
	}
	return path, nil
}

// cohortModelSuiteToken records HOW the model list was selected, so the manifest
// says "agent_suite" (a models.yml-tracked, re-freezable set) rather than only
// listing today's members. An explicit comma list records "" — the resolved
// members are identical, but there is no suite to re-resolve.
//
// The suite-name set comes from modelSuiteResolvers (eval_suite_types.go), the
// same table expandModelSuite resolves with.
func cohortModelSuiteToken(modelsFlag string, fullSuite bool) string {
	trimmed := strings.TrimSpace(modelsFlag)
	if trimmed == "" {
		// No --models: resolveEvalModelList falls back to a named suite.
		if fullSuite {
			return "extended_suite"
		}
		return "dev_models"
	}
	if isModelSuiteToken(trimmed) {
		return trimmed
	}
	return ""
}

// freezeCohortManifest builds the manifest and writes it BEFORE the run loop.
//
// A write failure here is FATAL (exit 1), unlike the finalize rewrite. At freeze
// time nothing has been spent yet, and the operator explicitly asked for
// reproducible release evidence — producing the cohort without the artifact that
// documents it would be exactly the silent-fallback CLAUDE.md §2 forbids. After
// the run, the trade-off inverts: a completed metered run must never be
// discarded over a manifest rewrite (see finalizeSuiteRun).
func freezeCohortManifest(dir string, p cohortManifestParams) *CohortManifest {
	m := buildCohortManifest(p)
	path, err := writeCohortManifest(dir, m)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: --baseline %s: could not write the cohort manifest: %v\n", p.baselineID, err)
		fmt.Fprintf(os.Stderr, "       Refusing to run a frozen cohort without its reproducibility artifact.\n")
		os.Exit(1)
	}
	fmt.Printf("  Cohort:     %s (hash %s)\n", p.baselineID, m.CohortHash)
	fmt.Printf("  Manifest:   %s\n", path)
	return m
}

// finalizeCohortManifest closes the run window out and rewrites the artifact.
// cohort_hash is unaffected (it excludes timestamps by construction).
//
// A nil manifest (the default, no --baseline) is a no-op: finalize must never
// invent a cohort that was not frozen.
func finalizeCohortManifest(dir string, m *CohortManifest, completedAt time.Time) (string, error) {
	if m == nil {
		return "", nil
	}
	t := completedAt.UTC()
	m.RunWindow.CompletedAt = &t
	return writeCohortManifest(dir, m)
}
