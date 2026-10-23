package eval_harness

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/proctree"
)

type MaintenanceAnalysisInput struct{ Binary, PackageRoot, PlanPath, RunsPath, UsagePath, OutputPath string }
type MaintenanceAnalysisProvenance struct {
	Schema              string            `json:"schema"`
	CompilerPath        string            `json:"compilerPath"`
	CompilerVersion     string            `json:"compilerVersion"`
	CompilerSHA256      string            `json:"compilerSHA256"`
	SourceRoot          string            `json:"sourceRoot"`
	SourceSHA256        string            `json:"sourceSHA256"`
	Inputs              map[string]string `json:"inputs"`
	OutputSHA256        string            `json:"outputSHA256"`
	DeclaredPackageBase string            `json:"declaredPackageBase"`
}

func maintenanceFileSHA(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	d := sha256.Sum256(b)
	return hex.EncodeToString(d[:]), nil
}

func maintenanceAnalysisSourceSHA(root string) (string, error) {
	type entry struct {
		Path   string `json:"path"`
		SHA256 string `json:"sha256"`
	}
	rows := []entry{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".ailang" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".ail") && d.Name() != "ailang.toml" && d.Name() != "ailang.lock" {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink in analysis source: %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		hash, err := maintenanceFileSHA(path)
		if err != nil {
			return err
		}
		rows = append(rows, entry{filepath.ToSlash(rel), hash})
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("analysis source is empty")
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Path < rows[j].Path })
	b, err := json.Marshal(rows)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(b)
	return hex.EncodeToString(hash[:]), nil
}

func maintenanceRunCommand(ctx context.Context, dir, bin string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	proctree.Configure(cmd)
	b, err := cmd.CombinedOutput()
	proctree.Kill(cmd)
	if err != nil {
		return b, fmt.Errorf("native analysis command failed: %w: %s", err, b)
	}
	return b, nil
}

func maintenanceWriteNewJSON(path string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(append(b, '\n'))
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// RunMaintenanceNativeAnalysis calls the source-pinned native package. All
// cohorts, costs and deltas remain AILANG computations. This external record
// describes observed compiler/source/input bytes; reportId itself is declared.
func RunMaintenanceNativeAnalysis(ctx context.Context, in MaintenanceAnalysisInput) (*MaintenanceAnalysisProvenance, error) {
	if in.Binary == "" || in.PackageRoot == "" || in.OutputPath == "" {
		return nil, fmt.Errorf("explicit compiler/source/output required")
	}
	bin, err := exec.LookPath(in.Binary)
	if err != nil {
		return nil, err
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		return nil, err
	}
	root, err := filepath.Abs(in.PackageRoot)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	compiler, err := maintenanceFileSHA(bin)
	if err != nil {
		return nil, err
	}
	source, err := maintenanceAnalysisSourceSHA(root)
	if err != nil {
		return nil, err
	}
	p := &MaintenanceAnalysisProvenance{Schema: "maintenance.analysis-provenance/1", CompilerPath: bin, CompilerSHA256: compiler, SourceRoot: root, SourceSHA256: source, DeclaredPackageBase: "a4163559a2e20dddc1a1bb0869a06e42372994a1", Inputs: map[string]string{}}
	paths := []*string{&in.PlanPath, &in.RunsPath, &in.UsagePath, &in.OutputPath}
	for _, path := range paths {
		if *path == "" {
			return nil, fmt.Errorf("missing analysis input/output path")
		}
		*path, err = filepath.Abs(*path)
		if err != nil {
			return nil, err
		}
	}
	for key, path := range map[string]string{"plan": in.PlanPath, "runs": in.RunsPath, "usage": in.UsagePath} {
		h, err := maintenanceFileSHA(path)
		if err != nil {
			return nil, err
		}
		p.Inputs[key] = h
	}
	if _, err = os.Lstat(in.OutputPath); !os.IsNotExist(err) {
		return nil, fmt.Errorf("analysis output must be a new path")
	}
	if _, err = os.Lstat(in.OutputPath + ".provenance.json"); !os.IsNotExist(err) {
		return nil, fmt.Errorf("analysis provenance must be a new path")
	}
	version, err := maintenanceRunCommand(ctx, root, bin, "version")
	if err != nil {
		return nil, err
	}
	p.CompilerVersion = strings.TrimSpace(string(version))
	if p.CompilerVersion == "" {
		return nil, fmt.Errorf("compiler identity unavailable")
	}
	if _, err = maintenanceRunCommand(ctx, root, bin, "run", "--caps", "IO,FS,Env", "--entry", "main", "ledger_cli.ail", "runs", in.PlanPath, in.RunsPath, in.UsagePath, p.CompilerVersion, in.OutputPath); err != nil {
		return nil, err
	}
	after, err := maintenanceAnalysisSourceSHA(root)
	if err != nil {
		return nil, err
	}
	if after != source {
		return nil, fmt.Errorf("analysis source changed during evaluation")
	}
	p.OutputSHA256, err = maintenanceFileSHA(in.OutputPath)
	if err != nil {
		return nil, err
	}
	if err = maintenanceWriteNewJSON(in.OutputPath+".provenance.json", p); err != nil {
		return nil, err
	}
	return p, nil
}
