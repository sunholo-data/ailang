# M0 spike image for RESIDENT-P1 (v6.40.0). NOT the production resident image —
# this exists to answer two gate questions and is expected to be deleted:
#   1. does herdr-server run with no controlling terminal at all?
#   2. does the shared-vCPU burst budget carry a real compile?
#
# Design: docs/design/v6.40.0/resident-agent-instances.md
FROM debian:bookworm-slim

ARG GO_VERSION=1.26.4
# herdr pinned by version AND checksum, per sprint M1: a supervisor fetched
# unpinned from the internet at build time is a supply chain we do not accept.
# Values from https://herdr.dev/latest.json, 2026-09-02.
ARG HERDR_VERSION=0.8.2
ARG HERDR_URL=https://github.com/herdrdev/herdr/releases/download/v0.8.2/herdr-linux-x86_64
ARG HERDR_SHA256=976150a14d490c94b243ea2e1a7eb2dfb67f12e36b182db90936f6728e6aecf4

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl git python3 procps bc \
    && rm -rf /var/lib/apt/lists/*

# Go, pinned to the same version as the comparison laptop so the wall-clock
# numbers are comparable rather than merely suggestive.
RUN curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" -o /tmp/go.tgz \
    && tar -C /usr/local -xzf /tmp/go.tgz && rm /tmp/go.tgz
ENV PATH=/usr/local/go/bin:/usr/local/bin:$PATH

RUN curl -fsSL "$HERDR_URL" -o /usr/local/bin/herdr \
    && echo "${HERDR_SHA256}  /usr/local/bin/herdr" | sha256sum -c - \
    && chmod +x /usr/local/bin/herdr

# Node, because the agent CLIs we intend to host (Claude Code, Codex) are Node
# processes and their heap sizing is the thing under test.
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && apt-get install -y --no-install-recommends nodejs && rm -rf /var/lib/apt/lists/*

COPY m0-bench.sh /usr/local/bin/m0-bench.sh
COPY mem-probe.sh /usr/local/bin/mem-probe.sh
RUN chmod +x /usr/local/bin/m0-bench.sh /usr/local/bin/mem-probe.sh

ENV PORT=8080
ENTRYPOINT ["/usr/local/bin/m0-bench.sh"]
