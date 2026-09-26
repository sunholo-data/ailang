package messaging

import "testing"

func TestDecodeMergedPRs_FiltersPrefixAndCarriesMerger(t *testing.T) {
	out := []byte(`[
	 {"number":182,"state":"MERGED","headRefName":"coordinator/task-70a33b83","mergedAt":"2026-09-23T11:45:52Z","mergedBy":{"login":"MarkEdmondson1234"}},
	 {"number":5,"state":"MERGED","headRefName":"feature/coordinator/x","mergedAt":"2026-09-01T00:00:00Z","mergedBy":null}
	]`)
	prs, err := decodeMergedPRs(out, "coordinator/")
	if err != nil {
		t.Fatal(err)
	}
	if len(prs) != 1 {
		t.Fatalf("got %d PRs, want 1 — head: search matches, it does not prefix", len(prs))
	}
	if prs[0].Number != 182 || prs[0].MergedBy != "MarkEdmondson1234" || prs[0].MergedAt == "" {
		t.Fatalf("decoded %+v", prs[0])
	}
	all, _ := decodeMergedPRs(out, "")
	if len(all) != 2 || all[1].MergedBy != "" {
		t.Fatalf("null mergedBy must decode to empty, got %+v", all)
	}
}
