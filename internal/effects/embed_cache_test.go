package effects

import (
	"errors"
	"sync"
	"testing"
)

type countedEmbedder struct {
	calls int
	fail  bool
}

func (e *countedEmbedder) Embed(s string) ([]float32, error) {
	e.calls++
	if e.fail {
		return nil, errors.New("provider failed")
	}
	return []float32{float32(len(s)), 1}, nil
}
func (e *countedEmbedder) EmbedBatch(s []string) ([][]float32, error) {
	v := make([][]float32, len(s))
	for i, t := range s {
		var err error
		v[i], err = e.Embed(t)
		if err != nil {
			return nil, err
		}
	}
	return v, nil
}
func (e *countedEmbedder) ModelName() string { return "test-model" }
func (e *countedEmbedder) Dimension() int    { return 2 }
func TestEmbedCacheCopiesAndConcurrent(t *testing.T) {
	e := &countedEmbedder{}
	c := NewCachingEmbedder(e, nil)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, err := c.Embed("hello")
			if err != nil {
				t.Error(err)
			}
			v[0] = 100
		}()
	}
	wg.Wait()
	v, err := c.Embed("hello")
	if err != nil || v[0] != 5 || e.calls != 1 {
		t.Fatalf("%v %v calls=%d", v, err, e.calls)
	}
	batch, err := c.EmbedBatch([]string{"hello", "world!", "hello"})
	if err != nil || len(batch) != 3 || e.calls != 2 {
		t.Fatalf("batch=%v %v calls=%d", batch, err, e.calls)
	}
	e.fail = true
	_, _ = c.Embed("bad")
	_, _ = c.Embed("bad")
	if e.calls != 4 {
		t.Fatal("cached failure")
	}
}
