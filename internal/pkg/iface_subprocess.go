package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/proctree"
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
		MaxExportedModules: 128, // sunholo/ailang_parse exports 56 (M-PKG-QUALITY-LADDER raised from 64)
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

// pinnedIfaceBinary is the binary the ailang CLI registered as itself (see
// UseIfaceBinary). Empty in every other process.
var pinnedIfaceBinary string

// UseIfaceBinary pins the binary BuildModuleIface execs. The ailang CLI pins
// its own executable at startup, so the identity it prints and sends is
// computed by the version that is running — whatever the file is called.
//
// This used to be decided by the file NAME: only an executable called exactly
// `ailang` counted as itself, and anything else fell back to the first
// `ailang` on PATH without saying so. Measured 2026-10-19: a v0.54.15 build
// run under another name hashed sunholo/email 0.10.0 with the installed
// v0.52.0, and the registry refused it (PUB005) against every "upgraded"
// publisher the author tried.
func UseIfaceBinary(path string) {
	pinnedIfaceBinary = path
}

// realResolveIfaceBinary is the production implementation. It is a named
// function so tests can reach it even while the resolveIfaceBinary var is
// overridden (TestMain replaces the var for every BuildModuleIface arm).
// Without a pin (the registry validator, tests) it is the `ailang` on PATH.
func realResolveIfaceBinary() (string, error) {
	if pinnedIfaceBinary != "" {
		return pinnedIfaceBinary, nil
	}
	path, lookupErr := exec.LookPath("ailang")
	if lookupErr != nil {
		return "", fmt.Errorf("resolve ailang binary (none pinned, none on PATH): %w", lookupErr)
	}
	return path, nil
}

var resolveIfaceBinary = realResolveIfaceBinary

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
	proctree.Configure(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if moduleCtx.Err() != nil {
		proctree.Kill(cmd)
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
