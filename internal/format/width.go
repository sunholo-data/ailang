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
	// Unlike holeText in interp.go, measurement deliberately inherits no attachments.
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
		panic("format: inline width measurement failed: " + err.Error())
	}
	line, _, _ := strings.Cut(measurement.w.string(), "\n")
	return utf8.RuneCountInString(line)
}

func (p *printer) exceedsWidth(n ast.Expr, pending int) bool {
	if p.measuring {
		return false
	}
	return p.w.col+pending+p.inlineWidth(n) > p.maxWidth
}
