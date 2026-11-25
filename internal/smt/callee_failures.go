package smt

import (
	"fmt"
	"strings"
)

// activeCalleeFailures records, for each callee the resolver could not define
// during the current EncodeFunction call, why it could not. A caller that
// reaches the callee then fails with that reason appended, rather than the bare
// "not SMT-encodable in this context". Callees are resolved in dependency order,
// so a helper that fails because ITS helper failed carries that reason too, and
// the end of the chain names the construct that actually blocks the proof.
// Its lifetime is activeUserFunctions' (beginUserFunctionScope).
var activeCalleeFailures map[string]string

// activeCalleeOrder holds every callee the resolver collected for the current
// EncodeFunction call, so a call to a helper with no recorded failure can say
// whether the helper was collected at all.
var activeCalleeOrder map[string]bool

// beginUserFunctionScope installs the user-function set and the callee-failure
// record for one EncodeFunction call; the returned func clears both.
func beginUserFunctionScope() func() {
	activeUserFunctions = make(map[string]bool)
	activeCalleeFailures = make(map[string]string)
	activeCalleeOrder = make(map[string]bool)
	return func() { activeUserFunctions, activeCalleeFailures, activeCalleeOrder = nil, nil, nil }
}

// noteCalleeOrder records the callees the resolver is about to define.
func noteCalleeOrder(order []string) {
	if activeCalleeOrder == nil {
		return
	}
	for _, name := range order {
		activeCalleeOrder[name] = true
	}
}

// recordCalleeFailure notes why a callee was not defined. The first reason
// recorded for a name wins.
func recordCalleeFailure(name, reason string) {
	if activeCalleeFailures == nil || reason == "" {
		return
	}
	if _, seen := activeCalleeFailures[name]; !seen {
		activeCalleeFailures[name] = reason
	}
}

// rejectionsReason joins the messages of the rejections that kept a callee
// from being defined.
func rejectionsReason(rs []SMTRejectionReason) string {
	msgs := make([]string, 0, len(rs))
	for _, r := range rs {
		msgs = append(msgs, r.Message)
	}
	return strings.Join(msgs, "; ")
}

// unencodableUserCallError is the leak guard's error: a call to a user function
// that was neither defined nor replaced by its contract.
func unencodableUserCallError(name string) error {
	msg := fmt.Sprintf("calls user function %q that is not SMT-encodable in this context", name)
	switch why := activeCalleeFailures[name]; {
	case why != "":
		msg += ": " + why
	case activeCalleeOrder[name]:
		// Collected, but not defined yet when its caller was encoded: the
		// callees call each other.
		msg += ": it is defined after its caller in dependency order (the helpers are mutually recursive)"
	case activeCalleeOrder != nil:
		msg += ": it was not collected as a callee of the function being verified"
	}
	return fmt.Errorf("%w: %s", ErrUnresolvableTypes, msg)
}

// encodingWrappers are the prefixes the encoder stacks onto an error as it
// unwinds; they say where the failure surfaced, not what it is.
var encodingWrappers = []string{
	ErrUnsupportedConstruct.Error() + ": ",
	ErrUnresolvableTypes.Error() + ": ",
	"cannot encode function body: ",
	"let body: ",
	"let value: ",
}

// failureReason is err's message without the encoder's leading wrappers.
func failureReason(err error) string {
	s := err.Error()
	for stripped := true; stripped; {
		stripped = false
		for _, w := range encodingWrappers {
			if strings.HasPrefix(s, w) {
				s, stripped = s[len(w):], true
			}
		}
	}
	return s
}
