package main

import "testing"

// M6 (Phase 3): binary and JSON ingest straight into the packed Array[float]
// store, on both backends with no VM fallback. The Result and the bytes cross
// the evaluator bridge, which is what keeps the VM on its own path.
func TestStdArrayIngest(t *testing.T) {
	const imports = `import std/array (make, fromList, length, sum, encodeF64LE, decodeF64LE, encodeF32LE, decodeF32LE)
import std/json (decodeFloatArray)
import std/bytes (fromString)

func showR(r: Result[Array[float], string]) -> string = match r { Ok(a) => show(a), Err(e) => "Err: ${e}" }`
	cases := []struct{ name, body, want string }{
		{"f64 round trip", `showR(decodeF64LE(encodeF64LE(fromList([0.1, 2.5, -3.0]))))`, "#[0.1, 2.5, -3.0]"},
		{"f32 lossy", `showR(decodeF32LE(encodeF32LE(fromList([0.1, 1.5]))))`, "#[0.10000000149011612, 1.5]"},
		{"f64 100k", `match decodeF64LE(encodeF64LE(make(100000, 0.5))) { Ok(a) => show(sum(a)), Err(e) => e }`, "50000.0"},
		{"f64 bad length", `showR(decodeF64LE(fromString("abcdefghi")))`, "Err: decodeF64LE: 9 bytes is not a whole number of 8-byte floats"},
		{"json", `showR(decodeFloatArray(" [1.5, -2, 3e2, 0.25E-1] "))`, "#[1.5, -2.0, 300.0, 0.025]"},
		{"json empty", `showR(decodeFloatArray("[]"))`, "#[]"},
		{"json nested", `showR(decodeFloatArray("[1, [2]]"))`, "Err: decodeFloatArray: expected a number at offset 4"},
		{"json string", `showR(decodeFloatArray("[\"1\"]"))`, "Err: decodeFloatArray: expected a number at offset 1"},
		{"json trailing comma", `showR(decodeFloatArray("[1,]"))`, "Err: decodeFloatArray: expected a number at offset 3"},
		{"json leading zero", `showR(decodeFloatArray("[01]"))`, "Err: decodeFloatArray: expected ',' or ']' at offset 2"},
		{"json trailing data", `showR(decodeFloatArray("[1] x"))`, "Err: decodeFloatArray: trailing data at offset 4"},
		{"json not array", `showR(decodeFloatArray("{}"))`, "Err: decodeFloatArray: expected '[' at offset 0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for backend, got := range runArrayCase(t, arrayProgram(imports, c.body), true) {
				if got != c.want {
					t.Errorf("%s: got %q, want %q", backend, got, c.want)
				}
			}
		})
	}
}
