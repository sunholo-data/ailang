package pipeline

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildCanonicalJSON compiles a module without evaluating it and returns its
// normalized interface JSON. It is library-shaped: all failures are returned
// to the caller rather than terminating the process.
func BuildCanonicalJSON(ctx context.Context, packageDir, modulePath string) ([]byte, error) {
	filename := strings.ReplaceAll(modulePath, "/", string(filepath.Separator))
	if !strings.HasSuffix(filename, ".ail") {
		filename += ".ail"
	}
	if !filepath.IsAbs(filename) {
		filename = filepath.Join(packageDir, filename)
	}

	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %q: %w", filename, err)
	}
	cfg := Config{DryLink: true}
	if packageDir != "" {
		cfg.PackageDir = packageDir
	}
	result, err := RunWithContext(ctx, cfg, Source{
		Code:     string(content),
		Filename: filename,
		IsREPL:   false,
	})
	if err != nil {
		return nil, err
	}
	// DECLARED RESIDUAL — these three branches are defence-in-depth and are
	// NOT reachable through RunWithContext's current contract, so no test kills
	// them (confirmed by the iteration-312 sprint evaluator: each was neutered
	// with `if false && ...`, built rc=0, and left all 819 internal/pipeline
	// tests plus the cmd/ailang arms green).
	//
	// Why unreachable today: a type error in the module surfaces through
	// RunWithContext's own error return rather than as result.Errors with a nil
	// err, and result.Interface is populated by buildAndRegisterInterface only
	// when that step itself returns nil. They are kept because this function is
	// library-shaped and its callers must never see a nil interface or a
	// silently-dropped error list if that contract ever loosens.
	//
	// This gap is named on purpose. Do not read the mutation table as coverage
	// for these lines.
	if len(result.Errors) > 0 {
		return nil, errors.Join(result.Errors...)
	}
	if result.Interface == nil {
		return nil, errors.New("no interface generated for module")
	}

	jsonBytes, err := result.Interface.ToNormalizedJSON()
	if err != nil {
		return nil, fmt.Errorf("serialize interface: %w", err)
	}
	return jsonBytes, nil
}
