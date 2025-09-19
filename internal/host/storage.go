package host

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kylape/host-manager/internal/state"
)

// detectStorage determines the best storage configuration for the host
func detectStorage() (*state.StorageConfig, error) {
	// Try to get instance type from EC2 metadata first
	instanceType, err := getInstanceType()
	if err == nil {
		return detectStorageFromInstanceType(instanceType), nil
	}

	// Fallback to device scanning if metadata is unavailable
	return scanForNVMeDevices()
}

// scanForNVMeDevices scans for available NVMe devices on the system
func scanForNVMeDevices() (*state.StorageConfig, error) {
	devices, err := ioutil.ReadDir("/dev/")
	if err != nil {
		return nil, fmt.Errorf("failed to scan /dev/: %w", err)
	}

	for _, device := range devices {
		if strings.HasPrefix(device.Name(), "nvme") &&
			strings.HasSuffix(device.Name(), "n1") {

			devicePath := "/dev/" + device.Name()

			// Check if it's an instance store (not EBS)
			if isInstanceStore(devicePath) {
				return &state.StorageConfig{
					HasNVMe: true,
					Device:  devicePath,
					Type:    "instance-store",
				}, nil
			}
		}
	}

	return &state.StorageConfig{
		HasNVMe: false,
		Type:    "ebs-only",
	}, nil
}

// isInstanceStore checks if a device is likely an instance store volume
func isInstanceStore(device string) bool {
	// Check if device has a filesystem (EBS volumes usually do)
	cmd := exec.Command("blkid", device)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if cmd.Run() == nil {
		return false // Has filesystem, likely EBS
	}

	// Check device size - instance store is usually large
	cmd = exec.Command("blockdev", "--getsize64", device)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	size, err := strconv.ParseInt(strings.TrimSpace(string(output)), 10, 64)
	if err != nil {
		return false
	}

	sizeGB := size / (1024 * 1024 * 1024)

	// Instance stores are typically 75GB+
	return sizeGB > 50
}

// setupNVMeStorage configures NVMe storage for containers
func setupNVMeStorage(device string) error {
	// Create BTRFS filesystem
	cmd := exec.Command("mkfs.btrfs", "-f", device)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create BTRFS filesystem: %w", err)
	}

	// Mount to /root
	cmd = exec.Command("mount", device, "/root")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to mount %s to /root: %w", device, err)
	}

	// Create required directories
	if err := os.MkdirAll("/root/kind", 0755); err != nil {
		return fmt.Errorf("failed to create /root/kind: %w", err)
	}

	if err := os.MkdirAll("/root/containers/storage", 0755); err != nil {
		return fmt.Errorf("failed to create /root/containers/storage: %w", err)
	}

	return setupContainerStorage()
}

// setupDefaultStorage configures default storage without NVMe
func setupDefaultStorage() error {
	// Create required directories with default storage
	if err := os.MkdirAll("/var/lib/containers/storage", 0755); err != nil {
		return fmt.Errorf("failed to create container storage directory: %w", err)
	}

	return setupContainerStorage()
}

// setupContainerStorage configures container storage settings
func setupContainerStorage() error {
	storageConf := `[storage]
driver = "overlay"
graphroot = "/root/containers/storage"
runroot = "/run/containers/storage"
`

	if err := ioutil.WriteFile("/etc/containers/storage.conf", []byte(storageConf), 0644); err != nil {
		return fmt.Errorf("failed to write storage.conf: %w", err)
	}

	// Set SELinux contexts if available
	cmd := exec.Command("semanage", "fcontext", "-a", "-t", "container_var_lib_t", "/root/containers/storage(/.*)?")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore errors - SELinux might not be enabled

	cmd = exec.Command("semanage", "fcontext", "-a", "-t", "container_file_t", "/root/containers/storage/overlay-containers(/.*)?")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore errors

	cmd = exec.Command("restorecon", "-R", "/root/containers/storage")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Run() // Ignore errors

	return nil
}