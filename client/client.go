package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/kylape/host-manager/internal/state"
)

// Client provides a client interface to the host manager HTTP API
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewClient creates a new host manager client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "http://host.docker.internal:8080"
	}

	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Health checks the service health
func (c *Client) Health() (*state.HealthResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("failed to get health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	var health state.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return &health, nil
}

// GetHostStatus returns the current host status
func (c *Client) GetHostStatus() (*state.HostState, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/host/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get host status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("get host status failed with status %d: %s", resp.StatusCode, string(body))
	}

	var hostState state.HostState
	if err := json.NewDecoder(resp.Body).Decode(&hostState); err != nil {
		return nil, fmt.Errorf("failed to decode host status: %w", err)
	}

	return &hostState, nil
}

// ListClusters returns all clusters
func (c *Client) ListClusters() ([]state.ClusterResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/clusters")
	if err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("list clusters failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Clusters []state.ClusterResponse `json:"clusters"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode clusters response: %w", err)
	}

	return response.Clusters, nil
}

// CreateCluster creates a new cluster
func (c *Client) CreateCluster(name string, kubevirt bool) (*state.ClusterResponse, error) {
	req := state.ClusterCreateRequest{
		Name:     name,
		KubeVirt: kubevirt,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/clusters", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("create cluster failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Success bool                   `json:"success"`
		Cluster state.ClusterResponse `json:"cluster"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return &response.Cluster, nil
}

// GetCluster returns details for a specific cluster
func (c *Client) GetCluster(name string) (*state.ClusterResponse, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/clusters/" + name)
	if err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("cluster %s not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("get cluster failed with status %d: %s", resp.StatusCode, string(body))
	}

	var cluster state.ClusterResponse
	if err := json.NewDecoder(resp.Body).Decode(&cluster); err != nil {
		return nil, fmt.Errorf("failed to decode cluster response: %w", err)
	}

	return &cluster, nil
}

// DeleteCluster deletes a cluster
func (c *Client) DeleteCluster(name string) error {
	req, err := http.NewRequest("DELETE", c.BaseURL+"/clusters/"+name, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("cluster %s not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("delete cluster failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetKubeconfig returns the kubeconfig for a cluster
func (c *Client) GetKubeconfig(name string) (string, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/clusters/" + name + "/kubeconfig")
	if err != nil {
		return "", fmt.Errorf("failed to get kubeconfig: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("cluster %s not found", name)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("get kubeconfig failed with status %d: %s", resp.StatusCode, string(body))
	}

	kubeconfig, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	return string(kubeconfig), nil
}

// LoadImage loads a Docker image into a cluster
func (c *Client) LoadImage(clusterName, imageName string) error {
	req := struct {
		Image string `json:"image"`
	}{
		Image: imageName,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := c.HTTPClient.Post(c.BaseURL+"/clusters/"+clusterName+"/load-image", "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("load image failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetRegistryStatus returns the registry status
func (c *Client) GetRegistryStatus() (*state.RegistryStatus, error) {
	resp, err := c.HTTPClient.Get(c.BaseURL + "/registry/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get registry status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("get registry status failed with status %d: %s", resp.StatusCode, string(body))
	}

	var status state.RegistryStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode registry status: %w", err)
	}

	return &status, nil
}

// StartRegistry starts the container registry
func (c *Client) StartRegistry() error {
	resp, err := c.HTTPClient.Post(c.BaseURL+"/registry/start", "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to start registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("start registry failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}