package state

import "time"

// HostState represents the current state of the host system
type HostState struct {
	Initialized      bool                   `json:"initialized"`
	InitializedAt    *time.Time             `json:"initialized_at,omitempty"`
	InstanceType     string                 `json:"instance_type,omitempty"`
	StorageType      string                 `json:"storage_type,omitempty"`     // "nvme", "ebs-only"
	StorageDevice    string                 `json:"storage_device,omitempty"`   // "/dev/nvme1n1"
	PackagesInstalled bool                  `json:"packages_installed"`
	BaseClusterReady bool                   `json:"base_cluster_ready"`
	RegistryRunning  bool                   `json:"registry_running"`
	Clusters         map[string]ClusterInfo `json:"clusters"`
}

// ClusterInfo represents information about a kind cluster
type ClusterInfo struct {
	Status    string     `json:"status"`    // "running", "stopped", "error"
	Created   *time.Time `json:"created,omitempty"`
	Type      string     `json:"type"`      // "infrastructure", "development"
	KubeVirt  bool       `json:"kubevirt"`  // whether cluster has KubeVirt enabled
}

// StorageConfig represents storage configuration for the host
type StorageConfig struct {
	HasNVMe bool   `json:"has_nvme"`
	Device  string `json:"device,omitempty"`
	Type    string `json:"type"` // "instance-store", "ebs-only"
}

// ClusterCreateRequest represents a request to create a new cluster
type ClusterCreateRequest struct {
	Name     string `json:"name"`
	KubeVirt bool   `json:"kubevirt,omitempty"`
}

// ClusterResponse represents a cluster in API responses
type ClusterResponse struct {
	Name     string     `json:"name"`
	Status   string     `json:"status"`
	Created  *time.Time `json:"created,omitempty"`
	Type     string     `json:"type"`
	KubeVirt bool       `json:"kubevirt"`
}

// RegistryStatus represents the status of the container registry
type RegistryStatus struct {
	Running bool   `json:"running"`
	Port    int    `json:"port"`
	URL     string `json:"url"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string `json:"status"`
	Initialized bool   `json:"initialized"`
	Version     string `json:"version"`
}