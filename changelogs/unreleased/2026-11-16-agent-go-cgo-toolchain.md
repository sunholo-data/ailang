### Fixed — agent-go executor image: make and a C toolchain for the cloud sprint-evaluator (2026-11-16)

D-FLEET-41 A (Mark, attended). `docker/Dockerfile.agent-go` now installs `make`, `gcc` and
`libc6-dev`, the same three packages #1753 gave the codex-go image. The cloud
sprint-evaluator has run on agent-go since 10-10, and at least 4 of its 8 logged runs hit a
missing `make` or C compiler; one (task-68e3cfb1) scored without the CGO tests at all.
`Dockerfile.agent-pi-go` is unchanged (0 evaluator starts since 10-10).
