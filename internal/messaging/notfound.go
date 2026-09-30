package messaging

import "errors"

// ErrMessageNotFound reports that an inbox message id resolves to nothing.
//
// It exists because the two backends disagreed on how to say "absent", and the
// daemon could not tell absence from a transient failure. SQLite's
// GetInboxMessage returns (nil, nil) for a missing row; Firestore's returned a
// bare fmt.Errorf. A caller that must distinguish "retry me" from "this will
// never resolve" had only a string to go on, so every absence was retried
// forever.
//
// Measured on the laptop 2026-09-30: 360 notifications published 2026-09-24/25
// named inbox docs that were never written to the prod store. The daemon nacked
// each one on every redelivery — 160,947 nacks against 10 acks in six hours —
// which saturated the subscription and starved the real traffic behind it. The
// loop only ever ended because Pub/Sub's 7-day retention dropped the messages.
//
// Backends MUST wrap this for a genuinely absent id, and MUST NOT wrap it for a
// network, permission or deadline failure — those are the retryable ones.
var ErrMessageNotFound = errors.New("inbox message not found")

// IsMessageNotFound reports whether err means "this id resolves to nothing", so
// a caller can ack instead of retrying forever. A nil error is not an absence:
// the (nil, nil) backends signal absence with a nil message, which is the
// caller's job to check.
func IsMessageNotFound(err error) bool {
	return err != nil && errors.Is(err, ErrMessageNotFound)
}
