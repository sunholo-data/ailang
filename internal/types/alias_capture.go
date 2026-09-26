package types

import "errors"

// M-TYPE-NAME-SHADOW: captured imported aliases.
//
// Interface alias bodies name other types by BARE name: links' `type Seen =
// {rows: [Row]}` crosses the module boundary as TRecord{rows: [TCon "Row"]}.
// In a module that declares its own `Row`, expanding that body would resolve
// "Row" to the LOCAL Row — a silent wrong type. Until alias bodies are closed
// over their defining module (M2 of m-type-name-shadow-and-cache.md), the
// pipeline withholds such aliases from the alias env and registers them here
// with a message naming both definitions; expanding one is a loud error.
//
// Using a captured alias is the only thing that errors. A module that merely
// has the alias in reach (every alias of every compiled module is) is fine.

// RegisterAliasCapture marks an imported alias as captured: expanding `name`
// during unification fails with `msg` instead of resolving to a wrong type.
func (tc *CoreTypeChecker) RegisterAliasCapture(name, msg string) {
	if tc.aliasCaptures == nil {
		tc.aliasCaptures = make(map[string]string)
	}
	tc.aliasCaptures[name] = msg
}

// SetAliasCaptures installs the captured-alias messages on a unifier.
func (u *Unifier) SetAliasCaptures(captures map[string]string) {
	u.aliasCaptures = captures
}

// latchAliasCapture latches the captured-alias error for `name`, if any, and
// reports whether it did. Surfaced by Unify through the same latch as the
// alias-arity diagnostic (expandAlias returns Type and cannot return an error).
func (u *Unifier) latchAliasCapture(name string) bool {
	msg, captured := u.aliasCaptures[name]
	if !captured {
		return false
	}
	u.aliasArityErr = errors.New(msg)
	return true
}
