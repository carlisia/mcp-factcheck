# Build stage
FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o mfc ./cmd/server

# Final stage
FROM alpine:latest

# Add MCP server label
LABEL io.modelcontextprotocol.server.name="io.github.carlisia/mcp-factcheck"

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Create non-root user
RUN addgroup -g 1000 -S mcp && \
    adduser -u 1000 -S mcp -G mcp

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/mfc .

# Copy data directory and embeddings
COPY --from=builder /app/data ./data
COPY --from=builder /app/internal/storage/embeddings ./internal/storage/embeddings

# Change ownership
RUN chown -R mcp:mcp /app

# Switch to non-root user
USER mcp

# Expose the default port (if applicable)
# EXPOSE 8080

# Set environment variables
ENV MCP_DATA_DIR=/app/data

# Run the server
ENTRYPOINT ["./mfc"]