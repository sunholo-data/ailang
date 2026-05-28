package messaging

import "testing"

func msgWithSlot(title, slot string, vec []float32) InboxMessage {
	return InboxMessage{
		Title: title,
		Envelope: &Envelope{
			Slots: map[string]*EnvelopeVector{
				slot: {Vector: vec, Model: "test", Dimension: len(vec)},
			},
		},
	}
}

func TestClusterMessages_Empty(t *testing.T) {
	if got := ClusterMessages(nil, "intent", 0.75); len(got) != 0 {
		t.Fatalf("expected no clusters for empty input, got %d", len(got))
	}
}

func TestClusterMessages_GroupsSimilar(t *testing.T) {
	// Two parallel vectors (cosine 1.0) + one orthogonal vector.
	msgs := []InboxMessage{
		msgWithSlot("a", "intent", []float32{1, 0}),
		msgWithSlot("b", "intent", []float32{2, 0}), // same direction as a
		msgWithSlot("c", "intent", []float32{0, 1}), // orthogonal
	}
	clusters := ClusterMessages(msgs, "intent", 0.75)
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}
	// First cluster seeds from "a" and absorbs "b".
	if clusters[0].Count != 2 {
		t.Fatalf("expected first cluster size 2, got %d", clusters[0].Count)
	}
	if clusters[0].Label != "a" {
		t.Fatalf("expected first cluster labelled 'a', got %q", clusters[0].Label)
	}
	if clusters[1].Count != 1 {
		t.Fatalf("expected second cluster size 1, got %d", clusters[1].Count)
	}
}

func TestClusterMessages_ThresholdSeparates(t *testing.T) {
	// Same vectors as above, but a threshold above their similarity (1.0 vs 0.0
	// for orthogonal) — identical pair still groups, orthogonal stays apart.
	msgs := []InboxMessage{
		msgWithSlot("a", "intent", []float32{1, 0}),
		msgWithSlot("b", "intent", []float32{0, 1}),
	}
	// threshold 0.5: cosine(a,b)=0 < 0.5 → two singleton clusters.
	if got := ClusterMessages(msgs, "intent", 0.5); len(got) != 2 {
		t.Fatalf("expected 2 singleton clusters below threshold, got %d", len(got))
	}
}

func TestClusterMessages_SkipsMissingSlot(t *testing.T) {
	msgs := []InboxMessage{
		msgWithSlot("a", "intent", []float32{1, 0}),
		{Title: "no-envelope"},                    // nil envelope
		msgWithSlot("c", "code", []float32{1, 0}), // has a different slot only
	}
	clusters := ClusterMessages(msgs, "intent", 0.75)
	if len(clusters) != 1 {
		t.Fatalf("expected 1 cluster (only 'a' has the intent slot), got %d", len(clusters))
	}
	if clusters[0].Label != "a" {
		t.Fatalf("expected cluster from 'a', got %q", clusters[0].Label)
	}
}
