package pipeline

import (
	"fmt"
	"sort"
	"strings"

	"github.com/sunholo/ailang/internal/core"
	"github.com/sunholo/ailang/internal/types"
)

// ValidationGap represents a Core node missing from CoreTypeInfo
type ValidationGap struct {
	NodeID   uint64
	ExprKind string // "Lit(Float)", "Intrinsic(OpLe)", "Var", etc.
	Position string // From OriginalSpan() if available
	Hint     string // Actionable suggestion for fixing
	IsStrict bool   // If false, only warn (for unreachable code in future)
}

// ValidateCoreTypeInfo walks the Core AST and verifies every node has a type in CoreTypeInfo.
//
// CONTRACT: This validator runs AFTER type checking and BEFORE lowering.
// Every Core node must have an entry in CoreTypeInfo (type variables are acceptable for
// polymorphic code; this checks presence, not concreteness).
//
// Why this matters:
// - Lowering relies on CoreTypeInfo for type-directed code generation
// - Missing types cause "cannot lower unknown variant" panics with no context
// - This fail-fast approach gives clear diagnostics at compile time
//
// Synthetic nodes: The validator skips compiler-generated nodes (e.g., Prelude builtins,
// injected symbols). To mark a node as synthetic, set node.CoreNode.Flags.Synthetic = true
// during elaboration. (TODO: Add Flags field to CoreNode in future PR)
//
// Strict mode: Currently always enabled. In the future, --strict-coreti flag will elevate
// warnings (unreachable code) to errors. Default is strict in CI.
//
// Returns:
//   - nil if all nodes are typed
//   - ValidationError listing all gaps (NodeID, kind, position, hint)
func ValidateCoreTypeInfo(prog *core.Program, coreTI types.CoreTypeInfo) error {
	validator := &validator{
		coreTI: coreTI,
		gaps:   []ValidationGap{},
		strict: true, // Always strict for now; add flag in future
	}

	// Walk all top-level declarations
	for _, decl := range prog.Decls {
		validator.walkExpr(decl)
	}

	// Return error if gaps found
	if len(validator.gaps) > 0 {
		return validator.formatError()
	}

	return nil
}

// validator holds state during AST walk
type validator struct {
	coreTI types.CoreTypeInfo
	gaps   []ValidationGap
	strict bool
}

// walkExpr recursively walks Core expressions and checks for CoreTypeInfo entries
func (v *validator) walkExpr(expr core.CoreExpr) {
	if expr == nil {
		return
	}

	// Check if this node has a type
	// NOTE: We check for presence, not concreteness. Type variables (α, β) are fine
	// for polymorphic code pre-monomorphization. The key is "has a type," not "concrete type."
	if !v.coreTI.Has(expr.ID()) {
		// TODO: Skip synthetic nodes (when CoreNode.Flags.Synthetic exists)
		// if expr.Flags().Synthetic { return }

		v.recordGap(expr)
	}

	// Recursively walk children
	switch e := expr.(type) {
	case *core.Var:
		// Leaf node - no children

	case *core.VarGlobal:
		// Leaf node - no children

	case *core.Lit:
		// Leaf node - no children

	case *core.Lambda:
		v.walkExpr(e.Body)

	case *core.Let:
		v.walkExpr(e.Value)
		v.walkExpr(e.Body)

	case *core.LetRec:
		for _, binding := range e.Bindings {
			v.walkExpr(binding.Value)
		}
		v.walkExpr(e.Body)

	case *core.App:
		v.walkExpr(e.Func)
		for _, arg := range e.Args {
			v.walkExpr(arg)
		}

	case *core.If:
		v.walkExpr(e.Cond)
		v.walkExpr(e.Then)
		v.walkExpr(e.Else)

	case *core.Match:
		v.walkExpr(e.Scrutinee)
		for _, arm := range e.Arms {
			if arm.Guard != nil {
				v.walkExpr(arm.Guard)
			}
			v.walkExpr(arm.Body)
		}

	case *core.BinOp:
		v.walkExpr(e.Left)
		v.walkExpr(e.Right)

	case *core.UnOp:
		v.walkExpr(e.Operand)

	case *core.Intrinsic:
		// NOTE: CoreTI must be populated with operand types here; lowering
		// relies on these to choose concrete builtins before eval.
		for _, arg := range e.Args {
			v.walkExpr(arg)
		}

	case *core.Record:
		for _, fieldVal := range e.Fields {
			v.walkExpr(fieldVal)
		}

	case *core.RecordAccess:
		v.walkExpr(e.Record)

	case *core.RecordUpdate:
		v.walkExpr(e.Base)
		for _, updateVal := range e.Updates {
			v.walkExpr(updateVal)
		}

	case *core.List:
		for _, elem := range e.Elements {
			v.walkExpr(elem)
		}

	case *core.Array:
		for _, elem := range e.Elements {
			v.walkExpr(elem)
		}

	case *core.Tuple:
		for _, elem := range e.Elements {
			v.walkExpr(elem)
		}

	case *core.DictAbs:
		v.walkExpr(e.Body)

	case *core.DictApp:
		v.walkExpr(e.Dict)
		for _, arg := range e.Args {
			v.walkExpr(arg)
		}

	case *core.DictRef:
		// M-DX19: DictRef is a leaf node representing a dictionary reference
		// It has ClassName and TypeName but no children to walk

	default:
		// Unknown Core node type - this is a compiler bug
		panic(fmt.Sprintf("ValidateCoreTypeInfo: unknown Core expression type %T (NodeID %d)", expr, expr.ID()))
	}
}

