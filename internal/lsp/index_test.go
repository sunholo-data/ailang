package lsp

import (
	"testing"
)

const indexFixture = `module fixture

import std/io (println)

export func main() -> () ! {IO} {
  println("hello")
}
`

func TestPositionIndexFindsTopLevelIdent(t *testing.T) {
	t.Parallel()
	idx := BuildPositionIndex("fixture.ail", indexFixture)
	if idx == nil {
		t.Fatal("BuildPositionIndex returned nil")
	}

	// `println` on the line `  println("hello")` — line 6, starts at col 3.
	// Cursor inside "println" should hit the identifier.
	for col := 3; col <= 9; col++ { // "println" spans cols 3..9 inclusive (7 chars)
		got := idx.Lookup(6, col)
		if got == nil {
			t.Errorf("Lookup(6, %d) returned nil; expected ident inside 'println'", col)
			continue
		}
		if got.Name != "println" {
			t.Errorf("Lookup(6, %d): got %q, want %q", col, got.Name, "println")
		}
	}
}

func TestPositionIndexReturnsNilOutsideIdent(t *testing.T) {
	t.Parallel()
	idx := BuildPositionIndex("fixture.ail", indexFixture)
	if idx == nil {
		t.Fatal("BuildPositionIndex returned nil")
	}

	// Whitespace at L6:C1 (before 'println') — should return nil.
	if got := idx.Lookup(6, 1); got != nil {
		t.Errorf("Lookup at whitespace L6:C1: got %v, want nil", got)
	}
	// Inside the string literal "hello" at L6:C12+ — should NOT match println.
	if got := idx.Lookup(6, 13); got != nil && got.Name == "println" {
		t.Errorf("Lookup inside string literal incorrectly matched 'println'")
	}
	// Way past EOL — nil.
	if got := idx.Lookup(6, 999); got != nil {
		t.Errorf("Lookup past EOL: got %v, want nil", got)
	}
	// Wrong line — nil.
	if got := idx.Lookup(99, 5); got != nil {
		t.Errorf("Lookup on non-existent line: got %v, want nil", got)
	}
}

func TestPositionIndexHandlesEmptyFile(t *testing.T) {
	t.Parallel()
	idx := BuildPositionIndex("empty.ail", "")
	// Either nil or empty index is acceptable; lookups must not panic.
	if idx != nil {
		if got := idx.Lookup(1, 1); got != nil {
			t.Errorf("empty file Lookup: got %v, want nil", got)
		}
	}
}
