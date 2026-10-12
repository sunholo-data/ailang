# Pure-row / iface-purity M1–M2 execution evidence

Refs #1443.

Branch: `coordinator/task-7861d01e`.
Base and remote dev: `7c43ece3ce7abac639aa04777c3abc56e4117e65`.
The landed row-unification validation and latent-function-value tests are present.
No model credential is needed; Z3 is available. Teaching prompt loaded: v0.16.7.
Cache base: v6 (alias-body closure); selected next unused version: v7.

## HOLD remeasurement before M2

Every std module checked on the base with canonical module paths and caching disabled.
50 modules, 484 function exports, 168 labelled outer rows, 17 open outer rows,
271 keyword-pure exports, and 27 rowless/unannotated-callback combinators.
The historical 26 grew by `std/list.mapAccumL`. Before M2 all exports carry the
stub `Purity: true`, so all 168 labelled functions are false positives.
Open and labelled counts overlap; they are not summed.

`std/list.flatMap`, `std/list.sortBy`, and `std/option.flatMap` have callback-shared
outer tails. The other 24 combinators have empty outer rows. All 27 should report
true. `std/list.mapE` has a declared shared `e` and should report false; all six
streaming exports have outer labels and should report false.

### Measured schemes

```text
std/ai/streaming.anthropicStream : (string, string, string) -> Result[StreamConn, StreamErrorKind] ! {AI[mode=fixed], Net, Stream}
std/ai/streaming.callStream : (string, string, string) -> Result[string, {code: string, message: string, retryable: bool}] ! {AI[mode=fixed], Net, Stream}
std/ai/streaming.disconnect : StreamConn -> () ! {Stream}
std/ai/streaming.onEvent : (StreamConn, StreamEvent -> bool ! {...e}) -> () ! {Stream, ...e}
std/ai/streaming.openaiCompatStream : (string, string, string) -> Result[StreamConn, StreamErrorKind] ! {AI[mode=fixed], Net, Stream}
std/ai/streaming.runEventLoop : StreamConn -> () ! {Stream}
std/list.any : (a -> bool, list[a]) -> bool
std/list.filter : (a -> bool, list[a]) -> list[a]
std/list.findIndex : (a -> bool, list[a]) -> Option[int]
std/list.flatMap : (a -> list[b] ! {...ρ2}, list[a]) -> list[b] ! {...ρ2}
std/list.foldl : ((α32, a) -> α32, α32, list[a]) -> α32
std/list.foldr : ((a, α38) -> α38, α38, list[a]) -> α38
std/list.map : (a -> b, list[a]) -> list[b]
std/list.mapAccumL : ((s, a) -> (b, s), s, list[a]) -> (list[b], s)
std/list.mapE : (a -> b ! {...e}, list[a]) -> [b] ! {...e}
std/list.sortBy : ((a, a) -> int ! {...ρ2}, list[a]) -> list[a] ! {...ρ2}
std/list.takeFlatMap : (int, a -> list[b], list[a]) -> list[b]
std/list.takeMap : (int, a -> b, list[a]) -> list[b]
std/list.zipWith : ((α128, α129) -> α130 ! {...ρ3}, list[α128], list[α129]) -> list[α130]
std/option.filter : (α53 -> bool ! {...ρ3}, Option[α53]) -> Option[α53]
std/option.flatMap : (α20 -> Option[b] ! {...ρ2}, Option[α20]) -> Option[b] ! {...ρ2}
std/option.map : (α9 -> α4 ! {...ρ3}, Option[α9]) -> Option[α4]
std/result.flatMap : (α58 -> Result[b, α59] ! {...ρ2}, Result[α58, α59]) -> Result[b, α59]
std/result.map : (α14 -> α6 ! {...ρ3}, Result[α14, α9]) -> Result[α6, α9]
std/result.mapErr : (α37 -> α31 ! {...ρ3}, Result[α36, α37]) -> Result[α36, α31]
std/string.foldChars : ((α47, string) -> α47, α47, string) -> α47
std/string.foldSlices : (string, string, α69, (α69, string) -> α69) -> α69
std/string.mapSlicesJoin : (string, string, string -> string) -> string
std/xml.flatMapChildren : (XmlNode, XmlNode -> list[a]) -> list[a]
std/xml.foldChildren : (XmlNode, α50, (α50, XmlNode) -> α50) -> α50
std/xml.foldChildrenStep : (XmlNode, α54, (α54, XmlNode) -> FoldStep[α54]) -> α54
std/xml.mapChildren : (XmlNode, XmlNode -> a) -> list[a]
std/xml.parseFold : (string, string, a, (a, XmlNode) -> a) -> Result[a, string]
std/xml.parseFoldStep : (string, string, a, (a, XmlNode) -> FoldStep[a]) -> Result[a, string]
```

## Scope

M1 and M2 only. M3–M4, #1718, SMT admission, corpus migrations and teaching
material migrations remain pending. This is a partial implementation for review;
the complete sprint is not ready for release.

## M1

The 16-case signature matrix established six failures against the old checker:
IO, Debug, Declassify, scoped Declassify, var-only and mixed open declarations.
The implemented check rejects all six with the function name and both repairs;
non-pure controls, absent rows and explicit empty rows remain accepted.

M1 checkpoint: focused `go test ./internal/pipeline/...` passes; `make lint`
reports zero issues; boundaries, file sizes and changelog checks pass.

## M2

Pending implementation and validation.
