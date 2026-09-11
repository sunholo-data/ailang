//go:build !js

package effects

import (
	"os/exec"
	"strings"
)

// ProcessDenial is why Authorize refused a command. Ctor is the ProcessError
// constructor (NotAllowed | NotFound); Detail is its payload — the command, plus
// the subcommand words that were tried when a subcommand chain was required
// ("git push"), so the caller can see what was refused without a log.
type ProcessDenial struct {
	Ctor   string
	Detail string
}

// Authorize is the single allowlist decision for every Process exec path —
// exec, spawnProcess and asyncExecProcess all call it (M-PROCESS-SUBCMD). It
// returns the path-pinned binary to run, or a denial.
//
// Order: no context / no allowlist → PATH lookup; command not listed →
// NotAllowed(cmd); listed but unresolved at startup → NotFound(cmd); a
// subcommand chain is required and none is a prefix of args → NotAllowed
// ("cmd arg…"). Matching is positional from args[0], so `git -C x status` is
// refused under `git:status` — fail closed rather than parse option grammars.
func (pc *ProcessContext) Authorize(cmdName string, args []string) (string, *ProcessDenial) {
	if pc == nil || !pc.HasAllowlist {
		resolved, err := exec.LookPath(cmdName)
		if err != nil {
			return "", &ProcessDenial{Ctor: "NotFound", Detail: cmdName}
		}
		return resolved, nil
	}

	resolved, allowed := pc.Allowlist[cmdName]
	if !allowed {
		return "", &ProcessDenial{Ctor: "NotAllowed", Detail: cmdName}
	}
	if resolved == "" {
		return "", &ProcessDenial{Ctor: "NotFound", Detail: cmdName}
	}

	chains, narrowed := pc.Subcommands[cmdName]
	if !narrowed {
		return resolved, nil
	}
	longest := 0
	for _, chain := range chains {
		if hasPrefix(args, chain) {
			return resolved, nil
		}
		if len(chain) > longest {
			longest = len(chain)
		}
	}
	return "", &ProcessDenial{Ctor: "NotAllowed", Detail: strings.TrimSpace(cmdName + " " + strings.Join(triedWords(args, longest), " "))}
}

// triedWords is the part of args a subcommand chain was compared against: at
// most the longest chain, with trailing flag tokens dropped so `git push --force`
// is reported as the subcommand it is ("git push"), not as an option string.
func triedWords(args []string, longest int) []string {
	if len(args) > longest {
		args = args[:longest]
	}
	for len(args) > 0 && strings.HasPrefix(args[len(args)-1], "-") {
		args = args[:len(args)-1]
	}
	return args
}

// Error renders the denial the way the Go-error exec paths report it.
func (d *ProcessDenial) Error() string {
	if d.Ctor == "NotFound" {
		return "command not found: " + d.Detail
	}
	return "command not allowed: " + d.Detail
}

func hasPrefix(args, chain []string) bool {
	if len(args) < len(chain) {
		return false
	}
	for i, want := range chain {
		if args[i] != want {
			return false
		}
	}
	return true
}
