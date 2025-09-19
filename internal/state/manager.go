package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

const StateFilePath = "/etc/host-manager-state.json"

// Manager handles persistence of host state
type Manager struct {
	statePath string
}

// NewManager creates a new state manager
func NewManager() *Manager {
	return &Manager{
		statePath: StateFilePath,
	}
}

// Load reads the current host state from disk
func (m *Manager) Load() (*HostState, error) {
	data, err := ioutil.ReadFile(m.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return fresh state if file doesn't exist
			return &HostState{
				Initialized: false,
				Clusters:    make(map[string]ClusterInfo),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state HostState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Initialize clusters map if nil
	if state.Clusters == nil {
		state.Clusters = make(map[string]ClusterInfo)
	}

	return &state, nil
}

// Save writes the host state to disk
func (m *Manager) Save(state *HostState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := ioutil.WriteFile(m.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// MarkInitialized marks the host as initialized
func (m *Manager) MarkInitialized(instanceType, storageType, storageDevice string) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	now := time.Now()
	state.Initialized = true
	state.InitializedAt = &now
	state.InstanceType = instanceType
	state.StorageType = storageType
	state.StorageDevice = storageDevice
	state.PackagesInstalled = true

	return m.Save(state)
}

// UpdateCluster updates information about a cluster
func (m *Manager) UpdateCluster(name, status, clusterType string, kubevirt bool) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	now := time.Now()
	state.Clusters[name] = ClusterInfo{
		Status:   status,
		Created:  &now,
		Type:     clusterType,
		KubeVirt: kubevirt,
	}

	return m.Save(state)
}

// RemoveCluster removes a cluster from state
func (m *Manager) RemoveCluster(name string) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	delete(state.Clusters, name)
	return m.Save(state)
}

// SetRegistryStatus updates the registry status
func (m *Manager) SetRegistryStatus(running bool) error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	state.RegistryRunning = running
	return m.Save(state)
}

// SetBaseClusterReady marks the base cluster as ready
func (m *Manager) SetBaseClusterReady() error {
	state, err := m.Load()
	if err != nil {
		return err
	}

	state.BaseClusterReady = true
	return m.Save(state)
}