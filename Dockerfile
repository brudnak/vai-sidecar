# ── build stage ─────────────────────────────────────────────
FROM --platform=linux/amd64 golang:1.24.3 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /snapshot

# ── final (debug-friendly) stage ────────────────────────────
FROM alpine:3.19
RUN apk add --no-cache ca-certificates curl jq
COPY --from=builder /snapshot /snapshot
EXPOSE 8080
ENTRYPOINT ["/snapshot"]
