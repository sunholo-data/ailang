package firestore

import (
	"context"
	"encoding/json"
	"fmt"

	"google.golang.org/api/iterator"

	obs "github.com/sunholo-data/ailang/internal/observatory"
)

// Firestore-side repair for trace/span IDs corrupted by the OTLP/JSON decode
// defect fixed in v0.33.1 (M-OPENROUTER-BROADCAST-INGEST M1).
//
// Why this exists separately from migrate_v18: the deployed observatories run
// AILANG_STORAGE=gcp, so their spans live in Firestore, and the
// internal/observatory migrations only run from the two SQLite paths. migrate_v18
// therefore repairs local and rig observatories and does nothing at all to dev or
// prod. This is the same transform against the other store.
//
// The transform itself is unchanged and deliberately NOT reimplemented — it calls
// obs.RecoverCorruptedID, the function migrate_v18 uses and whose tests are
// anchored to a real production value.
//
// ONE STRUCTURAL DIFFERENCE FROM SQLITE, and it is the risky part: in Firestore a
// span's ID is its DOCUMENT KEY, not a column. A document cannot be renamed, so
// repairing a span id means create-new + delete-old. Both writes go in the SAME
// atomic WriteBatch, so a span is never observed twice nor lost between them.
// trace_id and parent_span_id are ordinary fields and update in place.
//
// Unlike the SQLite migration, this is NOT one transaction end to end — a
// WriteBatch caps at 500 ops, so a large collection commits in chunks. Each chunk
// is atomic; a failure between chunks leaves earlier ones applied. That is safe
// only because the repair is idempotent, and it is: re-running finishes the job.

// SpanIDRepairReport summarizes what a repair pass did, or would do.
type SpanIDRepairReport struct {
	Scanned        int
	TraceIDsFixed  int
	SpanIDsFixed   int
	ParentIDsFixed int
	// Skipped counts documents whose repaired span id already exists — the same
	// span ingested under both encodings. Rewriting would clobber the good
	// document, so these keep their corrupted ids and are reported.
	Skipped int
	// DocsWritten is the number of documents actually modified. Zero on a dry run.
	DocsWritten int
	// Samples holds a few before/after pairs for eyeballing a dry run.
	Samples []string
}

