package sharedmem

import "github.com/sunholo-data/ailang/internal/effects"

func (i *SQLiteSharedIndex) Upsert(namespace, key string, simhash, version, timestamp int64) {
	_ = i.UpsertChecked(namespace, key, simhash, version, timestamp)
}
func (i *SQLiteSharedIndex) Delete(namespace, key string) { _ = i.DeleteChecked(namespace, key) }
func (i *SQLiteSharedIndex) UpsertWithEmbedding(namespace, key string, simhash int64, embedding []float64, version, timestamp int64) {
	_ = i.UpsertWithEmbeddingChecked(namespace, key, simhash, embedding, version, timestamp)
}
func (i *SQLiteSharedIndex) FindSimilarSimHash(namespace string, querySimHash int64, topK, maxScan int, mode effects.DeterminismMode) []effects.SearchResult {
	v, _ := i.FindSimilarSimHashChecked(namespace, querySimHash, topK, maxScan, mode)
	return v
}
func (i *SQLiteSharedIndex) FindSimilarByEmbedding(namespace string, queryEmbedding []float64, topK, maxScan int, mode effects.DeterminismMode) []effects.SearchResult {
	v, _ := i.FindSimilarByEmbeddingChecked(namespace, queryEmbedding, topK, maxScan, mode)
	return v
}
func (i *SQLiteSharedIndex) EntryCount(namespace string) int {
	v, _ := i.EntryCountChecked(namespace)
	return v
}
func (i *SQLiteSharedIndex) Namespaces() []string { v, _ := i.NamespacesChecked(); return v }
