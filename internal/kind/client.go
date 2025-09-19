package kind

import (
	"fmt"
	"os/exec"
	"strings"
)

// Client wraps kind CLI operations
type Client struct{}

// NewClient creates a new kind client
func NewClient() *Client {
	return &Client{}
}

// CreateCluster creates a new kind cluster
func (c *Client) CreateCluster(name string, withRegistry bool) error {
	var config string
	if withRegistry {
		config = c.getClusterConfigWithRegistry()
	} else {
		config = c.getBasicClusterConfig()
	}

	cmd := exec.Command("kind", "create", "cluster", "--name", name, "--config", "-")
	cmd.Stdin = strings.NewReader(config)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create cluster %s: %w\nOutput: %s", name, err, string(output))
	}

	// Connect to registry if it exists and this cluster should use it
	if withRegistry {
		if err := c.connectToRegistry(name); err != nil {
			return fmt.Errorf("failed to connect cluster to registry: %w", err)
		}
	}

	return nil
}

// DeleteCluster deletes a kind cluster
func (c *Client) DeleteCluster(name string) error {
	cmd := exec.Command("kind", "delete", "cluster", "--name", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete cluster %s: %w\nOutput: %s", name, err, string(output))
	}
	return nil
}

// ListClusters returns a list of kind clusters
func (c *Client) ListClusters() ([]string, error) {
	cmd := exec.Command("kind", "get", "clusters")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	clusters := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(clusters) == 1 && clusters[0] == "" {
		return []string{}, nil
	}

	return clusters, nil
}

// GetKubeconfig returns the kubeconfig for a cluster
func (c *Client) GetKubeconfig(name string) (string, error) {
	cmd := exec.Command("kind", "get", "kubeconfig", "--name", name)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get kubeconfig for %s: %w", name, err)
	}
	return string(output), nil
}

// CreateRegistry creates the shared container registry
func (c *Client) CreateRegistry() error {
	// Check if registry already exists
	cmd := exec.Command("podman", "inspect", "kind-registry")
	if cmd.Run() == nil {
		// Registry already exists, check if it's running
		cmd = exec.Command("podman", "inspect", "-f", "{{.State.Running}}", "kind-registry")
		output, err := cmd.Output()
		if err == nil && strings.TrimSpace(string(output)) == "true" {
			return nil // Registry is already running
		}

		// Start existing registry
		cmd = exec.Command("podman", "start", "kind-registry")
		return cmd.Run()
	}

	// Create new registry
	cmd = exec.Command("podman", "run",
		"-d", "--restart=always",
		"-p", "127.0.0.1:5001:5000",
		"--network", "bridge",
		"--name", "kind-registry",
		"registry:2")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create registry: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// LoadImage loads a Docker image into a cluster
func (c *Client) LoadImage(clusterName, imageName string) error {
	cmd := exec.Command("kind", "load", "docker-image", imageName, "--name", clusterName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to load image %s into cluster %s: %w\nOutput: %s", imageName, clusterName, err, string(output))
	}
	return nil
}

// getClusterConfigWithRegistry returns kind config that connects to the shared registry
func (c *Client) getClusterConfigWithRegistry() string {
	return `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry]
    config_path = "/etc/containerd/certs.d"
kubeadmConfigPatches:
- |
  apiVersion: kubeadm.k8s.io/v1
  kind: ClusterConfiguration
  metadata:
    name: config
  kubernetesVersion: "v1.32.0"
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 32222
    hostPort: 2222
  extraMounts:
  - containerPath: /local
    hostPath: /root/kind`
}

// getBasicClusterConfig returns a basic kind config
func (c *Client) getBasicClusterConfig() string {
	return `kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
kubeadmConfigPatches:
- |
  apiVersion: kubeadm.k8s.io/v1
  kind: ClusterConfiguration
  metadata:
    name: config
  kubernetesVersion: "v1.32.0"
nodes:
- role: control-plane`
}

// connectToRegistry connects a cluster to the shared registry
func (c *Client) connectToRegistry(clusterName string) error {
	// Get cluster nodes
	cmd := exec.Command("kind", "get", "nodes", "--name", clusterName)
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get cluster nodes: %w", err)
	}

	nodes := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Configure registry for each node
	for _, node := range nodes {
		if node == "" {
			continue
		}

		// Create registry config directory
		cmd = exec.Command("podman", "exec", node, "mkdir", "-p", "/etc/containerd/certs.d/localhost:5001")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create registry config dir in node %s: %w", node, err)
		}

		// Write registry config
		config := "[host.\"http://kind-registry:5000\"]"
		cmd = exec.Command("podman", "exec", "-i", node, "cp", "/dev/stdin", "/etc/containerd/certs.d/localhost:5001/hosts.toml")
		cmd.Stdin = strings.NewReader(config)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to write registry config in node %s: %w", node, err)
		}
	}

	// Connect registry to cluster network
	cmd = exec.Command("podman", "network", "connect", "kind", "kind-registry")
	cmd.Run() // Ignore errors - might already be connected

	return nil
}