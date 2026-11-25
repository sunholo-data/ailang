### Added — verify: list patterns are encoded

- **A match on list patterns (`[]`, `[a, b]`, `[x, ...rest]`, `x :: rest`) reaches the solver** (M-VERIFY-SOUNDNESS-COVERAGE). Before, every function using one, or calling a helper that does, was skipped with `match on list patterns (x :: rest) is outside the decodable fragment`. Each arm becomes a test on the list's length and its named elements, and binds its variables to `seq.nth` and `seq.extract` terms. The arms form a first-match chain, so guards, literal elements and lists inside constructors (`Some([x, ...rest])`) work too. A recursive helper that matches on a list unrolls like any other.
- The last arm is taken untested only when the verifier's own coverage check proves the unguarded arms cover every length. A match it cannot prove exhaustive is skipped; no fall-through value is invented.
- Downstream corpora: all 9 skips this blocked now get past it. 3 become inconclusive: `entropy`, `pptxTakeFitting` and `apiDisplayFilename` need a fact about lists longer than the unrolling depth. The other 6 stop at other blockers: `stringToInt`, `std/list.reverse`, an anonymous `{count, rest}` record, and two solver timeouts. No verified result changed.

### Fixed — verify: pattern variables no longer collide with record accessors

- **A pattern variable named like a record field was a Z3 error.** `match o { Some(p) => p.p, … }` bound `p` under its raw name, which the accessor `p` already declares. Pattern variables now bind under the `$l_` symbol that `let` uses (#215), in every match encoding.

### Fixed — verify: an inline record nested in another inline record is declared

- **An ADT whose constructor field is an inline record that itself contains an inline record** skipped every function touching that ADT. An example is `decisions`' `ChoiceA({ choice, probabilities: [{ key, p }] })`, which skipped with `sort "Record_key_p" referenced in declaration of "Record_choice_probabilities" is not declared`. The per-function sort filter followed an inline record's references to ADTs and aliases, but not to another inline record, so it dropped the inner one. It now follows those too.

### Fixed — verify: tuples, lambdas, local recursion and unit literals are skips, not errors

- **A tuple expression, lambda, `letrec`, dictionary reference or unit literal inside a recursive function's unrolled body was a verifier ERROR** (`encoding error: … tuple expressions cannot be encoded in SMT-LIB`). Each is a boundary of the encodable fragment, not a verifier fault, so it is now a skip that names the construct, as #757 made list patterns. A recursive nullary function (its unrolling passes `()`) is one such case.
- The CLI tests that used a list pattern as their example of an unsupported construct now use a tuple scrutinee.
