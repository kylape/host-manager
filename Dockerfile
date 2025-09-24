# Build stage
FROM quay.io/centos/centos:stream9 AS builder

# Install Go
RUN dnf update -y && dnf install -y golang git && dnf clean all

WORKDIR /app

# Copy go mod files first for better layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/hm-server .

# Final stage
FROM quay.io/centos/centos:stream9

# Install ca-certificates for HTTPS requests
RUN dnf update -y && dnf install -y ca-certificates && dnf clean all

WORKDIR /app

# Create a non-root user
RUN useradd -r -u 1001 hostmanager

# Copy the binary from builder stage
COPY --from=builder /app/bin/hm-server .

# Change ownership to non-root user
RUN chown hostmanager:hostmanager /app/hm-server

# Switch to non-root user
USER hostmanager

# Expose port (adjust if needed)
EXPOSE 8080

# Run the binary
CMD ["./hm-server"]