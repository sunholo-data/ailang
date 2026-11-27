package runner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/iface"
	"github.com/sunholo-data/ailang/internal/types"
)

// ExpandCapsFlag expands any "auto" token inside a comma-separated --caps flag
// (e.g. "auto" or "auto,Process") against the entrypoint's declared effect row.
//
// Under --caps auto, Process is never granted implicitly; if the entrypoint
// declares Process and the caller did not also pass Process explicitly, this
// returns an actionable error pointing to --caps auto,Process.
func ExpandCapsFlag(moduleIface *iface.Iface, entry, rawCaps string) (resolved string, usedAuto bool, err error) {
	if strings.TrimSpace(rawCaps) == "" {
		return "", false, nil
	}

	explicit := map[string]bool{}
	for _, tok := range strings.Split(rawCaps, ",") {
		name := strings.TrimSpace(tok)
		if name == "" {
			continue
		}
		if name == "auto" {
			usedAuto = true
			continue
		}
		explicit[name] = true
	}
	if !usedAuto {
		return rawCaps, false, nil
	}

	if EntryDeclaresEffect(moduleIface, entry, "Process") && !explicit["Process"] {
		return "", true, fmt.Errorf("auto will not grant Process (it can run any program); pass --caps auto,Process")
	}

	merged := map[string]bool{}
	for _, c := range ResolveAutoCaps(moduleIface, entry) {
		merged[c] = true
	}
	for c := range explicit {
		merged[c] = true
	}
	out := make([]string, 0, len(merged))
	for c := range merged {
		out = append(out, c)
	}
	sort.Strings(out)
	return strings.Join(out, ","), true, nil
}

// CheckMissingEntryCaps performs the static pre-flight capability check before
// executing user code: declared - granted - {Debug}. If any required effect is
// missing, it returns an E_CAPS_MISSING error naming the exact --caps list
// needed so no partial side effects can run.
//
// Unknown capability names in caps are skipped here so GrantCapabilities can
// surface its dedicated typo diagnostic (#1116). Open row variables only count
// their concrete labels.
func CheckMissingEntryCaps(moduleIface *iface.Iface, entry, caps string) error {
	declared := EntryDeclaredEffects(moduleIface, entry)
	if len(declared) == 0 {
		return nil
	}

	granted := map[string]bool{}
	var grantedList []string
	for _, tok := range strings.Split(caps, ",") {
		name := strings.TrimSpace(tok)
		if name == "" {
			continue
		}
		if !types.IsKnownEffect(name) {
			return nil // let GrantCapabilities report unknown capability typo
		}
		if !granted[name] {
			granted[name] = true
			grantedList = append(grantedList, name)
		}
	}
	sort.Strings(grantedList)

	var required []string
	var missing []string
	for _, eff := range declared {
		if eff == "Debug" {
			continue // ghost effect: always granted in NewEffContext / erased in --release
		}
		required = append(required, eff)
		if !granted[eff] {
			missing = append(missing, eff)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	neededSet := map[string]bool{}
	for _, c := range grantedList {
		neededSet[c] = true
	}
	for _, c := range required {
		neededSet[c] = true
	}
	needed := make([]string, 0, len(neededSet))
	for c := range neededSet {
		needed = append(needed, c)
	}
	sort.Strings(needed)

	grantedDisplay := "none"
	if len(grantedList) > 0 {
		grantedDisplay = strings.Join(grantedList, ", ")
	}
	entryName := entry
	if entryName == "" {
		entryName = "main"
	}
	return fmt.Errorf(
		"E_CAPS_MISSING: entrypoint %q declares {%s}; granted {%s}; missing %s. Re-run with --caps %s",
		entryName,
		strings.Join(required, ", "),
		grantedDisplay,
		strings.Join(missing, ", "),
		strings.Join(needed, ","),
	)
}
