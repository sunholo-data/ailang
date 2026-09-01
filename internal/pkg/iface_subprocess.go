package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/iface"
)

// PublishLimits bounds the work performed while constructing package interfaces.
type PublishLimits struct {
	Overall            time.Duration
	PerModule          time.Duration
	MaxExportedModules int
}

// DefaultPublishLimits returns the default limits for a publish operation.
func DefaultPublishLimits() PublishLimits {
	return PublishLimits{
		Overall:            60 * time.Second,
		PerModule:          10 * time.Second,
		MaxExportedModules: 64,
	}
}

// ModuleIfaceTimeoutError reports that one module exceeded its compile deadline.
type ModuleIfaceTimeoutError struct {
	ModulePath string
	Limit      time.Duration
}

func (e *ModuleIfaceTimeoutError) Error() string {
	return fmt.Sprintf("building interface for module %q timed out after %s", e.ModulePath, e.Limit)
}

func (e *ModuleIfaceTimeoutError) Unwrap() error { return context.DeadlineExceeded }

var resolveIfaceBinary = func() (string, error) {
	executablePath, executableErr := os.Executable()
	if executableErr == nil && strings.TrimSuffix(filepath.Base(executablePath), filepath.Ext(executablePath)) == "ailang" {
		return executablePath, nil
	}
	path, lookupErr := exec.LookPath("ailang")
	if lookupErr != nil {
		return "", fmt.Errorf("resolve ailang binary (current executable %q is not the publisher): %w", executablePath, lookupErr)
	}
	return path, nil
}

// BuildModuleIface compiles modulePath in an isolated ailang subprocess and
// returns its canonical interface representation.
func BuildModuleIface(ctx context.Context, packageDir, modulePath string, lim PublishLimits) (*iface.InterfaceJSON, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	packageRoot, err := filepath.Abs(packageDir)
	if err != nil {
		return nil, fmt.Errorf("resolve package directory: %w", err)
	}
	manifest, err := LoadManifest(packageRoot)
	if err != nil {
		return nil, fmt.Errorf("load package manifest: %w", err)
	}
	if lim.MaxExportedModules > 0 && len(manifest.Exports.Modules) > lim.MaxExportedModules {
		return nil, fmt.Errorf("package exports %d modules, exceeding limit of %d", len(manifest.Exports.Modules), lim.MaxExportedModules)
	}

	moduleCtx := ctx
	cancel := func() {}
	if lim.PerModule > 0 {
		moduleCtx, cancel = context.WithTimeout(ctx, lim.PerModule)
	}
	defer cancel()

	binary, err := resolveIfaceBinary()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(moduleCtx, binary, "internal-dump-iface", packageRoot, modulePath)
	cmd.Dir = packageRoot
	setProcessGroup(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if moduleCtx.Err() != nil {
		if cmd.Process != nil {
			_ = killProcessGroup(cmd.Process.Pid)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if errors.Is(moduleCtx.Err(), context.DeadlineExceeded) {
			return nil, &ModuleIfaceTimeoutError{ModulePath: modulePath, Limit: lim.PerModule}
		}
		return nil, moduleCtx.Err()
	}
	if runErr != nil {
		return nil, fmt.Errorf("build interface for module %q: %w: %s", modulePath, runErr, stderr.String())
	}

	var result iface.InterfaceJSON
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("decode interface for module %q: %w", modulePath, err)
	}
	return &result, nil
}
