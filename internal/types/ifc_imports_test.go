package types_test

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/types"
)

func importedGate() *types.ImportedIFCSig {
	s := types.EmptyIFCSummary()
	s.Params = []types.IFCParam{{Name: "body", Label: types.EncodeIFCLabel(types.LabelBottom()), Forbidden: []string{"email"}, Applied: types.EncodeIFCLabel(types.LabelBottom())}}
	return s
}

// #1134: an imported {not ℓ} parameter is a sink at the caller; the diagnostic
// names the canonical imported callee and the caller's call site.
func TestIFCImportedSinkNamesEdge(t *testing.T) {
	file := parseIFC(t, `module app
func src(raw: string) -> string<email> { raw }
func leak() -> string { gate(src("evil")) }
func ok() -> string { gate("hello") }
`)
	var im types.IFCImports
	im.Add("gate", "lib.gate", importedGate())
	errs := types.CheckModuleIFCWithImports(file, im)
	if len(errs) != 1 || errs[0].Kind != types.SinkRefinementError {
		t.Fatalf("want one sink error, got %v", errs)
	}
	if msg := errs[0].Message; !strings.Contains(msg, `parameter "body" of imported function lib.gate`) || !strings.HasPrefix(errs[0].Position, "ifc_test.ail:3:") {
		t.Fatalf("diagnostic does not name the edge: %s at %s", msg, errs[0].Position)
	}
	if errs := types.CheckModuleIFC(file); len(errs) != 0 {
		t.Fatalf("import-free check must stay per-module: %v", errs)
	}
}

// Missing or malformed imported metadata fails closed: it is never read as clean.
func TestIFCImportedMetadataFailsClosed(t *testing.T) {
	file := parseIFC(t, `module app
func ok() -> string { gate("hello") }
`)
	malformed := importedGate()
	malformed.Params[0].Forbidden = nil
	for name, sig := range map[string]*types.ImportedIFCSig{"missing": nil, "malformed": malformed} {
		var im types.IFCImports
		im.Add("gate", "lib.gate", sig)
		errs := types.CheckModuleIFCWithImports(file, im)
		if len(errs) != 1 || !strings.Contains(errs[0].Message, "imported lib.gate is unusable") {
			t.Fatalf("%s metadata accepted: %v", name, errs)
		}
	}
}
