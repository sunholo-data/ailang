package pipeline

import (
	"strings"
	"testing"
)

// M-TYPE-NAME-SHADOW regression pack (Daneel bug report, 2026-09-26).
//
// A module's OWN type declaration must win over a same-named type that reaches
// it from another module. Pre-fix, compileFreshModule registered the module's
// local aliases on the type checker first and then re-registered every imported
// alias on top (pipeline_module_compile.go), so the imported body silently
// replaced the local one. The imported set includes every alias of every
// directly imported module (M-TYPE-ALIAS) and, for any module with at least one
// import, every alias of every module already compiled (M-TRANSITIVE-ALIAS-ENV-
// IMPORT) — which is why the report looked "transitive": the victim was compiled
// after an unrelated module that happened to export the same name.
//
// Design: design_docs/planned/v0_44_0/m-type-name-shadow-and-cache.md

// TestTypeNameShadow_DaneelTransitiveShape is the report's graph in four
// modules: main imports functions from links (which exports Row) and reaches
// heart (which exports a DIFFERENT Row) only through brief. heart is compiled
// after links in topo order and has an import of its own, so pre-fix its own
// `-> Row` annotation resolved to links' Row.
func TestTypeNameShadow_DaneelTransitiveShape(t *testing.T) {
	files := map[string]string{
		"links.ail": `module links

export type Row = {date: string, from: string}
export type Seen = {url: string, rows: [Row]}

export pure func seenAll(u: string) -> Seen { {url: u, rows: [{date: "d", from: "f"}]} }
`,
		"heart.ail": `module heart

import std/string (trim)

export type Row = {verb: string, lane: string, notes: string}

export pure func parseRow(line: string) -> Row { {verb: trim(line), lane: "l", notes: "n"} }

export pure func table(line: string) -> string { parseRow(line).verb }
`,
		"brief.ail": `module brief

import heart (table)

export pure func brief(line: string) -> string { table(line) }
`,
		"main.ail": `module main

import links (Seen, seenAll)
import brief (brief)

pure func firstDate(s: Seen) -> string {
  match s.rows {
    [] => "",
    r :: _ => r.date
  }
}

export pure func main() -> string { firstDate(seenAll(brief("x"))) }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("heart's own Row must not be replaced by links' Row: %v", err)
	}
}

// TestTypeNameShadow_DirectImportUnexportedLocal is the smallest form: two
// modules, the importer's Row is not even exported, and it imports only a
// function from a module that exports its own Row.
func TestTypeNameShadow_DirectImportUnexportedLocal(t *testing.T) {
	files := map[string]string{
		"ma.ail": `module ma

export type Row = {date: string}

export pure func mkA(d: string) -> Row { {date: d} }
`,
		"main.ail": `module main

import ma (mkA)

type Row = {verb: string, lane: string}

export pure func mkD(v: string) -> Row { {verb: (mkA(v)).date, lane: "x"} }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("a local Row must shadow an imported module's Row: %v", err)
	}
}

// TestTypeNameShadow_LocalADTNotExpandedByImportedAlias: a local sum type named
// like an imported RECORD alias must stay nominal — the imported alias must not
// expand the local TCon into a record.
func TestTypeNameShadow_LocalADTNotExpandedByImportedAlias(t *testing.T) {
	files := map[string]string{
		"ma.ail": `module ma

export type Row = {date: string}

export pure func mkA(d: string) -> Row { {date: d} }
`,
		"main.ail": `module main

import ma (mkA)

type Row = Empty | Full(string)

export pure func mk(v: string) -> Row {
  if (mkA(v)).date == "" then Empty else Full(v)
}
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("a local ADT Row must not be expanded by an imported record alias Row: %v", err)
	}
}

// TestTypeNameShadow_ImportedAliasStillExpandsWithoutLocal is the non-regression
// control: with NO local Row, the imported Row must still expand (M-TYPE-ALIAS).
func TestTypeNameShadow_ImportedAliasStillExpandsWithoutLocal(t *testing.T) {
	files := map[string]string{
		"ma.ail": `module ma

export type Row = {date: string}

export pure func mkA(d: string) -> Row { {date: d} }
`,
		"main.ail": `module main

import ma (Row, mkA)

export pure func f(v: string) -> Row { {date: (mkA(v)).date} }
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("imported Row must still expand when there is no local Row: %v", err)
	}
}

