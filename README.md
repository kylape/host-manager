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

## State File

After initialization, host-manager creates `/etc/host-manager-state.json` to track system state:

```json
{
  "initialized": true,
  "initialized_at": "2024-01-15T10:30:00Z",
  "instance_type": "m5.xlarge",
  "storage_type": "nvme",
  "storage_device": "/dev/nvme1n1",
  "packages_installed": true,
  "base_cluster_ready": true,
  "registry_running": true,
  "clusters": {
    "kind": {
      "status": "running",
      "created": "2024-01-15T10:30:00Z",
      "type": "infrastructure",
      "kubevirt": false
    }
  }
}
```

This state file prevents re-initialization on subsequent runs and tracks:

* Host initialization status and timestamp
* EC2 instance type and storage configuration
* Package installation status
* Base infrastructure cluster status
* Local registry status
* All managed Kind clusters with their configuration
