package types

// Kind represents kinds in the type system
type Kind interface {
	kind()
	String() string
	Equals(Kind) bool
}

// KStar represents the kind of types (* in type theory)
type KStar struct{}

func (k KStar) kind()          {}
func (k KStar) String() string { return "*" }
func (k KStar) Equals(other Kind) bool {
	_, ok := other.(KStar)
	return ok
}

// KRow represents the kind of rows (Row k)
type KRow struct {
	ElemKind Kind
}

func (k KRow) kind()          {}
func (k KRow) String() string { return "Row " + k.ElemKind.String() }
func (k KRow) Equals(other Kind) bool {
	if o, ok := other.(KRow); ok {
		return k.ElemKind.Equals(o.ElemKind)
	}
	return false
}

// KEffect represents the kind of effects
type KEffect struct{}

func (k KEffect) kind()          {}
func (k KEffect) String() string { return "Effect" }
func (k KEffect) Equals(other Kind) bool {
	_, ok := other.(KEffect)
	return ok
}

// KRecord represents the kind of record labels
type KRecord struct{}

func (k KRecord) kind()          {}
func (k KRecord) String() string { return "Record" }
func (k KRecord) Equals(other Kind) bool {
	_, ok := other.(KRecord)
	return ok
}

// Common kinds
var (
	Star      = KStar{}
	Effect    = KEffect{}
	Record    = KRecord{}
	EffectRow = KRow{ElemKind: Effect}
	RecordRow = KRow{ElemKind: Record}
)
