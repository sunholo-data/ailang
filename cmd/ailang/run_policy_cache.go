package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sunholo-data/ailang/internal/config"
	"github.com/sunholo-data/ailang/internal/policy"
)

// startPolicyWorker allocates only after the caller has set up all pipes
// and registered the worker lifetime. The caller sets cmd.Env (workerEnv with
// no default cache) before runlife.Prepare appends its lifetime metadata, so
// the private cache is APPENDED here rather than replacing the environment;
// start is the lifetime's admission (cmd.Start + AfterStart). Its errors are
// safe to pass to refusePolicy, which exits without defers.
func startPolicyWorker(cmd *exec.Cmd, res *policy.Resolved, start func() error) (string, error) {
	var cache string
	if res.Restricted() && config.CacheDir() == "" {
		var err error
		cache, err = os.MkdirTemp("", "ailang-policy-cache-*")
		if err != nil {
			return "", fmt.Errorf("cannot create worker compile cache: %w", err)
		}
		if res.Root != "" && entryInsideSandbox(res.Root, cache) {
			_ = os.RemoveAll(cache)
			return "", fmt.Errorf("temporary compile cache is inside fs_sandbox; choose a TMPDIR outside the sandbox")
		}
	}
	if cmd.Env == nil {
		cmd.Env = workerEnv(res, "")
	}
	if cache != "" {
		// exec dedups by key, last wins: this overrides an empty operator value.
		cmd.Env = append(cmd.Env, "AILANG_CACHE_DIR="+cache)
	}
	if err := start(); err != nil {
		if cache != "" {
			_ = os.RemoveAll(cache)
		}
		return "", fmt.Errorf("cannot start the worker: %w", err)
	}
	return cache, nil
}

// Resolve the nearest existing ancestor so missing leaves below a symlink
// cannot hide an operator cache inside the sandbox. Entry admission is unchanged.
func cacheInsideSandbox(root, path string) bool {
	if root == "" {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	ancestor := abs
	var missing []string
	for {
		if real, err := filepath.EvalSymlinks(ancestor); err == nil {
			for i := len(missing) - 1; i >= 0; i-- {
				real = filepath.Join(real, missing[i])
			}
			return entryInsideSandbox(root, real)
		}
		parent := filepath.Dir(ancestor)
		if parent == ancestor {
			return entryInsideSandbox(root, abs)
		}
		missing = append(missing, filepath.Base(ancestor))
		ancestor = parent
	}
}
