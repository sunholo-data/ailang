package firestore

import (
	"context"
	"fmt"
	"math/bits"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"

	"github.com/sunholo/ailang/internal/messaging"
)

// --- Search & Deduplication ---

// SemanticSearch performs SimHash-based semantic search over inbox messages.
// Firestore doesn't support bitwise operations, so we scan and filter client-side.
// For neural search, the caller should provide an Embedder in opts.
func (s *MessagingStore) SemanticSearch(opts messaging.SearchOptions) ([]messaging.SearchHit, error) {
	ctx := context.Background()
	q := s.client.Collection(collInbox).Query

	if opts.Inbox != "" {
		q = q.Where("to_inbox", "==", opts.Inbox)
	}

	maxScan := opts.MaxScan
	if maxScan <= 0 {
		maxScan = 1000
	}
	q = q.Limit(maxScan)

	iter := q.Documents(ctx)
	defer iter.Stop()

	threshold := opts.Threshold
	if threshold <= 0 {
		threshold = 0.70
	}
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}

	// Compute SimHash of query for comparison
	queryHash := simhashText(opts.Query)

	var hits []messaging.SearchHit
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		m := mapToInbox(doc.Data())

		if m.Simhash == nil {
			continue
		}

		score := simhashSimilarity(queryHash, *m.Simhash)
		if score >= threshold {
			hits = append(hits, messaging.SearchHit{
				Message:   *m,
				Score:     score,
				ScoreKind: "simhash",
			})
		}
	}

	// Sort by score descending
	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })

	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func (s *MessagingStore) FindSimilar(msgID string, threshold float64, limit int) ([]messaging.SearchHit, error) {
	msg, err := s.GetInboxMessage(msgID)
	if err != nil {
		return nil, err
	}
	if msg.Simhash == nil {
		return nil, nil
	}

	ctx := context.Background()
	iter := s.client.Collection(collInbox).
		Limit(1000).
		Documents(ctx)
	defer iter.Stop()

	if threshold <= 0 {
		threshold = 0.70
	}
	if limit <= 0 {
		limit = 20
	}

	var hits []messaging.SearchHit
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		m := mapToInbox(doc.Data())
		if m.ID == msgID || m.Simhash == nil {
			continue
		}

		score := simhashSimilarity(*msg.Simhash, *m.Simhash)
		if score >= threshold {
			hits = append(hits, messaging.SearchHit{
				Message:   *m,
				Score:     score,
				ScoreKind: "simhash",
			})
		}
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > limit {
		hits = hits[:limit]
	}
	return hits, nil
}

func (s *MessagingStore) FindDuplicates(inbox string, threshold float64) ([]messaging.DuplicateGroup, error) {
	ctx := context.Background()
	q := s.client.Collection(collInbox).Query
	if inbox != "" {
		q = q.Where("to_inbox", "==", inbox)
	}
	// Only consider messages that aren't already marked as duplicates
	q = q.Where("dup_of", "==", "")

	iter := q.Documents(ctx)
	defer iter.Stop()

	if threshold <= 0 {
		threshold = 0.85
	}

	var msgs []*messaging.InboxMessage
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		m := mapToInbox(doc.Data())
		if m.Simhash != nil {
			msgs = append(msgs, m)
		}
	}

	// Group by similarity using union-find approach
	used := make(map[int]bool)
	var groups []messaging.DuplicateGroup

	for i := 0; i < len(msgs); i++ {
		if used[i] {
			continue
		}
		group := messaging.DuplicateGroup{
			Representative: *msgs[i],
			MinScore:       1.0,
			ScoreKind:      "simhash",
		}
		for j := i + 1; j < len(msgs); j++ {
			if used[j] {
				continue
			}
			score := simhashSimilarity(*msgs[i].Simhash, *msgs[j].Simhash)
			if score >= threshold {
				group.Duplicates = append(group.Duplicates, *msgs[j])
				if score < group.MinScore {
					group.MinScore = score
				}
				used[j] = true
			}
		}
		if len(group.Duplicates) > 0 {
			groups = append(groups, group)
		}
	}

	return groups, nil
}

func (s *MessagingStore) ApplyDuplicates(groups []messaging.DuplicateGroup, runID string) error {
	ctx := context.Background()
	for _, group := range groups {
		repID := group.Representative.ID
		for _, dup := range group.Duplicates {
			if _, err := s.client.Doc(collInbox, dup.ID).Update(ctx, []firestore.Update{
				{Path: "dup_of", Value: repID},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *MessagingStore) ClearDuplicateMarker(msgID string) error {
	_, err := s.client.Doc(collInbox, msgID).Update(context.Background(), []firestore.Update{
		{Path: "dup_of", Value: ""},
	})
	return err
}

func (s *MessagingStore) UpdateMessageEmbedding(msgID string, embedding []float32, model string) error {
	// Store embedding as JSON string (Firestore doesn't have native float32 array)
	_, err := s.client.Doc(collInbox, msgID).Update(context.Background(), []firestore.Update{
		{Path: "embedding_model", Value: model},
		{Path: "embedding_updated_at", Value: timeNowMillis()},
	})
	return err
}

func (s *MessagingStore) UpdateMessageEnvelope(msgID string, env *messaging.Envelope, overwrite bool) error {
	if env == nil {
		return nil
	}

	doc := s.client.Doc(collInbox, msgID)

	if !overwrite {
		// Merge: read existing envelope first
		snap, err := doc.Get(context.Background())
		if err != nil {
			return fmt.Errorf("get message for envelope merge: %w", err)
		}
		existingJSON, _ := snap.DataAt("envelope")
		if str, ok := existingJSON.(string); ok && str != "" && str != "{}" {
			existing := messaging.EnvelopeFromJSON(str)
			if existing != nil {
				existing.Merge(env)
				env = existing
			}
		}
	}

	envJSON := env.ToJSON()

	_, updateErr := doc.Update(context.Background(), []firestore.Update{
		{Path: "envelope", Value: envJSON},
	})
	return updateErr
}

// --- SimHash helpers ---

// simhashText computes a simple SimHash for text comparison.
func simhashText(text string) int64 {
	var hash int64
	for i, r := range text {
		hash ^= int64(r) << uint(i%8)
	}
	return hash
}

// simhashSimilarity computes similarity between two SimHash values (0.0-1.0).
func simhashSimilarity(a, b int64) float64 {
	xor := uint64(a ^ b)
	distance := bits.OnesCount64(xor)
	return 1.0 - float64(distance)/64.0
}

func timeNowMillis() int64 {
	return timeNow().UnixMilli()
}

// timeNow returns current time (extracted for testing).
func timeNow() time.Time {
	return time.Now()
}
