package emitgo

import (
	"fmt"
	"strings"

	"github.com/sunholo/ailang/internal/gen/stmt"
)

func (e *emitter) emitTypeDecl(td stmt.TypeDecl) {
	switch kind := td.Kind.(type) {
	case stmt.ADTDecl:
		e.emitADT(td.Name, kind)
	case stmt.RecordDecl:
		e.emitRecord(td.Name, kind, td.Exported)
	case stmt.TypeAliasDecl:
		name := exportName(td.Name, td.Exported)
		e.writeLine("type %s = %s", name, kind.Target.GoString())
	}
}

// emitADT generates a discriminated union pattern for an ADT.
//
//	type ColorKind int
//	const (
//	    ColorKindRed ColorKind = iota
//	    ColorKindGreen
//	)
//
//	type Color struct {
//	    Kind ColorKind
//	    Red  *ColorRed
//	    Green *ColorGreen
//	}
//
//	type ColorRed struct { ... }
//	func NewColorRed(...) *Color { ... }
func (e *emitter) emitADT(name string, adt stmt.ADTDecl) {
	if len(adt.Variants) == 0 {
		return
	}

	capName := capitalize(name)
	kindType := capName + "Kind"

	// Kind enum.
	e.writeLine("type %s int", kindType)
	e.writeLine("")
	e.writeLine("const (")
	e.indent++
	for i, v := range adt.Variants {
		constName := capName + "Kind" + capitalize(v.Tag)
		if i == 0 {
			e.writeLine("%s %s = iota", constName, kindType)
		} else {
			e.writeLine("%s", constName)
		}
	}
	e.indent--
	e.writeLine(")")
	e.writeLine("")

	// Variant structs (only for constructors with fields).
	for _, v := range adt.Variants {
		if len(v.Fields) == 0 {
			continue
		}
		structName := capName + capitalize(v.Tag)
		e.writeLine("type %s struct {", structName)
		e.indent++
		for i, f := range v.Fields {
			e.writeLine("Value%d %s", i, f.GoString())
		}
		e.indent--
		e.writeLine("}")
		e.writeLine("")
	}

	// Main union struct.
	e.writeLine("type %s struct {", capName)
	e.indent++
	e.writeLine("Kind %s", kindType)
	for _, v := range adt.Variants {
		if len(v.Fields) > 0 {
			e.writeLine("%s *%s%s", capitalize(v.Tag), capName, capitalize(v.Tag))
		}
	}
	e.indent--
	e.writeLine("}")
	e.writeLine("")

	// Constructor functions.
	for _, v := range adt.Variants {
		ctorName := "New" + capName + capitalize(v.Tag)
		if len(v.Fields) == 0 {
			// Nullary constructor.
			e.writeLine("func %s() *%s {", ctorName, capName)
			e.indent++
			e.writeLine("return &%s{Kind: %sKind%s}", capName, capName, capitalize(v.Tag))
			e.indent--
			e.writeLine("}")
		} else {
			// Constructor with fields.
			var params []string
			for i, f := range v.Fields {
				params = append(params, fmt.Sprintf("field%d %s", i, f.GoString()))
			}
			e.writeLine("func %s(%s) *%s {", ctorName, strings.Join(params, ", "), capName)
			e.indent++
			structName := capName + capitalize(v.Tag)
			var fieldInits []string
			for i := range v.Fields {
				fieldInits = append(fieldInits, fmt.Sprintf("Value%d: field%d", i, i))
			}
			e.writeLine("return &%s{Kind: %sKind%s, %s: &%s{%s}}",
				capName, capName, capitalize(v.Tag), capitalize(v.Tag), structName,
				strings.Join(fieldInits, ", "))
			e.indent--
			e.writeLine("}")
		}
		e.writeLine("")
	}
}

// emitRecord generates a Go struct for a record type.
func (e *emitter) emitRecord(name string, rec stmt.RecordDecl, exported bool) {
	structName := exportName(name, exported)

	e.writeLine("type %s struct {", structName)
	e.indent++
	for _, f := range rec.Fields {
		e.writeLine("%s %s", capitalize(f.Name), f.Type.GoString())
	}
	e.indent--
	e.writeLine("}")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func exportName(name string, exported bool) string {
	if exported {
		return capitalize(name)
	}
	return name
}
