package pipeline

import (
	"fmt"
	"strings"
	"testing"
)

// Both streaming APIs must publish the callback row, rather than either
// requiring IO from pure callbacks or silently discharging IO for their caller.
func TestAIStreamCallbackEffectRows(t *testing.T) {
	for _, api := range []string{"stepWithStream", "stepWithStreamRecorded"} {
		for _, tt := range []struct {
			name, callback, row string
			reject              bool
		}{
			{"pure callback", "quiet", "AI", false},
			{"IO callback charged", "loud", "AI", true},
			{"declared IO callback", "loud", "AI, IO", false},
		} {
			t.Run(api+"/"+tt.name, func(t *testing.T) {
				src := fmt.Sprintf(`module stream_rows
import std/ai (%s, StreamChunk)
import std/io (println)
func quiet(c: StreamChunk) -> () = ()
func loud(c: StreamChunk) -> () ! {IO} = println("callback")
export func caller() -> bool ! {%s} {
 let _ = %s("stub", [], [], [], %s);
 true
}
`, api, tt.row, api, tt.callback)
				err := checkModuleSource(t, "stream_rows", src)
				if !tt.reject {
					if err != nil {
						t.Fatal(err)
					}
					return
				}
				if err == nil || !strings.Contains(err.Error(), "Missing effects: IO") || !strings.Contains(err.Error(), "caller") {
					t.Fatalf("caller must account for callback IO: %v", err)
				}
			})
		}
	}
}
