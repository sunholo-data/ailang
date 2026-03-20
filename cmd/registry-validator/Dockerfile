# AILANG Registry Validator — Cloud Run service
# Builds both ailang CLI (for check/verify) and the validator HTTP server.

FROM golang:1.24 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -o ailang ./cmd/ailang/
RUN CGO_ENABLED=0 go build -o registry-validator ./cmd/registry-validator/

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y z3 ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/ailang /usr/local/bin/
COPY --from=builder /app/registry-validator /usr/local/bin/
# Bundle stdlib for ailang check/verify
COPY std/ /usr/local/share/ailang/std/

EXPOSE 8080
ENV PORT=8080
CMD ["registry-validator"]
