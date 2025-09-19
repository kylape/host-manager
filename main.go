package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/kylape/host-manager/internal/host"
	"github.com/kylape/host-manager/internal/server"
	"github.com/kylape/host-manager/internal/state"
)

func main() {
	// Parse command line flags
	var (
		help       = flag.Bool("help", false, "Show help message")
		port       = flag.String("port", "8080", "HTTP server port")
		foreground = flag.Bool("foreground", false, "Run in foreground instead of background")
	)
	flag.Parse()

	// Exit immediately if running in a container
	if isRunningInContainer() {
		fmt.Println("Error: This service cannot run in containers.")
		os.Exit(1)
	}

	// Exit immediately if not running as root
	if os.Geteuid() != 0 {
		fmt.Println("Error: This service must run as root.")
		os.Exit(1)
	}

	if *help {
		showHelp()
		return
	}

	// Handle daemon mode (background by default)
	if !*foreground {
		daemonize()
		return
	}

	log.Println("Starting host manager...")

	// Initialize state manager
	stateManager := state.NewManager()

	// Check if host is already initialized
	hostState, err := stateManager.Load()
	if err != nil {
		log.Printf("Failed to load state (assuming fresh host): %v", err)
		hostState = &state.HostState{Initialized: false}
	}

	if !hostState.Initialized {
		log.Println("Fresh host detected, running initialization...")

		// Initialize host with auto-detection
		hostManager := host.NewManager(stateManager)
		if err := hostManager.Initialize(); err != nil {
			log.Fatalf("Host setup failed: %v", err)
		}

		log.Println("Host initialization complete")
	} else {
		log.Printf("Host already initialized (at %v), skipping setup", hostState.InitializedAt)
	}

	// Start HTTP server for runtime operations
	srv := server.New(stateManager)
	if err := srv.Start(":" + *port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func showHelp() {
	fmt.Printf(`Host Manager - Unified host management service for EC2-based development environments

Usage: %s [options]

Options:
  --help          Show this help message
  --port PORT     HTTP server port (default: 8080)
  --foreground    Run in foreground instead of background

Features:
  - Auto-initialization: Complete host setup on first run
  - Kind cluster management: HTTP API for multiple development clusters
  - Smart storage: Automatic NVMe detection and configuration
  - State persistence: Tracks what's been initialized

API Endpoints:
  GET  /health                      Service health check
  GET  /clusters                    List all clusters
  POST /clusters                    Create new cluster
  GET  /clusters/{name}/kubeconfig  Get kubeconfig for cluster
  DELETE /clusters/{name}           Delete cluster

Example Usage:
  # Start service (auto-initializes on fresh host)
  %s

  # Start on custom port
  %s --port 9090

  # Create development cluster
  curl -X POST http://localhost:8080/clusters -d '{"name": "my-dev-cluster"}'

For more information, see README.md
`, os.Args[0], os.Args[0], os.Args[0])
}

// daemonize runs the process in the background
func daemonize() {
	// In container environments, just redirect output and continue
	// Re-run the same command with --foreground flag
	args := append(os.Args[1:], "--foreground")
	cmd := exec.Command(os.Args[0], args...)

	// Redirect stdout/stderr to discard output in background mode
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	fmt.Printf("Host manager daemon started with PID %d\n", cmd.Process.Pid)
}

// isRunningInContainer detects if the process is running inside a container
func isRunningInContainer() bool {
	// Check for container-specific files/environment variables
	containerIndicators := []string{
		"/.dockerenv",           // Docker
		"/run/.containerenv",    // Podman
	}

	for _, indicator := range containerIndicators {
		if _, err := os.Stat(indicator); err == nil {
			return true
		}
	}

	// Check environment variables
	if os.Getenv("container") != "" || os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		return true
	}

	return false
}