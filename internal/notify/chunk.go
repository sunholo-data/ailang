package notify

import "unicode/utf8"

// chunkMessage splits s into pieces no longer than limit runes, so a channel
// can respect its transport's per-message length cap (Discord 2000, etc.).
// Splitting is rune-safe (never cuts a multi-byte character). A string already
// within the limit (or a non-positive limit) is returned as a single chunk.
// Port of Aitana's channels/_chunk.py::chunk_message.
func chunkMessage(s string, limit int) []string {
	if limit <= 0 || utf8.RuneCountInString(s) <= limit {
		return []string{s}
	}
	runes := []rune(s)
	var chunks []string
	for len(runes) > 0 {
		n := limit
		if n > len(runes) {
			n = len(runes)
		}
		chunks = append(chunks, string(runes[:n]))
		runes = runes[n:]
	}
	return chunks
}
