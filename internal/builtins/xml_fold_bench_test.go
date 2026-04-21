package builtins

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/effects"
	"github.com/sunholo-data/ailang/internal/eval"
)

// ============================================================================
// Benchmarks: parseFold vs parseElements
// ============================================================================

func generateLargeXML(count int) string {
	var sb strings.Builder
	sb.WriteString("<sst>")
	for i := 0; i < count; i++ {
		fmt.Fprintf(&sb, `<si><t>shared_string_%d_with_some_extra_text_for_realism</t></si>`, i)
	}
	sb.WriteString("</sst>")
	return sb.String()
}

func BenchmarkParseElements_1K(b *testing.B) {
	benchmarkParseElements(b, 1000)
}

func BenchmarkParseElements_10K(b *testing.B) {
	benchmarkParseElements(b, 10000)
}

func BenchmarkParseFold_1K(b *testing.B) {
	benchmarkParseFold(b, 1000)
}

func BenchmarkParseFold_10K(b *testing.B) {
	benchmarkParseFold(b, 10000)
}

func benchmarkParseElements(b *testing.B, count int) {
	xmlStr := generateLargeXML(count)
	ctx := xmlFoldTestCtx()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := xmlParseElementsImpl(ctx, []eval.Value{
			&eval.StringValue{Value: xmlStr},
			&eval.StringValue{Value: "si"},
			&eval.IntValue{Value: count},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkParseFold(b *testing.B, count int) {
	xmlStr := generateLargeXML(count)
	ctx := xmlFoldTestCtx()
	counter := &eval.BuiltinFunction{
		Name: "counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.IntValue{Value: acc.Value + 1}, nil
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := xmlParseFoldImpl(ctx, []eval.Value{
			&eval.StringValue{Value: xmlStr},
			&eval.StringValue{Value: "si"},
			&eval.IntValue{Value: 0},
			counter,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

// ============================================================================
// Benchmarks: zipXmlScanFold vs readEntry+parseElements
// ============================================================================

func createBenchZip(b *testing.B, count int) string {
	b.Helper()
	dir := b.TempDir()
	path := filepath.Join(dir, "bench.zip")
	f, err := os.Create(path)
	if err != nil {
		b.Fatal(err)
	}
	w := zip.NewWriter(f)
	entry, err := w.Create("xl/sharedStrings.xml")
	if err != nil {
		b.Fatal(err)
	}
	entry.Write([]byte("<sst>"))
	for i := 0; i < count; i++ {
		fmt.Fprintf(entry, `<si><t>shared_string_%d_with_some_extra_text</t></si>`, i)
	}
	entry.Write([]byte("</sst>"))
	w.Close()
	f.Close()
	return path
}

func BenchmarkZipReadEntry_ParseElements_1K(b *testing.B) {
	zipPath := createBenchZip(b, 1000)
	ctx := xmlFoldTestCtx()
	ctx.Grant(effects.NewCapability("FS"))

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Step 1: readEntry (full materialization)
		result, err := zipReadEntryImpl(ctx, []eval.Value{
			&eval.StringValue{Value: zipPath},
			&eval.StringValue{Value: "xl/sharedStrings.xml"},
		})
		if err != nil {
			b.Fatal(err)
		}
		xmlStr := result.(*eval.TaggedValue).Fields[0].(*eval.StringValue)

		// Step 2: parseElements
		_, err = xmlParseElementsImpl(ctx, []eval.Value{
			xmlStr,
			&eval.StringValue{Value: "si"},
			&eval.IntValue{Value: 1000},
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkZipXmlScanFold_1K(b *testing.B) {
	zipPath := createBenchZip(b, 1000)
	ctx := xmlFoldTestCtx()
	ctx.Grant(effects.NewCapability("FS"))
	counter := &eval.BuiltinFunction{
		Name: "counter",
		Fn: func(args []eval.Value) (eval.Value, error) {
			acc := args[0].(*eval.IntValue)
			return &eval.IntValue{Value: acc.Value + 1}, nil
		},
	}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := zipXmlScanFoldImpl(ctx, []eval.Value{
			&eval.StringValue{Value: zipPath},
			&eval.StringValue{Value: "xl/sharedStrings.xml"},
			&eval.StringValue{Value: "si"},
			&eval.IntValue{Value: 0},
			counter,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
