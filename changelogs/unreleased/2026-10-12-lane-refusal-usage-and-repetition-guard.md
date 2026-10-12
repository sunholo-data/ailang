### Fixed — lane CLI refusals show the correct call; motoko's ailang_only lane breaks repeat loops

- `ailang policy-tool` argument refusals now end with the right request shape, e.g. `op check
  requires path — call it as {"op":"check","path":"<in-sandbox path>"}`. They used to name only
  the problem: on 2026-10-12, GLM-5.3-Flash on motoko sent `{op: "check", module: …}` 83 times and
  `{op: "test", module: …}` 70 times on one real task.
- `docker/Dockerfile.agent-motoko` pins motoko_agent `a1ece42e`, whose `ailang_only` profile loads
  `repetition_guard`. A call repeated with byte-identical output is refused, and the model is told
  to change course.
- The motoko pin moves to `6de44ada`, where `repetition_guard` compares results without the
  envelope's `tool_call_id`. Before this it could never fire on extension tools: the 215-repeat run
  above had the guard loaded the whole time. Upstream: arniwesth/motoko_agent#260.
