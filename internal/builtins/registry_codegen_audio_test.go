package builtins

import "testing"

// Audio requires real codegen implementations, not capitalization fallbacks
// that generate unresolved names for a standard-library call.
func TestAudioCodegenBoundariesHaveSpecs(t *testing.T) {
	for _, key := range []StdlibKey{{"std/audio", "wavHeader"}, {"std/audio", "wavHeaderChecked"}, {"std/audio", "wavFromPcm"}, {"std/audio", "wavFromPcmChecked"}, {"std/audio", "wavInfo"}, {"std/audio", "pcm16ToFloats"}, {"std/audio", "floatsToPcm16"}, {"std/bytes", "fromInts"}, {"std/bytes", "toInts"}, {"std/bytes", "slice"}, {"std/bytes", "byteAt"}} {
		if s := GetCodegenSpecByStdlibName(key.Module, key.Name); s == nil || s.Helper == nil {
			t.Errorf("missing generated-Go helper for %s.%s", key.Module, key.Name)
		}
	}
}
