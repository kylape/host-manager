package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/kylape/host-manager/client"
)

func main() {
	var (
		serverURL = flag.String("server", "http://host.docker.internal:8080", "Host manager server URL")
		help      = flag.Bool("help", false, "Show help message")
	)

	flag.Parse()

	if *help || len(flag.Args()) == 0 {
		showHelp()
		return
	}

	hmc := client.NewClient(*serverURL)
	command := flag.Args()[0]

	switch command {
	case "health":
		handleHealth(hmc)
	case "status":
		handleHostStatus(hmc)
	case "clusters":
		handleClusters(hmc, flag.Args()[1:])
	case "registry":
		handleRegistry(hmc, flag.Args()[1:])
	default:
		fmt.Printf("Unknown command: %s\n", command)
		showHelp()
		os.Exit(1)
	}
}

func handleHealth(hmc *client.Client) {
	health, err := hmc.Health()
	if err != nil {
		log.Fatalf("Failed to get health: %v", err)
	}

	fmt.Printf("Status: %s\n", health.Status)
	fmt.Printf("Initialized: %v\n", health.Initialized)
	fmt.Printf("Version: %s\n", health.Version)
}

func handleHostStatus(hmc *client.Client) {
	status, err := hmc.GetHostStatus()
	if err != nil {
		log.Fatalf("Failed to get host status: %v", err)
	}

	data, _ := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(data))
}

func handleClusters(hmc *client.Client, args []string) {
	if len(args) == 0 {
		// List clusters
		clusters, err := hmc.ListClusters()
		if err != nil {
			log.Fatalf("Failed to list clusters: %v", err)
		}

		if len(clusters) == 0 {
			fmt.Println("No clusters found")
			return
		}

		fmt.Printf("%-20s %-10s %-15s %-8s\n", "NAME", "STATUS", "TYPE", "KUBEVIRT")
		fmt.Printf("%-20s %-10s %-15s %-8s\n", "----", "------", "----", "--------")
		for _, cluster := range clusters {
			fmt.Printf("%-20s %-10s %-15s %-8v\n", cluster.Name, cluster.Status, cluster.Type, cluster.KubeVirt)
		}
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "create":
		if len(args) < 2 {
			fmt.Println("Usage: clusters create <name> [--kubevirt]")
			os.Exit(1)
		}
		name := args[1]
		kubevirt := len(args) > 2 && args[2] == "--kubevirt"

		cluster, err := hmc.CreateCluster(name, kubevirt)
		if err != nil {
			log.Fatalf("Failed to create cluster: %v", err)
		}

		fmt.Printf("Cluster %s created successfully\n", cluster.Name)

	case "delete":
		if len(args) < 2 {
			fmt.Println("Usage: clusters delete <name>")
			os.Exit(1)
		}
		name := args[1]

		if err := hmc.DeleteCluster(name); err != nil {
			log.Fatalf("Failed to delete cluster: %v", err)
		}

		fmt.Printf("Cluster %s deleted successfully\n", name)

	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: clusters get <name>")
			os.Exit(1)
		}
		name := args[1]

		cluster, err := hmc.GetCluster(name)
		if err != nil {
			log.Fatalf("Failed to get cluster: %v", err)
		}

		data, _ := json.MarshalIndent(cluster, "", "  ")
		fmt.Println(string(data))

	case "kubeconfig":
		if len(args) < 2 {
			fmt.Println("Usage: clusters kubeconfig <name>")
			os.Exit(1)
		}
		name := args[1]

		kubeconfig, err := hmc.GetKubeconfig(name)
		if err != nil {
			log.Fatalf("Failed to get kubeconfig: %v", err)
		}

		fmt.Print(kubeconfig)

	default:
		fmt.Printf("Unknown clusters subcommand: %s\n", subcommand)
		showHelp()
		os.Exit(1)
	}
}

func handleRegistry(hmc *client.Client, args []string) {
	if len(args) == 0 {
		// Show registry status
		status, err := hmc.GetRegistryStatus()
		if err != nil {
			log.Fatalf("Failed to get registry status: %v", err)
		}

		fmt.Printf("Running: %v\n", status.Running)
		fmt.Printf("Port: %d\n", status.Port)
		fmt.Printf("URL: %s\n", status.URL)
		return
	}

	subcommand := args[0]
	switch subcommand {
	case "start":
		if err := hmc.StartRegistry(); err != nil {
			log.Fatalf("Failed to start registry: %v", err)
		}
		fmt.Println("Registry started successfully")

	default:
		fmt.Printf("Unknown registry subcommand: %s\n", subcommand)
		showHelp()
		os.Exit(1)
	}
}

func showHelp() {
	fmt.Printf(`Host Manager Client - CLI tool for managing the host manager service

Usage: %s [options] <command> [args...]

Options:
  --server URL    Host manager server URL (default: http://host.docker.internal:8080)
  --help          Show this help message

Commands:
  health                          Check service health
  status                          Show detailed host status
  clusters                        List all clusters
  clusters create <name> [--kubevirt]  Create new cluster
  clusters delete <name>          Delete cluster
  clusters get <name>             Get cluster details
  clusters kubeconfig <name>      Get cluster kubeconfig
  registry                        Show registry status
  registry start                  Start registry

Examples:
  # Check if service is healthy
  %s health

  # List all clusters
  %s clusters

  # Create a new development cluster
  %s clusters create my-dev-cluster

  # Create cluster with KubeVirt
  %s clusters create vm-cluster --kubevirt

  # Get kubeconfig for a cluster
  %s clusters kubeconfig my-dev-cluster > ~/.kube/config

  # Delete a cluster
  %s clusters delete my-dev-cluster

  # Check registry status
  %s registry
`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}