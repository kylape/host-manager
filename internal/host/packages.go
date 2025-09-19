package host

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"runtime"
)

// installPackages installs required system packages
func installPackages() error {
	// Update system packages
	cmd := exec.Command("dnf", "update", "-y")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to update packages: %w", err)
	}

	// Install required packages
	packages := []string{
		"jq", "tmux", "iotop", "htop", "vim",
		"curl", "wget", "git", "podman", "buildah", "skopeo",
	}

	args := append([]string{"install", "-y"}, packages...)
	cmd = exec.Command("dnf", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	// Configure system settings
	if err := configureSystemSettings(); err != nil {
		return fmt.Errorf("failed to configure system settings: %w", err)
	}

	// Install Kubernetes tools
	if err := installKubernetesTools(); err != nil {
		return fmt.Errorf("failed to install Kubernetes tools: %w", err)
	}

	return nil
}

// configureSystemSettings configures various system settings
func configureSystemSettings() error {
	// Enable lingering for current user
	cmd := exec.Command("loginctl", "enable-linger", os.Getenv("USER"))
	cmd.Run() // Ignore errors

	// Set inotify limits
	cmd = exec.Command("sysctl", "fs.inotify.max_user_watches=524288")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set inotify max_user_watches: %w", err)
	}

	cmd = exec.Command("sysctl", "fs.inotify.max_user_instances=512")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set inotify max_user_instances: %w", err)
	}

	return nil
}

// installKubernetesTools installs kind and kubectl
func installKubernetesTools() error {
	arch := runtime.GOARCH
	var kindURL, kubectlURL string

	switch arch {
	case "amd64":
		kindURL = "https://kind.sigs.k8s.io/dl/v0.29.0/kind-linux-amd64"
		kubectlURL = "https://dl.k8s.io/release/stable.txt"
	case "arm64":
		kindURL = "https://kind.sigs.k8s.io/dl/v0.29.0/kind-linux-arm64"
		kubectlURL = "https://dl.k8s.io/release/stable.txt"
	default:
		return fmt.Errorf("unsupported architecture: %s", arch)
	}

	// Download and install kind
	if err := downloadAndInstall(kindURL, "/usr/local/bin/kind"); err != nil {
		return fmt.Errorf("failed to install kind: %w", err)
	}

	// Get latest kubectl version
	resp, err := http.Get(kubectlURL)
	if err != nil {
		return fmt.Errorf("failed to get kubectl version: %w", err)
	}
	defer resp.Body.Close()

	versionBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read kubectl version: %w", err)
	}
	version := string(versionBytes)

	// Build kubectl download URL
	var kubectlBinary string
	if arch == "amd64" {
		kubectlBinary = fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/amd64/kubectl", version)
	} else {
		kubectlBinary = fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/arm64/kubectl", version)
	}

	// Download and install kubectl
	if err := downloadAndInstall(kubectlBinary, "/usr/local/bin/kubectl"); err != nil {
		return fmt.Errorf("failed to install kubectl: %w", err)
	}

	return nil
}

// downloadAndInstall downloads a binary and installs it to the specified path
func downloadAndInstall(url, path string) error {
	// Download the binary
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Read the binary data
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read binary data: %w", err)
	}

	// Write to target path
	if err := ioutil.WriteFile(path, data, 0755); err != nil {
		return fmt.Errorf("failed to write binary to %s: %w", path, err)
	}

	return nil
}