package pkg

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SmokeFile is the conventional name of a package's pre-publish smoke test.
// When present, RunSmokeInTempDir executes it and ailang publish requires it
// to pass before uploading the tarball.
const SmokeFile = "_smoke.ail"

// DefaultSmokeTimeout is the wall-clock cap for smoke tests when the manifest
// does not override it. Chosen to be generous for cold starts (module compile,
// dependency loading) but short enough that a hung test fails CI fast.
const DefaultSmokeTimeout = 30 * time.Second

// SmokeResult reports the outcome of a single smoke run.
type SmokeResult struct {
	// Passed is true iff the smoke binary exited with code 0 within Timeout.
	Passed bool
	// Output is the combined stdout+stderr captured during the run, truncated
	// to a reasonable size for inclusion in CLI error messages.
	Output string
	// TimedOut is true iff the run was killed because it exceeded Timeout.
	TimedOut bool
	// ExitCode is the process exit code (0 when passed, non-zero on failure,
	// -1 on timeout/signal).
	ExitCode int
	// Duration is the wall-clock time the smoke test ran before completion or
	// timeout.
	Duration time.Duration
}

// RunSmokeInTempDir runs the package's _smoke.ail in an isolated temp
// directory using the provided ailang binary. The package source is copied
// (only the published file set: ailang.toml, *.ail, AGENT.md, assets/) so
// the smoke test sees nothing inherited from the developer's workdir.
//
// Returns:
//   - (*SmokeResult, nil) when the smoke run completed (pass or fail).
//     Inspect SmokeResult.Passed.
//   - (nil, error) for infrastructure failures (cannot copy files, cannot
//     find ailang binary, cannot mkdir temp). These are rare and should
//     abort publish independent of the smoke result.
//
// The spawned process is placed in its own process group so the entire group
// can be killed on timeout — this prevents `find`, `sleep`, or other helpers
// inside the smoke test from outliving the publish command.
func RunSmokeInTempDir(packageDir, ailangBin string, timeout time.Duration) (*SmokeResult, error) {
	if timeout <= 0 {
		timeout = DefaultSmokeTimeout
	}

	smokePath := filepath.Join(packageDir, SmokeFile)
	if _, err := os.Stat(smokePath); err != nil {
		return nil, fmt.Errorf("no %s in %s", SmokeFile, packageDir)
	}

	tmpDir, err := os.MkdirTemp("", "ailang-smoke-")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := copyPackageContents(packageDir, tmpDir); err != nil {
		return nil, fmt.Errorf("failed to stage smoke workspace: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{
		"run",
		"--caps", "Net,AI,SharedMem,IO,Env,Clock,FS,Process,Stream",
		"--ai-stub",
		"--entry", "main",
		SmokeFile,
	}

	cmd := exec.CommandContext(ctx, ailangBin, args...)
	cmd.Dir = tmpDir
	// Smoke tests use canonical module paths (e.g. `module sunholo/foo/_smoke`)
	// even though the file lives at <tmpdir>/_smoke.ail, so the canonical-path
	// MOD010 check would otherwise reject the load. The smoke is a one-shot
	// verifier, not production code; relax checks for this run only.
	cmd.Env = append(os.Environ(), "AILANG_RELAX_MODULES=1")
	setProcessGroup(cmd)

	var combined bytes.Buffer
	cmd.Stdout = &combined
	cmd.Stderr = &combined

	start := time.Now()
	runErr := cmd.Run()
	elapsed := time.Since(start)

	out := truncateOutput(combined.String(), 8*1024)

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		// Belt-and-braces: CommandContext already kills the leader, but the
		// children survive unless we reap the whole group.
		if cmd.Process != nil {
			_ = killProcessGroup(cmd.Process.Pid)
		}
		return &SmokeResult{
			Passed:   false,
			Output:   out,
			TimedOut: true,
			ExitCode: -1,
			Duration: elapsed,
		}, nil
	}

	exitCode := 0
	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			exitCode = ee.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &SmokeResult{
		Passed:   runErr == nil,
		Output:   out,
		TimedOut: false,
		ExitCode: exitCode,
		Duration: elapsed,
	}, nil
}

// copyPackageContents mirrors the file set CreateTarball would publish into
// destDir, so the smoke test sees the same view the registry will install.
//
// Includes: ailang.toml, *.ail (incl. _smoke.ail), AGENT.md, assets/**.
// Excludes: .git, tests/, ailang.lock.
func copyPackageContents(srcDir, destDir string) error {
	srcDir, err := filepath.Abs(srcDir)
	if err != nil {
		return err
	}

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "tests" || base == "test" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(destDir, rel), 0755)
		}

		if !shouldStageFile(rel) {
			return nil
		}
		return copyFile(path, filepath.Join(destDir, rel))
	})
}

func shouldStageFile(rel string) bool {
	relForward := filepath.ToSlash(rel)
	switch {
	case rel == ManifestFile:
		return true
	case rel == LockFileName:
		// ailang.lock isn't published, but the smoke needs it to resolve deps
		// from the developer's local registry cache. Without it, packages
		// with [dependencies] cannot import even themselves through pkg/...
		return true
	case rel == "AGENT.md":
		return true
	case rel == SmokeFile:
		return true
	case filepath.Ext(rel) == ".ail":
		return true
	case len(relForward) > len(AssetsDir)+1 && relForward[:len(AssetsDir)+1] == AssetsDir+"/":
		return true
	}
	return false
}

func copyFile(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func truncateOutput(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	keep := s[len(s)-maxBytes:]
	return "[output truncated; showing last " + fmt.Sprintf("%d", maxBytes) + " bytes]\n" + keep
}

// HasExtensionBlock reports whether the manifest declares an [extension] section.
// Extension packages MUST have a smoke test (M-EXT-PORTABILITY-GATE).
func HasExtensionBlock(m *PackageManifest) bool {
	if m == nil {
		return false
	}
	if v, ok := m.Metadata["extension"]; ok && v != nil {
		return true
	}
	return false
}