// TestTypeNameShadow_LocalStillTypeChecks is the other control: shadowing must
// not turn the local alias off — a literal that does not match the LOCAL Row
// is still an error, and the error names the local fields.
func TestTypeNameShadow_LocalStillTypeChecks(t *testing.T) {
	files := map[string]string{
		"ma.ail": `module ma

export type Row = {date: string}

export pure func mkA(d: string) -> Row { {date: d} }
`,
		"main.ail": `module main

import ma (mkA)

type Row = {verb: string, lane: string}

export pure func mkD(v: string) -> Row { {date: (mkA(v)).date} }
`,
	}
	err := checkModules(t, files)
	if err == nil {
		t.Fatal("expected a record mismatch against the LOCAL Row {lane, verb}")
	}
	if !strings.Contains(err.Error(), "lane") || !strings.Contains(err.Error(), "verb") {
		t.Fatalf("error should name the local Row's fields, got: %v", err)
	}
}

// linksModule exports a Row and a Seen whose body refers to that Row by name.
const linksModule = `module links

export type Row = {date: string, from: string}
export type Seen = {url: string, rows: [Row]}

export pure func seenAll(u: string) -> Seen { {url: u, rows: [{date: "d", from: "f"}]} }
`

// TestTypeNameShadow_CapturedImportedAliasIsLoud: interface alias bodies still
// name other types by BARE name (links' Seen = {rows: [Row]}), so in a module
// that declares its own Row, expanding Seen would silently pick up the LOCAL Row.
// Until alias bodies are closed over their defining module (M2), using such an
// alias must fail loudly and name both definitions — never resolve silently.
func TestTypeNameShadow_CapturedImportedAliasIsLoud(t *testing.T) {
	files := map[string]string{
		"links.ail": linksModule,
		"main.ail": `module main

import links (Seen, seenAll)

type Row = {verb: string}

pure func firstDate(s: Seen) -> string {
  match s.rows {
    [] => "",
    r :: _ => r.date
  }
}

pure func mine(v: string) -> Row { {verb: v} }

export pure func main() -> string { firstDate(seenAll(mine("x").verb)) }
`,
	}
	err := checkModules(t, files)
	if err == nil {
		t.Fatal("expected a loud error: links.Seen refers to links.Row, which main's own Row shadows")
	}
	msg := err.Error()
	for _, want := range []string{"Seen", "Row", "links", "main"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error should name %q, got: %v", want, msg)
		}
	}
	if !strings.Contains(msg, "shadow") {
		t.Errorf("error should say the local type shadows the imported one, got: %v", msg)
	}
}

// TestTypeNameShadow_CapturedAliasUnusedIsFine: the capture error fires only
// when the captured alias is actually expanded. A module that declares Row and
// merely has links loaded (here: imports a function whose signature is already
// fully expanded) must still check — this is the Daneel heartbeat position.
func TestTypeNameShadow_CapturedAliasUnusedIsFine(t *testing.T) {
	files := map[string]string{
		"links.ail": linksModule,
		"main.ail": `module main

import links (seenAll)

type Row = {verb: string}

pure func mine(v: string) -> Row { {verb: v} }

export pure func main() -> string {
  match (seenAll(mine("x").verb)).rows {
    [] => "",
    r :: _ => r.date
  }
}
`,
	}
	if err := checkModules(t, files); err != nil {
		t.Fatalf("an unexpanded captured alias must not error: %v", err)
	}
}
