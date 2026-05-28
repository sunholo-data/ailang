package messaging

// Cluster is a group of inbox messages judged semantically similar on a
// single envelope slot. It is the unit produced by ClusterMessages and
// consumed by the `ailang messages triage` CLI and the coordinator's
// triage router.
type Cluster struct {
	Label    string         `json:"label"`
	Slot     string         `json:"slot"`
	Count    int            `json:"count"`
	Messages []InboxMessage `json:"messages"`
}

// ClusterMessages performs greedy threshold-based clustering on messages
// using the named envelope slot. Messages whose envelope lacks the slot are
// skipped (callers handle them separately, e.g. as "Uncategorized").
//
// The algorithm is intentionally simple and deterministic: walk messages in
// order, seed a cluster from the first unassigned message, and absorb any
// later message whose slot vector has cosine similarity >= threshold.
func ClusterMessages(messages []InboxMessage, slot string, threshold float64) []Cluster {
	assigned := make([]bool, len(messages))
	var clusters []Cluster

	for i := 0; i < len(messages); i++ {
		if assigned[i] {
			continue
		}
		assigned[i] = true

		vec := messages[i].Envelope.Get(slot)
		if vec == nil {
			continue
		}

		cluster := Cluster{
			Label:    messages[i].Title,
			Slot:     slot,
			Count:    1,
			Messages: []InboxMessage{messages[i]},
		}

		for j := i + 1; j < len(messages); j++ {
			if assigned[j] {
				continue
			}
			other := messages[j].Envelope.Get(slot)
			if other == nil {
				continue
			}
			if CosineSimilarity(vec.Vector, other.Vector) >= threshold {
				assigned[j] = true
				cluster.Messages = append(cluster.Messages, messages[j])
				cluster.Count++
			}
		}

		clusters = append(clusters, cluster)
	}

	return clusters
}
