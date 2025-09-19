package host

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/kylape/host-manager/internal/state"
)

// getInstanceType fetches the EC2 instance type from metadata service
func getInstanceType() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	// First, get IMDSv2 token
	token, err := getIMDSv2Token(client)
	if err != nil {
		return "", fmt.Errorf("failed to get IMDSv2 token: %w", err)
	}

	// Use token to fetch instance type
	req, err := http.NewRequest("GET", "http://169.254.169.254/latest/meta-data/instance-type", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("X-aws-ec2-metadata-token", token)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch instance type: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("metadata service returned status %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}

// getIMDSv2Token obtains a session token for IMDSv2
func getIMDSv2Token(client *http.Client) (string, error) {
	req, err := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600") // 6 hours

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("token request failed with status %d", resp.StatusCode)
	}

	token, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read token: %w", err)
	}

	return strings.TrimSpace(string(token)), nil
}

// detectStorageFromInstanceType determines storage configuration based on instance type
func detectStorageFromInstanceType(instanceType string) *state.StorageConfig {
	if hasNVMeStorage(instanceType) {
		return &state.StorageConfig{
			HasNVMe: true,
			Device:  "/dev/nvme1n1", // Standard location for instance store
			Type:    "instance-store",
		}
	}

	return &state.StorageConfig{
		HasNVMe: false,
		Type:    "ebs-only",
	}
}

// hasNVMeStorage checks if an instance type has NVMe instance store
func hasNVMeStorage(instanceType string) bool {
	// Instance families with NVMe instance store
	nvmePatterns := []string{
		// Compute optimized with NVMe
		"c5d.", "c5ad.", "c6gd.", "c6id.", "c7gd.",

		// Memory optimized with NVMe
		"m5d.", "m5ad.", "m5dn.", "m5zn.", "m6gd.", "m6id.", "m6idn.",
		"r5d.", "r5ad.", "r5dn.", "r6gd.", "r6id.", "r6idn.",
		"x1e.", "x2gd.", "x2idn.", "x2iedn.",

		// Storage optimized
		"i3.", "i3en.", "i4i.", "d2.", "d3.", "d3en.",

		// General purpose with NVMe
		"a1.", "t3.", "t4g.",

		// High performance computing
		"hpc6a.", "hpc6id.", "hpc7a.", "hpc7g.",

		// Metal instances (all have NVMe)
		".metal",
	}

	for _, pattern := range nvmePatterns {
		if strings.Contains(instanceType, pattern) {
			return true
		}
	}

	return false
}