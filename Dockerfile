# Build stage
FROM --platform=linux/amd64 golang:latest AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY main.go .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o vai-sidecar main.go

# Final stage
FROM --platform=linux/amd64 alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/vai-sidecar /usr/local/bin/
EXPOSE 8080
CMD ["/usr/local/bin/vai-sidecar"]