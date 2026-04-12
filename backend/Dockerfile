# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install git (needed by some go modules)
RUN apk add --no-cache git

# Download dependencies first for layer caching.
# go.sum must exist locally — run `go mod tidy` if it is missing.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags="-s -w" \
      -o server \
      ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:3.19

# ca-certificates: for HTTPS calls to data.go.kr
# tzdata: for Asia/Seoul timezone (used by the cron scheduler)
RUN apk add --no-cache ca-certificates tzdata

# Run as non-root user
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app

COPY --from=builder /app/server .
# Include SQL migration files so the binary can apply them on startup
COPY --from=builder /app/migrations ./migrations

RUN chown -R app:app /app
USER app

EXPOSE 8080
CMD ["./server"]
