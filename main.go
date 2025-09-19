package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"syscall"

	"github.com/kylape/host-manager/internal/host"
	"github.com/kylape/host-manager/internal/logger"
	"github.com/kylape/host-manager/internal/server"
	"github.com/kylape/host-manager/internal/state"
)

func main() {
	// Parse command line flags
	var (
		help       = flag.Bool("help", false, "Show help message")
		port       = flag.String("port", "8080", "HTTP server port")
		foreground = flag.Bool("foreground", false, "Run in foreground instead of background")
		auditLog   = flag.Bool("audit", false, "Enable HTTP request audit logging")
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

	// Initialize logging
	logger := logger.New(*foreground)

	// Handle daemon mode (background by default)
	if !*foreground {
		logger.Info("Starting daemonization process")
		if err := daemonize(); err != nil {
			logger.Error("Failed to daemonize", "error", err)
			os.Exit(1)
		}
		return
	}

	logger.Info("Starting host manager", "port", *port, "audit", *auditLog)

	// Initialize state manager
	stateManager := state.NewManager()

	// Check if host is already initialized
	hostState, err := stateManager.Load()
	if err != nil {
		logger.Warn("Failed to load state, assuming fresh host", "error", err)
		hostState = &state.HostState{Initialized: false}
	}

	if !hostState.Initialized {
		logger.Info("Fresh host detected, running initialization")

		// Initialize host with auto-detection
		hostManager := host.NewManager(stateManager)
		if err := hostManager.Initialize(); err != nil {
			logger.Error("Host setup failed", "error", err)
			os.Exit(1)
		}

		logger.Info("Host initialization complete")
	} else {
		logger.Info("Host already initialized, skipping setup", "initialized_at", hostState.InitializedAt)
	}

	// Start HTTP server for runtime operations
	srv := server.New(stateManager, logger, *auditLog)
	logger.Info("HTTP server ready", "address", ":"+*port)
	if err := srv.Start(":" + *port); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf(`Host Manager - Unified host management service for EC2-based development environments

Usage: %s [options]

Options:
  --help          Show this help message
  --port PORT     HTTP server port (default: 8080)
  --foreground    Run in foreground instead of background
  --audit         Enable HTTP request audit logging

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

// daemonize implements proper POSIX daemonization
func daemonize() error {
	// Filter environment variables to include only valid ones
	validEnv := filterValidEnvVars(os.Environ())

	// First fork
	pid, err := syscall.ForkExec(os.Args[0], append(os.Args, "--foreground"), &syscall.ProcAttr{
		Dir:   "/",
		Env:   validEnv,
		Files: []uintptr{0, 1, 2}, // stdin, stdout, stderr
	})
	if err != nil {
		return fmt.Errorf("fork failed: %v", err)
	}

	fmt.Printf("Host manager daemon started with PID %d\n", pid)
	return nil
}

// filterValidEnvVars filters environment variables to include only those with valid names
func filterValidEnvVars(environ []string) []string {
	// Valid environment variable names: [A-Za-z_][A-Za-z0-9_]*
	validNameRegex := regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

	var validEnv []string
	var invalidCount int

	for _, env := range environ {
		if validNameRegex.MatchString(env) {
			validEnv = append(validEnv, env)
		} else {
			invalidCount++
		}
	}

	// Log warning if invalid environment variables were filtered
	if invalidCount > 0 {
		fmt.Fprintf(os.Stderr, "Warning: Filtered %d environment variables with invalid names during daemonization\n", invalidCount)
	}

	// Check for critical environment variables and warn if missing
	criticalVars := []string{"PATH"}
	for _, critical := range criticalVars {
		found := false
		for _, env := range validEnv {
			if regexp.MustCompile(`^` + critical + `=`).MatchString(env) {
				found = true
				break
			}
		}
		if !found {
			fmt.Fprintf(os.Stderr, "Warning: Critical environment variable %s not found - daemon may not function correctly\n", critical)
		}
	}

	return validEnv
}

// isRunningInContainer detects if the process is running inside a container
func isRunningInContainer() bool {
	// Check for container-specific files/environment variables
	containerIndicators := []string{
		"/.dockerenv",        // Docker
		"/run/.containerenv", // Podman
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
