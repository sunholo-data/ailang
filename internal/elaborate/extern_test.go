package elaborate

import (
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

// #2390: an `extern func` declaration is EXT004 at the declaration, as a
// structured report, whether or not anything calls it, and also in a file the
// module-less branch would otherwise take (no imports, no other funcs).
func TestExternFuncIsEXT004(t *testing.T) {
	for _, src := range []string{
		"module m\nextern func hostAdd(x: int, y: int) -> int\nexport func f() -> int = 1\n",
		"module m\nextern func hostAdd(x: int, y: int) -> int\nexport func f() -> int = hostAdd(1, 2)\n",
		"module m\nextern pure func hostAdd(x: int) -> int\n",
	} {
		p := parser.New(lexer.New(src, "m.ail"))
		file := p.ParseFile()
		if len(p.Errors()) > 0 {
			t.Fatalf("parse errors: %v", p.Errors())
		}
		_, err := NewElaborator().ElaborateFile(file)
		if err == nil {
			t.Fatalf("%q: elaborated without error", src)
		}
		rep, ok := errors.AsReport(err)
		if !ok || rep.Code != errors.EXT004 || rep.Span == nil || rep.Span.Start.Line != 2 || rep.Span.Start.Column != 1 {
			t.Fatalf("%q: report %+v (ok=%v), want EXT004 at 2:1", src, rep, ok)
		}
		if !strings.Contains(err.Error(), "'hostAdd'") || !strings.Contains(err.Error(), "Go emitter") ||
			rep.Fix == nil || !strings.Contains(rep.Fix.Suggestion, "AILANG func") {
			t.Errorf("%q: message %q / fix %+v", src, err.Error(), rep.Fix)
		}
		if strings.Contains(err.Error(), "#366") {
			t.Errorf("%q: still the misleading #366 text: %v", src, err)
		}
	}
}

// A file with no extern elaborates as before.
func TestNoExternElaborates(t *testing.T) {
	p := parser.New(lexer.New("module m\nexport func hostAdd(x: int, y: int) -> int = x + y\n", "m.ail"))
	file := p.ParseFile()
	if _, err := NewElaborator().ElaborateFile(file); err != nil {
		t.Fatal(err)
	}
}

// The report's span names the file even when the parsed position carries
// none (it falls back to the file's own path).
func TestExternFuncReportNamesTheFile(t *testing.T) {
	p := parser.New(lexer.New("module m\nextern func hostAdd(x: int) -> int\n", ""))
	file := p.ParseFile()
	file.Path = "lib/m.ail"
	_, err := NewElaborator().ElaborateFile(file)
	rep, ok := errors.AsReport(err)
	if !ok || rep.Span == nil || rep.Span.Start.File != "lib/m.ail" || !strings.Contains(err.Error(), "lib/m.ail:2:1") {
		t.Fatalf("report %+v, err %v: want the span in lib/m.ail at 2:1", rep, err)
	}
}
