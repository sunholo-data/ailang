package types

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/sunholo-data/ailang/internal/ast"
)

// IFCLabel preserves the lattice's constant and variable constituents separately.
// Empty arrays encode bottom; nil arrays are missing metadata, never clean.
type IFCLabel struct {
	Constants []string `json:"constants"`
	Variables []string `json:"variables"`
}

func EncodeIFCLabel(l Label) IFCLabel {
	out := IFCLabel{Constants: []string{}, Variables: []string{}}
	for _, p := range normalisedParts(l) {
		switch v := p.(type) {
		case labelConst:
			out.Constants = append(out.Constants, v.name)
		case labelVar:
			out.Variables = append(out.Variables, v.name)
		}
	}
	sort.Strings(out.Constants)
	sort.Strings(out.Variables)
	return out
}

// Decode reconstructs a validated label without a lossy display-string parser.
func (l IFCLabel) Decode() Label {
	out := LabelBottom()
	for _, name := range l.Constants {
		out = LabelJoin(out, LabelConst(name))
	}
	for _, name := range l.Variables {
		out = LabelJoin(out, LabelVar(name))
	}
	return out
}

type IFCParam struct {
	Name      string   `json:"name"`
	Label     IFCLabel `json:"label"`
	Forbidden string   `json:"forbidden"`
}

// ImportedIFCSig is types-owned so the checker need not import iface.
type ImportedIFCSig struct {
	Params         []IFCParam `json:"params"`
	Return         IFCLabel   `json:"return"`
	HasReturnLabel bool       `json:"has_return_label"`
	Intrinsic      IFCLabel   `json:"intrinsic"`
	Declassify     []string   `json:"declassify"`
}

func EmptyIFCSummary() *ImportedIFCSig {
	return &ImportedIFCSig{Params: []IFCParam{}, Return: EncodeIFCLabel(LabelBottom()), Intrinsic: EncodeIFCLabel(LabelBottom()), Declassify: []string{}}
}

// IsUnlabelled reports whether the summary carries no security information: every
// parameter is bottom with no forbidden sink, there is no declared return label, the
// intrinsic body label is bottom, and there is no Declassify authority. Parameter
// names are not security information. A nil summary is missing metadata, not
// unlabelled, so it reports false.
func (s *ImportedIFCSig) IsUnlabelled() bool {
	if s == nil || s.HasReturnLabel || len(s.Declassify) > 0 {
		return false
	}
	if !s.Return.isBottom() || !s.Intrinsic.isBottom() {
		return false
	}
	for _, p := range s.Params {
		if p.Forbidden != "" || !p.Label.isBottom() {
			return false
		}
	}
	return true
}

func (l IFCLabel) isBottom() bool { return len(l.Constants) == 0 && len(l.Variables) == 0 }

// Validate rejects absent and noncanonical metadata at persistence boundaries.
func (s *ImportedIFCSig) Validate() error {
	if s == nil || s.Params == nil || s.Declassify == nil {
		return fmt.Errorf("missing IFC summary; recompile module from source")
	}
	for _, l := range []IFCLabel{s.Return, s.Intrinsic} {
		if err := l.validate(); err != nil {
			return err
		}
	}
	if !s.HasReturnLabel && !LabelEqual(s.Return.Decode(), LabelBottom()) {
		return fmt.Errorf("IFC return label without declaration")
	}
	for _, p := range s.Params {
		if p.Name == "" {
			return fmt.Errorf("missing IFC parameter name")
		}
		if p.Forbidden == "*" {
			return fmt.Errorf("reserved IFC forbidden label")
		}
		if err := p.Label.validate(); err != nil {
			return err
		}
	}
	if len(s.Declassify) == 1 && s.Declassify[0] == "*" {
		return nil
	}
	return validateIFCNames(s.Declassify)
}

func (l IFCLabel) validate() error {
	if l.Constants == nil || l.Variables == nil {
		return fmt.Errorf("missing IFC label constituents")
	}
	if err := validateIFCNames(l.Constants); err != nil {
		return err
	}
	return validateIFCNames(l.Variables)
}

func validateIFCNames(names []string) error {
	for i, name := range names {
		if name == "" || name == "*" || (i > 0 && names[i-1] >= name) {
			return fmt.Errorf("invalid or noncanonical IFC label list")
		}
	}
	return nil
}

// AnalyzeModuleIFC computes reusable summaries. Imported names must already be
// resolved by the caller; lexical/local declarations take precedence. This API
// does not enable imported call-edge checks (the pipeline wiring is separate).
func AnalyzeModuleIFC(file *ast.File, imports map[string]*ImportedIFCSig) map[string]*ImportedIFCSig {
	out := make(map[string]*ImportedIFCSig)
	if file == nil {
		return out
	}
	c := newIFCChecker(file)
	c.imports = imports
	// Iterate in source order to a fixed point. Keep previous approximations for
	// recursive edges and recompute all bodies; a DFS memo alone is order-dependent.
	for {
		changed := false
		for _, fn := range file.Funcs {
			if fn == nil {
				continue
			}
			old := c.effLabel[fn.Name]
			delete(c.effLabel, fn.Name)
			next := c.effectiveBodyLabel(fn.Name)
			if old == nil || !LabelEqual(old, next) {
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	for _, fn := range file.Funcs {
		if fn == nil {
			continue
		}
		sig := c.sigs[fn.Name]
		s := EmptyIFCSummary()
		for _, p := range sig.params {
			s.Params = append(s.Params, IFCParam{Name: p.name, Label: EncodeIFCLabel(p.label), Forbidden: p.notLabel})
		}
		s.Return = EncodeIFCLabel(sig.returnLabel)
		s.HasReturnLabel = sig.hasReturnLabel
		// Local calls also add deepLabel(return type) through labelOf.
		// Carry that declared component here for consumers without this AST.
		s.Intrinsic = EncodeIFCLabel(LabelJoin(c.effectiveBodyLabel(fn.Name), c.types.deepLabel(fn.ReturnType)))
		s.Declassify = append(s.Declassify, sig.authorizedLabels...)
		sort.Strings(s.Declassify)
		out[fn.Name] = s
	}
	return out
}

func importedIFCResult(s *ImportedIFCSig, args []Label) Label {
	result := s.Return.Decode()
	if len(s.Declassify) > 0 {
		for _, part := range normalisedParts(joinLabels(args)) {
			if unauthorizedLabel(part, result, s.Declassify) != "" {
				result = LabelJoin(result, part)
			}
		}
		return result
	}
	if s.HasReturnLabel {
		return result
	}
	return LabelJoin(s.Intrinsic.Decode(), joinLabels(args))
}

// GobEncode preserves empty arrays (gob ordinarily decodes them as nil).
func (s *ImportedIFCSig) GobEncode() ([]byte, error) { return json.Marshal(s) }
func (s *ImportedIFCSig) GobDecode(data []byte) error {
	return json.Unmarshal(data, s)
}
