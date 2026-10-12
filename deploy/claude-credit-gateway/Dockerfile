# Optional minimal service image. Preferred production route reuses the verified
# coordinator buildpack image and passes credit-gateway serve as Cloud Run args.
# Build context MUST be the ailang repository root.
FROM golang:1.26.9 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/ailang ./cmd/ailang
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ailang /ailang
ENTRYPOINT ["/ailang"]
CMD ["credit-gateway", "serve"]
