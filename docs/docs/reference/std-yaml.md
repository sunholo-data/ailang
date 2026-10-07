---
reviewBy: 2026-12-28
---

# std/yaml — YAML decoding and encoding

The `std/yaml` module (v0.30.0+) reads YAML into AILANG's existing `Json` ADT. It is a thin, **pure** bridge: a Go-backed builtin parses YAML with `gopkg.in/yaml.v3` and re-emits it as a JSON string, which `std/json.decode` turns into the same `Json` value you'd get from decoding JSON directly. Because the parser is pure Go with no syscalls, `std/yaml` is **WASM-portable** — the same code runs in the browser.

## When to use

Use `std/yaml` when a program needs to read a YAML source — a config file, a tool manifest, a corpus catalogue — inside a single-language AILANG pipeline, instead of shelling out to an external YAML tool. Since YAML 1.2's core schema maps onto JSON types, anything that could equally have been JSON round-trips cleanly.

## API

### `yamlToJson(s: string) -> Result[string, string]`

Converts a YAML string to an equivalent JSON string. Returns `Ok(jsonString)` on success, `Err(message)` on failure.

```ailang
import std/yaml (yamlToJson)
import std/result (Ok, Err)

match yamlToJson("a: 1\nb: [x, y]\n") {
  Ok(j)  => println(j),          -- {"a":1,"b":["x","y"]}
  Err(e) => println("err: ${e}")
}
```

### `decode(s: string) -> Result[Json, string]`

Decodes a YAML string into the `std/json` `Json` ADT. Equivalent to `yamlToJson` followed by `std/json.decode`, so the entire `std/json` accessor surface (`get`, `getString`, `getInt`, `asNumber`, …) works on the result.

```ailang
import std/yaml (decode)
import std/json (getString)
import std/option (Some, None)
import std/result (Ok, Err)

match decode("title: Fysik A\nyear: 2026\n") {
  Ok(cat) => match getString(cat, "title") {
    Some(t) => println("title: ${t}"),
    None    => println("no title")
  },
  Err(e) => println("parse failed: ${e}")
}
```

### `encode(j: Json) -> Result[string, string]`

Encodes the shared `Json` ADT into one deterministic block YAML document
(v0.53.0+). It is pure and WASM-portable. Returns `Ok(yamlString)` or
`Err(message)` for non-finite numbers, including nested NaN or infinities.

```ailang
import std/yaml (encode)
import std/json (JObject, JString, JNumber)
import std/result (Ok, Err)

match encode(JObject([
  {key: "title", value: JString("Fysik A")},
  {key: "year", value: JNumber(2027.0)}
])) {
  Ok(s) => println(s),
  Err(e) => println("encode failed: ${e}")
}
```

The YAML bytes are:

```yaml
"title": "Fysik A"
"year": 2027
```

All keys and string values use double quotes and JSON-compatible escaping.
Multiline strings contain escaped `\n`, with YAML-sensitive control and line
separator characters emitted as Unicode escapes (also U+FFFE/U+FFFF).
Keys whose quoted representation reaches 1024 characters use explicit `? "key"` / `: value` block syntax to avoid YAML’s
simple-key length limit. Nonempty collections use block style and two-space indentation; only empty collections use `[]` or `{}`.
Output ends in exactly one newline, with no wrapping, tags, document markers,
anchors, or block literal/folded strings.

Object pairs emit in their original list order, including duplicate keys.
Re-decoding a duplicate-key object returns `Err` naming the duplicated key.
For finite numbers and recursively unique object keys, matching an encoded
`Ok(s)` and then calling `decode(s)` preserves the Json value **modulo recursive
object key order**: today's decode sorts keys. Exact equality holds for sorted
objects; encode does not depend on the sibling decode-order improvement.
Numbers follow `std/json`'s canonical formatting, including signed zero.
Integer payloads from the WASM bridge emit without conversion to float; decoding
still uses the shared Json number representation and its precision limits.
Arbitrary AILANG records and other ADTs are not accepted as input.

## Semantics and limits

- **Single document.** Only the first document of a multi-document stream (`---` separated) is read. `decodeAll` is future work.
- **No silent coercion.** Any YAML that has no JSON representation returns `Err` rather than a guessed value:
  - **non-string map keys** (`1: a`, `true: x`) — JSON object keys must be strings;
  - **`NaN` / `±Inf` floats** — not representable in JSON.
- **Anchors and aliases work but are not preserved.** `a: &x 1\nb: *x` decodes to `{"a":1,"b":1}`; the output is plain JSON with no anchor structure.
- **Numbers decode as `JNumber(float)`.** Like `std/json`, integers become floats, so integers beyond 2^53 lose precision. This is inherited from the shared `Json` ADT, not specific to YAML.
- **Empty input decodes to `JNull`** (`yamlToJson("")` returns `Ok("null")`).

## Run the examples

```bash
ailang run --caps IO --entry main examples/runnable/yaml_config.ail
# Output:
#   title: Fysik A
#   year: 2026
```

For decode → edit → encode:

```bash
ailang run --caps IO --entry main examples/runnable/yaml_roundtrip.ail
# "title": "Fysik A"
# "year": 2027
```

## See also

- [`std/json`](./stdlib) — the `Json` ADT and its accessors, which `std/yaml.decode` reuses.
