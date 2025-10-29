# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o kubeforge ./cmd/kubeforge-server

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates openssh-client

# Create app user
RUN addgroup -g 1000 kubeforge && \
    adduser -D -u 1000 -G kubeforge kubeforge

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/kubeforge .

# Create data directory for SQLite
RUN mkdir -p /app/data && chown -R kubeforge:kubeforge /app

# Switch to non-root user
USER kubeforge

# Expose port
EXPOSE 8080

# Set environment variables
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080
ENV DB_DRIVER=sqlite
ENV DB_DSN=/app/data/kubeforge.db
ENV LOG_LEVEL=info
ENV LOG_FORMAT=json

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/healthz || exit 1

# Run the application
ENTRYPOINT ["./kubeforge"]
