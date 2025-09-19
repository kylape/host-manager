package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kylape/host-manager/internal/host"
	"github.com/kylape/host-manager/internal/server"
	"github.com/kylape/host-manager/internal/state"
)

func main() {
	// Parse command line flags
	var (
		help = flag.Bool("help", false, "Show help message")
		port = flag.String("port", "8080", "HTTP server port")
	)
	flag.Parse()

	if *help {
		showHelp()
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
	log.Printf("Starting HTTP server on :%s", *port)
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