package smt

import (
	"fmt"
	"strings"

	"github.com/sunholo-data/ailang/internal/core"
	"github.com/sunholo-data/ailang/internal/types"
)

type parametricContext struct {
	resultSort string
	adts       map[string][]ADTVariant
	params     map[string][]string
	ti         types.CoreTypeInfo
	programTI  map[*core.Program]types.CoreTypeInfo
}

var activeADTContext *parametricContext

func activateADTContext(adts map[string][]ADTVariant, opts []EncodeFunctionOpts) func() {
	previous := activeADTContext
	if len(opts) > 0 {
		o := opts[0]
		c := &parametricContext{adts: adts, params: o.ADTTypeParams, ti: o.CoreTI, programTI: make(map[*core.Program]types.CoreTypeInfo)}
		if o.Program != nil {
			c.programTI[o.Program] = o.CoreTI
		}
		for path, p := range o.ImportedPrograms {
			c.programTI[p] = o.ImportedCoreTI[path]
		}
		if o.ReturnType != nil {
			c.resultSort, _ = MapType(o.ReturnType)
		}
		activeADTContext = c
	} else {
		activeADTContext = nil
	}
	return func() { activeADTContext = previous }
}

func constructorSymbol(expr core.CoreExpr, name string) (string, error) {
	c := activeADTContext
	if c == nil {
		return name, nil
	}
	var owners []string
	for owner, variants := range c.adts {
		for _, v := range variants {
			if v.Name == name {
				owners = append(owners, owner)
			}
		}
	}
	needsType := len(owners) > 1
	for _, owner := range owners {
		needsType = needsType || len(c.params[owner]) > 0
	}
	if !needsType {
		return name, nil
	}
	typ, ok := c.ti.GetForExpr(expr)
	if !ok {
		return "", fmt.Errorf("%w: constructor %s has no checked node type", ErrUnresolvableTypes, name)
	}
	if fn, ok := typ.(*types.TFunc2); ok {
		typ = fn.Return
	}
	sort, err := MapType(typ)
	if err != nil {
		return "", fmt.Errorf("%w: constructor %s has unsupported/open type: %v", ErrUnresolvableTypes, name, err)
	}
	head := sort
	if strings.HasPrefix(sort, "(") {
		parts := smtListParts(sort)
		if len(parts) > 0 {
			head = parts[0]
		}
	}
	found := false
	for _, owner := range owners {
		if head == owner {
			found = true
		}
	}
	if !found {
		return "", fmt.Errorf("%w: checked constructor %s type %s does not match declaration", ErrUnresolvableTypes, name, sort)
	}
	return fmt.Sprintf("(as %s %s)", name, sort), nil
}

func encodeCheckedConstructor(expr core.CoreExpr, name string, args []core.CoreExpr) (string, error) {
	symbol, err := constructorSymbol(expr, name)
	if err != nil {
		return "", err
	}
	return encodeConstructorApp(symbol, args)
}
