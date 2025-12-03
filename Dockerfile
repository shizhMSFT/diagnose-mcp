# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o diagnose-mcp ./cmd/diagnose-mcp

# Runtime stage
FROM node:25-alpine

# Install Node.js and npm (already included in node image)
# npx comes with npm, so no additional installation needed

# Copy the binary from builder
COPY --from=builder /build/diagnose-mcp /usr/local/bin/diagnose-mcp

# Set working directory
WORKDIR /workspace

# Make diagnose-mcp executable
RUN chmod +x /usr/local/bin/diagnose-mcp

# Default command
ENTRYPOINT ["diagnose-mcp"]
CMD ["--help"]
