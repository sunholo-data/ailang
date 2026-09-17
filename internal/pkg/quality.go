package pkg

import (
	"fmt"
	"sort"
	"strings"
)

// M-PKG-QUALITY-LADDER M3 — the ONE quality report behind `ailang pkg quality`,
// `ailang publish --dry-run` and the registry validator.
//
// Two provenances, one report. Fields marked "server" are static and bounded
// (compile, Z3, interface identity, manifest checks) and are recomputed by the
// validator; only they can ever be registry gates. Fields marked "attested"
// (tests, smoke) EXECUTE package code and therefore run only on the
// publisher's machine — the validator banks what the upload claimed, stamped
// with who claimed it, and never gates on it. This keeps the validator free of
// arbitrary-code execution (round-1 quorum objection, design §Architecture).
//
// The report is assembled from QualityInputs rather than measured here so the
// package stays import-cycle-free (internal/pipeline imports internal/pkg) and
// so publisher and validator provably assemble identical `server` sections
// from identical measurements (seam test in quality_test.go).

// QualitySchema tags the report JSON.
const QualitySchema = "ailang.package-quality/v1"

// AttestedFormField is the multipart field the publisher uses to send its
// attested block alongside the tarball.
const AttestedFormField = "attested"

// Provenance labels a report section.
const (
	SourceServer   = "server"
	SourceAttested = "attested"
)

// QualityMode says which side is assembling the report.
type QualityMode int

const (
	// ModePublisher: `ailang pkg quality` / `publish` on the author's machine.
	ModePublisher QualityMode = iota
	// ModeServer: the registry validator; attested sections come from the upload.
	ModeServer
)

// QualityInputs are the measurements the caller made before assembling.
type QualityInputs struct {
	CompileOK    bool
	CompileFiles int
	CompileError string

	Verify    *PackageVerifyReport // nil when the run itself failed
	VerifyErr string

	InterfaceHashV1 string
	InterfaceHashV2 string
	Signatures      []string
	InterfaceV2Err  string

	HasAgentDoc bool

	// ChangelogNotes / HasChangelogSection come from ChangelogSection(dir, version).
	ChangelogNotes      string
	HasChangelogSection bool
	// ReleaseGatesHard: PUB001/PUB002 block (true from ReleaseGateHardFrom,
	// see ReleaseGatesHard); false = grace window, badges only.
	ReleaseGatesHard bool

	// Attested is nil when the publisher did not run tests/smoke (or the
	// upload carried none).
	Attested *AttestedBlock
}

// AttestedBlock is what the publisher executed and vouches for.
type AttestedBlock struct {
	AttestedBy string        `json:"attested_by"`
	Tests      *TestsSection `json:"tests,omitempty"`
	Smoke      *SmokeSection `json:"smoke,omitempty"`
}

// QualityReport is the assembled document.
type QualityReport struct {
	Schema    string           `json:"schema"`
	Package   string           `json:"package"`
	Version   string           `json:"version"`
	Stability string           `json:"stability"`
	Mode      string           `json:"mode"`
	Compile   CompileSection   `json:"compile"`
	Contracts ContractsSection `json:"contracts"`
	Tests     *TestsSection    `json:"tests,omitempty"`
	Smoke     *SmokeSection    `json:"smoke,omitempty"`
	Interface InterfaceSection `json:"interface"`
	Effects   EffectsSection   `json:"effects"`
	Release   ReleaseSection   `json:"release"`
	Docs      DocsSection      `json:"docs"`
	Style     StyleSection     `json:"style"`
	Gates     []Finding        `json:"gates"`
	Badges    []Finding        `json:"badges"`
}

// Finding is one PUBnnn observation.
type Finding struct {
	Code  string `json:"code"`
	Level string `json:"level"` // "gate" | "warn" | "info"
	Msg   string `json:"msg"`
}

type CompileSection struct {
	Source string `json:"source"`
	OK     bool   `json:"ok"`
	Files  int    `json:"files,omitempty"`
	Error  string `json:"error,omitempty"`
}

type ContractsSection struct {
	Source              string  `json:"source"`
	Verified            int     `json:"verified"`
	Total               int     `json:"total"`
	Counterexample      int     `json:"counterexample"`
	Skipped             int     `json:"skipped"`
	Errors              int     `json:"errors"`
	UncontractedExports int     `json:"uncontracted_exports"`
	WallSeconds         float64 `json:"wall_seconds,omitempty"`
	Error               string  `json:"error,omitempty"`
}

