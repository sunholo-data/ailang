// Package gitexec resolves and executes git by absolute path.
package gitexec

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
)

var ErrUnresolvable = errors.New("git is not resolvable to an absolute path")

var (
	cacheMu   sync.Mutex
	cachedGit string
	lookPath  = func() (string, error) { return exec.LookPath("git") }
)

func resolveWith(look func() (string, error)) (string, error) {
	p, err := look()
	if err != nil {
		return "", fmt.Errorf("gitexec: %w: %v", ErrUnresolvable, err)
	}
	if !filepath.IsAbs(p) {
		return "", fmt.Errorf("gitexec: %w: lookup returned %q", ErrUnresolvable, p)
	}
	return p, nil
}

// Path returns a process-wide cached absolute git path. Only success is cached.
func Path() (string, error) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if cachedGit != "" {
		return cachedGit, nil
	}
	p, err := resolveWith(lookPath)
	if err == nil {
		cachedGit = p
	}
	return p, err
}

func commandWith(ctx context.Context, resolve func() (string, error), args ...string) *exec.Cmd {
	p, resolveErr := resolve()
	cmd := exec.CommandContext(ctx, p, args...)
	if resolveErr != nil {
		cmd.Err = resolveErr
	}
	return cmd
}

// Command returns a git command whose resolution failure is deferred through Cmd.Err.
func Command(args ...string) *exec.Cmd {
	return commandWith(context.Background(), Path, args...)
}

// CommandContext is Command with a context.
func CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	return commandWith(ctx, Path, args...)
}
