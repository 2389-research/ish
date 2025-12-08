# ABOUTME: Multi-stage Dockerfile for ISH - builds Go binary and creates minimal runtime image
# ABOUTME: Stage 1 builds the binary, Stage 2 creates the final lightweight container

# Stage 1: Build the Go binary
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with optimizations
# -ldflags="-s -w" strips debug info for smaller binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o ish ./cmd/ish

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Create non-root user for security
RUN addgroup -g 1000 ish && \
    adduser -D -u 1000 -G ish ish

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/ish /app/ish

# Create data directory for SQLite database
RUN mkdir -p /app/data && \
    chown -R ish:ish /app

# Switch to non-root user
USER ish

# Expose default port
EXPOSE 9000

# Default command: serve with database in /app/data
# Users can override with docker run commands
CMD ["./ish", "serve", "--db", "/app/data/ish.db", "--port", "9000"]
