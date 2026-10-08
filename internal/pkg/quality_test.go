package pkg

import (
	"encoding/json"
	"strings"
	"testing"
)

func qualityManifest(stability string, effects []string) *PackageManifest {
	m := &PackageManifest{}
	m.Package.Name = "test/q"
	m.Package.Version = "0.1.0"
	m.Package.Edition = "1"
	m.Stability.Level = stability
	m.Effects.Max = effects
	m.Metadata = map[string]interface{}{"ai_summary": "x", "repository": "https://github.com/test/q/tree/main/packages/q"}
	m.Metadata["license_url"] = "https://example.test/LICENSE"
	m.Exports.Modules = []string{"test/q/a"}
	return m
}

// M6: a package whose repository cannot be parsed cannot get a derived inbox agent.
func TestBuildQualityReport_RepositoryBadge(t *testing.T) {
	m := qualityManifest("experimental", []string{})
	if r := BuildQualityReport(m, ModeServer, cleanInputs(), false); hasCode(r.Badges, "PUB021") {
		t.Errorf("GitHub tree URL must not badge: %v", r.Badges)
	}
	delete(m.Metadata, "repository")
	if r := BuildQualityReport(m, ModeServer, cleanInputs(), false); !hasCode(r.Badges, "PUB021") {
		t.Errorf("missing repository must badge PUB021: %v", r.Badges)
	}
}

func cleanInputs() QualityInputs {
	return QualityInputs{
		CompileOK: true, CompileFiles: 2,
		Verify:          &PackageVerifyReport{Schema: PackageVerifySchema, Verified: 2, Total: 2, Uncontracted: 1},
		InterfaceHashV1: "sha256:v1", InterfaceHashV2: "sha256:ifacev2:abc",
		Signatures:      []string{"test/q/a:func:capAt:(int,int)->int:", "test/q/a:func:load:(string)->string:FS", "test/q/a:type:Money/0"},
		HasAgentDoc:     true,
		AgentDocContent: "Use test/q/a for all operations.",
	}
}

func TestBuildQualityReport_DiscoverabilityFindings(t *testing.T) {
	m := qualityManifest("experimental", []string{})
	in := cleanInputs()
	delete(m.Metadata, "ai_summary")
	delete(m.Metadata, "license_url")
	in.AgentDocContent = "stale guide"
	in.Overlap = []string{"z/pkg", "a/pkg"}
	r := BuildQualityReport(m, ModePublisher, in, false)
	for _, code := range []string{"PUB008", "PUB018", "PUB022", "PUB023"} {
		if !hasCode(r.Badges, code) {
			t.Errorf("missing %s: %+v", code, r.Badges)
		}
	}
	if got := strings.Join(r.Docs.ExportOverlap, ","); got != "a/pkg,z/pkg" {
		t.Errorf("overlap = %q", got)
	}
	strict := BuildQualityReport(m, ModePublisher, in, true)
	if !hasCode(strict.Gates, "PUB018") || !hasCode(strict.Gates, "PUB022") {
		t.Errorf("strict warnings not promoted: %+v", strict.Gates)
	}
	if hasCode(strict.Gates, "PUB008") || hasCode(strict.Gates, "PUB023") {
		t.Errorf("info findings promoted: %+v", strict.Gates)
	}
	clean := BuildQualityReport(qualityManifest("experimental", []string{}), ModePublisher, cleanInputs(), false)
	for _, code := range []string{"PUB008", "PUB018", "PUB022", "PUB023"} {
		if hasCode(clean.Badges, code) {
			t.Errorf("clean input emitted %s: %+v", code, clean.Badges)
		}
	}
}

func hasCode(fs []Finding, code string) bool {
	for _, f := range fs {
		if f.Code == code {
			return true
		}
	}
	return false
}

// The seam the design's success metric #3 names: publisher and validator
// assemble byte-identical `server` sections from identical measurements, with
// the attested block being the only difference.
func TestBuildQualityReport_ServerSectionsIdenticalAcrossModes(t *testing.T) {
	m := qualityManifest("experimental", []string{"FS"})
	in := cleanInputs()
	pub := BuildQualityReport(m, ModePublisher, withAttested(in, "daneel", 3, 0), false)
	srv := BuildQualityReport(m, ModeServer, withAttested(in, "daneel", 3, 0), false)

	type serverView struct {
		Compile   CompileSection
		Contracts ContractsSection
		Interface InterfaceSection
		Effects   EffectsSection
		Release   ReleaseSection
		Docs      DocsSection
		Style     StyleSection
		Gates     []Finding
	}
	view := func(r *QualityReport) string {
		b, _ := json.Marshal(serverView{r.Compile, r.Contracts, r.Interface, r.Effects, r.Release, r.Docs, r.Style, r.Gates})
		return string(b)
	}
	if view(pub) != view(srv) {
		t.Fatalf("server sections differ across modes:\npublisher: %s\nserver:    %s", view(pub), view(srv))
	}
	if pub.Tests == nil || pub.Tests.Source != SourceAttested || pub.Tests.AttestedBy != "daneel" {
		t.Errorf("attested tests not stamped: %+v", pub.Tests)
	}
	if pub.Style.ExportedFuncs != 2 || pub.Style.PureExports != 1 || pub.Style.PureRatio != 0.5 {
		t.Errorf("style = %+v, want 2 funcs / 1 pure / 0.5", pub.Style)
	}
	if pub.Effects.RankMax != "FS" {
		t.Errorf("rank_max = %q", pub.Effects.RankMax)
	}
}

func withAttested(in QualityInputs, by string, passed, failed int) QualityInputs {
	in.Attested = &AttestedBlock{AttestedBy: by, Tests: &TestsSection{Files: 1, Passed: passed, Failed: failed}, Smoke: &SmokeSection{Present: true, Passed: true, Seconds: 0.5}}
	return in
}

