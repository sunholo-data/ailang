package eval_harness

// The execution envelope binds controller-owned inputs to the existing native
// trajectory ledger. It contains paths the model must never receive wholesale.
import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

const MaintenancePackageSourceRevision = "a4163559a2e20dddc1a1bb0869a06e42372994a1"

type MaintenanceFileRef struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type MaintenanceAsset struct {
	Path        string `json:"path"`
	Destination string `json:"destination"`
	SHA256      string `json:"sha256"`
	Executable  bool   `json:"executable"`
}
type MaintenanceCommandBinding struct {
	Command []string             `json:"command"`
	Files   []MaintenanceFileRef `json:"files"`
}
type MaintenanceStageBinding struct {
	ID          string                    `json:"id"`
	Request     MaintenanceFileRef        `json:"request"`
	PublicCheck MaintenanceCommandBinding `json:"publicCheck"`
	Oracle      MaintenanceCommandBinding `json:"oracle"`
}
type MaintenanceTrajectoryBinding struct {
	ID         string                    `json:"id"`
	Starter    []MaintenanceAsset        `json:"starter"`
	CarryFiles []string                  `json:"carryFiles"`
	Stages     []MaintenanceStageBinding `json:"stages"`
}
type MaintenanceToolchain struct {
	Language string             `json:"language"`
	Command  []string           `json:"command"`
	Version  string             `json:"version"`
	Evidence MaintenanceFileRef `json:"evidence"`
}
type MaintenanceLimits struct {
	MaxTrajectories    int `json:"maxTrajectories"`
	MaxStages          int `json:"maxStages"`
	MaxStageSeconds    int `json:"maxStageSeconds"`
	MaxCampaignSeconds int `json:"maxCampaignSeconds"`
	MaxRequests        int `json:"maxRequests"`
	MaxTokens          int `json:"maxTokens"`
}
type MaintenanceIsolation struct {
	Backend      string `json:"backend"`
	Status       string `json:"status"`
	PolicyDigest string `json:"policyDigest"`
}
type MaintenanceExecutionManifest struct {
	Schema                string                         `json:"schema"`
	PackageSourceRevision string                         `json:"packageSourceRevision"`
	PackageRoot           string                         `json:"packageRoot"`
	Plan                  MaintenanceFileRef             `json:"plan"`
	Settings              MaintenanceFileRef             `json:"settings"`
	Analysis              MaintenanceFileRef             `json:"analysis"`
	Toolchains            []MaintenanceToolchain         `json:"toolchains"`
	Trajectories          []MaintenanceTrajectoryBinding `json:"trajectories"`
	ProtectedRoots        []string                       `json:"protectedRoots"`
	Limits                MaintenanceLimits              `json:"limits"`
	Isolation             MaintenanceIsolation           `json:"isolation"`
}

type MaintenanceStagePlan struct {
	ID                string `json:"id"`
	RequirementDigest string `json:"requirementDigest"`
	OracleDigest      string `json:"oracleDigest"`
}
type MaintenanceTrajectoryPlan struct {
	ID               string                 `json:"id"`
	Family           string                 `json:"family"`
	Variant          string                 `json:"variant"`
	Repeat           int                    `json:"repeat"`
	Language         string                 `json:"language"`
	StarterDigest    string                 `json:"starterDigest"`
	NativeToolDigest string                 `json:"nativeToolDigest"`
	Stages           []MaintenanceStagePlan `json:"stages"`
}
type MaintenanceRunPlan struct {
	Schema         string                      `json:"schema"`
	ProtocolID     string                      `json:"protocolId"`
	Role           string                      `json:"role"`
	Model          string                      `json:"model"`
	Harness        string                      `json:"harness"`
	SettingsDigest string                      `json:"settingsDigest"`
	Retention      string                      `json:"retention"`
	Treatment      string                      `json:"treatment"`
	LeftLanguage   string                      `json:"leftLanguage"`
	RightLanguage  string                      `json:"rightLanguage"`
	Trajectories   []MaintenanceTrajectoryPlan `json:"trajectories"`
}
type MaintenanceExecution struct {
	Manifest       MaintenanceExecutionManifest
	Plan           MaintenanceRunPlan
	Root           string
	ManifestSHA256 string
}

