package host

import (
	"fmt"
	"log"

	"github.com/kylape/host-manager/internal/kind"
	"github.com/kylape/host-manager/internal/state"
)

// Manager handles host initialization and management
type Manager struct {
	stateManager *state.Manager
}

// NewManager creates a new host manager
func NewManager(stateManager *state.Manager) *Manager {
	return &Manager{
		stateManager: stateManager,
	}
}

// Initialize performs complete host initialization
func (m *Manager) Initialize() error {
	log.Println("Starting host initialization...")

	// Detect storage configuration
	storage, err := detectStorage()
	if err != nil {
		return fmt.Errorf("failed to detect storage: %w", err)
	}

	var instanceType string
	if metaInstanceType, err := getInstanceType(); err == nil {
		instanceType = metaInstanceType
		log.Printf("Detected instance type: %s", instanceType)
	} else {
		instanceType = "unknown"
		log.Printf("Could not detect instance type: %v", err)
	}

	log.Printf("Storage configuration: type=%s, device=%s", storage.Type, storage.Device)

	// Install system packages
	log.Println("Installing system packages...")
	if err := installPackages(); err != nil {
		return fmt.Errorf("failed to install packages: %w", err)
	}

	// Configure storage
	log.Println("Configuring storage...")
	if err := m.configureStorage(storage); err != nil {
		return fmt.Errorf("failed to configure storage: %w", err)
	}

	// Setup SSH keys
	log.Println("Configuring SSH...")
	if err := m.configureSSH(); err != nil {
		return fmt.Errorf("failed to configure SSH: %w", err)
	}

	// Create base infrastructure
	log.Println("Creating base infrastructure...")
	if err := m.createBaseInfrastructure(); err != nil {
		return fmt.Errorf("failed to create base infrastructure: %w", err)
	}

	// Mark host as initialized
	if err := m.stateManager.MarkInitialized(instanceType, storage.Type, storage.Device); err != nil {
		return fmt.Errorf("failed to save initialization state: %w", err)
	}

	log.Println("Host initialization completed successfully")
	return nil
}

// configureStorage sets up storage based on detected configuration
func (m *Manager) configureStorage(storage *state.StorageConfig) error {
	if storage.HasNVMe {
		log.Printf("Configuring NVMe storage: %s", storage.Device)
		return setupNVMeStorage(storage.Device)
	}

	log.Println("Configuring default storage")
	return setupDefaultStorage()
}

// configureSSH sets up SSH keys and configuration
func (m *Manager) configureSSH() error {
	// This is a simplified version - in a real implementation you might
	// want to configure SSH keys from a secure source
	log.Println("SSH configuration completed (placeholder)")
	return nil
}

// createBaseInfrastructure creates the base kind cluster and registry
func (m *Manager) createBaseInfrastructure() error {
	kindClient := kind.NewClient()

	// Create shared registry
	log.Println("Creating shared container registry...")
	if err := kindClient.CreateRegistry(); err != nil {
		return fmt.Errorf("failed to create registry: %w", err)
	}

	if err := m.stateManager.SetRegistryStatus(true); err != nil {
		return fmt.Errorf("failed to update registry status: %w", err)
	}

	// Create base infrastructure cluster
	log.Println("Creating base infrastructure cluster...")
	if err := kindClient.CreateCluster("kind", true); err != nil {
		return fmt.Errorf("failed to create base cluster: %w", err)
	}

	if err := m.stateManager.UpdateCluster("kind", "running", "infrastructure", false); err != nil {
		return fmt.Errorf("failed to update cluster state: %w", err)
	}

	if err := m.stateManager.SetBaseClusterReady(); err != nil {
		return fmt.Errorf("failed to mark base cluster ready: %w", err)
	}

	return nil
}