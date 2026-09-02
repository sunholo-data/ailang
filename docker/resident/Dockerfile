# The resident agent image (v6.40.0, RESIDENT-P1 M1).
#
# A thin layer over AILANG's published agent images: they already carry
# debian-bookworm-slim, Node 22, git, the ailang binary and the pi CLI, and
# their `ailang` user is uid 1000 — which matches the uid Cloud Run's gcsfuse
# mount uses, so /agent-home is writable with no change. This adds only what a
# RESIDENT agent needs on top: a supervisor, a health endpoint and a boot path
# that is safe to run on every restart.
#
# Design: docs/design/v6.40.0/resident-agent-instances.md
ARG BASE=europe-west1-docker.pkg.dev/ailang-multivac/ailang/agent-pi:latest
FROM ${BASE}

# herdr pinned by version AND checksum. A supervisor fetched unpinned from the
# internet at build time is a supply chain we do not accept, and the upstream
# install.sh resolves "latest" from a manifest — fine for a laptop, not for an
# image that must rebuild identically. Values from https://herdr.dev/latest.json
# (2026-09-02); bump deliberately, never floatingly.
ARG HERDR_VERSION=0.8.2
ARG HERDR_URL=https://github.com/herdrdev/herdr/releases/download/v0.8.2/herdr-linux-x86_64
ARG HERDR_SHA256=976150a14d490c94b243ea2e1a7eb2dfb67f12e36b182db90936f6728e6aecf4

USER root
RUN curl -fsSL "$HERDR_URL" -o /usr/local/bin/herdr \
    && echo "${HERDR_SHA256}  /usr/local/bin/herdr" | sha256sum -c - \
    && chmod 0755 /usr/local/bin/herdr

COPY boot.sh /usr/local/bin/boot.sh
COPY server.mjs /usr/local/bin/server.mjs
COPY resident-run /usr/local/bin/resident-run
COPY lib/ /usr/local/bin/lib/
RUN chmod 0755 /usr/local/bin/boot.sh /usr/local/bin/server.mjs /usr/local/bin/resident-run

USER ailang

# The socket path is set EXPLICITLY rather than inherited from $HOME. herdr
# 0.8.2 ignores both HOME and XDG_CONFIG_HOME when resolving it, so leaving it
# implicit means the path depends on which user the container happens to run
# as — and HERDR_SOCKET_PATH is the only supported lever (Phase 0 §6).
ENV HERDR_SOCKET_PATH=/home/ailang/.herdr/herdr.sock \
    RESIDENT_PORT=8080 \
    AGENT_HOME=/agent-home \
    AILANG_FS_SANDBOX=/workspace \
    PI_HOME=/home/ailang/.pi \
    PROGRAM_ALLOWLIST_FILE=/etc/resident/allowlist.json

EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/boot.sh"]
