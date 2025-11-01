# =============================
# Build stage
# =============================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy the entire source code
COPY . .

# Create temporary directory for pdfcpu (distroless can't create it later)
RUN mkdir -p /tmp

# Build the application with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always 2>/dev/null || echo 'dev')" \
    -a -installsuffix cgo \
    -o dealsense-manager \
    ./cmd/server

# =============================
# Final stage - distroless
# =============================
FROM gcr.io/distroless/static-debian12

# Copy certificates and timezone data
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy pre-created /tmp directory (distroless cannot RUN mkdir)
COPY --from=builder /tmp /tmp

# Set working directory
WORKDIR /app

# Copy the compiled binary and required config files
COPY --from=builder /app/dealsense-manager .
COPY --from=builder /app/genai-exchange-475318-84af2a035486.json .
COPY --from=builder /app/config.yaml .

# Switch to non-root user for security
USER 65534:65534

# Expose port
EXPOSE 8001

# Run the binary
CMD ["/app/dealsense-manager"]
