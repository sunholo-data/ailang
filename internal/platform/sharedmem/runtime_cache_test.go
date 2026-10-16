package sharedmem

import (
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestCheckedSharedMemCASConcurrentConnections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	a, err := NewSQLiteSharedCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	b, err := NewSQLiteSharedCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	for _, old := range [][]byte{nil, []byte("initial")} {
		if old != nil {
			if err := a.PutChecked("key", old); err != nil {
				t.Fatal(err)
			}
		}
		var successes atomic.Int32
		var wg sync.WaitGroup
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c := a
				if i%2 == 1 {
					c = b
				}
				ok, err := c.CASChecked("key", old, []byte("updated"))
				if err != nil {
					t.Error(err)
				}
				if ok {
					successes.Add(1)
				}
			}(i)
		}
		wg.Wait()
		if successes.Load() != 1 {
			t.Fatalf("successful swaps = %d", successes.Load())
		}
	}
}

func TestCheckedSharedMemFailureAndEmptyBytes(t *testing.T) {
	c, err := NewSQLiteSharedCache(filepath.Join(t.TempDir(), "cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := c.PutChecked("empty", nil); err != nil {
		t.Fatal(err)
	}
	v, ok, err := c.GetChecked("empty")
	if err != nil || !ok || len(v) != 0 {
		t.Fatalf("get empty = %v %v %v", v, ok, err)
	}
	ok, err = c.CASChecked("empty", []byte{}, nil)
	if err != nil || !ok {
		t.Fatalf("CAS empty = %v %v", ok, err)
	}
	c.Close()
	if _, _, err := c.GetChecked("x"); err == nil {
		t.Error("closed get succeeded")
	}
	if err := c.PutChecked("x", nil); err == nil {
		t.Error("closed put succeeded")
	}
	if _, err := c.CASChecked("x", nil, nil); err == nil {
		t.Error("closed CAS succeeded")
	}
	if err := c.DeleteChecked("x"); err == nil {
		t.Error("closed delete succeeded")
	}
	if _, err := c.KeysChecked(); err == nil {
		t.Error("closed keys succeeded")
	}
}
