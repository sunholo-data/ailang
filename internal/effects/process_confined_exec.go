//go:build !js

package effects

import (
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
func (pc *ProcessContext) confine(cmdName string, args []string) (path string, argv []string, env []string, denial *ProcessDenial) {
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
	return gitPath, argv, confinedEnv(), nil
}