// RepairCorruptedSpanIDs rewrites corrupted trace/span ids in the spans
// collection.
//
// dryRun reports what would change WITHOUT writing. It is the caller's default
// for a reason: this walks a production collection and rewrites primary keys.
func (s *ObservatoryStore) RepairCorruptedSpanIDs(ctx context.Context, dryRun bool) (*SpanIDRepairReport, error) {
	report := &SpanIDRepairReport{}

	// Collect first, mutate second. Rewriting document keys while iterating the
	// same collection would make the iteration order undefined mid-pass.
	type pending struct {
		oldID, newID       string
		data               map[string]any
		renameDoc          bool
		newTrace, newParen string
		setTrace, setParen bool
	}
	var work []pending

	it := s.client.Collection(collObsSpans).Documents(ctx)
	defer it.Stop()
	for {
		doc, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return report, fmt.Errorf("iterate spans: %w", err)
		}
		report.Scanned++

		data := doc.Data()
		p := pending{oldID: doc.Ref.ID, data: data}

		newID, renamed, err := obs.RecoverCorruptedID(doc.Ref.ID, obs.CorrectSpanIDHexLen, obs.CorruptedSpanIDHexLen)
		if err != nil {
			return report, fmt.Errorf("span id %q: %w", doc.Ref.ID, err)
		}
		p.newID, p.renameDoc = newID, renamed

		if raw, ok := data["trace_id"].(string); ok {
			v, changed, err := obs.RecoverCorruptedID(raw, obs.CorrectTraceIDHexLen, obs.CorruptedTraceIDHexLen)
			if err != nil {
				return report, fmt.Errorf("trace id %q: %w", raw, err)
			}
			p.newTrace, p.setTrace = v, changed
		}
		if raw, ok := data["parent_span_id"].(string); ok {
			v, changed, err := obs.RecoverCorruptedID(raw, obs.CorrectSpanIDHexLen, obs.CorruptedSpanIDHexLen)
			if err != nil {
				return report, fmt.Errorf("parent span id %q: %w", raw, err)
			}
			p.newParen, p.setParen = v, changed
		}

		if !p.renameDoc && !p.setTrace && !p.setParen {
			continue
		}
		if p.setTrace {
			report.TraceIDsFixed++
		}
		if p.setParen {
			report.ParentIDsFixed++
		}
		if p.renameDoc {
			report.SpanIDsFixed++
		}
		if len(report.Samples) < 5 {
			report.Samples = append(report.Samples,
				fmt.Sprintf("%s -> %s (trace %s -> %s)", trunc(p.oldID), trunc(p.newID), trunc(fmt.Sprint(data["trace_id"])), trunc(p.newTrace)))
		}
		work = append(work, p)
	}

	if dryRun || len(work) == 0 {
		return report, nil
	}

	// WriteBatch is atomic and capped at 500 operations AND ~11.5 MB of request
	// payload. The operation cap is the obvious limit; the BYTE cap is the one
	// that actually bites here, because these spans carry gen_ai.prompt and
	// gen_ai.completion — whole prompts and whole generated programs. A 400-op
	// batch of them blew the limit on the first real run.
	//
	// So chunk on BOTH, and keep a wide margin: a repaired document is re-sent in
	// full, and the size estimate below is approximate.
	const (
		maxOpsPerBatch   = 50
		maxBytesPerBatch = 4 << 20 // 4 MiB against an ~11.5 MB ceiling
	)
	batch := s.client.Batch()
	ops, batchBytes, pendingDocs := 0, 0, 0

	// commit flushes the batch. DocsWritten is credited ONLY here, after the
	// commit succeeds — crediting it at queue time made a failed run report 190
	// documents written when the atomic batch had in fact written none.
	commit := func() error {
		if ops == 0 {
			return nil
		}
		if _, err := batch.Commit(ctx); err != nil {
			return fmt.Errorf("commit repair batch: %w", err)
		}
		report.DocsWritten += pendingDocs
		batch = s.client.Batch()
		ops, batchBytes, pendingDocs = 0, 0, 0
		return nil
	}

	for _, p := range work {
		size := approxDocSize(p.data)
		if ops >= maxOpsPerBatch || (ops > 0 && batchBytes+size > maxBytesPerBatch) {
			if err := commit(); err != nil {
				return report, err
			}
		}
		batchBytes += size
		if p.setTrace {
			p.data["trace_id"] = p.newTrace
		}
		if p.setParen {
			p.data["parent_span_id"] = p.newParen
		}

		if !p.renameDoc {
			batch = batch.Set(s.client.Doc(collObsSpans, p.oldID), p.data)
			ops++
			pendingDocs++
			continue
		}

		// Renaming: refuse to clobber an existing document. The same span
		// present under both encodings is a duplicate, not a repair target, and
		// deleting either copy is not this pass's call to make.
		if _, err := s.client.Doc(collObsSpans, p.newID).Get(ctx); err == nil {
			report.Skipped++
			report.SpanIDsFixed--
			continue
		}

		p.data["id"] = p.newID
		batch = batch.Create(s.client.Doc(collObsSpans, p.newID), p.data)
		batch = batch.Delete(s.client.Doc(collObsSpans, p.oldID))
		ops += 2
		pendingDocs++
	}

	if err := commit(); err != nil {
		return report, err
	}
	return report, nil
}

// approxDocSize estimates a document's serialized size, so batches can be
// chunked by BYTES and not just by operation count. Firestore's ~11.5 MB request
// cap is the limit that actually bites on span documents carrying whole prompts
// and completions.
//
// Approximate on purpose: it only has to be good enough to keep a wide margin,
// and a JSON round-trip is far cheaper than the write it is protecting.
func approxDocSize(data map[string]any) int {
	b, err := json.Marshal(data)
	if err != nil {
		// Unmeasurable: assume large so it lands in a batch of its own rather
		// than silently counting as free.
		return maxBytesPerDocFallback
	}
	return len(b)
}

// maxBytesPerDocFallback is the pessimistic size assumed for a document whose
// size cannot be measured.
const maxBytesPerDocFallback = 1 << 20

// trunc shortens an id for display without hiding its length.
func trunc(s string) string {
	if len(s) <= 14 {
		return s
	}
	return s[:10] + "…" + fmt.Sprint(len(s))
}
