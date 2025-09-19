.PHONY: all clean hm-client hm-server

# Default target
all: hm-client hm-server

# Variables
GO_FILES := $(shell find . -name "*.go" -type f)
CLIENT_FILES := $(shell find ./cmd/hm-client ./client ./internal -name "*.go" -type f 2>/dev/null || true)
SERVER_FILES := $(shell find . -name "*.go" -type f -not -path "./cmd/hm-client/*")

# Build hm-client binary
bin/hm-client: $(CLIENT_FILES) go.mod go.sum
	@mkdir -p bin
	go build -o bin/hm-client ./cmd/hm-client

# Build hm-server binary (main.go)
bin/hm-server: $(SERVER_FILES) go.mod go.sum
	@mkdir -p bin
	go build -o bin/hm-server .

# Convenience targets
hm-client: bin/hm-client

hm-server: bin/hm-server

# Clean build artifacts
clean:
	rm -rf bin/