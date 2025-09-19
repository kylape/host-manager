package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kylape/host-manager/internal/kind"
	"github.com/kylape/host-manager/internal/state"
)

// Server handles HTTP requests for host management
type Server struct {
	stateManager *state.Manager
	kindClient   *kind.Client
	router       *mux.Router
}

// New creates a new HTTP server
func New(stateManager *state.Manager) *Server {
	s := &Server{
		stateManager: stateManager,
		kindClient:   kind.NewClient(),
		router:       mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	log.Printf("Starting HTTP server on %s", addr)
	return http.ListenAndServe(addr, s.router)
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Health and status endpoints
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/host/status", s.handleHostStatus).Methods("GET")
	s.router.HandleFunc("/version", s.handleVersion).Methods("GET")

	// Cluster management endpoints
	s.router.HandleFunc("/clusters", s.handleListClusters).Methods("GET")
	s.router.HandleFunc("/clusters", s.handleCreateCluster).Methods("POST")
	s.router.HandleFunc("/clusters/{name}", s.handleGetCluster).Methods("GET")
	s.router.HandleFunc("/clusters/{name}", s.handleDeleteCluster).Methods("DELETE")
	s.router.HandleFunc("/clusters/{name}/kubeconfig", s.handleGetKubeconfig).Methods("GET")
	s.router.HandleFunc("/clusters/{name}/load-image", s.handleLoadImage).Methods("POST")

	// Registry management endpoints
	s.router.HandleFunc("/registry/status", s.handleRegistryStatus).Methods("GET")
	s.router.HandleFunc("/registry/start", s.handleRegistryStart).Methods("POST")

	// Enable CORS for all routes
	s.router.Use(corsMiddleware)
	s.router.Use(loggingMiddleware)
}

// handleHealth returns service health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	hostState, err := s.stateManager.Load()
	if err != nil {
		http.Error(w, "Failed to load host state", http.StatusInternalServerError)
		return
	}

	response := state.HealthResponse{
		Status:      "healthy",
		Initialized: hostState.Initialized,
		Version:     "1.0.0",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHostStatus returns detailed host status
func (s *Server) handleHostStatus(w http.ResponseWriter, r *http.Request) {
	hostState, err := s.stateManager.Load()
	if err != nil {
		http.Error(w, "Failed to load host state", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(hostState)
}

// handleVersion returns version information
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"service_version": "1.0.0",
		"api_version":     "v1",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleListClusters returns all clusters
func (s *Server) handleListClusters(w http.ResponseWriter, r *http.Request) {
	hostState, err := s.stateManager.Load()
	if err != nil {
		http.Error(w, "Failed to load host state", http.StatusInternalServerError)
		return
	}

	var clusters []state.ClusterResponse
	for name, info := range hostState.Clusters {
		clusters = append(clusters, state.ClusterResponse{
			Name:     name,
			Status:   info.Status,
			Created:  info.Created,
			Type:     info.Type,
			KubeVirt: info.KubeVirt,
		})
	}

	response := map[string][]state.ClusterResponse{
		"clusters": clusters,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleCreateCluster creates a new cluster
func (s *Server) handleCreateCluster(w http.ResponseWriter, r *http.Request) {
	var req state.ClusterCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Cluster name is required", http.StatusBadRequest)
		return
	}

	// Check if cluster already exists
	hostState, err := s.stateManager.Load()
	if err != nil {
		http.Error(w, "Failed to load host state", http.StatusInternalServerError)
		return
	}

	if _, exists := hostState.Clusters[req.Name]; exists {
		http.Error(w, fmt.Sprintf("Cluster %s already exists", req.Name), http.StatusConflict)
		return
	}

	// Create the cluster
	if err := s.kindClient.CreateCluster(req.Name, true); err != nil {
		http.Error(w, fmt.Sprintf("Failed to create cluster: %v", err), http.StatusInternalServerError)
		return
	}

	// Update state
	clusterType := "development"
	if req.Name == "kind" {
		clusterType = "infrastructure"
	}

	if err := s.stateManager.UpdateCluster(req.Name, "running", clusterType, req.KubeVirt); err != nil {
		log.Printf("Failed to update cluster state: %v", err)
	}

	response := map[string]interface{}{
		"success": true,
		"cluster": state.ClusterResponse{
			Name:     req.Name,
			Status:   "running",
			Type:     clusterType,
			KubeVirt: req.KubeVirt,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// handleGetCluster returns details for a specific cluster
func (s *Server) handleGetCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	hostState, err := s.stateManager.Load()
	if err != nil {
		http.Error(w, "Failed to load host state", http.StatusInternalServerError)
		return
	}

	info, exists := hostState.Clusters[name]
	if !exists {
		http.Error(w, "Cluster not found", http.StatusNotFound)
		return
	}

	response := state.ClusterResponse{
		Name:     name,
		Status:   info.Status,
		Created:  info.Created,
		Type:     info.Type,
		KubeVirt: info.KubeVirt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleDeleteCluster deletes a cluster
func (s *Server) handleDeleteCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	if name == "kind" {
		http.Error(w, "Cannot delete infrastructure cluster", http.StatusForbidden)
		return
	}

	// Delete the cluster
	if err := s.kindClient.DeleteCluster(name); err != nil {
		http.Error(w, fmt.Sprintf("Failed to delete cluster: %v", err), http.StatusInternalServerError)
		return
	}

	// Remove from state
	if err := s.stateManager.RemoveCluster(name); err != nil {
		log.Printf("Failed to remove cluster from state: %v", err)
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Cluster %s deleted", name),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleGetKubeconfig returns kubeconfig for a cluster
func (s *Server) handleGetKubeconfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	kubeconfig, err := s.kindClient.GetKubeconfig(name)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get kubeconfig: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Write([]byte(kubeconfig))
}

// handleLoadImage loads an image into a cluster
func (s *Server) handleLoadImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var req struct {
		Image string `json:"image"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Image == "" {
		http.Error(w, "Image name is required", http.StatusBadRequest)
		return
	}

	if err := s.kindClient.LoadImage(name, req.Image); err != nil {
		http.Error(w, fmt.Sprintf("Failed to load image: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success": true,
		"message": fmt.Sprintf("Image %s loaded into cluster %s", req.Image, name),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRegistryStatus returns registry status
func (s *Server) handleRegistryStatus(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual registry status check
	response := state.RegistryStatus{
		Running: true,
		Port:    5001,
		URL:     "localhost:5001",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRegistryStart starts the registry
func (s *Server) handleRegistryStart(w http.ResponseWriter, r *http.Request) {
	if err := s.kindClient.CreateRegistry(); err != nil {
		http.Error(w, fmt.Sprintf("Failed to start registry: %v", err), http.StatusInternalServerError)
		return
	}

	if err := s.stateManager.SetRegistryStatus(true); err != nil {
		log.Printf("Failed to update registry status: %v", err)
	}

	response := map[string]interface{}{
		"success": true,
		"message": "Registry started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// corsMiddleware adds CORS headers
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}