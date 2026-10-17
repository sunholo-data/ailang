# M-TYPED-MESSAGE-PLANE: typed, budgeted messages, without disturbing the plane that runs today

**Status**: Planned (r4: M1 corrected after Mark pointed out that the cascade already runs on the cloud plane (#1730 filed); Quorum guardrail spent: r1 and r2 both BLOCKED; every r2 objection is dispositioned below with new evidence rows. The sprint-planner re-verifies at plan time. Design Freeze open for Mark)
**Target**: v1.0.0, clause 4 (the orchestration half of the 1.0 claim)
**Priority**: P0 for v1.0
**Estimated**: ~2–3 weeks across M0–M5; each milestone ships and can be rolled back on its own
**Dependencies**: [m-json-codecs](m-json-codecs.md) (AILANG type ↔ JSON; needed by M3) · [M-CLI-MSG-HANDLER](../v0_38_6/m-cli-msg-handler.md) (designed, not built; it is M2 here) · [m-message-plane-trust](../m-message-plane-trust.md) (in progress; this doc adds no new seam to it)
**Created**: 2026-10-09 · **Author**: attended session with Mark (Claude Opus 5.5)
**Quorum trigger**: #1 (Design Freeze) and #2 (it extends shared machinery that every lane uses) → the quorum runs.

**Mark's constraint, 2026-10-09:** *"typing the message plane would be awesome … if we don't disrupt
what is happening today."* Every milestone below is **additive**. An untyped message stays valid
forever, no sender has to change, and routing does not move.

---

## What exists today (verified 2026-10-09)

- **The plane runs and is used.** In 7 days it carried 759 messages: 61 handoffs, 99 approval
  requests and 127 completions, with 98 tasks completed (`ailang messages activity --hours 168`, V1).
  Flows include the design → plan → execute → evaluate chain, Daneel's mail flows, feedback →
  `pkg:` agent → PR → approval, and triage.
- **A message is a row with a string payload.** `InboxMessage` (`internal/messaging/inbox.go:17-42`)
  has `MessageType` (a string), `Title`, `Payload` (a string), `CorrelationID`, `ParentTaskID`,
  `ChainID` and `Iteration`. `--type` accepts any string (V2).
- **One typed envelope exists, and it rides inside the payload.** `ailang.package-message/v1`
  (`internal/messaging/pkg_schema.go:13`) has 11 kinds and is validated by `ValidatePackageMessage`
  (`:113`). It is converted to an `InboxMessage` with `MessageType: notification` and the envelope as
  JSON in `Payload` (`pkg_schema.go:269-271`) (V3).
- **That envelope has no live traffic on the prod plane.** None of the last 3,000 prod messages,
  back to 2026-08-31, carries it (V4). The cause: `ailang publish` emits it through
  `openPkgMsgStore()`, which opens the **publisher's local SQLite** (`cmd/ailang/pkg_msg.go:266-268`),
  not the prod Firestore plane (V5).
- **Completions are parsed from text.** Handoffs are read from `PR_URL:` / `VERSION:` output markers,
  and handoff topology comes from `config.cloud.yaml`, *"never from model output"* (V6).
- **AILANG programs cannot use the plane through the effect system.** `std/cognition`
  `sendMsg(string, string)` / `recvMsg(string)` are typed records behind `! {Msg}`, with
  `budget_remaining`, but no CLI handler exists: `--caps Msg` fails with `ErrNoMsgHandler` (#1130).
  Daneel therefore shells out to `ailang messages send` under a process allowlist, which its own docs
  call "outside the effect system" (V7).
- **The SQLite store constrains `message_type`** with a CHECK list (`internal/messaging/schema.go:384`).
  Firestore has no constraint (V8).

## The design: a schema id beside the payload, shadow first, enforce per kind

**Two new optional fields; nothing existing changes.**
- `InboxMessage.Schema string` (`json:"schema,omitempty"`) names a type, e.g. `ailang.completion/v1`.
- `InboxMessage.Body string` (`json:"body,omitempty"`) holds that type's JSON value.
- **`Payload` is untouched.** It keeps carrying exactly the text it carries today, including the
  `PR_URL:` / `VERSION:` markers that `internal/coordinator/stage_execution.go:120` parses through
  `GetEffectiveOutputMarkers()` (V6). A typed message therefore never changes what a line-based
  marker scanner sees.
- `MessageType` keeps its meaning and its values, so the SQLite CHECK constraint is untouched.

| State | Meaning |
|---|---|
| No `Schema` / `Body` (every message today) | Untyped; valid forever; handled exactly as now |
| `Schema` + `Body` | `Body` is JSON that must match the named type; `Payload` still carries its text |

**Where a schema is defined.** It is an AILANG type in a published package (`sunholo/message_schemas`,
F1), so it gets the package system's v2 interface hash, versioning and cascade for free (V9). The
message-schema registry maps a schema id to that type. **Validation uses existing code only:**
1. The registered type is mapped to a JSON Schema by the existing
   `internal/apiserver/schema.TypeToSchema` (V10), extended for named ADTs by [m-json-codecs](m-json-codecs.md).
2. That schema is resolved and checked with `github.com/google/jsonschema-go` v0.4.3, which is
   already a direct dependency in `go.mod`: `(*Schema).Resolve` then `(*Resolved).Validate` (V11).

That gives no third mapper and no new validation engine. The coarse duplicate in `mcp.go:582` is
retired onto the same mapper (M5).

**Validation modes, per schema id**, set in the plane's config:

| Mode | On a mismatch | Use |
|---|---|---|
| `off` | nothing | Default for any new schema id |
| `shadow` | the message is delivered unchanged, and the mismatch is recorded (count, sample, sender) in `messages health` | Proving real traffic is clean |
| `enforce` | the send is refused with a typed error naming the field | Only after a clean shadow window (F3) |

Rollback is flipping a kind back to `shadow` or `off`; no data migration runs in either direction.

## Milestones (each additive and independently shippable)

**M0: Field and registry, in shadow** (~2 days)
- Add the `Schema` and `Body` fields to both stores. Firestore needs nothing. SQLite gets two additive, nullable
  columns through the existing migration path. No CHECK change.
- Add the registry and the three validation modes.
- Accept: every existing message round-trips unchanged; one test per mode.

**M1: One package envelope** (~2 days)

*Corrected 2026-10-09 (Mark):* the package cascade **already** reaches the cloud plane.
- `publish` sends a typed `CascadeEnvelopeFields` on the IAM-restricted `ailang-cascade` topic, and
  only its legacy inbox copy goes to local SQLite (V13).
- The earlier M1 ("send package messages to the plane behind a switch") was wrong and is removed.

What remains is **duplication**:
- Two package envelopes describe the same event: `PackageMessageEnvelope` (`ailang.package-message/v1`,
  inbox, local) and `CascadeEnvelopeFields` (Pub/Sub, cloud).
- Their change-class vocabularies differ (local `patch` / `minor` / `major` / `U`, mapped by
  `mapChangeClassToSchema` to the cascade's A/B/C).
- **The fix:** make `CascadeEnvelopeFields` the one schema, registered as `ailang.cascade/v1` in
  this doc's registry. The legacy inbox copy is derived from it rather than built separately.
- **Depends on #1730**, whose fix decides whether the legacy row moves to the cloud store or the
  adapter dispatches from the envelope.
- **Acceptance:** one builder, both outputs; a golden test that today's cascade envelope bytes are
  unchanged.

**M2: `Msg` runs from the CLI** (this is [M-CLI-MSG-HANDLER](../v0_38_6/m-cli-msg-handler.md), ~2–3 days)
- `sendMsg` / `recvMsg` against the configured plane, with the Msg capability and budget.
- A program's sends are then inside its effect row, its `run --policy` admission and its `@limit`.
- Daneel's shell-out path keeps working; moving onto `Msg` is Daneel's choice.

**M3: Typed send and receive in AILANG** (~3 days; needs [m-json-codecs](m-json-codecs.md))
- `sendTyped[a](to: string, schemaId: string, c: JsonCodec[a], v: a) -> Result[…, MsgError] ! {Msg}`
  encodes `v` with `c.encode` into `Body` and sets `Schema`.
- `recvTyped[a](inbox: string, schemaId: string, c: JsonCodec[a]) -> Result[…, MsgError] ! {Msg}`
  refuses a message whose `Schema` differs, and decodes `Body` with `c.decode`.
- Codecs are explicit values, so no type-directed resolution is needed (m-json-codecs V3–V5).
- The type checker checks that the sender's value matches the codec; the plane's validator (M0)
  checks it matches the registered schema id.

**M4: Coordinator kinds go typed, by dual-write, with a loud cross-check** (~3 days)
- The coordinator emits `completion`, `handoff` and `approval_request` with `Schema` + `Body` set
  (`ailang.completion/v1`, …). `Payload` and its markers are unchanged.
- **Consumers read both and compare. Nothing falls back silently:**
  - When `Body` is present, the consumer reads the typed field **and** parses the markers.
  - Agreement increments `typed_marker_agree`, and disagreement increments `typed_marker_disagree`.
    Both are counted per kind in `messages health`, and each disagreement is logged with the message
    id and both values.
  - On disagreement the consumer uses the **markers**, which are today's behaviour, so routing cannot
    change, and the disagreement is visible.
  - `typed_reads` / `marker_only_reads` counters record which path each consumer ran.
- **Kind promotion:** a kind moves `shadow` → `enforce` after a clean window (F3). The later,
  separate ruling to retire markers requires zero disagreements and zero marker-only reads for that
  kind over the window, which is evidence these counters collect.

**M5: One schema mapper** (~1 day)
- `mcp.go`'s `ailangTypeToJSONSchema` (maps every non-primitive to `"string"`, V10) delegates to
  `schema.TypeToSchema`. That removes a duplicate and gives MCP tools typed parameter schemas. MCP
  clients see richer `inputSchema`s, so this ships behind the same shadow-first rule: compare both
  mappers' output on every exported function in CI before switching.

## Non-disruption guarantees (each is an acceptance test)

1. A message with no `Schema` is stored, routed and delivered exactly as before (golden test on the M0 branch).
2. No `MessageType` value is added, removed or renamed; the SQLite CHECK constraint is unchanged.
3. `config.cloud.yaml` routing, `trigger_on_complete` and `auto_approve_handoff_to` are unchanged.
4. Existing CLI sends (`ailang messages send …`) behave identically; `--schema` is a new, optional flag.
5. Marker-based completion parsing keeps working while M4 dual-writes: `Payload` bytes are
   identical for typed and untyped messages (golden test), and on any disagreement the markers win.
6. Every enforcement is per schema id and reversible by config, with no deploy.

## Out of scope (v1.1)

- **Declared contracts per agent or package** (which kinds an inbox accepts and emits, checked at
  registry load), so the routing graph itself type-checks. Designed as
  [m-package-protocol-manifests](../m-package-protocol-manifests.md); it has an open authority question.
- **Retiring text markers.** That is a separate ruling after M4's shadow data.

## Design Freeze

- [ ] **F1. Where schemas live:** recommend a published package, `sunholo/message_schemas`, so
  schemas inherit interface hashing and the cascade. The alternative is `std/messages`, which is
  versioned with the binary.
- [ ] **F2. The typed API shape (M3):** recommend explicit `schemaId` strings bound to a type in the
  package, for v1. Inferring the id from the type needs type-directed elaboration and can follow.
- [ ] **F3. The clean-window rule for `shadow` → `enforce`:** recommend 7 days with zero mismatches
  for that kind on the prod plane, reported by `messages health`.


## Changes since r1 (quorum BLOCKED 3/3, 2026-10-09)

- **gemini (M4's fallback is silent):** M4 now reads both paths, counts agreement and disagreement
  per kind, logs each disagreement, and uses today's markers on conflict.
- **glm (markers and JSON cannot share one payload; no read-path evidence):** the typed value moves
  to a new `Body` field, so `Payload` and its markers are byte-identical. Read-path counters supply
  the evidence a marker retirement would need. The parser is cited (V6).
- **kimi (a mapper is not a validator):** validation uses `google/jsonschema-go`'s
  `Resolve` / `Validate`, already a dependency (V11).

## Round-2 objections and dispositions (quorum r2 BLOCKED, 2026-10-09; guardrail spent)

*Superseded 2026-10-09: the M1 these two dispositions answer was removed (the cascade already reaches the plane; see the new M1). They are kept as history.*

- **glm (M1 not additive; local readers would starve; prod notification consumers change):** M1
  keeps the local write (its reader is cited, V12). The plane write is behind a default-off switch
  (guarantee 7), and enabling it is ruling F4, which names the consequence: package agents wake.
- **kimi (M1 enforce from day one, with an unproven validator swap):** the hand-written validator
  stays authoritative. The registry validator runs in shadow with per-kind agreement counts and is
  promoted only after the clean window, the same gate M5 uses.

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | 0 | Delivery semantics unchanged |
| A2 Replayability | +1 | Typed payloads in traces replay as values, not strings |
| A3 Effect Legibility | +1 | M2 puts sends inside the effect row |
| A4 Explicit Authority | +1 | M2 puts sends under `run --policy` and budgets |
| A5 Bounded Verification | +1 | Decoding against a declared type is local and total |
| A6 Safe Concurrency | 0 | No concurrency change |
| A7 Machines First | +1 | Agents read typed fields instead of parsing `PR_URL:` text |
| A8 Minimal Syntax | 0 | No syntax; APIs and one optional field |
| A9 Cost Visibility | +1 | Message budgets (`budget_remaining`) become enforced on the real plane |
| A10 Composability | +1 | Schemas are packages; one mapper serves the API server, MCP and messages |
| A11 Structured Failure | +1 | Mismatches are typed errors naming the field |
| A12 System Boundary | +1 | The edge between agents becomes a checked boundary |

**Net +9.** No −1.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| A typed kind's real traffic doesn't match its schema | That is what `shadow` is for; enforcement waits for a clean window |
| SQLite migration on rigs with old schemas | Two additive nullable columns through the existing migration path; covered by a migration test like `schema_migration_v190_test.go` |
| MCP clients react to richer schemas (M5) | CI diff of both mappers on every export before switching; schema stays a superset |
| `m-message-plane-trust` is mid-flight | No milestone here touches the seams it is fixing; M0 is a field and a registry |

## Verification Log (2026-10-09, `origin/dev` `c92739681`)

| # | Claim | Evidence |
|---|---|---|
| V1 | Volume and live flows | `ailang messages activity --hours 168` (research pass, 2026-10-09): 759 messages, 61 handoffs, 99 approval requests, 127 completions; 98 completed / 8 failed / 6 blocked |
| V2 | Message row shape | `internal/messaging/inbox.go:17-42`; `cmd/ailang/messages_send.go:43` (`--type` any string) |
| V3 | The package envelope rides in the payload | `internal/messaging/pkg_schema.go:13` (`PackageMessageSchema = "ailang.package-message/v1"`), `:18-29` (11 kinds), `:113` (`ValidatePackageMessage`), `:269-271` (`MessageType: InboxTypeNotification`, `Payload: payload`) |
| V4 | No prod traffic carries the envelope | Research pass 2026-10-09: 0 of the last 3,000 prod messages (back to 2026-08-31) |
| V5 | Publish emits to the local store | `cmd/ailang/pkg_publish.go:310` `openPkgMsgStore()`; `cmd/ailang/pkg_msg.go:266-268` → `messaging.OpenStore(messaging.GetDefaultDatabasePath())` |
| V6 | Completions are text markers; topology is config | `internal/coordinator/stage_execution.go:120` (`effectiveMarkers := agent.GetEffectiveOutputMarkers()`), `agent_registry_effective.go:67`; `config.cloud.yaml:372` ("handoff topology comes from this registry and never from model output") |
| V7 | `Msg` has no CLI handler | `ailang docs std/cognition`: `sendMsg(string, string) -> { msg_id, clock, budget_remaining } ! {Msg}`; #1130 `ErrNoMsgHandler`; `design_docs/planned/v0_38_6/m-cli-msg-handler.md`; Daneel `authority.md:166-180` |
| V8 | SQLite CHECK on `message_type` | `internal/messaging/schema.go:384` |
| V9 | Packages give hashing and the cascade | `internal/pkg/hasher_v2.go` (`sha256:ifacev2:` over function names, types and effects); change classes A/B/C drive the cascade |
| V13 | The cascade already reaches the cloud plane | `cmd/ailang/pkg_publish.go` ~455–510 (the legacy inbox via `EmitUpgradeAvailable(store, …)`, then the "Cascade-topic publish (M-PKG-AUTONOMOUS-CASCADE-SAFE M2) … the authoritative, IAM-restricted path that agents act on" with `CascadeEnvelopeFields`); `internal/pubsub/publisher.go:70-100`; topic `ailang-cascade`; #1730 |
| V12 | The local package-message store has a reader | `cmd/ailang/pkg_msg.go:166` (`store, err := openPkgMsgStore()` in the `pkg msg` command) |
| V11 | A JSON Schema validator is already a dependency | `go.mod`: `github.com/google/jsonschema-go v0.4.3`; module source `jsonschema/resolve.go:152` `func (root *Schema) Resolve(opts *ResolveOptions) (*Resolved, error)`, `jsonschema/validate.go:37` `func (rs *Resolved) Validate(instance any) error`; used today for MCP tool `InputSchema` (`internal/apiserver/feedback_tool.go:51`) |
| V10 | Two type → schema mappers exist | `internal/apiserver/schema/schema.go:64` `TypeToSchema` (rich); `internal/apiserver/mcp.go:581-582` `ailangTypeToJSONSchema` (non-primitive → `"string"`) |

---

**Document created**: 2026-10-09