type TestsSection struct {
	Source     string `json:"source"`
	AttestedBy string `json:"attested_by,omitempty"`
	Files      int    `json:"files"`
	Passed     int    `json:"passed"`
	Failed     int    `json:"failed"`
	Skipped    int    `json:"skipped,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type SmokeSection struct {
	Source     string  `json:"source"`
	AttestedBy string  `json:"attested_by,omitempty"`
	Present    bool    `json:"present"`
	Passed     bool    `json:"passed"`
	Seconds    float64 `json:"seconds,omitempty"`
}

type InterfaceSection struct {
	Source     string `json:"source"`
	HashV1     string `json:"hash_v1,omitempty"`
	HashV2     string `json:"hash_v2,omitempty"`
	Signatures int    `json:"signatures"`
	Error      string `json:"error,omitempty"`
}

type EffectsSection struct {
	Source          string   `json:"source"`
	Max             []string `json:"max"`
	CeilingDeclared bool     `json:"ceiling_declared"`
	RankMax         string   `json:"rank_max,omitempty"`
}

// ReleaseSection is filled by M4 ([release] + CHANGELOG); present from M3 so
// the schema is stable.
type ReleaseSection struct {
	Source           string `json:"source"`
	Kind             string `json:"kind,omitempty"`
	ChangelogSection bool   `json:"changelog_section"`
	Notes            string `json:"notes,omitempty"`
}

type DocsSection struct {
	AgentMD   bool `json:"agent_md"`
	AISummary bool `json:"ai_summary"`
}

type StyleSection struct {
	ExportedFuncs int     `json:"exported_funcs"`
	PureExports   int     `json:"pure_exports"`
	PureRatio     float64 `json:"pure_ratio"`
}

// EffectPrivilegeRank orders effects by the authority they confer (design D4,
// ratified 2026-09-17): a widening TOWARD a higher-ranked effect is an
// escalation. Declassify is a taint sink, not a capability, and is always
// Tier 3; it is ranked above everything so RankMax surfaces it.
var EffectPrivilegeRank = []string{"Declassify", "Process", "Net", "FS", "Env", "AI", "SharedMem", "Stream", "IO", "Clock", "Rand"}

// RankOf returns the privilege rank (0 = highest) or -1 for unknown effects.
func RankOf(effect string) int {
	for i, e := range EffectPrivilegeRank {
		if e == effect {
			return i
		}
	}
	return -1
}

// HighestRanked returns the most privileged effect in the set ("" if none).
func HighestRanked(effects []string) string {
	best, bestRank := "", len(EffectPrivilegeRank)+1
	for _, e := range effects {
		r := RankOf(e)
		if r < 0 {
			r = len(EffectPrivilegeRank) // unknown effects sort last
		}
		if r < bestRank {
			best, bestRank = e, r
		}
	}
	return best
}

// StabilityGates reports whether badges become gates at this stability
// (design D3: stable and frozen are held to the full table).
func StabilityGates(level string) bool {
	return level == "stable" || level == "frozen"
}

// BuildQualityReport assembles the report from measurements. strict promotes
// every badge to a gate (the publisher-local `--strict`).
func BuildQualityReport(m *PackageManifest, mode QualityMode, in QualityInputs, strict bool) *QualityReport {
	r := &QualityReport{
		Schema:    QualitySchema,
		Package:   m.Package.Name,
		Version:   m.Package.Version,
		Stability: m.Stability.Level,
		Mode:      map[QualityMode]string{ModePublisher: "publisher", ModeServer: "server"}[mode],
		Gates:     []Finding{},
		Badges:    []Finding{},
	}
	hard := StabilityGates(m.Stability.Level) || strict

	// compile (server)
	r.Compile = CompileSection{Source: SourceServer, OK: in.CompileOK, Files: in.CompileFiles, Error: in.CompileError}
	if !in.CompileOK {
		r.gate("PUB000", "compile failed: "+firstLine(in.CompileError))
	}

	// contracts (server)
	r.Contracts = ContractsSection{Source: SourceServer, Error: in.VerifyErr}
	if in.Verify != nil {
		r.Contracts.Verified = in.Verify.Verified
		r.Contracts.Total = in.Verify.Total
		r.Contracts.Counterexample = in.Verify.Counterexample
		r.Contracts.Skipped = in.Verify.Skipped
		r.Contracts.Errors = in.Verify.Errors
		r.Contracts.UncontractedExports = in.Verify.Uncontracted
		r.Contracts.WallSeconds = in.Verify.WallSeconds
		if in.Verify.Counterexample > 0 {
			r.gate("PUB006", fmt.Sprintf("%d contract(s) refuted by Z3", in.Verify.Counterexample))
		}
		if in.Verify.Uncontracted > 0 {
			r.finding("PUB011", hard && m.Stability.Level == "frozen", "warn",
				fmt.Sprintf("%d exported function(s) carry no contract", in.Verify.Uncontracted))
		}
	} else if in.CompileOK {
		r.badge("PUB011", "warn", "contract verification did not run: "+firstLine(in.VerifyErr))
	}

	// interface (server)
	r.Interface = InterfaceSection{Source: SourceServer, HashV1: in.InterfaceHashV1, HashV2: in.InterfaceHashV2, Signatures: len(in.Signatures), Error: in.InterfaceV2Err}
	if in.InterfaceV2Err != "" && in.CompileOK {
		// Shadow mode (design D6): a badge until the D6 streak flips it.
		r.badge("PUB005", "warn", "interface identity (v2) could not be built: "+firstLine(in.InterfaceV2Err))
	}

	// effects (server) — measured footprint and scopes are Sprint 3.
	r.Effects = EffectsSection{Source: SourceServer, Max: append([]string{}, m.Effects.Max...), CeilingDeclared: m.Effects.Max != nil, RankMax: HighestRanked(m.Effects.Max)}
	sort.Strings(r.Effects.Max)
	if m.Effects.Max == nil {
		r.finding("PUB010", hard, "warn", "no [effects] ceiling declared — an absent ceiling is UNLIMITED; declare max = [] for a pure package")
	}

	// release (server) — design D2. Grace: badges below ReleaseGateHardFrom.
	r.Release = ReleaseSection{Source: SourceServer, Kind: m.Release.Kind, ChangelogSection: in.HasChangelogSection, Notes: in.ChangelogNotes}
	if !in.HasChangelogSection {
		r.finding("PUB001", in.ReleaseGatesHard, "warn",
			fmt.Sprintf("%s has no non-empty `## %s` section — describe what this version changes", ChangelogFile, m.Package.Version))
	}
	if m.Release.Kind == "" {
		r.finding("PUB002", in.ReleaseGatesHard, "warn",
			"[release] kind is not declared — one of "+strings.Join(ReleaseKinds, "|"))
	}

	// docs
	_, hasSummary := m.Metadata["ai_summary"]
	r.Docs = DocsSection{AgentMD: in.HasAgentDoc, AISummary: hasSummary}
	if !in.HasAgentDoc {
		r.badge("PUB020", "info", "no AGENT.md — agents get no usage guidance for this package")
	}
	// M6: the package's inbox agent is DERIVED from metadata.repository; without
	// a GitHub tree URL the inbox is served by the pkg:* template's default
	// workspace, which may be the wrong repo.
	repoURL, _ := m.Metadata["repository"].(string)
	if _, ok := ParseRepositoryURL(repoURL); !ok {
		r.badge("PUB021", "warn", "[metadata] repository is not a GitHub tree URL — the pkg:"+m.Package.Name+" inbox agent cannot be derived (workspace/subdirectory unknown)")
	}

	// style — from the signature set: `mod:func:name:type:<effects>`; an
	// empty trailing field is a pure function.
	r.Style = styleFromSignatures(in.Signatures)

	// attested — executed by the publisher, never by the server.
	if in.Attested != nil {
		if in.Attested.Tests != nil {
			t := *in.Attested.Tests
			t.Source, t.AttestedBy = SourceAttested, in.Attested.AttestedBy
			r.Tests = &t
			if t.Failed > 0 {
				r.finding("PUB012", mode == ModePublisher, "warn", fmt.Sprintf("%d test(s) failed", t.Failed))
			}
		}
		if in.Attested.Smoke != nil {
			s := *in.Attested.Smoke
			s.Source, s.AttestedBy = SourceAttested, in.Attested.AttestedBy
			r.Smoke = &s
			if s.Present && !s.Passed {
				r.finding("PUB015", mode == ModePublisher, "warn", "_smoke.ail failed")
			}
		}
	}
	if r.Tests == nil || r.Tests.Files+r.Tests.Passed+r.Tests.Failed == 0 {
		// Publisher-local gate at stable/frozen; always a badge on the server.
		r.finding("PUB012", hard && mode == ModePublisher, "info", "no tests discovered (*_test.ail or inline test blocks)")
	}
	if r.Smoke == nil || !r.Smoke.Present {
		// M-EXT-PORTABILITY-GATE (v0.19.0): extension packages MUST ship a
		// smoke test; publisher-local gate, badge elsewhere.
		if HasExtensionBlock(m) {
			r.finding("PUB014", mode == ModePublisher, "warn", "package declares [extension] but has no "+SmokeFile+" (required for extension packages)")
		} else {
			r.badge("PUB014", "info", "no "+SmokeFile+" — publish runs no boot check in a clean workdir")
		}
	}
	if strict {
		// --strict: every warn-level badge blocks; info stays informational.
		kept := r.Badges[:0]
		for _, b := range r.Badges {
			if b.Level == "warn" {
				r.gate(b.Code, b.Msg)
				continue
			}
			kept = append(kept, b)
		}
		r.Badges = kept
	}
	return r
}

// HasGates reports whether anything blocks.
func (r *QualityReport) HasGates() bool { return len(r.Gates) > 0 }

func (r *QualityReport) gate(code, msg string) {
	r.Gates = append(r.Gates, Finding{Code: code, Level: "gate", Msg: msg})
}
func (r *QualityReport) badge(code, level, msg string) {
	r.Badges = append(r.Badges, Finding{Code: code, Level: level, Msg: msg})
}

// finding routes to gates when hard, else to badges at the given level.
func (r *QualityReport) finding(code string, hard bool, level, msg string) {
	if hard {
		r.gate(code, msg)
		return
	}
	r.badge(code, level, msg)
}

func styleFromSignatures(sigs []string) StyleSection {
	var s StyleSection
	for _, sig := range sigs {
		parts := strings.Split(sig, ":")
		if len(parts) < 5 || parts[1] != "func" {
			continue
		}
		s.ExportedFuncs++
		if parts[len(parts)-1] == "" {
			s.PureExports++
		}
	}
	if s.ExportedFuncs > 0 {
		s.PureRatio = float64(s.PureExports) / float64(s.ExportedFuncs)
	}
	return s
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
