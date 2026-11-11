package errors

// RT010 is raised by `unreachable(msg)` (M-V1-MATCH-EXHAUSTIVENESS §8): the
// program reached a case its author declared impossible. Both engines and
// generated Go raise this one type with the same text.
const RT010 = "RT010"

// UnreachableError is RT010. Msg is the author's own explanation, passed to
// `unreachable` verbatim.
type UnreachableError struct {
	Msg string
}

func (e *UnreachableError) Error() string {
	return RT010 + ": unreachable code reached: " + e.Msg
}

// Code returns the registry code, RT010.
func (e *UnreachableError) Code() string { return RT010 }