// LoadMaintenanceExecutionManifest performs only local immutable input reads.
// No model registry, executor, canary, credentials, rig lock or child process is
// accessed, so dry-run uses exactly the admission input binding without dispatch.
func LoadMaintenanceExecutionManifest(path string) (*MaintenanceExecution, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if err = maintenanceRegularPath(absolute, false); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(absolute)
	if err != nil {
		return nil, err
	}
	e := &MaintenanceExecution{Root: filepath.Dir(absolute)}
	if err = maintenanceStrictJSON(raw, &e.Manifest); err != nil {
		return nil, fmt.Errorf("execution manifest: %w", err)
	}
	sum := sha256.Sum256(raw)
	e.ManifestSHA256 = hex.EncodeToString(sum[:])
	if err = e.validate(); err != nil {
		return nil, fmt.Errorf("execution manifest: %w", err)
	}
	return e, nil
}

func (e *MaintenanceExecution) validate() error {
	m := &e.Manifest
	if m.Schema != "maintenance.execution/1" || m.PackageSourceRevision != MaintenancePackageSourceRevision {
		return fmt.Errorf("require maintenance.execution/1 and pinned package source %s", MaintenancePackageSourceRevision)
	}
	if len(m.ProtectedRoots) == 0 {
		return fmt.Errorf("protected source roots must be explicit")
	}
	for i, p := range m.ProtectedRoots {
		p = e.resolve(p)
		if err := maintenanceRegularPath(p, true); err != nil {
			return err
		}
		m.ProtectedRoots[i] = p
	}
	m.PackageRoot = e.resolve(m.PackageRoot)
	if !e.protected(m.PackageRoot) {
		return fmt.Errorf("package source is not under a protected root")
	}
	if err := maintenanceRegularPath(m.PackageRoot, true); err != nil {
		return err
	}
	for _, ref := range []*MaintenanceFileRef{&m.Plan, &m.Settings, &m.Analysis} {
		if err := e.bind(ref); err != nil {
			return err
		}
	}
	b, err := os.ReadFile(m.Plan.Path)
	if err != nil {
		return err
	}
	if err = maintenanceStrictJSON(b, &e.Plan); err != nil {
		return fmt.Errorf("trajectory plan: %w", err)
	}
	p := &e.Plan
	if p.Schema != "maintenance.trajectory/2" || p.Retention != "fresh-session-owned-artifacts" || p.Treatment != "native-tools" || p.ProtocolID == "" || p.Model == "" || p.Harness == "" {
		return fmt.Errorf("invalid native trajectory/2 identity or retention")
	}
	if p.Role != "instrument_control" && p.Role != "public_development_control" {
		return fmt.Errorf("unsupported evidence role %q", p.Role)
	}
	if p.SettingsDigest != m.Settings.SHA256 {
		return fmt.Errorf("settings digest differs from native plan")
	}
	if p.LeftLanguage == "" || p.RightLanguage == "" || p.LeftLanguage == p.RightLanguage {
		return fmt.Errorf("two distinct language arms required")
	}
	l := m.Limits
	if l.MaxTrajectories < 1 || l.MaxStages < 1 || l.MaxRequests < 1 || l.MaxTokens < 1 || l.MaxStageSeconds < 1 || l.MaxCampaignSeconds < 1 {
		return fmt.Errorf("all schedule/request/token/time limits must be positive")
	}
	if m.Isolation.Backend != "pi" || (m.Isolation.Status != "unqualified" && m.Isolation.Status != "qualified") || !maintenanceSHA(m.Isolation.PolicyDigest) {
		return fmt.Errorf("explicit pi isolation status/policy digest required")
	}
	if len(p.Trajectories) == 0 || len(p.Trajectories) > l.MaxTrajectories || len(m.Trajectories) != len(p.Trajectories) {
		return fmt.Errorf("trajectory schedule exceeds bound or bindings are missing")
	}
	tools := map[string]string{}
	for i := range m.Toolchains {
		tool := &m.Toolchains[i]
		if tool.Language == "" || tool.Version == "" || !maintenanceCommand(tool.Command) || tools[tool.Language] != "" {
			return fmt.Errorf("invalid or duplicate native toolchain")
		}
		if err = e.bind(&tool.Evidence); err != nil {
			return err
		}
		tools[tool.Language] = tool.Evidence.SHA256
	}
	arms := map[string]map[string]bool{}
	ids := map[string]bool{}
	count := 0
	for i := range p.Trajectories {
		t := &p.Trajectories[i]
		binding := &m.Trajectories[i]
		if t.ID == "" || ids[t.ID] || t.Family == "" || t.Variant == "" || t.Repeat < 1 || (t.Language != p.LeftLanguage && t.Language != p.RightLanguage) || binding.ID != t.ID {
			return fmt.Errorf("invalid, duplicate or unbound trajectory %q", t.ID)
		}
		ids[t.ID] = true
		key := fmt.Sprintf("%s\x00%s\x00%d", t.Family, t.Variant, t.Repeat)
		if arms[key] == nil {
			arms[key] = map[string]bool{}
		}
		if arms[key][t.Language] {
			return fmt.Errorf("duplicate language arm %s", t.ID)
		}
		arms[key][t.Language] = true
		if tools[t.Language] == "" || tools[t.Language] != t.NativeToolDigest {
			return fmt.Errorf("native tool identity missing or mismatched for %s", t.ID)
		}
		if err = e.validateBinding(t, binding); err != nil {
			return err
		}
		count += len(t.Stages)
	}
	for key, languages := range arms {
		if len(languages) != 2 {
			return fmt.Errorf("missing native counterpart for %q", key)
		}
	}
	if count > l.MaxStages {
		return fmt.Errorf("stage schedule exceeds bound")
	}
	return nil
}

