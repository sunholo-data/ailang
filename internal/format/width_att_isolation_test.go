package format

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
)

func TestMeasurementIgnoresInheritedAttachments(t *testing.T) {
	fixtures := map[string]string{
		"leading":  "module m\nfunc f() =\n  let a = 1 in\n  -- inner note\n  let b = 2 in\n  b\n",
		"trailing": "module m\nfunc f() =\n  let a = 1 in -- trailing note\n  let b = 2 in\n  b\n",
	}

	for name, src := range fixtures {
		prog := mustParse(t, src)
		env, err := NewEnvelope([]byte(src))
		if err != nil {
			t.Fatalf("%s: NewEnvelope: %v", name, err)
		}
		atts, err := AttachComments(env, prog.File)
		if err != nil {
			t.Fatalf("%s: AttachComments: %v", name, err)
		}
		if len(atts) == 0 {
			t.Fatalf("%s: fixture produced no attachments", name)
		}

		body, ok := prog.File.Funcs[0].Body.(*ast.Block)
		if !ok || len(body.Exprs) == 0 {
			t.Fatalf("%s: function body = %T, want non-empty *ast.Block", name, prog.File.Funcs[0].Body)
		}
		chain := body.Exprs[0]
		idx := newAttachIndex(env, atts)

		reference := (&printer{w: newWriter("  "), maxWidth: 120}).inlineWidth(chain)
		isolated := (&printer{w: newWriter("  "), att: idx, maxWidth: 120}).inlineWidth(chain)

		inheritingPrinter := &printer{
			w:                newWriter("  "),
			att:              idx,
			measuring:        true,
			measurementDepth: 1,
		}
		if err := inheritingPrinter.expr(chain, precLowest); err != nil {
			t.Fatalf("%s: inheriting measurement: %v", name, err)
		}
		line, _, _ := strings.Cut(inheritingPrinter.w.string(), "\n")
		inheriting := utf8.RuneCountInString(line)

		if isolated != reference {
			t.Fatalf("%s: isolated width = %d, reference = %d", name, isolated, reference)
		}
		if inheriting == reference {
			t.Fatalf("%s: fixture no longer carries an observable attachment: inheriting width = reference = %d", name, reference)
		}
	}
}

// This test exists so that widening the attachment boundary set (M2/M3) goes
// red here and forces re-checking m-fmt-measurement-att-isolation-unpinned,
// because nested-interior ownership is the one shape that could defeat the
// exact-owner hasAnyAttachment gate.
func TestNestedInteriorCommentsDoNotAttach(t *testing.T) {
	fixtures := map[string]string{
		"list interior": "module m\nfunc f() = [1,\n  -- inner note\n  2]\n",
		"if interior":   "module m\nfunc f() = if true then\n  -- inner note\n  1\nelse 2\n",
	}

	for name, src := range fixtures {
		prog := mustParse(t, src)
		env, err := NewEnvelope([]byte(src))
		if err != nil {
			t.Fatalf("%s: NewEnvelope: %v", name, err)
		}
		if _, err := AttachComments(env, prog.File); err == nil {
			t.Fatalf("%s: AttachComments unexpectedly accepted a nested-interior comment", name)
		}
	}
}