// D3 — gate/badge split by stability, and attested findings never gate on the server.
func TestBuildQualityReport_GateBadgeSplit(t *testing.T) {
	in := cleanInputs()
	in.Attested = &AttestedBlock{AttestedBy: "me", Tests: &TestsSection{Files: 1, Passed: 1, Failed: 1}}

	exp := BuildQualityReport(qualityManifest("experimental", nil), ModePublisher, in, false)
	if hasCode(exp.Gates, "PUB010") || !hasCode(exp.Badges, "PUB010") {
		t.Errorf("experimental nil ceiling must be a badge, got gates=%v badges=%v", exp.Gates, exp.Badges)
	}
	if !hasCode(exp.Gates, "PUB012") {
		t.Errorf("a FAILED attested test must gate on the publisher's own machine: %v", exp.Gates)
	}

	stable := BuildQualityReport(qualityManifest("stable", nil), ModePublisher, in, false)
	if !hasCode(stable.Gates, "PUB010") {
		t.Errorf("stable nil ceiling must gate: %v", stable.Gates)
	}

	srv := BuildQualityReport(qualityManifest("stable", nil), ModeServer, in, false)
	if hasCode(srv.Gates, "PUB012") || !hasCode(srv.Badges, "PUB012") {
		t.Errorf("attested test failure must NEVER gate on the server (RCE boundary): gates=%v", srv.Gates)
	}

	// Compile failure and a Z3 counterexample gate everywhere.
	broken := cleanInputs()
	broken.CompileOK, broken.CompileError = false, "type error: x\nmore"
	if r := BuildQualityReport(qualityManifest("experimental", []string{}), ModeServer, broken, false); !hasCode(r.Gates, "PUB000") {
		t.Errorf("compile failure must gate: %v", r.Gates)
	}
	refuted := cleanInputs()
	refuted.Verify.Counterexample = 1
	if r := BuildQualityReport(qualityManifest("experimental", []string{}), ModeServer, refuted, false); !hasCode(r.Gates, "PUB006") {
		t.Errorf("counterexample must gate: %v", r.Gates)
	}
}

// --strict promotes warn-level badges to gates (the phantom flag's promise).
func TestBuildQualityReport_StrictPromotesWarnings(t *testing.T) {
	in := cleanInputs()
	in.InterfaceV2Err = "boom"
	loose := BuildQualityReport(qualityManifest("experimental", []string{}), ModePublisher, in, false)
	if !hasCode(loose.Badges, "PUB005") || hasCode(loose.Gates, "PUB005") {
		t.Fatalf("shadow-mode v2 failure must be a badge: %v / %v", loose.Gates, loose.Badges)
	}
	strict := BuildQualityReport(qualityManifest("experimental", []string{}), ModePublisher, in, true)
	if !hasCode(strict.Gates, "PUB005") {
		t.Errorf("--strict must promote PUB005: %v", strict.Gates)
	}
	if hasCode(strict.Gates, "PUB020") { // info-level
		t.Errorf("info-level badges stay informational under --strict: %v", strict.Gates)
	}
}

func TestEffectPrivilegeRank_D4(t *testing.T) {
	if got := HighestRanked([]string{"IO", "Net", "Clock"}); got != "Net" {
		t.Errorf("HighestRanked = %q, want Net", got)
	}
	if got := HighestRanked([]string{"FS", "Declassify"}); got != "Declassify" {
		t.Errorf("Declassify must outrank everything: %q", got)
	}
	if RankOf("Process") >= RankOf("Net") || RankOf("Net") >= RankOf("FS") || RankOf("FS") >= RankOf("Env") {
		t.Errorf("rank order broken: %v", EffectPrivilegeRank)
	}
	if HighestRanked(nil) != "" {
		t.Error("empty set has no rank")
	}
}

func TestQualityReport_JSONHasSchemaAndProvenance(t *testing.T) {
	r := BuildQualityReport(qualityManifest("experimental", []string{}), ModeServer, cleanInputs(), false)
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{`"schema":"ailang.package-quality/v1"`, `"compile":{"source":"server"`, `"contracts":{"source":"server"`, `"gates":[]`} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON missing %s:\n%s", want, s)
		}
	}
}

// #1305: a smoke discovered but not executed (`pkg quality --no-run`) is
// "not run", not "failed" — a badge, never the PUB015 gate. A smoke that DID
// run and failed still gates on the publisher's machine.
func TestBuildQualityReport_SmokeNotRunIsNotAFailure(t *testing.T) {
	m := qualityManifest("experimental", []string{})
	notRun := cleanInputs()
	notRun.Attested = &AttestedBlock{Smoke: &SmokeSection{Present: true, NotRun: true, Notes: "not run (--no-run)"}}
	r := BuildQualityReport(m, ModePublisher, notRun, false)
	if hasCode(r.Gates, "PUB015") || r.HasGates() {
		t.Errorf("un-run smoke must not gate: %v", r.Gates)
	}
	if !hasCode(r.Badges, "PUB015") {
		t.Errorf("un-run smoke should be reported as a not-run badge: %v", r.Badges)
	}
	if strict := BuildQualityReport(m, ModePublisher, notRun, true); hasCode(strict.Gates, "PUB015") {
		t.Errorf("--strict must not promote a not-run smoke (info): %v", strict.Gates)
	}

	failed := cleanInputs()
	failed.Attested = &AttestedBlock{Smoke: &SmokeSection{Present: true, Passed: false}}
	if r := BuildQualityReport(m, ModePublisher, failed, false); !hasCode(r.Gates, "PUB015") {
		t.Errorf("a smoke that ran and failed must still gate: %v", r.Gates)
	}
}
