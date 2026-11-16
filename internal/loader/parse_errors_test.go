package loader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/parser"
)

// A module that does not parse returns ParseErrors: the same text as before
// ("parse errors in <path>:" then each error), unwrapping to every
// *parser.ParserError so a caller can read the first one's code.
func TestParseErrors_TextAndUnwrap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.ail")
	if err := os.WriteFile(path, []byte("module bad\ntype float = float\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ml := NewModuleLoader(dir)
	_, err := ml.Load(path)
	if err == nil {
		t.Fatal("a module that does not parse must fail to load")
	}
	var pes *ParseErrors
	if !errors.As(err, &pes) || len(pes.Errs) != 1 {
		t.Fatalf("want *ParseErrors with one error, got %T: %v", err, err)
	}
	var pe *parser.ParserError
	if !errors.As(err, &pe) || pe.Code != "PAR_TYPE_DECL_BUILTIN" || pe.Pos.Line != 2 {
		t.Fatalf("want the PAR_TYPE_DECL_BUILTIN ParserError at line 2, got %v", pe)
	}
	if !strings.Contains(err.Error(), "parse errors in ") || !strings.Contains(err.Error(), "PAR_TYPE_DECL_BUILTIN at ") {
		t.Errorf("text changed: %v", err)
	}
	two := &ParseErrors{Path: "x.ail", Errs: []error{fmt.Errorf("first"), fmt.Errorf("second")}}
	if two.Error() != "parse errors in x.ail:\nfirst\n\nsecond" {
		t.Errorf("multi-error text: %q", two.Error())
	}
}
