package firestore

import (
	"encoding/json"

	"github.com/sunholo-data/ailang/internal/coordinator"
)

// Finalisation-ledger conversion for the Firestore task mapping
// (M-COMPLETION-PATH-PARITY C1).
//
// The task mapping in coordinator_convert.go is hand-written in both directions
// rather than reflected, so a field absent from either half is silently dropped.
// For the ledger that failure is invisible and total: every redelivery would read
// an empty ledger, conclude nothing had been done, and re-run every effect — the
// exact behaviour the ledger exists to prevent, with no error anywhere.
//
// The ledger is stored as a JSON string rather than a nested map so its shape is
// owned by one place (coordinator.MarshalLedger) and cannot drift between
// backends.

func ledgerToMap(l coordinator.FinalizationLedger) interface{} {
	if len(l) == 0 {
		return nil
	}
	b, err := json.Marshal(l)
	if err != nil {
		// A ledger that cannot be serialised must not be written as a partial
		// value that would later read back as "nothing done".
		return nil
	}
	return string(b)
}

func ledgerFromMap(v interface{}) coordinator.FinalizationLedger {
	s, ok := v.(string)
	if !ok || s == "" {
		return coordinator.FinalizationLedger{}
	}
	l, err := coordinator.UnmarshalLedger(s)
	if err != nil {
		return coordinator.FinalizationLedger{}
	}
	return l
}
