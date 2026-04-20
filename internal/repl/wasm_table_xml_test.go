package repl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo/ailang/internal/eval"
)

// TestWasmTableXmlExtraction reproduces the docparse-benchmark bug:
// getCellText() returns empty strings for all table cells when running
// through the ModuleRegistry (WASM) code path.
//
// The native CLI evaluator handles this correctly, so the bug is
// specific to the WASM/ModuleRegistry evaluation path.
func TestWasmTableXmlExtraction(t *testing.T) {
	reg := NewModuleRegistry()

	// Load stdlib dependencies (same order as WASM loadEmbeddedStdlib)
	stdlibs := []string{"option", "result", "list", "math", "json", "string", "xml"}
	for _, modName := range stdlibs {
		path := filepath.Join("..", "..", "std", modName+".ail")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read std/%s.ail: %v", modName, err)
		}
		_, err = reg.LoadModule("std/"+modName, string(content))
		if err != nil {
			t.Fatalf("Failed to load std/%s: %v", modName, err)
		}
	}

	// Module that mimics docparse's getCellText pattern
	code := `
module test/table_xml
import std/xml (parse, findAll, getText, getChildren, getTag)
import std/result (Ok, Err)
import std/list (map, length as listLength)
import std/string (join)

-- Same pattern as docparse/services/docx_parser.ail getCellText
pure func getCellText(tc: XmlNode) -> string {
  let paragraphs = findAll(tc, "w:p");
  joinParagraphTexts(paragraphs)
}

pure func joinParagraphTexts(ps: [XmlNode]) -> string =
  join("\n", map(extractParagraphText, ps))

pure func extractParagraphText(p: XmlNode) -> string {
  let children = getChildren(p);
  join("", map(childNodeText, children))
}

pure func childNodeText(node: XmlNode) -> string {
  let tag = getTag(node);
  if tag == "w:r" then extractRunText(node)
  else ""
}

pure func extractRunText(run: XmlNode) -> string {
  let textElems = findAll(run, "w:t");
  join("", map(getText, textElems))
}

-- Test: parse table XML and extract cell texts
export pure func parseTableCells(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      let cells = findAll(root, "w:tc");
      let texts = map(getCellText, cells);
      join("|", texts)
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test: count rows and cells
export pure func countRowsAndCells(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      let rows = findAll(root, "w:tr");
      let cells = findAll(root, "w:tc");
      "${show(listLength(rows))}rows,${show(listLength(cells))}cells"
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test: direct getText on w:t elements
export pure func directTextExtract(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      let textElems = findAll(root, "w:t");
      join("|", map(getText, textElems))
    },
    Err(e) => "ERROR:${e}"
  }
}
`
	_, err := reg.LoadModule("test/table_xml", code)
	if err != nil {
		t.Fatalf("Failed to load test/table_xml: %v", err)
	}

	tableXml := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:tbl>
      <w:tr>
        <w:tc>
          <w:p><w:r><w:t>Cell A1</w:t></w:r></w:p>
        </w:tc>
        <w:tc>
          <w:p><w:r><w:t>Cell B1</w:t></w:r></w:p>
        </w:tc>
      </w:tr>
      <w:tr>
        <w:tc>
          <w:p><w:r><w:t>Cell A2</w:t></w:r></w:p>
        </w:tc>
        <w:tc>
          <w:p><w:r><w:t xml:space="preserve">Cell B2</w:t></w:r></w:p>
        </w:tc>
      </w:tr>
    </w:tbl>
  </w:body>
</w:document>`

	// Test 1: Count rows and cells
	t.Run("countRowsAndCells", func(t *testing.T) {
		result, err := reg.InvokeExport("test/table_xml", "countRowsAndCells", []eval.Value{
			&eval.StringValue{Value: tableXml},
		})
		if err != nil {
			t.Fatalf("countRowsAndCells failed: %v", err)
		}
		sv, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		if sv.Value != "2rows,4cells" {
			t.Errorf("Expected '2rows,4cells', got %q", sv.Value)
		}
	})

	// Test 2: Direct text extraction from w:t elements
	t.Run("directTextExtract", func(t *testing.T) {
		result, err := reg.InvokeExport("test/table_xml", "directTextExtract", []eval.Value{
			&eval.StringValue{Value: tableXml},
		})
		if err != nil {
			t.Fatalf("directTextExtract failed: %v", err)
		}
		sv, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		expected := "Cell A1|Cell B1|Cell A2|Cell B2"
		if sv.Value != expected {
			t.Errorf("Expected %q, got %q", expected, sv.Value)
		}
	})

	// Test 3: findFirst vs findAll discrepancy (THE CORE BUG)
	// findAll(root, "w:body") returns 1 result but findFirst(root, "w:body") returns None
	t.Run("findFirst_vs_findAll", func(t *testing.T) {
		// Add findFirst to the module for this test
		findFirstCode := `
