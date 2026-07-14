package importextract

import (
	"reflect"
	"testing"
)

func TestExtractModules(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "selective imports",
			src: `module t
import std/io (println)
import std/list (map, foldl)
export func main() -> () ! {IO} { println("hi") }`,
			want: []string{"std/io", "std/list"},
		},
		{
			name: "module alias",
			src: `module t
import std/list as L
export func main() -> () ! {IO} { () }`,
			want: []string{"std/list"},
		},
		{
			name: "duplicate imports deduped",
			src: `module t
import std/io (println)
import std/io (print)
export func main() -> () ! {IO} { println("x") }`,
			want: []string{"std/io"},
		},
		{
			name: "non-std imports excluded",
			src: `module t
import std/io (println)
export func main() -> () ! {IO} { println("x") }`,
			want: []string{"std/io"},
		},
		{
			name: "no imports",
			src: `module t
export func main() -> int = 42`,
			want: []string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ExtractModulesFromSource(tc.src, tc.name+".ail")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExtractModules_ParseError(t *testing.T) {
	// Malformed source must error, never silently return partial results.
	_, err := ExtractModulesFromSource("module t\nimport std/io (", "bad.ail")
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestEqual(t *testing.T) {
	if !Equal([]string{"std/io", "std/list"}, []string{"std/list", "std/io"}) {
		t.Error("Equal should be order-independent")
	}
	if Equal([]string{"std/io"}, []string{"std/io", "std/list"}) {
		t.Error("differing sets must not be equal")
	}
}
