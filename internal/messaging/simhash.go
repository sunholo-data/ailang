package messaging

import "github.com/sunholo-data/ailang/internal/builtins"

// SearchText builds the text that a message's simhash is computed over.
//
// Exported so every backend derives the hash from the SAME bytes. The SQLite
// insert path built "title + \" \" + payload" inline while the Firestore backend
// built nothing at all, and a divergence here is invisible: it does not fail, it
// just returns similarity scores that mean nothing.
func SearchText(title, payload string) string {
	if payload == "" {
		return title
	}
	return title + " " + payload
}

// ComputeSimhash returns the canonical simhash for a message.
//
// This is the ONLY simhash a message store may use, at write time or at query
// time. The Firestore backend previously carried a private `simhashText` that
// XOR-folded runes into eight shift positions — not a simhash at all, and not
// comparable with builtins.SimHash. Combined with a write path that never
// populated the field, semantic search over the canonical prod store matched
// nothing and reported it as an empty result.
func ComputeSimhash(title, payload string) int64 {
	return builtins.SimHash(SearchText(title, payload))
}
