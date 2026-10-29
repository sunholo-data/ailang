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
	Forbidden []string `json:"forbidden"`
	Applied   IFCLabel `json:"applied"` // reserved for future applied-label export semantics
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
	if s == nil || s.Validate() != nil || s.HasReturnLabel || len(s.Declassify) > 0 {
		return false
	}
	if !s.Return.isBottom() || !s.Intrinsic.isBottom() {
		return false
	}
	for _, p := range s.Params {
		if len(p.Forbidden) > 0 || !p.Label.isBottom() || !p.Applied.isBottom() {
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
		if p.Forbidden == nil {
			return fmt.Errorf("missing IFC forbidden label list")
		}
		if err := validateIFCNames(p.Forbidden); err != nil {
			return err
		}
		if err := p.Applied.validate(); err != nil {
			return err
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
func AnalyzeModuleIFC(file *ast.File, imports map[string]*ImportedIFCSig) (out map[string]*ImportedIFCSig, err error) {
	defer func() {
		if p := recover(); p != nil {
			if limit, ok := p.(ifcAnalysisLimit); ok {
				out, err = nil, limit.diagnostic()
			} else {
				panic(p)
			}
		}
	}()
	out = make(map[string]*ImportedIFCSig)
	if file == nil {
		return out, nil
	}
	names := make([]string, 0, len(imports))
	for name := range imports {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := imports[name].Validate(); err != nil {
			return nil, fmt.Errorf("import %s: %w", name, err)
		}
	}
	c := newIFCChecker(file)
	c.imports = imports
	c.exportSummaries = true
	c.convergeExportIntrinsics(file)
	// Returned producer values need the existing module callable-result fixed
	// point. Without any source facts the intrinsic is bottom; avoid expensive
	// symbolic summaries for ordinary unlabelled modules (e.g. std/fs).
	if c.hasExportProducerFacts(file) {
		c.summariseModule(file)
		c.convergeExportIntrinsics(file)
	}

	for _, fn := range file.Funcs {
		if fn == nil {
			continue
		}
		sig := c.sigs[fn.Name]
		s := EmptyIFCSummary()
		for _, p := range sig.params {
			forbidden := []string{}
			if p.hasNot {
				forbidden = append(forbidden, p.notLabel)
			}
			s.Params = append(s.Params, IFCParam{Name: p.name, Label: EncodeIFCLabel(p.label), Forbidden: forbidden, Applied: EncodeIFCLabel(LabelBottom())})
		}
		s.Return = EncodeIFCLabel(sig.returnLabel)
		s.HasReturnLabel = sig.hasReturnLabel
		// Local calls also add deepLabel(return type) through labelOf.
		// Carry that declared component here for consumers without this AST.
		s.Intrinsic = EncodeIFCLabel(LabelJoin(c.effectiveBodyLabel(fn.Name), c.types.deepLabel(fn.ReturnType)))
		s.Declassify = append(s.Declassify, sig.authorizedLabels...)
		sort.Strings(s.Declassify)
		if err := s.Validate(); err != nil {
			return nil, fmt.Errorf("export %s: %w", fn.Name, err)
		}
		out[fn.Name] = s
	}
	return out, nil
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
func (s *ImportedIFCSig) GobEncode() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(s)
}
func (s *ImportedIFCSig) GobDecode(data []byte) error {
	return json.Unmarshal(data, s)
}

// producerCallableResult resolves producer facts at the application of a
// callable value. The export API additionally follows local fnTop alternatives;
// ordinary caller checking retains its existing result policy until M3. Bare
// function storage does not acquire the label of an unapplied producer.
func (c *ifcChecker) producerCallableResult(fn *ifcCallable, args, flows []Label, callee Label, at *ast.FuncCall, env ifcEnv) (Label, bool) {
	if fn == nil {
		return nil, false
	}
	// Container-held/labelled callables already carry information, even when
	// their producer summary returns bottom. Preserve that flow label.
	result := callee
	known := false
	for _, alt := range fn.alts {
		switch {
		case alt.imported != nil:
			known = true
			result = LabelJoin(result, importedIFCResult(alt.imported, args))
		case c.exportSummaries && alt.kind == fnTop && c.sigs[alt.name] != nil:
			known = true
			sig := c.sigs[alt.name]
			passed := append([]Label(nil), args...)
			for i, p := range sig.params {
				if i >= len(args) {
					break
				}
				if sameType(c.staticType(at.Args[i], env), p.typ) {
					passed[i] = flows[i]
				}
			}
			result = LabelJoin(result, c.calleeResultLabel(sig, passed, args))
		default:
			// Unknown alternatives retain the conservative argument join.
			result = LabelJoin(result, joinLabels(args))
		}
	}
	return result, known
}

func (c *ifcChecker) convergeExportIntrinsics(file *ast.File) {
	// Iterate in source order to a fixed point. Keep previous approximations for
	// recursive edges and recompute all bodies; a DFS memo alone is order-dependent.
	for {
		c.analysisStep()
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
}

func (c *ifcChecker) hasExportProducerFacts(file *ast.File) bool {
	for _, fn := range file.Funcs {
		if fn == nil {
			continue
		}
		if !LabelEqual(c.effLabel[fn.Name], LabelBottom()) ||
			!LabelEqual(c.sigs[fn.Name].returnLabel, LabelBottom()) ||
			!LabelEqual(c.types.deepLabel(fn.ReturnType), LabelBottom()) {
			return true
		}
	}
	for _, imported := range c.imports {
		if !imported.Intrinsic.isBottom() || !imported.Return.isBottom() {
			return true
		}
	}
	return false
}
