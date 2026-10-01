//go:build !js

package effects

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/gitexec"
)

// The confined-git SCHEMAS live in process_confined.go, untagged, because
// internal/policy resolves restricted Process entries against them on every
// platform including wasm. Only this step — turning an admitted call into a
// real invocation — needs the Process runtime (ProcessDenial), which the wasm
// build does not have.

// confine turns an authorized (cmdName, args) into the hardened invocation:
// the absolute git path, the built argv and the environment. Only git has
// a schema today; the allowlist has already admitted the subcommand.
// sandbox is the fs_sandbox the child runs in (its cwd); repository
// discovery is bounded there, and without one there is no invocation.
func (pc *ProcessContext) confine(cmdName string, args []string, sandbox string) (path string, argv []string, env []string, denial *ProcessDenial) {
	if cmdName != "git" && !strings.HasSuffix(cmdName, "/git") {
		return "", nil, nil, &ProcessDenial{Ctor: "NotAllowed", Detail: cmdName + " (restricted mode confines Process to " + strings.Join(ConfinedProcessEntries(), ", ") + ")"}
	}
	argv, err := confineGit(args)
	if err != nil {
		return "", nil, nil, &ProcessDenial{Ctor: "NotAllowed", Detail: "git " + strings.Join(args, " ") + " (" + err.Error() + ")"}
	}
	gitPath, err := gitexec.Path()
	if err != nil {
		return "", nil, nil, &ProcessDenial{Ctor: "NotFound", Detail: "git (" + err.Error() + ")"}
	}
	env, err = confinedEnv(sandbox)
	if err != nil {
		return "", nil, nil, &ProcessDenial{Ctor: "NotAllowed", Detail: "git " + strings.Join(args, " ") + " (" + err.Error() + ")"}
	}
	// Outside a repository `git diff <a> <b>` silently becomes a --no-index
	// diff of two arbitrary paths. With discovery bounded at the sandbox, a
	// repository is found only through <sandbox>/.git (read-only to the
	// program; a bare sandbox is refused by safe.bareRepository), so diff is
	// admitted only when that exists.
	if args[0] == "diff" {
		if _, err := os.Lstat(filepath.Join(sandbox, ".git")); err != nil {
			return "", nil, nil, &ProcessDenial{Ctor: "NotAllowed", Detail: "git " + strings.Join(args, " ") + " (the fs_sandbox is not a repository root, and outside a repository git diff compares arbitrary paths)"}
		}
	}
	return gitPath, argv, env, nil
}