// recordGap records a missing CoreTypeInfo entry with context
func (v *validator) recordGap(expr core.CoreExpr) {
	gap := ValidationGap{
		NodeID:   expr.ID(),
		ExprKind: v.exprKind(expr),
		Position: v.formatPosition(expr),
		Hint:     v.getHint(expr),
		IsStrict: v.strict,
	}
	v.gaps = append(v.gaps, gap)
}

// exprKind returns a human-readable kind string for diagnostics
// Examples: "Lit(Float)", "Intrinsic(OpLe)", "Var(x)", "Lambda", "Let"
func (v *validator) exprKind(expr core.CoreExpr) string {
	switch e := expr.(type) {
	case *core.Lit:
		kindStr := map[core.LitKind]string{
			core.IntLit:    "Int",
			core.FloatLit:  "Float",
			core.StringLit: "String",
			core.BoolLit:   "Bool",
			core.UnitLit:   "Unit",
		}
		return fmt.Sprintf("Lit(%s)", kindStr[e.Kind])

	case *core.Intrinsic:
		opStr := map[core.IntrinsicOp]string{
			core.OpAdd: "OpAdd", core.OpSub: "OpSub", core.OpMul: "OpMul",
			core.OpDiv: "OpDiv", core.OpMod: "OpMod",
			core.OpEq: "OpEq", core.OpNe: "OpNe",
			core.OpLt: "OpLt", core.OpLe: "OpLe", core.OpGt: "OpGt", core.OpGe: "OpGe",
			core.OpConcat: "OpConcat", core.OpAnd: "OpAnd", core.OpOr: "OpOr",
			core.OpNot: "OpNot", core.OpNeg: "OpNeg",
		}
		return fmt.Sprintf("Intrinsic(%s)", opStr[e.Op])

	case *core.Var:
		return fmt.Sprintf("Var(%s)", e.Name)

	case *core.VarGlobal:
		return fmt.Sprintf("VarGlobal(%s.%s)", e.Ref.Module, e.Ref.Name)

	case *core.Lambda:
		return fmt.Sprintf("Lambda(%v)", e.Params)

	case *core.Let:
		return fmt.Sprintf("Let(%s)", e.Name)

	case *core.LetRec:
		names := []string{}
		for _, b := range e.Bindings {
			names = append(names, b.Name)
		}
		return fmt.Sprintf("LetRec(%s)", strings.Join(names, ", "))

	case *core.App:
		return "App"

	case *core.If:
		return "If"

	case *core.Match:
		return fmt.Sprintf("Match(%d arms)", len(e.Arms))

	case *core.BinOp:
		return fmt.Sprintf("BinOp(%s)", e.Op)

	case *core.UnOp:
		return fmt.Sprintf("UnOp(%s)", e.Op)

	case *core.Record:
		return fmt.Sprintf("Record(%d fields)", len(e.Fields))

	case *core.RecordAccess:
		return fmt.Sprintf("RecordAccess(.%s)", e.Field)

	case *core.RecordUpdate:
		return fmt.Sprintf("RecordUpdate(%d updates)", len(e.Updates))

	case *core.List:
		return fmt.Sprintf("List(%d elements)", len(e.Elements))

	case *core.Array:
		return fmt.Sprintf("Array(%d elements)", len(e.Elements))

	case *core.Tuple:
		return fmt.Sprintf("Tuple(%d elements)", len(e.Elements))

	case *core.DictAbs:
		return fmt.Sprintf("DictAbs(%d params)", len(e.Params))

	case *core.DictApp:
		return fmt.Sprintf("DictApp(%s.%s)", e.Dict, e.Method)

	default:
		return fmt.Sprintf("%T", expr)
	}
}

