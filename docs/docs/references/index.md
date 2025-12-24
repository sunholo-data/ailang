---
sidebar_position: 3
title: Citations & Bibliography
description: Complete academic citations with DOIs, arXiv links, and PDF sources
---

# Citations & Bibliography

This page provides complete academic citations for the research papers underlying AILANG's design.
All entries include DOIs, arXiv links, and direct PDF sources where available.

For design rationale and adopted/rejected decisions, see [Foundations & Design Lineage](./design-lineage.mdx).

---

## Type Systems

### Hindley-Milner Type Inference

AILANG uses Hindley-Milner type inference with let-polymorphism, providing principal types without type annotations.

> **Damas, L., & Milner, R.** (1982). Principal type-schemes for functional programs. *Proceedings of the 9th ACM SIGPLAN-SIGACT Symposium on Principles of Programming Languages (POPL '82)*, 207–212. ACM.
> DOI: [10.1145/582153.582176](https://doi.org/10.1145/582153.582176)
> PDF: [damas-milner-1982.pdf](/references/damas-milner-1982.pdf)

**Historical context**: The type inference algorithm for the simply typed lambda calculus was devised by Curry and Feys (1958). In 1969, J. Roger Hindley extended this work and proved the algorithm always inferred the most general type. In 1978, Robin Milner independently provided an equivalent algorithm (Algorithm W). In 1982, Luis Damas proved Milner's algorithm is complete and extended it to support polymorphic references.

> **Hindley, J. R.** (1969). The principal type-scheme of an object in combinatory logic. *Transactions of the American Mathematical Society*, 146, 29–60.

> **Milner, R.** (1978). A theory of type polymorphism in programming. *Journal of Computer and System Sciences*, 17(3), 348–375.
> DOI: [10.1016/0022-0000(78)90014-4](https://doi.org/10.1016/0022-0000(78)90014-4)

### Row Polymorphism

AILANG uses row polymorphism for extensible records and effect types.

> **Wand, M.** (1989). Type inference for record concatenation and multiple inheritance. *Proceedings of the Fourth Annual Symposium on Logic in Computer Science (LICS '89)*, 92–97.
> DOI: [10.1109/LICS.1989.39162](https://doi.org/10.1109/LICS.1989.39162)

> **Wand, M.** (1991). Type inference for record concatenation and multiple inheritance. *Information and Computation*, 93(1), 1–15.
> DOI: [10.1016/0890-5401(91)90050-C](https://doi.org/10.1016/0890-5401(91)90050-C)

> **Rémy, D.** (1993). Type inference for records in a natural extension of ML. *Theoretical Aspects of Object-Oriented Programming: Types, Semantics and Language Design*. MIT Press.

**See also**: [Wikipedia: Row polymorphism](https://en.wikipedia.org/wiki/Row_polymorphism)

### Type Classes

AILANG implements type class elaboration via dictionary passing for overloaded operations.

> **Wadler, P., & Blott, S.** (1989). How to make ad-hoc polymorphism less ad hoc. *Proceedings of the 16th ACM SIGPLAN-SIGACT Symposium on Principles of Programming Languages (POPL '89)*, 60–76. ACM.
> DOI: [10.1145/75277.75283](https://doi.org/10.1145/75277.75283)

**Impact**: This paper introduced type classes, which became a foundational implementation technique. The dictionary-passing elaboration presented in this paper influenced Haskell's type classes, Scala's implicits, Agda's instance arguments, Coq's type classes, and Rust's traits.

---

## Effect Systems

### Polymorphic Effect Systems

AILANG's effect system builds on foundational work in polymorphic effect types.

> **Lucassen, J. M., & Gifford, D. K.** (1988). Polymorphic effect systems. *Proceedings of the 15th ACM SIGPLAN-SIGACT Symposium on Principles of Programming Languages (POPL '88)*, 47–57. ACM.
> DOI: [10.1145/73560.73564](https://doi.org/10.1145/73560.73564)
> PDF: [lucassen-gifford-1988.pdf](/references/lucassen-gifford-1988.pdf)

**Key contribution**: This paper presents a 'kinded' type system with three base kinds: types (describing values), effects (describing side-effects), and regions (describing store areas). This enables type, effect, and region polymorphism.

### Algebraic Effects and Handlers

AILANG's effect handlers are inspired by the algebraic effects tradition.

> **Plotkin, G., & Pretnar, M.** (2009). Handlers of Algebraic Effects. *Programming Languages and Systems: ESOP 2009*, LNCS 5502, 80–94. Springer.
> DOI: [10.1007/978-3-642-00590-9_7](https://doi.org/10.1007/978-3-642-00590-9_7)
> PDF: [plotkin-pretnar-2009.pdf](/references/plotkin-pretnar-2009.pdf)

> **Plotkin, G., & Pretnar, M.** (2013). Handling Algebraic Effects. *Logical Methods in Computer Science*, 9(4), Article 23.
> arXiv: [1312.1399](https://arxiv.org/abs/1312.1399) · PDF: [plotkin-pretnar-2013-arxiv.pdf](/references/plotkin-pretnar-2013-arxiv.pdf)

**Key insight**: Algebraic effects include exceptions, state, nondeterminism, interactive I/O, and time. Handling a computation amounts to composing it with a unique homomorphism guaranteed by universality.

### Koka: Row-Polymorphic Effect Types

AILANG's row-polymorphic effect types are directly influenced by Koka.

> **Leijen, D.** (2014). Koka: Programming with Row Polymorphic Effect Types. *Mathematically Structured Functional Programming 2014 (MSFP 2014)*, EPTCS 153, 100–126.
> arXiv: [1406.2061](https://arxiv.org/abs/1406.2061)
> PDF: [leijen-koka-2014-arxiv.pdf](/references/leijen-koka-2014-arxiv.pdf)

**Key features**: Polymorphic effects through row-polymorphism with duplicate labels. If an expression can be typed without an `exn` effect, it will never throw an unhandled exception. State can be safely combined with let-polymorphism without imperative type variables.

---

## Compilation

### A-Normal Form (ANF)

AILANG compiles to Core AST in A-Normal Form for optimization and evaluation.

> **Flanagan, C., Sabry, A., Duba, B. F., & Felleisen, M.** (1993). The Essence of Compiling with Continuations. *Proceedings of the ACM SIGPLAN 1993 Conference on Programming Language Design and Implementation (PLDI '93)*, 237–247. ACM.
> DOI: [10.1145/155090.155113](https://doi.org/10.1145/155090.155113)
> PDF: [flanagan-anf-1993.pdf](/references/flanagan-anf-1993.pdf)

**Key insight**: One can forgo a standard 3-pass CPS transformation while still capturing the essence of compiling with continuations by doing a single source-level transformation to A-Normal Form. ANF assigns names to every intermediate computation via `let` bindings.

### Pattern Matching Compilation

AILANG compiles pattern matching to decision trees for efficiency.

> **Maranget, L.** (2008). Compiling Pattern Matching to Good Decision Trees. *Proceedings of the 2008 ACM SIGPLAN Workshop on ML*, 35–46. ACM.
> DOI: [10.1145/1411304.1411311](https://doi.org/10.1145/1411304.1411311)
> PDF: [maranget-2008.pdf](/references/maranget-2008.pdf)

**Key contribution**: Novel heuristics inspired by "necessity" from lazy pattern matching. Decision trees never test a given subterm more than once (at the cost of potential code size explosion). The necessity heuristic provides a straightforward connection to runtime efficiency.

---

## Verification

### Automated Reasoning Checks (ARC)

AILANG's M-VERIFY contract system implements techniques from AWS's neurosymbolic verification research.

> **Bayless, S., et al.** (2025). A Neurosymbolic Approach to Natural Language Formalization and Verification.
> arXiv: [2511.09008](https://arxiv.org/abs/2511.09008) · PDF: [bayless-2025-neurosymbolic.pdf](/references/bayless-2025-neurosymbolic.pdf)

**Key approach**: A two-stage framework that (1) uses LLMs with optional human guidance to formalize natural language policies, and (2) uses inference-time autoformalization to validate logical correctness. Multiple redundant formalization steps cross-check for semantic equivalence, achieving 99.2% soundness.

**AILANG implementation**: The [Contracts & Verification](../guides/contracts.mdx) guide shows how AILANG brings these techniques into the language itself via `requires`/`ensures` clauses. The [park admission example](../guides/contracts.mdx#policy-example-park-admission) directly reproduces the policy scenario from the ARC paper.

**Note**: This is AWS's "ARC" (Automated Reasoning Checks), distinct from François Chollet's ARC-AGI benchmark.

---

## AI Reasoning Benchmarks

### Abstraction and Reasoning Corpus (ARC-AGI)

While distinct from AWS ARC, Chollet's ARC benchmark informs AILANG's approach to AI-friendly language design.

> **Chollet, F.** (2019). On the Measure of Intelligence.
> arXiv: [1911.01547](https://arxiv.org/abs/1911.01547) · PDF: [chollet-2019-measure-of-intelligence.pdf](/references/chollet-2019-measure-of-intelligence.pdf)
> DOI: [10.48550/arXiv.1911.01547](https://doi.org/10.48550/arXiv.1911.01547)

**Key insight**: Intelligence is an agent's ability to adapt to changing environments and respond appropriately in novel situations. ARC consists of tasks requiring solvers to infer transformation rules from limited examples, then generalize to novel inputs. Despite requiring only four types of prior knowledge (objectness, goal-directedness, arithmetic, geometric topology), it presents high reasoning difficulty.

> **Chollet, F., et al.** (2025). ARC-AGI-2: A New Challenge for Frontier AI Reasoning Systems.
> arXiv: [2505.11831](https://arxiv.org/abs/2505.11831) · PDF: [chollet-2025-arc-agi-2.pdf](/references/chollet-2025-arc-agi-2.pdf)

**GitHub**: [fchollet/ARC-AGI](https://github.com/fchollet/ARC-AGI)

---

## AI & Software Engineering

### Long-Context Code Synthesis

AILANG's design addresses challenges identified in LLM code generation research.

> **Qiu, J., et al.** (2025). LoCoBench: A Benchmark for Long-Context Large Language Models in Complex Software Engineering.
> arXiv: [2509.09614](https://arxiv.org/abs/2509.09614) · PDF: [qiu-2025-locobench.pdf](/references/qiu-2025-locobench.pdf)

**Key finding**: LLMs struggle with tightly-coupled systems and implicit dependencies. AILANG's explicit effects and module boundaries directly address these failure modes.

**See**: [Why AILANG?](/docs/why-ailang#research-validation-long-context-code-synthesis) for detailed analysis and figures from this research.

---

## Functional Programming Foundations

### Algebraic Laws and Fold Theory

AILANG's iteration primitives obey equational laws enabling safe transformation.

> **Bird, R., & Wadler, P.** (1988). *Introduction to Functional Programming*. Prentice Hall.
> Available: [Oxford CS](https://www.cs.ox.ac.uk/publications/books/functional/)

### Stream Fusion

AILANG's compiler may apply fusion optimizations to eliminate intermediate data structures.

> **Coutts, D., Leshchinskiy, R., & Stewart, D.** (2007). Stream Fusion: From Lists to Streams to Nothing at All. *Proceedings of the 12th ACM SIGPLAN International Conference on Functional Programming (ICFP '07)*, 315–326.
> PDF: [coutts-stream-fusion-2007.pdf](/references/coutts-stream-fusion-2007.pdf)

### Totality and Termination

AILANG aims for total functions with provable termination.

> **Brady, E.** (2013). Idris, a general-purpose dependently typed programming language: Design and implementation. *Journal of Functional Programming*, 23(5), 552–593.
> Documentation: [Idris Totality Checking](https://idris2.readthedocs.io/en/latest/tutorial/theorems.html#totality-checking)

---

## Language Design Influences

### ML Family

AILANG's syntax and semantics draw from the ML tradition.

> **Milner, R., Tofte, M., Harper, R., & MacQueen, D.** (1997). *The Definition of Standard ML (Revised)*. MIT Press.

### Haskell

Effect handling and type class design influenced by Haskell.

> **Marlow, S. (ed.)** (2010). *Haskell 2010 Language Report*.
> PDF: [haskell-2010-report.pdf](/references/haskell-2010-report.pdf)

### Koka Effect Handlers (Tutorial)

> **Leijen, D.** Koka Effect Handlers Tutorial.
> Available: [koka-lang.github.io](https://koka-lang.github.io/koka/doc/book.html#why-effects)

---

## Further Reading

### Textbooks

- **Pierce, B. C.** (2002). *Types and Programming Languages*. MIT Press. — Foundational text on type systems.
- **Harper, R.** (2016). *Practical Foundations for Programming Languages* (2nd ed.). Cambridge University Press.
- **Appel, A. W.** (1992). *Compiling with Continuations*. Cambridge University Press.

### Survey Papers

- **Pretnar, M.** (2015). An Introduction to Algebraic Effects and Handlers.
  PDF: [pretnar-2015-tutorial.pdf](/references/pretnar-2015-tutorial.pdf)

---

## Citing AILANG

### Suggested Citation

When referencing AILANG in academic work, please use the following format:

**Plain text:**

> Edmondson, M. (2025–2026). AILANG: A Deterministic Language for AI Code Synthesis. GitHub repository. https://github.com/sunholo-data/ailang

**BibTeX:**

```bibtex
@software{ailang2025,
  author       = {Edmondson, Mark},
  title        = {{AILANG}: A Deterministic Language for {AI} Code Synthesis},
  year         = {2025--2026},
  url          = {https://github.com/sunholo-data/ailang},
  note         = {Version 0.6.1}
}
```

**Author ORCID:** [0000-0002-8434-3881](https://orcid.org/0000-0002-8434-3881)

**For specific features**, cite the underlying research papers (listed above) alongside AILANG. For example:

> AILANG implements Hindley-Milner type inference (Damas & Milner, 1982) with row-polymorphic effect types inspired by Koka (Leijen, 2014).

### What to Cite

| If referencing... | Cite |
|-------------------|------|
| The language design | This page + relevant foundational papers |
| The implementation | GitHub repository |
| Specific features | Feature documentation + underlying papers |
| Benchmark results | [Evaluation documentation](/docs/guides/evaluation) |

### Future Publication

As AILANG evolves, we plan to publish a comprehensive design paper documenting the complete system. This page will be updated with the formal citation when available.

---

## See Also

- [Architecture Overview](../architecture/index.md) — Internal structure
- [Type System](../architecture/types.md) — Implementation details
- [Why No Loops?](../reference/no-loops.md) — Design rationale with references
- [Design Documents](../design-docs.md) — Detailed feature specifications
