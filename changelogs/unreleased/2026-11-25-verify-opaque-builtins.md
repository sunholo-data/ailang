### Added — verify: JSON, SHA-256 and XML escaping are opaque functions with facts

- **Contracts through `std/json.encode`/`decode`, `std/crypto.sha256Hex` and `std/xml.escapeXml` are now checked** (M-VERIFY-SOUNDNESS-COVERAGE). Their Go builtins (`_json_encode`, `_json_decode`, `_crypto_sha256hex`, `_escapeXml`) were unencodable, so every function reaching one was skipped. Each is now an uninterpreted function, which is sound because the builtins are pure. The solver also gets the facts their implementations guarantee for every input:
  - a SHA-256 hex digest is 64 characters;
  - escaped XML has no `<`, `>`, `"` or `'`, is never shorter than its input, and is unchanged when the input has no special character;
  - encoded JSON is never empty, and its outer shape follows the top constructor (`JObject` → `{…}`, `JArray([])` → `[]`, `JNull` → `null`, …).

  A contract that needs more than these facts is `inconclusive` until witness replay runs the real builtin: a replayed refutation is a confirmed violation. A function that does not declare the sort a builtin needs (`Json`) is skipped with a reason naming the builtin.
- Downstream corpora: 6 skips now verify (`agui` `encodeEvent`, `jsonLines`, `sseFrame`; `linkedinDerivePersona`; `xmlEscape`; `apiKeyHash`).

### Fixed — verify: two Z3 errors on records and empty lists in constructors

- **A record literal no longer takes another record's sort because the field names match.** Field names alone chose the sort, so a package alias `type A2UIProp = {key: string, value: string}` captured `std/json`'s `{key, value: Json}` literals (`kv`), and Z3 rejected the script. The literal's checked type now picks between records with the same field names; if it cannot, the function is skipped.
- **An empty list passed straight to a constructor takes the field's sort.** `JArray([])` was sorted `(Seq Int)`, and Z3 rejected `JArray`'s argument.
