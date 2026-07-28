package types

import (
	"sort"
	"strings"
)

type subsumptionEdge struct {
	Effect   string
	Declared string
	Required string
}

// subsumptionEdges is deliberately opt-in. Defaults do not create edges.
var subsumptionEdges = []subsumptionEdge{
	{Effect: "Rand", Declared: "seeded", Required: "os"},
	{Effect: "Rand", Declared: "crypto", Required: "os"},
}

// ModeSubsumes reports whether declared covers every required mode. The
// pipe-separated required spelling is the validation collector's stable
// representation of multiple requirements for the same effect.
func ModeSubsumes(effect, declared, required string) bool {
	for _, requiredMode := range strings.Split(required, "|") {
		if declared == requiredMode {
			continue
		}
		covered := false
		for _, edge := range subsumptionEdges {
			if edge.Effect == effect && edge.Declared == declared && edge.Required == requiredMode {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

// EffectParamMismatch identifies one validation parameter incompatibility.
type EffectParamMismatch struct {
	Effect        string
	Key           string
	RequiredValue string
	DeclaredValue string
}

// EffectRowDiff distinguishes absent effects from incompatible parameters.
type EffectRowDiff struct {
	Missing         []string
	ParamMismatches []EffectParamMismatch
}

// DiffEffectRows returns a deterministic structured validation difference.
func DiffEffectRows(required, declared *Row) EffectRowDiff {
	var diff EffectRowDiff
	if required == nil {
		return diff
	}
	for effect := range required.Labels {
		if declared == nil {
			diff.Missing = append(diff.Missing, effect)
			continue
		}
		if _, ok := declared.Labels[effect]; !ok {
			diff.Missing = append(diff.Missing, effect)
			continue
		}
		requiredParams := effectiveParamsOf(required, effect)
		declaredParams := effectiveParamsOf(declared, effect)
		keys := make(map[string]struct{}, len(requiredParams)+len(declaredParams))
		for key := range requiredParams {
			keys[key] = struct{}{}
		}
		for key := range declaredParams {
			keys[key] = struct{}{}
		}
		for key := range keys {
			requiredValue, requiredOK := requiredParams[key]
			declaredValue, declaredOK := declaredParams[key]
			compatible := requiredOK == declaredOK && requiredValue == declaredValue
			if key == "mode" && requiredOK && declaredOK {
				compatible = ModeSubsumes(effect, declaredValue, requiredValue)
			}
			if !compatible {
				diff.ParamMismatches = append(diff.ParamMismatches, EffectParamMismatch{
					Effect: effect, Key: key, RequiredValue: requiredValue, DeclaredValue: declaredValue,
				})
			}
		}
	}
	sort.Strings(diff.Missing)
	sort.Slice(diff.ParamMismatches, func(i, j int) bool {
		a, b := diff.ParamMismatches[i], diff.ParamMismatches[j]
		if a.Effect != b.Effect {
			return a.Effect < b.Effect
		}
		return a.Key < b.Key
	})
	return diff
}
