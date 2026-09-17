package trace

import (
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// Retention policy for the in-memory collector (M-TRACE-TIER-NOT-ENFORCED M2).
//
// `deep` exists to capture everything, and that is worth preserving — but
// unbounded retention made it an OOM: 4934 MB on an 800-iteration accumulator
// loop, because each call recorded the whole accumulator as a string.
//
// Two bounds, addressing different halves:
//
//  1. maxValueBytes caps each rendered value. This is the one that matters
//     asymptotically: it turns a recursion over a growing accumulator from
//     O(n^2) into O(n) while keeping EVERY event, span and parent link. The
//     shape of the execution — which is most of the debugging value — is
//     untouched; only the tail of pathologically large values is elided.
//  2. maxRetainedBytes caps the total held. This is the backstop for a long
//     run, and it evicts the OLDEST events: for a debugger the tail is what
//     matters, since the interesting thing usually happened last.
//
// Both announce themselves. A silently truncated trace is a measurement that
// lies, which for a provenance artifact is worse than no trace at all — so an
// elided value carries its own marker, and eviction inserts a synthetic event
// into the stream so the artifact declares its own discontinuity.
const (
	// DefaultMaxValueBytes bounds one rendered argument or result. Generous:
	// ordinary values are far below it, and only a pathological accumulator hits it.
	DefaultMaxValueBytes = 1024

	// DefaultMaxRetainedBytes bounds the total retained in memory.
	//
	// 32 MB (M-V1-MEMORY-FOOTPRINT D-B, ratified 2026-09-16). It was 256 MB
	// while eventSize charged only string bytes; the real heap for that was
	// 1-2 GB. eventSize is honest now, so the cap means what it says, and
	// exporters read the observer stream — which retention never touches — so
	// nothing that writes a trace to disk sees less.
	DefaultMaxRetainedBytes = 32 << 20 // 32 MB

	// stringHeaderBytes is what one string costs beyond its contents: the
	// pointer+len header held by whichever struct or slice references it.
	stringHeaderBytes = int(unsafe.Sizeof(""))

	// evictionLowWater is the fraction of the cap eviction drains down to.
	// Evicting one event per new event would be O(n) per insert; batching keeps
	// it amortized O(1).
	evictionLowWater = 3 // evict to 3/4 of the cap
)

// ValueMode controls whether rendered VALUES are recorded at all. It is
// deliberately orthogonal to Tier: the tier decides which events exist, this
// decides whether those events carry payloads.
//
// The two are independent because the most useful configuration for a
// confidentiality-bound workload is `deep` + `ValuesRedacted`: the COMPLETE call
// tree — every function, every effect, arity, durations, parentage — with no
// content whatsoever. Most of a trace's audit value is structural. Knowing that
// `readMail -> extractAttachment -> httpPost` ran, in that order, at those
// depths, is what answers "what did it do"; the message body is not needed to
// answer it and is exactly what must not be written down.
type ValueMode int

const (
	// ValuesFull records arguments and results as rendered. The default.
	ValuesFull ValueMode = iota
	// ValuesRedacted replaces every argument and result with a size descriptor.
	//
	// This is a BLUNT control, and for a confidentiality engagement that is a
	// feature rather than a compromise. Label-aware redaction
	// (M-TRACE-LABEL-AWARE) is more precise — it would redact only values the
	// checker flagged — but its guarantee is "the trace contains only values the
	// label checker did not flag", which is worth exactly as much as the
	// checker. ValuesRedacted's guarantee is "the trace contains no values", and
	// it is explicable in one sentence to someone reading a contract.
	ValuesRedacted
)

// SetValueMode selects whether values are recorded. See ValueMode.
func (c *Collector) SetValueMode(m ValueMode) { c.valueMode = m }

// redactValue replaces a rendered value with a size descriptor.
//
// The length is retained deliberately: it costs no confidentiality — a byte
// count is not the content — and it preserves real debugging signal. An empty
// response, a body that grew between retries, a token of an unexpected size are
// all visible without a single byte of payload.
func redactValue(s string) string {
	if s == "" || isRedactedDescriptor(s) {
		return s
	}
	return RedactedDescriptor(len(s))
}

// RedactedDescriptor is the string a redacted value is recorded as. Exposed so
// a render site can produce it from a byte count (eval.RenderedLen) without
// ever rendering the payload.
func RedactedDescriptor(n int) string {
	return fmt.Sprintf("<redacted:%d bytes>", n)
}

// isRedactedDescriptor recognises a value already recorded as a descriptor so
// redactValue is idempotent: a site that pre-redacted must not have its
// descriptor redacted again into "<redacted:19 bytes>".
func isRedactedDescriptor(s string) bool {
	const pre, suf = "<redacted:", " bytes>"
	if !strings.HasPrefix(s, pre) || !strings.HasSuffix(s, suf) {
		return false
	}
	_, err := strconv.Atoi(s[len(pre) : len(s)-len(suf)])
	return err == nil
}

// SetLimits overrides the retention bounds. A value <= 0 disables that bound.
// Exposed for tests and for callers that genuinely want everything.
func (c *Collector) SetLimits(maxValueBytes, maxRetainedBytes int) {
	c.maxValueBytes = maxValueBytes
	c.maxRetainedBytes = maxRetainedBytes
}

// DroppedEvents reports how many events the retention cap evicted. Non-zero
// means the trace is incomplete at the HEAD.
func (c *Collector) DroppedEvents() int { return c.dropped }

// truncateValue bounds one rendered value, marking the elision explicitly so a
// reader can tell a truncated value from a short one.
func truncateValue(s string, max int) string {
	if max <= 0 || len(s) <= max || isBoundedAt(s, max) {
		return s
	}
	return fmt.Sprintf("%s…(+%d bytes elided)", s[:max], len(s)-max)
}

// isBoundedAt reports whether s is already the output of bounding at exactly
// max bytes — a max-byte prefix followed by the elision marker. The check is
// anchored at offset max, so the marker text appearing anywhere else in a
// value is ordinary content. This is what makes truncateValue idempotent for
// values a render site bounded with eval.ShowBounded before recording them.
func isBoundedAt(s string, max int) bool {
	const pre, suf = "…(+", " bytes elided)"
	tail := s[max:]
	if !strings.HasPrefix(tail, pre) || !strings.HasSuffix(tail, suf) {
		return false
	}
	_, err := strconv.Atoi(tail[len(pre) : len(tail)-len(suf)])
	return err == nil
}

// boundValues applies maxValueBytes to every rendered value on an event.
func (c *Collector) boundValues(evt *TraceEvent) {
	if c.valueMode == ValuesRedacted {
		c.redactEventValues(evt)
		return
	}
	if c.maxValueBytes <= 0 {
		return
	}
	if evt.Function != nil {
		for i, a := range evt.Function.Args {
			evt.Function.Args[i] = truncateValue(a, c.maxValueBytes)
		}
		evt.Function.Result = truncateValue(evt.Function.Result, c.maxValueBytes)
	}
	if evt.Effect != nil {
		for i, a := range evt.Effect.Args {
			evt.Effect.Args[i] = truncateValue(a, c.maxValueBytes)
		}
		evt.Effect.Result = truncateValue(evt.Effect.Result, c.maxValueBytes)
	}
}

// eventSize is an event's retained heap cost: the TraceEvent struct itself,
// the payload struct it points at, one header per string field and per args
// element, and the string contents. The previous version charged 128 bytes plus
// contents and nothing else, which under-counted by 4x or more on ordinary
// events — so the 256 MB cap it enforced admitted 1-2 GB of real heap
// (M-V1-MEMORY-FOOTPRINT F3). The ID/timestamp strings on the event are
// counted through the header rule like any other.
func eventSize(evt TraceEvent) int {
	n := int(unsafe.Sizeof(evt))
	n += strBytes(evt.Version) + strBytes(evt.TraceID) + strBytes(evt.SpanID) + strBytes(evt.ParentSpanID)
	if evt.Function != nil {
		n += int(unsafe.Sizeof(*evt.Function))
		n += strBytes(evt.Function.Name) + strBytes(evt.Function.Result) + strsBytes(evt.Function.Args)
	}
	if evt.Effect != nil {
		n += int(unsafe.Sizeof(*evt.Effect))
		n += strBytes(evt.Effect.EffectName) + strBytes(evt.Effect.OpName) + strBytes(evt.Effect.Result) + strsBytes(evt.Effect.Args)
	}
	if evt.Module != nil {
		n += int(unsafe.Sizeof(*evt.Module))
	}
	if evt.Contract != nil {
		n += int(unsafe.Sizeof(*evt.Contract))
	}
	if evt.Budget != nil {
		n += int(unsafe.Sizeof(*evt.Budget))
	}
	if evt.Error != nil {
		n += int(unsafe.Sizeof(*evt.Error))
	}
	if evt.Truncation != nil {
		n += int(unsafe.Sizeof(*evt.Truncation))
	}
	return n
}

// strBytes is a string's content plus the header that references it. Version
// and the like are short constants, but charging them uniformly keeps the
// accounting rule simple enough to trust.
func strBytes(s string) int { return len(s) + stringHeaderBytes }

func strsBytes(ss []string) int {
	n := 0
	for _, s := range ss {
		n += strBytes(s)
	}
	return n
}

// isPinned reports whether an event must survive eviction.
//
// module_start carries the granted capability set — the single most useful line
// for answering "what was this program allowed to do", and the FIRST thing a
// naive oldest-first ring would discard.
func isPinned(evt TraceEvent) bool {
	return evt.Event == EventModuleStart
}

// record is the single point where an event enters the collector.
//
// Order matters: observers are notified BEFORE retention is applied, so a
// streaming consumer (Collector.OnEvent) receives the COMPLETE event stream even
// when the in-memory cap is evicting. That is what lets `deep` genuinely keep
// everything — on disk, where the constraint is not RAM.
func (c *Collector) record(evt TraceEvent) {
	c.boundValues(&evt)
	c.notify(evt)

	c.events = append(c.events, evt)
	c.retainedBytes += eventSize(evt)
	c.evictIfNeeded()
}

// evictIfNeeded drops the oldest unpinned events until the retained total is
// back under the low-water mark, and records the discontinuity in the stream.
func (c *Collector) evictIfNeeded() {
	if c.maxRetainedBytes <= 0 || c.retainedBytes <= c.maxRetainedBytes {
		return
	}
	target := c.maxRetainedBytes / 4 * evictionLowWater

	keepFrom := 0
	for keepFrom < len(c.events) && c.retainedBytes > target {
		if isPinned(c.events[keepFrom]) {
			keepFrom++
			continue
		}
		c.retainedBytes -= eventSize(c.events[keepFrom])
		c.dropped++
		keepFrom++
	}
	if keepFrom == 0 {
		return
	}

	// Compact, preserving pinned events at the front so the capability set and
	// module identity survive however long the run goes on.
	kept := make([]TraceEvent, 0, len(c.events)-keepFrom+2)
	for _, e := range c.events[:keepFrom] {
		if isPinned(e) {
			kept = append(kept, e)
		}
	}
	kept = append(kept, c.truncationMarker())
	kept = append(kept, c.events[keepFrom:]...)
	c.events = kept
}

// truncationMarker is the synthetic event that makes the artifact declare its
// own discontinuity, rather than looking like a complete trace that merely
// starts late.
func (c *Collector) truncationMarker() TraceEvent {
	return TraceEvent{
		Version:     traceVersion,
		Event:       EventTraceTruncated,
		TimestampNS: c.nowNS(),
		TraceID:     c.traceID,
		Truncation: &TruncationEvent{
			DroppedEvents:    c.dropped,
			MaxRetainedBytes: c.maxRetainedBytes,
			Reason: fmt.Sprintf(
				"retention cap of %d bytes reached; %d oldest events evicted. "+
					"The tail is retained. Raise the cap, or attach an exporter to stream the full trace.",
				c.maxRetainedBytes, c.dropped),
		},
	}
}

// redactEventValues strips payloads from every value-bearing field.
//
// Applied at the collector, not at the call sites, so no recorder can bypass it
// — the guarantee has to hold for paths added later, not just the ones that
// exist today.
func (c *Collector) redactEventValues(evt *TraceEvent) {
	if evt.Function != nil {
		for i, a := range evt.Function.Args {
			evt.Function.Args[i] = redactValue(a)
		}
		evt.Function.Result = redactValue(evt.Function.Result)
	}
	if evt.Effect != nil {
		for i, a := range evt.Effect.Args {
			evt.Effect.Args[i] = redactValue(a)
		}
		evt.Effect.Result = redactValue(evt.Effect.Result)
	}
}