func (e *MaintenanceExecution) validateBinding(t *MaintenanceTrajectoryPlan, b *MaintenanceTrajectoryBinding) error {
	if len(b.Starter) == 0 || len(b.CarryFiles) == 0 || len(t.Stages) != 3 || len(b.Stages) != 3 {
		return fmt.Errorf("%s needs starter, owned files and exactly three stages", t.ID)
	}
	paths := map[string]bool{}
	for i := range b.Starter {
		a := &b.Starter[i]
		if !maintenanceArtifactPath(a.Destination) || paths[a.Destination] {
			return fmt.Errorf("invalid or duplicate carried starter path %q", a.Destination)
		}
		paths[a.Destination] = true
		ref := MaintenanceFileRef{Path: a.Path, SHA256: a.SHA256}
		if err := e.bind(&ref); err != nil {
			return err
		}
		a.Path = ref.Path
		info, err := os.Stat(a.Path)
		if err != nil {
			return err
		}
		if info.Mode().Perm()&0111 != 0 != a.Executable {
			return fmt.Errorf("starter executable identity differs for %s", a.Path)
		}
	}
	digest, err := maintenanceStarterDigest(b.Starter)
	if err != nil {
		return err
	}
	if digest != t.StarterDigest {
		return fmt.Errorf("starter digest differs for %s", t.ID)
	}
	carry := map[string]bool{}
	for _, path := range b.CarryFiles {
		if !maintenanceArtifactPath(path) || carry[path] {
			return fmt.Errorf("invalid or duplicate carried-artifact path %q", path)
		}
		carry[path] = true
	}
	for _, a := range b.Starter {
		if !carry[a.Destination] {
			return fmt.Errorf("starter %q is outside owned-file declaration", a.Destination)
		}
	}
	for i := range t.Stages {
		s, bs := &t.Stages[i], &b.Stages[i]
		if s.ID != fmt.Sprintf("S%d", i+1) || bs.ID != s.ID || !maintenanceSHA(s.RequirementDigest) || !maintenanceSHA(s.OracleDigest) {
			return fmt.Errorf("invalid three-stage contract for %s", t.ID)
		}
		if err = e.bind(&bs.Request); err != nil {
			return err
		}
		if bs.Request.SHA256 != s.RequirementDigest {
			return fmt.Errorf("requirement identity differs for %s/%s", t.ID, s.ID)
		}
		for _, cmd := range []*MaintenanceCommandBinding{&bs.PublicCheck, &bs.Oracle} {
			if !maintenanceCommand(cmd.Command) || len(cmd.Files) == 0 {
				return fmt.Errorf("check/oracle command and source evidence required")
			}
			for j := range cmd.Files {
				if err = e.bind(&cmd.Files[j]); err != nil {
					return err
				}
			}
		}
		pipeline, err := maintenanceOracleDigest(bs.Oracle.Files)
		if err != nil || pipeline != s.OracleDigest {
			return fmt.Errorf("oracle pipeline identity differs for %s/%s", t.ID, s.ID)
		}
	}
	return nil
}

