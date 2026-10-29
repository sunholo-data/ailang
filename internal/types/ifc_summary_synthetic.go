package types

import "fmt"

// IFCSummaryFromScheme supplies explicit metadata for synthetic exports that
// have no surface body. It reads only the outer type and each immediate label
// wrapper, never recursively traverses types, and is therefore cycle-safe.
func IFCSummaryFromScheme(scheme *Scheme) *ImportedIFCSig {
	s := EmptyIFCSummary()
	if scheme == nil {
		return s
	}
	fn, ok := scheme.Type.(*TFunc2)
	if !ok {
		return s
	}
	for i, typ := range fn.Params {
		label := LabelBottom()
		if lt, ok := typ.(*TLabelled); ok && lt.L != nil {
			label = lt.L
		}
		s.Params = append(s.Params, IFCParam{Name: fmt.Sprintf("arg%d", i), Label: EncodeIFCLabel(label), Forbidden: []string{}, Applied: EncodeIFCLabel(LabelBottom())})
	}
	if lt, ok := fn.Return.(*TLabelled); ok && lt.L != nil {
		s.Return = EncodeIFCLabel(lt.L)
		s.HasReturnLabel = true
	}
	return s
}
