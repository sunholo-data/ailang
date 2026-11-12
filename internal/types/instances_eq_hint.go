package types

import (
	"fmt"
	"strings"
)

// The missing-Eq diagnostic for a nominal type (#2232/#2248 review, ruling of
// 2026-11-12: `==` on a type without `deriving (Eq)` is an error in contracts
// as in function bodies). Contracts written for the verifier compared enums
// with == for months while the checker skipped them, so the message must say
// exactly what to change, and where: an AI author fixes from this text alone.

// RecordTypeDecl notes where module declares type name, for diagnostics only.
// The module pipeline records every loaded module's type declarations.
func (env *InstanceEnv) RecordTypeDecl(module, name, site string, record bool) {
	if env == nil || site == "" {
		return
	}
	if env.typeDeclSites == nil {
		env.typeDeclSites = make(map[string]typeDeclSite)
	}
	env.typeDeclSites[NominalKey(module, name)] = typeDeclSite{site: site, record: record}
}

// typeDeclSite is where a nominal type is declared and whether it is a record.
type typeDeclSite struct {
	site   string
	record bool
}

// instanceHint is actionableInstanceHint with this environment's knowledge of
// where each nominal type is declared.
func (env *InstanceEnv) instanceHint(class string, typ Type) string {
	if c, ok := typ.(*TCon); ok && class == "Eq" {
		var decl typeDeclSite
		if env != nil {
			decl = env.typeDeclSites[NominalKey(c.Module, c.Name)]
		}
		return eqTConHint(c, decl)
	}
	return actionableInstanceHint(class, typ)
}

// eqTConHint is the Eq hint for a nominal (or builtin) type constructor; decl
// says where it is declared ("file:line") and whether it is a record, when known.
func eqTConHint(c *TCon, decl typeDeclSite) string {
	prefix := fmt.Sprintf("Equality (==, !=) is not defined on %s.", c.Name)
	if c.Name == "()" {
		return prefix + " () has a single value, so comparing it says nothing: drop the comparison (`result == ()` in an ensures is always true). `deriving (Eq)` does not apply to ()."
	}
	if pkg, foreign := foreignPackage(c.Module); foreign {
		return prefix + fmt.Sprintf(" Type %s is declared in %s (module %s), another package: that package must add `deriving (Eq)` to type %s, and until it does no code here, a contract included, can use == on %s values. Match on its constructors instead.",
			c.Name, pkg, c.Module, c.Name, c.Name)
	}
	where := ""
	switch {
	case decl.site != "":
		where = " (declared at " + decl.site + ")"
	case c.Module != "":
		where = " (declared in module " + c.Module + ")"
	}
	example := fmt.Sprintf("`type %s = A | B deriving (Eq)`", c.Name)
	if decl.record {
		example = fmt.Sprintf("`type %s = {x: int, y: int} deriving (Eq)`", c.Name)
	}
	return prefix + fmt.Sprintf(" Add `deriving (Eq)` to type %s%s, e.g. %s.", c.Name, where, example)
}

// foreignPackage reports whether module belongs to a package this code cannot
// edit (the standard library or an installed registry package) and names it.
func foreignPackage(module string) (string, bool) {
	switch {
	case strings.HasPrefix(module, "std/"):
		return "the standard library", true
	case strings.HasPrefix(module, "pkg/"):
		parts := strings.SplitN(strings.TrimPrefix(module, "pkg/"), "/", 3)
		if len(parts) >= 2 {
			return "package " + parts[0] + "/" + parts[1], true
		}
		return "package " + module, true
	}
	return "", false
}
