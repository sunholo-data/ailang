package core

import "strings"

// ADTModule is the pseudo-module of constructor factories: a constructor use
// `A(x)` elaborates to App{VarGlobal{$adt.make_T_A}, [x]}.
const ADTModule = "$adt"

// ADTFactoryName names the $adt factory of constructor ctor of the ADT typeName
// declared in module (M-V1-NOMINAL-TYPE-IDENTITY): "make_<T>_<C>@<module>".
// Two modules that each declare `type T = A(...)` used to share the factory key
// make_T_A, so a later import overwrote the earlier one's scheme and the
// evaluator reported an ambiguous constructor for a correct program. An empty
// module gives the legacy unqualified "make_<T>_<C>" (hand-built Core in tests).
func ADTFactoryName(module, typeName, ctor string) string {
	name := "make_" + typeName + "_" + ctor
	if module != "" {
		name += "@" + module
	}
	return name
}

// ADTFactoryKey is the global type-environment key of a factory:
// "$adt." + ADTFactoryName(module, typeName, ctor).
func ADTFactoryKey(module, typeName, ctor string) string {
	return ADTModule + "." + ADTFactoryName(module, typeName, ctor)
}

// SplitADTFactoryModule splits the "@<module>" qualifier off a factory name:
// "make_T_C@q/a" -> ("make_T_C", "q/a"). A name without one is returned
// unchanged with module "". Back ends that only need the ADT and constructor
// names (bytecode lowering, Go codegen, SMT) strip it and keep their own split.
func SplitADTFactoryModule(name string) (base, module string) {
	if i := strings.Index(name, "@"); i >= 0 {
		return name[:i], name[i+1:]
	}
	return name, ""
}

// ParseADTFactoryName splits a factory name into its declaring module (""
// when unqualified), ADT name and constructor name. ok is false when name is
// not of the form make_<T>_<C>[@<module>].
func ParseADTFactoryName(name string) (module, typeName, ctor string, ok bool) {
	if !strings.HasPrefix(name, "make_") {
		return "", "", "", false
	}
	base, module := SplitADTFactoryModule(name)
	rest := base[len("make_"):]
	parts := strings.SplitN(rest, "_", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", false
	}
	return module, parts[0], parts[1], true
}

// IsADTFactory reports whether ref names a constructor factory.
func IsADTFactory(ref GlobalRef) bool {
	if ref.Module != ADTModule {
		return false
	}
	_, _, _, ok := ParseADTFactoryName(ref.Name)
	return ok
}

// ADTFactoryDisplayName is a factory name without its module qualifier, for
// diagnostics and runtime function names: "make_T_C@q/a" -> "make_T_C".
func ADTFactoryDisplayName(name string) string {
	base, _ := SplitADTFactoryModule(name)
	return base
}
