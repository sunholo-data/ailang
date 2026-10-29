package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sunholo-data/ailang/internal/testutil"
)

// nestedDeepCall places the callback past the old sixteen-container cutoff.
func nestedDeepCall(depth int, callback, caller string) string {
	typ, value := "(int) -> int", callback
	body := fmt.Sprintf("v%d(x)", depth)
	for i := depth; i > 0; i-- {
		previous := fmt.Sprintf("v%d", i-1)
		if i == 1 {
			previous = "fs"
		}
		body = fmt.Sprintf("match %s { [] => x, [v%d, ..._] => %s }", previous, i, body)
		typ, value = "["+typ+"]", "["+value+"]"
	}
	return fmt.Sprintf("func applyDeep(fs: %s, x: int) -> int = %s\n%s = applyDeep(%s, x)\n", typ, body, caller, value)
}

func TestLatentNested_DeepCallbackControls(t *testing.T) {
	runNestedCases(t, []nestedCase{
		{name: "deep_pure_rejected", body: nestedDeepCall(17, "logIt", "pure func sneaky(x: int) -> int"), blame: "sneaky"},
		{name: "deep_pure_callback", body: nestedDeepCall(17, "inc", "pure func ok(x: int) -> int")},
		{name: "deep_declared_effect", body: nestedDeepCall(17, "logIt", "func ok(x: int) -> int ! {IO}")},
	})
}

func TestLatentNested_DeepRunRejectsBeforeExecution(t *testing.T) {
	binary := testutil.FindAilangBinary(t)
	t.Setenv("AILANG_NO_CACHE", "1")
	source := nestedSource("deep", nestedDeepCall(17, "logIt", "pure func sneaky(x: int) -> int")+"export func main() -> () ! {IO} = println(show(sneaky(1)))\n")
	path := filepath.Join(t.TempDir(), "deep.ail")
	if err := os.WriteFile(path, []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	for _, bytecode := range []bool{false, true} {
		args := []string{"run", "--caps", "IO", "--entry", "main"}
		if bytecode {
			args = append(args, "--bytecode")
		}
		output, err := exec.Command(binary, append(args, path)...).CombinedOutput()
		text := string(output)
		if err == nil || performedLine.MatchString(text) {
			t.Fatalf("executed deep callback (bytecode=%v): %s", bytecode, output)
		}
		if !strings.Contains(text, "Missing effects: IO") || !strings.Contains(text, "function 'sneaky'") || strings.Contains(text, "panic:") {
			t.Fatalf("wrong rejection (bytecode=%v): %s", bytecode, output)
		}
	}
}
