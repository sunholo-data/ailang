package format

import (
	"strings"
	"unicode/utf8"

	"github.com/sunholo-data/ailang/internal/ast"
)

const defaultMaxWidth = 120

const (
	prefixLetIn            = 0
	prefixEquationBody     = len(" = ")
	prefixTopLevelLetValue = len(" = ")
)

// measurementPrinterHook is a test hook for observing measurement-printer
// construction depth. Tests use it to fail immediately if depth reaches two.
var measurementPrinterHook func(depth int)

func resolveMaxWidth(options Options) int {
	if options.MaxWidth == 0 {
		return defaultMaxWidth
	}
	return options.MaxWidth
}

func (p *printer) newMeasurementPrinter() *printer {
	depth := p.measurementDepth + 1
	if measurementPrinterHook != nil {
		measurementPrinterHook(depth)
	}
	// Measurement deliberately inherits no attachments: inline width is a property
	// of the expression, independent of comments owned by the rendering printer.
	// TestMeasurementIgnoresInheritedAttachments pins that invariant. At product
	// level it is currently double-masked by the hasAnyAttachment(X) ||
	// p.exceedsWidth(X, ...) short-circuits in expr.go:266, decl.go:174, and
	// decl.go:572, and by the fail-closed attachment boundary set. The measured
	// corpus differential saw 88 measurements, 82 with a populated attachment
	// index, and 0 divergences. M2's continuation layout must re-run that
	// differential because it may lift either mask.
	return &printer{
		w:                newWriter(p.w.indent), // depth intentionally remains zero
		att:              nil,
		measuring:        true,
		measurementDepth: depth,
	}
}

func (p *printer) inlineWidth(n ast.Expr) int {
	measurement := p.newMeasurementPrinter()
	if err := measurement.expr(n, precLowest); err != nil {
		p.measurementErr = err
		return 0
	}
	line, _, _ := strings.Cut(measurement.w.string(), "\n")
	return utf8.RuneCountInString(line)
}

func (p *printer) exceedsWidth(n ast.Expr, pending int) bool {
	if p.measuring {
		return false
	}
	width := p.inlineWidth(n)
	if p.measurementErr != nil {
		return false
	}
	return p.w.effectiveCol()+pending+width > p.maxWidth
}