func (e *MaintenanceExecution) resolve(p string) string {
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(e.Root, p)
}
func (e *MaintenanceExecution) protected(p string) bool {
	for _, root := range e.Manifest.ProtectedRoots {
		rel, err := filepath.Rel(root, p)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
func (e *MaintenanceExecution) bind(ref *MaintenanceFileRef) error {
	if ref.Path == "" || !maintenanceSHA(ref.SHA256) {
		return fmt.Errorf("file path and lowercase SHA256 required")
	}
	ref.Path = e.resolve(ref.Path)
	if !e.protected(ref.Path) {
		return fmt.Errorf("controller source %q is outside protected roots", ref.Path)
	}
	if err := maintenanceRegularPath(ref.Path, false); err != nil {
		return err
	}
	b, err := os.ReadFile(ref.Path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != ref.SHA256 {
		return fmt.Errorf("source hash mismatch: %s", ref.Path)
	}
	return nil
}
func maintenanceStarterDigest(assets []MaintenanceAsset) (string, error) {
	type entry struct {
		Path       string `json:"path"`
		SHA256     string `json:"sha256"`
		Executable bool   `json:"executable"`
	}
	files := make([]entry, len(assets))
	for i, a := range assets {
		files[i] = entry{a.Destination, a.SHA256, a.Executable}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	b, err := json.Marshal(files)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
func maintenanceSHA(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func maintenanceCommand(cmd []string) bool {
	if len(cmd) == 0 {
		return false
	}
	for _, s := range cmd {
		if s == "" || strings.ContainsRune(s, 0) {
			return false
		}
	}
	return true
}
func maintenanceArtifactPath(s string) bool {
	return utf8.ValidString(s) && s != "" && s != "." && !filepath.IsAbs(s) && !strings.ContainsAny(s, "\\:\x00") && !strings.HasPrefix(s, "/") && filepath.ToSlash(filepath.Clean(s)) == s && s != ".." && !strings.HasPrefix(s, "../")
}
func maintenanceRegularPath(p string, directory bool) error {
	for current := p; ; current = filepath.Dir(current) {
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink source rejected: %s", current)
		}
		if current == p && ((directory && !info.IsDir()) || (!directory && !info.Mode().IsRegular())) {
			return fmt.Errorf("unexpected source file type: %s", p)
		}
		if filepath.Dir(current) == current {
			break
		}
	}
	return nil
}

// maintenanceOracleDigest binds the entire ordered grading source inventory.
// A wrapper file alone does not identify its imported oracle/expected fixtures.
// Paths and native-language argv are excluded so counterpart arms share one
// pipeline identity across movable source checkouts.
func maintenanceOracleDigest(files []MaintenanceFileRef) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("empty oracle pipeline")
	}
	hashes := make([]string, len(files))
	for i, file := range files {
		if !maintenanceSHA(file.SHA256) {
			return "", fmt.Errorf("invalid oracle dependency digest")
		}
		hashes[i] = file.SHA256
	}
	raw, err := json.Marshal(hashes)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
