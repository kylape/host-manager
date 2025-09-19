# Host Manager

Unified host management service for EC2-based development environments.

## Features

- **Auto-initialization**: Complete host setup on first run
- **Kind cluster management**: HTTP API for multiple development clusters
- **Smart storage**: Automatic NVMe detection and configuration
- **State persistence**: Tracks what's been initialized

## Quick Start

```bash
# Deploy to fresh EC2 host
scp host-manager ec2-host:/usr/local/bin/
ssh ec2-host 'sudo /usr/local/bin/host-manager'
```

The binary will:
1. Detect it's running on a fresh host
2. Automatically configure storage, packages, and base infrastructure
3. Start HTTP server on port 8080 for cluster management

## API Usage

```bash
# Create new development cluster
curl -X POST http://localhost:8080/clusters \
  -H "Content-Type: application/json" \
  -d '{"name": "my-dev-cluster"}'

# Get kubeconfig
curl http://localhost:8080/clusters/my-dev-cluster/kubeconfig

# List clusters
curl http://localhost:8080/clusters

# Delete cluster
curl -X DELETE http://localhost:8080/clusters/my-dev-cluster
```

## Build

```bash
# Build the main server
go build -o host-manager .

# Build the client CLI
go build -o hm-client ./cmd/hm-client
```