module test/find_compare
import std/xml (parse, findAll, findFirst, getText, getTag)
import std/result (Ok, Err)
import std/option (Some, None)
import std/list (length as listLength)

-- Test findAll for w:body
export pure func findAllBody(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      let results = findAll(root, "w:body");
      "findAll:w:body=${show(listLength(results))}"
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test findFirst for w:body
export pure func findFirstBody(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      match findFirst(root, "w:body") {
        Some(body) => "findFirst:w:body=Some(${getTag(body)})",
        None => "findFirst:w:body=None"
      }
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test findFirst for w:tbl
export pure func findFirstTable(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      match findFirst(root, "w:tbl") {
        Some(tbl) => "findFirst:w:tbl=Some(${getTag(tbl)})",
        None => "findFirst:w:tbl=None"
      }
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test findAll for w:tbl
export pure func findAllTable(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      let results = findAll(root, "w:tbl");
      "findAll:w:tbl=${show(listLength(results))}"
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test findFirst for w:tr
export pure func findFirstRow(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      match findFirst(root, "w:tr") {
        Some(tr) => "findFirst:w:tr=Some(${getTag(tr)})",
        None => "findFirst:w:tr=None"
      }
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test findAll for w:tr
export pure func findAllRows(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      let results = findAll(root, "w:tr");
      "findAll:w:tr=${show(listLength(results))}"
    },
    Err(e) => "ERROR:${e}"
  }
}

-- Test findFirst for w:tc
export pure func findFirstCell(xml: string) -> string {
  match parse(xml) {
    Ok(root) => {
      match findFirst(root, "w:tc") {
        Some(tc) => {
          let textElems = findAll(tc, "w:t");
          match textElems {
            [] => "findFirst:w:tc=Some(empty)",
            t :: _ => "findFirst:w:tc=Some(${getText(t)})"
          }
        },
        None => "findFirst:w:tc=None"
      }
    },
    Err(e) => "ERROR:${e}"
  }
}
`
		reg2 := NewModuleRegistry()
		for _, modName := range stdlibs {
			path := filepath.Join("..", "..", "std", modName+".ail")
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("Failed to read std/%s.ail: %v", modName, err)
			}
			_, err = reg2.LoadModule("std/"+modName, string(content))
			if err != nil {
				t.Fatalf("Failed to load std/%s: %v", modName, err)
			}
		}
		_, err := reg2.LoadModule("test/find_compare", findFirstCode)
		if err != nil {
			t.Fatalf("Failed to load test/find_compare: %v", err)
		}

		tests := []struct {
			name     string
			funcName string
			expected string
		}{
			{"findAll_w:body", "findAllBody", "findAll:w:body=1"},
			{"findFirst_w:body", "findFirstBody", "findFirst:w:body=Some(w:body)"},
			{"findAll_w:tbl", "findAllTable", "findAll:w:tbl=1"},
			{"findFirst_w:tbl", "findFirstTable", "findFirst:w:tbl=Some(w:tbl)"},
			{"findAll_w:tr", "findAllRows", "findAll:w:tr=2"},
			{"findFirst_w:tr", "findFirstRow", "findFirst:w:tr=Some(w:tr)"},
			{"findFirst_w:tc", "findFirstCell", "findFirst:w:tc=Some(Cell A1)"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := reg2.InvokeExport("test/find_compare", tt.funcName, []eval.Value{
					&eval.StringValue{Value: tableXml},
				})
				if err != nil {
					t.Fatalf("%s failed: %v", tt.funcName, err)
				}
				sv, ok := result.(*eval.StringValue)
				if !ok {
					t.Fatalf("Expected StringValue, got %T: %v", result, result)
				}
				if sv.Value != tt.expected {
					t.Errorf("Expected %q, got %q", tt.expected, sv.Value)
				}
			})
		}
	})

	// Test 4: getCellText extraction (the exact pattern that fails in docparse)
	t.Run("parseTableCells", func(t *testing.T) {
		result, err := reg.InvokeExport("test/table_xml", "parseTableCells", []eval.Value{
			&eval.StringValue{Value: tableXml},
		})
		if err != nil {
			t.Fatalf("parseTableCells failed: %v", err)
		}
		sv, ok := result.(*eval.StringValue)
		if !ok {
			t.Fatalf("Expected StringValue, got %T: %v", result, result)
		}
		// Each cell should have text; joined by |
		if !strings.Contains(sv.Value, "Cell A1") {
			t.Errorf("Expected cell text to contain 'Cell A1', got %q", sv.Value)
		}
		if !strings.Contains(sv.Value, "Cell B2") {
			t.Errorf("Expected cell text to contain 'Cell B2', got %q", sv.Value)
		}
		expected := "Cell A1|Cell B1|Cell A2|Cell B2"
		if sv.Value != expected {
			t.Errorf("Expected %q, got %q", expected, sv.Value)
		}
	})
}
