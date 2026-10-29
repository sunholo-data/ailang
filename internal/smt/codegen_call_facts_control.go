package smt

import (
	"github.com/sunholo-data/ailang/internal/core"
)

func encodedCore(value string) core.CoreExpr {
	return &core.VarGlobal{Ref: core.GlobalRef{Module: "$encoded", Name: value}}
}

// Mirror the existing match encoder, preserving its supported pattern/exhaustion
// boundary. Guard facts belong to the first reachable matching pattern, even
// when that guard is false; body facts belong only to the selected arm.
func encodeFactMatch(m *core.Match) (string, error) {
	scrutinee, err := EncodeExpr(m.Scrutinee)
	if err != nil {
		return "", err
	}
	values, facts := *m, *m
	values.Scrutinee = &core.VarGlobal{CoreNode: core.CoreNode{NodeID: m.Scrutinee.ID()}, Ref: core.GlobalRef{Module: "$encoded", Name: scrutinee}}
	facts.Scrutinee = values.Scrutinee
	values.Arms = append([]core.MatchArm(nil), m.Arms...)
	facts.Arms = append([]core.MatchArm(nil), m.Arms...)
	var guards []string
	anyBodyFacts := false
	for i, arm := range m.Arms {
		value, bodyFacts, err := captureCallFacts(func() (string, error) { return encodePatternScoped(arm.Body, arm.Pattern) })
		if err != nil {
			return "", err
		}
		values.Arms[i].Body = encodedCore(value)
		facts.Arms[i].Body = encodedCore(bodyFacts)
		anyBodyFacts = anyBodyFacts || bodyFacts != "true"
		if arm.Guard == nil {
			continue
		}
		guard, guardFacts, err := captureCallFacts(func() (string, error) { return encodePatternScoped(arm.Guard, arm.Pattern) })
		if err != nil {
			return "", err
		}
		values.Arms[i].Guard = encodedCore(guard)
		facts.Arms[i].Guard = encodedCore(guard)
		if guardFacts == "true" {
			continue
		}
		reach := core.Match{Scrutinee: values.Scrutinee}
		for j := 0; j < i; j++ {
			a := values.Arms[j]
			a.Body = encodedCore("true")
			reach.Arms = append(reach.Arms, a)
		}
		reach.Arms = append(reach.Arms, core.MatchArm{Pattern: arm.Pattern, Body: encodedCore(guardFacts)}, core.MatchArm{Pattern: &core.WildcardPattern{}, Body: encodedCore("true")})
		reachedFacts, err := encodeFactMatchMirror(&reach, scrutinee, true)
		if err != nil {
			return "", err
		}
		guards = append(guards, reachedFacts)
	}
	value, err := encodeMatch(&values)
	if err != nil {
		return "", err
	}
	if anyBodyFacts {
		selectedFacts, err := encodeFactMatchMirror(&facts, scrutinee, needsIteMatch(m))
		if err != nil {
			return "", err
		}
		guards = append(guards, selectedFacts)
	}
	addCallFact(conjoinFacts(guards...))
	return value, nil
}

// A guard-free Boolean mirror of a primitive match still needs the original
// ite route: SMT match accepts datatypes only. Preserve typed nested matches.
func encodeFactMatchMirror(m *core.Match, scrutinee string, forceIte bool) (string, error) {
	for _, arm := range m.Arms {
		if hasNestedConstructorPattern(arm.Pattern) {
			return encodeMatch(m)
		}
	}
	if forceIte {
		return encodeMatchAsIte(scrutinee, m.Arms)
	}
	return encodeMatch(m)
}
