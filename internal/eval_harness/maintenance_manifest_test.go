package eval_harness

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func maintenanceManifestFixture(t *testing.T) (string, MaintenanceExecutionManifest) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	file := func(name, content string) MaintenanceFileRef {
		p := filepath.Join(root, name)
		if err := os.WriteFile(p, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256([]byte(content))
		return MaintenanceFileRef{Path: name, SHA256: hex.EncodeToString(h[:])}
	}
	settings := file("settings.json", `{"temperature":0}`)
	analysis := file("analysis.ail", "native analysis fixture")
	tool := file("tool.json", `{"version":"fixture-only"}`)
	starter := file("main.py", "broken initial source")
	request := file("request.md", "repair real retained source")
	check := file("check.py", "trusted public check")
	oracle := file("oracle.py", "trusted external oracle")
	assets := []MaintenanceAsset{{Path: starter.Path, Destination: "main.py", SHA256: starter.SHA256}}
	starterDigest, err := maintenanceStarterDigest(assets)
	if err != nil {
		t.Fatal(err)
	}
	plan := MaintenanceRunPlan{Schema: "maintenance.trajectory/2", ProtocolID: "fixture", Role: "instrument_control", Model: "scripted-control", Harness: "scripted", SettingsDigest: settings.SHA256, Retention: "fresh-session-owned-artifacts", Treatment: "native-tools", LeftLanguage: "ailang", RightLanguage: "python"}
	m := MaintenanceExecutionManifest{Schema: "maintenance.execution/1", PackageSourceRevision: MaintenancePackageSourceRevision, PackageRoot: root, Settings: settings, Analysis: analysis, ProtectedRoots: []string{root}, Limits: MaintenanceLimits{MaxTrajectories: 2, MaxStages: 6, MaxStageSeconds: 10, MaxCampaignSeconds: 60, MaxRequests: 6, MaxTokens: 100}, Isolation: MaintenanceIsolation{Backend: "pi", Status: "unqualified", PolicyDigest: strings.Repeat("a", 64)}}
	for _, lang := range []string{"ailang", "python"} {
		stages := []MaintenanceStagePlan{}
		bindings := []MaintenanceStageBinding{}
		for _, id := range []string{"S1", "S2", "S3"} {
			stages = append(stages, MaintenanceStagePlan{ID: id, RequirementDigest: request.SHA256, OracleDigest: oracle.SHA256})
			bindings = append(bindings, MaintenanceStageBinding{ID: id, Request: request, PublicCheck: MaintenanceCommandBinding{Command: []string{"fixture-check"}, Files: []MaintenanceFileRef{check}}, Oracle: MaintenanceCommandBinding{Command: []string{"fixture-oracle"}, Files: []MaintenanceFileRef{oracle}}})
		}
		plan.Trajectories = append(plan.Trajectories, MaintenanceTrajectoryPlan{ID: lang, Family: "real-family", Variant: "development", Repeat: 1, Language: lang, StarterDigest: starterDigest, NativeToolDigest: tool.SHA256, Stages: stages})
		m.Trajectories = append(m.Trajectories, MaintenanceTrajectoryBinding{ID: lang, Starter: assets, CarryFiles: []string{"main.py", "notes.md"}, Stages: bindings})
		m.Toolchains = append(m.Toolchains, MaintenanceToolchain{Language: lang, Command: []string{"fixture"}, Version: "fixture-only", Evidence: tool})
	}
	b, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	m.Plan = file("plan.json", string(b))
	return filepath.Join(root, "maintenance.execution.json"), m
}

func writeMaintenanceManifest(t *testing.T, path string, m MaintenanceExecutionManifest) {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMaintenanceManifestLoadsBoundSchedule(t *testing.T) {
	path, m := maintenanceManifestFixture(t)
	writeMaintenanceManifest(t, path, m)
	got, err := LoadMaintenanceExecutionManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Plan.Trajectories) != 2 || got.Plan.Model != "scripted-control" {
		t.Fatalf("wrong schedule: %+v", got.Plan)
	}
	if !filepath.IsAbs(got.Manifest.Trajectories[0].Stages[0].Request.Path) {
		t.Fatal("controller refs must resolve explicitly")
	}
}

func TestMaintenanceManifestRejectsAlterations(t *testing.T) {
	cases := []struct {
		name string
		edit func(*MaintenanceExecutionManifest)
	}{
		{"source revision", func(m *MaintenanceExecutionManifest) { m.PackageSourceRevision = "registry-0.1" }},
		{"unbounded", func(m *MaintenanceExecutionManifest) { m.Limits.MaxRequests = 0 }},
		{"too small stage cap", func(m *MaintenanceExecutionManifest) { m.Limits.MaxStages = 5 }},
		{"duplicate arm", func(m *MaintenanceExecutionManifest) { m.Trajectories[1].ID = m.Trajectories[0].ID }},
		{"stage order", func(m *MaintenanceExecutionManifest) { m.Trajectories[0].Stages[0].ID = "S2" }},
		{"requirement tamper", func(m *MaintenanceExecutionManifest) {
			m.Trajectories[0].Stages[0].Request.SHA256 = strings.Repeat("b", 64)
		}},
		{"oracle tamper", func(m *MaintenanceExecutionManifest) {
			m.Trajectories[0].Stages[0].Oracle.Files[0].SHA256 = strings.Repeat("b", 64)
		}},
		{"traversal destination", func(m *MaintenanceExecutionManifest) { m.Trajectories[0].Starter[0].Destination = "../escape" }},
		{"absolute carried", func(m *MaintenanceExecutionManifest) { m.Trajectories[0].CarryFiles = []string{"/tmp/escape"} }},
		{"windows carried", func(m *MaintenanceExecutionManifest) { m.Trajectories[0].CarryFiles = []string{`C:\escape`} }},
		{"empty command", func(m *MaintenanceExecutionManifest) { m.Trajectories[0].Stages[0].PublicCheck.Command = nil }},
		{"no oracle evidence", func(m *MaintenanceExecutionManifest) { m.Trajectories[0].Stages[0].Oracle.Files = nil }},
		{"settings mismatch", func(m *MaintenanceExecutionManifest) { m.Settings = m.Analysis }},
		{"unknown protection", func(m *MaintenanceExecutionManifest) { m.ProtectedRoots = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path, m := maintenanceManifestFixture(t)
			tc.edit(&m)
			writeMaintenanceManifest(t, path, m)
			if _, err := LoadMaintenanceExecutionManifest(path); err == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
}

func TestMaintenanceManifestStrictJSONAndSymlink(t *testing.T) {
	for _, extra := range []string{`,"schema":"maintenance.execution/1"`, `,"surprise":true`} {
		path, m := maintenanceManifestFixture(t)
		writeMaintenanceManifest(t, path, m)
		b, _ := os.ReadFile(path)
		b = append(b[:len(b)-1], []byte(extra+"}")...)
		_ = os.WriteFile(path, b, 0600)
		if _, err := LoadMaintenanceExecutionManifest(path); err == nil {
			t.Fatalf("accepted %s", extra)
		}
	}
	path, m := maintenanceManifestFixture(t)
	original := filepath.Join(filepath.Dir(path), m.Settings.Path)
	alias := filepath.Join(filepath.Dir(path), "alias.json")
	if err := os.Symlink(original, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	m.Settings.Path = alias
	writeMaintenanceManifest(t, path, m)
	if _, err := LoadMaintenanceExecutionManifest(path); err == nil {
		t.Fatal("accepted symlink source")
	}
}