// formatPosition returns a human-readable position string
func (v *validator) formatPosition(expr core.CoreExpr) string {
	pos := expr.OriginalSpan()
	if pos.Line == 0 {
		return "(unknown)"
	}
	return fmt.Sprintf("line %d, col %d", pos.Line, pos.Column)
}

// getHint returns an actionable hint based on the expression type
func (v *validator) getHint(expr core.CoreExpr) string {
	switch e := expr.(type) {
	case *core.Lit:
		if e.Kind == core.FloatLit || e.Kind == core.BoolLit {
			return "This usually means defaulting/substitution wasn't applied to CoreTI. Check that ApplySubstitution() was called after type inference."
		}
		return "Literal types should be populated during type checking. Check typechecker_core.go for missing CoreTI.Set() calls."

	case *core.Intrinsic:
		return "Intrinsic operations (comparisons, arithmetic) must have types before lowering. Check that operand types are populated in typechecker_core.go."

	case *core.Let, *core.LetRec:
		return "Nested let bindings must have types populated. Verify that Let/LetRec type inference calls CoreTI.Set() for each binding."

	case *core.Lambda:
		return "Lambda types should be inferred during type checking. Check that lambda type inference populates CoreTI."

	default:
		return "This node should have a type populated during type checking. Look for the corresponding inference site in typechecker_core.go."
	}
}

// formatError returns a formatted error listing all gaps
func (v *validator) formatError() error {
	// Sort gaps by NodeID for stable output
	sort.Slice(v.gaps, func(i, j int) bool {
		return v.gaps[i].NodeID < v.gaps[j].NodeID
	})

	var sb strings.Builder
	sb.WriteString("CoreTypeInfo validation failed: missing type information for Core nodes\n\n")

	// Group gaps by kind for better readability
	kindGroups := make(map[string][]ValidationGap)
	for _, gap := range v.gaps {
		kindGroups[gap.ExprKind] = append(kindGroups[gap.ExprKind], gap)
	}

	// Output grouped gaps
	for kind, gaps := range kindGroups {
		sb.WriteString(fmt.Sprintf("Missing %s types (%d nodes):\n", kind, len(gaps)))
		for _, gap := range gaps {
			sb.WriteString(fmt.Sprintf("  • NodeID %d at %s\n", gap.NodeID, gap.Position))
			sb.WriteString(fmt.Sprintf("    Hint: %s\n", gap.Hint))
		}
		sb.WriteString("\n")
	}

	// Add helpful debugging command
	sb.WriteString("Debug with:\n")
	sb.WriteString("  ailang debug ast <file> --show-types --compact\n\n")

	sb.WriteString("This is a compiler bug. The type checker should populate CoreTypeInfo for all Core nodes.\n")
	sb.WriteString("See: https://ailang.sunholo.com/docs/internals/type-system\n")

	return fmt.Errorf("%s", sb.String())
}
