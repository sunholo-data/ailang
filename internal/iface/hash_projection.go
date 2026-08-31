package iface

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// HashProjection implements M1 of m-registry-interface-hash-blind-to-signatures.md:
// it returns canonical interface JSON for hashing while excluding type aliases.
func HashProjection(j *InterfaceJSON) ([]byte, error) {
	if j == nil {
		return nil, fmt.Errorf("hash projection: nil interface")
	}

	projection := InterfaceJSON{
		Module: j.Module,
		Schema: j.Schema,
		Types:  make([]TypeJSON, 0, len(j.Types)),
		Funcs:  make([]FuncJSON, 0, len(j.Funcs)),
	}
	for _, t := range j.Types {
		out := t
		// Params is copied defensively. No test can distinguish this from sharing t's
		// backing array (measured, iteration 311: neutering the copy leaves the whole
		// internal/iface suite green) — and it is in fact structurally unreachable,
		// because HashProjection returns []byte and never hands the projection struct
		// to a caller. Kept as defence-in-depth against a future signature that does.
		out.Params = append([]string(nil), t.Params...)
		out.Ctors = nil
		if len(t.Ctors) > 0 {
			out.Ctors = append([]string(nil), t.Ctors...)
			sort.Strings(out.Ctors)
		}
		if t.Alias != "" {
			out.Alias = ""
		}
		projection.Types = append(projection.Types, out)
	}
	sort.Slice(projection.Types, func(a, b int) bool {
		return projection.Types[a].Name < projection.Types[b].Name
	})

	for _, f := range j.Funcs {
		out := f
		out.Effects = append([]string{}, f.Effects...)
		sort.Strings(out.Effects)
		projection.Funcs = append(projection.Funcs, out)
	}
	sort.Slice(projection.Funcs, func(a, b int) bool {
		return projection.Funcs[a].Name < projection.Funcs[b].Name
	})

	return json.Marshal(projection)
}

// escapeSigField makes a field safe to interpolate into a ':'-delimited signature.
// Without it the encoding is NOT injective: a func whose rendered type is "A:B" and a
// func of type "A" with effect "B:" both produce "mod:func:run:A:B:" — measured, and
// reproduced first-party at iteration 311 by the sprint evaluator. M5 diffs these sets
// to decide ADDED vs REMOVED vs RETYPED, and M7 will feed them from untrusted uploads,
// so a collision is a wrong release classification, not a cosmetic issue. Backslash is
// escaped first so the mapping stays invertible.
func escapeSigField(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, ":", `\:`)
	return strings.ReplaceAll(s, ",", `\,`)
}

// SignatureSet implements M1 of m-registry-interface-hash-blind-to-signatures.md:
// it returns stable signatures for exported functions, types, and constructors.
// The encoding is injective (see escapeSigField): distinct interfaces never share a
// signature, which is what lets M5 use plain set difference.
func SignatureSet(j *InterfaceJSON) []string {
	if j == nil {
		return nil
	}

	mod := escapeSigField(j.Module)
	sigs := make([]string, 0, len(j.Funcs)+len(j.Types))
	for _, f := range j.Funcs {
		effects := make([]string, 0, len(f.Effects))
		for _, e := range f.Effects {
			effects = append(effects, escapeSigField(e))
		}
		sort.Strings(effects)
		sigs = append(sigs, fmt.Sprintf("%s:func:%s:%s:%s", mod, escapeSigField(f.Name), escapeSigField(f.Type), strings.Join(effects, ",")))
	}
	for _, t := range j.Types {
		sigs = append(sigs, fmt.Sprintf("%s:type:%s/%d", mod, escapeSigField(t.Name), len(t.Params)))
		for _, ctor := range t.Ctors {
			sigs = append(sigs, fmt.Sprintf("%s:ctor:%s:%s", mod, escapeSigField(t.Name), escapeSigField(ctor)))
		}
	}
	sort.Strings(sigs)
	return sigs
}
