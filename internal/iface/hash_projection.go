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
		// Params is copied defensively: HashProjection never writes to it today, so no
		// test can distinguish this from sharing t's backing array (measured, iteration
		// 311: neutering the copy leaves the whole internal/iface suite green). Kept so a
		// future caller mutating the projection cannot reach back into the input.
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

// SignatureSet implements M1 of m-registry-interface-hash-blind-to-signatures.md:
// it returns stable signatures for exported functions, types, and constructors.
func SignatureSet(j *InterfaceJSON) []string {
	if j == nil {
		return nil
	}

	sigs := make([]string, 0, len(j.Funcs)+len(j.Types))
	for _, f := range j.Funcs {
		effects := append([]string(nil), f.Effects...)
		sort.Strings(effects)
		sigs = append(sigs, fmt.Sprintf("%s:func:%s:%s:%s", j.Module, f.Name, f.Type, strings.Join(effects, ",")))
	}
	for _, t := range j.Types {
		sigs = append(sigs, fmt.Sprintf("%s:type:%s/%d", j.Module, t.Name, len(t.Params)))
		for _, ctor := range t.Ctors {
			sigs = append(sigs, fmt.Sprintf("%s:ctor:%s:%s", j.Module, t.Name, ctor))
		}
	}
	sort.Strings(sigs)
	return sigs
}
