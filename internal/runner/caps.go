package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// ResolveAutoCaps returns the effect labels on the entry function's declared
// effect row, sorted — what `--caps auto` grants.
func ResolveAutoCaps(moduleIface *iface.Iface, entry string) []string {
	if moduleIface == nil {
		return []string{}
	}
	item, ok := moduleIface.GetExport(entry)
	if !ok || item == nil || item.Type == nil {
		return []string{}
	}
	funcType, ok := item.Type.Type.(*types.TFunc2)
	if !ok || funcType == nil || funcType.EffectRow == nil {
		return []string{}
	}

	caps := make([]string, 0, len(funcType.EffectRow.Labels))
	for name := range funcType.EffectRow.Labels {
		caps = append(caps, name)
	}
	sort.Strings(caps)
	return caps
}

// CapsList is the comma-separated list of capabilities `--caps` accepts, in the
// order the help text shows them. Every `--caps` usage string must be built from
// this constant: four hand-typed copies drifted apart and none named Process,
// so agents grepping `ailang --help` concluded the effect did not exist (#1137).
const CapsList = "IO,FS,Net,Env,Process,Clock,AI,Stream,SharedMem,SharedIndex,Secret,Trace,DOM,Msg,Cog"

// GrantCapabilities parses the --caps string and grants each name to the effect
// context. An unknown name is an error and NOTHING is granted: --caps is a
// security boundary (a script's `IO,FS` is the reason it cannot reach the
// network), and a typo used to run silently with the capability missing (#1116).
func GrantCapabilities(effCtx *effects.EffContext, caps string) error {
	var names []string
	for _, capName := range strings.Split(caps, ",") {
		capName = strings.TrimSpace(capName)
		if capName == "" {
			continue
		}
		if !types.IsKnownEffect(capName) {
			msg := fmt.Sprintf("unknown capability %q in --caps", capName)
			if hint := closestCapability(capName); hint != "" {
				msg += fmt.Sprintf(" (did you mean %s?)", hint)
			}
			return fmt.Errorf("%s; valid: %s", msg, CapsList)
		}
		names = append(names, capName)
	}
	for _, capName := range names {
		effCtx.Grant(effects.NewCapability(capName))
	}
	// M-SECRET-REMOTE-APPROVAL-WIRING: in cloud mode, gate secret() behind a
	// networked human approval. No-op (un-gated) for local runs. Covers every
	// run path, since they all configure capabilities through here.
	return attachCloudSecretApprover(effCtx)
}

// closestCapability suggests a documented capability for a misspelt one: a
// case-insensitive match first, else the nearest by edit distance within 2.
func closestCapability(name string) string {
	best, bestDist := "", 3
	for _, c := range strings.Split(CapsList, ",") {
		if strings.EqualFold(c, name) {
			return c
		}
		if d := editDistance(strings.ToLower(c), strings.ToLower(name)); d < bestDist {
			best, bestDist = c, d
		}
	}
	return best
}

func editDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}
