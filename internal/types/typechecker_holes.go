package types

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo-data/ailang/internal/ast"
	"github.com/sunholo-data/ailang/internal/core"
	ailerrors "github.com/sunholo-data/ailang/internal/errors"
	"github.com/sunholo-data/ailang/internal/typedast"
)

// TC_HOLE is the diagnostic code emitted for typed hole expressions (`_?`, `?name`).
const TC_HOLE = "TC_HOLE"

// HoleVarPrefix marks sentinel core.Var nodes lowered from *ast.Hole.
const HoleVarPrefix = "__ailang_hole__:"

type holeState struct {
	holes []recordedHole
}

type recordedHole struct {
	NodeID uint64
	Name   string
	Span   ast.Pos
	Locals []holeLocalBinding
}

type holeLocalBinding struct {
	Name string
	Type Type
}

func parseHoleVarName(name string) (string, bool) {
	if !strings.HasPrefix(name, HoleVarPrefix) {
		return "", false
	}
	return strings.TrimPrefix(name, HoleVarPrefix), true
}

func (tc *CoreTypeChecker) inferHole(ctx *InferenceContext, v *core.Var, holeName string) (*typedast.TypedVar, *TypeEnv, error) {
	tv := ctx.freshType(Star)
	locals := snapshotHoleLocals(ctx.env)
	tc.holes = append(tc.holes, recordedHole{
		NodeID: v.ID(),
		Name:   holeName,
		Span:   v.Span(),
		Locals: locals,
	})
	tc.CoreTI.Set(v.ID(), tv)
	return &typedast.TypedVar{
		TypedExpr: typedast.TypedExpr{
			NodeID:    v.ID(),
			Span:      v.Span(),
			Type:      tv,
			EffectRow: EmptyEffectRow(),
			Core:      v,
		},
		Name: v.Name,
	}, ctx.env, nil
}

// snapshotHoleLocals captures visible non-builtin, non-synthetic bindings in scope.
func snapshotHoleLocals(env *TypeEnv) []holeLocalBinding {
	seen := make(map[string]bool)
	var out []holeLocalBinding
	for cur := env; cur != nil && cur.parent != nil; cur = cur.parent {
		for name, raw := range cur.bindings {
			if seen[name] || name == "_" || strings.HasPrefix(name, "$") || strings.HasPrefix(name, "__") {
				continue
			}
			seen[name] = true
			var t Type
			switch v := raw.(type) {
			case Type:
				t = v
			case *Scheme:
				if v != nil {
					t = v.Type
				}
			}
			if t != nil {
				out = append(out, holeLocalBinding{Name: name, Type: t})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// HoleErrors aggregates multiple TC_HOLE diagnostics from a single module so
// `ailang check --json` / `--format=agent` can report every typed hole in one pass.
type HoleErrors []error

func (he HoleErrors) Error() string {
	parts := make([]string, len(he))
	for i, e := range he {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "\n")
}

func (he HoleErrors) Unwrap() []error { return []error(he) }

// VerifyTypedHoles runs after FinalizeSubstitutions so CoreTI carries the
// fully solved expected type for every typed hole (`_?` / `?name`) in the module.
func (tc *CoreTypeChecker) VerifyTypedHoles() error {
	if len(tc.holes) == 0 {
		return nil
	}
	errs := make(HoleErrors, 0, len(tc.holes))
	for _, h := range tc.holes {
		errs = append(errs, tc.buildHoleError(h))
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return errs
}

func (tc *CoreTypeChecker) buildHoleError(h recordedHole) error {
	expected := "unknown"
	if t, ok := tc.CoreTI.Get(h.NodeID); ok && t != nil {
		expected = displayType(t)
	}
	label := "_?"
	if h.Name != "" {
		label = "?" + h.Name
	}
	msg := fmt.Sprintf("typed hole `%s` has expected type `%s`", label, expected)

	var suggestion string
	if len(h.Locals) > 0 {
		const maxLocals = 8
		parts := make([]string, 0, len(h.Locals))
		for i, b := range h.Locals {
			if i >= maxLocals {
				parts = append(parts, fmt.Sprintf("... (+%d more)", len(h.Locals)-maxLocals))
				break
			}
			t := b.Type
			if len(tc.pendingCoreTISub) > 0 {
				t = ApplySubstitution(tc.pendingCoreTISub, t)
			}
			parts = append(parts, fmt.Sprintf("%s: %s", b.Name, displayType(t)))
		}
		suggestion = "in scope: " + strings.Join(parts, ", ")
	} else {
		suggestion = fmt.Sprintf("replace `%s` with an expression of type `%s`", label, expected)
	}

	span := &ast.Span{Start: h.Span, End: h.Span}
	return &ailerrors.ReportError{
		Rep: &ailerrors.Report{
			Code:    TC_HOLE,
			Message: msg,
			Span:    span,
			Fix:     &ailerrors.Fix{Suggestion: suggestion},
		},
	}
}
