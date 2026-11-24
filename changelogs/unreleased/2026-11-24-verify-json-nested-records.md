### Fixed — `ailang verify` handles `std/json`'s `Json` and records nested in ADT payloads

Every contracted function that touched `std/json`'s `Json` was `⚠ SKIPPED`
with `sort "Record_key_value" referenced in declaration of "Json" is not
declared`. `Json`'s `JObject([{ key: string, value: Json }])` holds a record
inside a list, and only a constructor field whose type *was* a record got its
record declared. A record nested anywhere in a payload (a list element, a type
argument such as `Option[{…}]`, a tuple, another record) is now declared, so
such functions verify or get a counterexample. The same applied to user ADTs of
that shape (`type Tree = Leaf(int) | Node([{ k: string, t: Tree }])`).

`std/json.get` and the accessors built on it still skip: they search the
object with `findIndex`, which the verifier does not encode yet.

A skip caused by mutually recursive helpers now names the same cycle on every
run; the helper lists were collected in map order.

Across the eight downstream corpora (`make verify-gap-map`), 2 results moved
from skipped to verified and nothing regressed.
