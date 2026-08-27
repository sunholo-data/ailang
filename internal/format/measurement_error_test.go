package format

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/lexer"
	"github.com/sunholo-data/ailang/internal/parser"
)

func TestMeasurementErrorAlwaysAccompaniesRenderError(t *testing.T) {
	const injectedMessage = "injected"
	widths := []int{120, 40, 20}
	roots := []string{"../../examples", "../../std"}
	injections := 0
	pristineRenders := 0

	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			t.Fatalf("corpus root %s is missing: %v", root, err)
		}
		err := filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if info.IsDir() || !strings.HasSuffix(path, ".ail") {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			prs := parser.New(lexer.New(string(data), path))
			prog := prs.Parse()
			if len(prs.Errors()) > 0 || prog == nil || prog.File == nil {
				return nil
			}

			for _, width := range widths {
				p := &printer{w: newWriter("  "), maxWidth: width}
				ferr := p.file(prog.File)
				pristineRenders++
				if p.measurementErr != nil || ferr != nil {
					t.Fatalf("%s width %d: pristine render has measurementErr=%v fileErr=%v", path, width, p.measurementErr, ferr)
				}
			}

			for _, fn := range prog.File.Funcs {
				blk, ok := fn.Body.(*ast.Block)
				if !ok {
					continue
				}
				for _, expr := range blk.Exprs {
					let, ok := expr.(*ast.Let)
					if !ok || let.Body == nil {
						continue
					}

					for _, width := range widths {
						originalValue := let.Value
						let.Value = &ast.Error{Msg: injectedMessage}
						assertInjectedMeasurementError(t, path, fn.Name, "Value", width, prog.File)
						let.Value = originalValue
						injections++

						originalBody := let.Body
						let.Body = &ast.Error{Msg: injectedMessage}
						assertInjectedMeasurementError(t, path, fn.Name, "Body", width, prog.File)
						let.Body = originalBody
						injections++
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}

	if injections < 100 {
		t.Fatalf("only %d injections; want at least 100 to keep the sweep non-vacuous", injections)
	}
	if pristineRenders == 0 {
		t.Fatal("no pristine corpus renders")
	}
	t.Logf("measurement-error sweep: injections=%d pristine-renders=%d", injections, pristineRenders)
}

func assertInjectedMeasurementError(t *testing.T, path, function, field string, width int, file *ast.File) {
	t.Helper()
	p := &printer{w: newWriter("  "), maxWidth: width}
	ferr := p.file(file)
	if p.measurementErr == nil {
		t.Fatalf("%s %s.%s width %d: injected error did not set measurementErr", path, function, field, width)
	}
	if ferr == nil {
		t.Fatalf("%s %s.%s width %d: measurementErr set without p.file error; measured redundancy declaration is refuted", path, function, field, width)
	}
}
