package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"kubeforge/internal/db"
	"kubeforge/internal/provision"
)

// CreateClusterRequest represents the request to create a new cluster
type CreateClusterRequest struct {
	Name             string                `json:"name"`
	K8sVersion       string                `json:"k8s_version"`
	PodNetworkCIDR   string                `json:"pod_network_cidr"`
	ServiceCIDR      string                `json:"service_cidr"`
	CNI              string                `json:"cni"`
	ContainerRuntime string                `json:"container_runtime"`
	APIServerEndpoint string               `json:"api_server_endpoint,omitempty"`
	ControlPlanes    []provision.HostSpec  `json:"control_planes"`
	Workers          []provision.HostSpec  `json:"workers"`
}

// ClusterHandler handles cluster-related API requests
type ClusterHandler struct{}

// NewClusterHandler creates a new cluster handler
func NewClusterHandler() *ClusterHandler {
	return &ClusterHandler{}
}

// RegisterRoutes registers cluster API routes
func (h *ClusterHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/clusters", h.ListClusters).Methods("GET")
	router.HandleFunc("/api/clusters", h.CreateCluster).Methods("POST")
	router.HandleFunc("/api/clusters/{id}", h.GetCluster).Methods("GET")
	router.HandleFunc("/api/clusters/{id}", h.DeleteCluster).Methods("DELETE")
	router.HandleFunc("/api/clusters/{id}/nodes", h.AddNode).Methods("POST")
	router.HandleFunc("/api/clusters/{id}/nodes/{nodeId}", h.RemoveNode).Methods("DELETE")
	router.HandleFunc("/api/clusters/{id}/kubeconfig", h.GetKubeconfig).Methods("GET")
	router.HandleFunc("/api/clusters/{id}/events", h.GetEvents).Methods("GET")
}

// ListClusters lists all clusters
func (h *ClusterHandler) ListClusters(w http.ResponseWriter, r *http.Request) {
	var clusters []db.Cluster

	result := db.DB.Preload("Nodes").Find(&clusters)
	if result.Error != nil {
		WriteInternalError(w, "Failed to retrieve clusters")
		return
	}

	WriteSuccess(w, clusters)
}

// GetCluster retrieves a single cluster by ID
func (h *ClusterHandler) GetCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		WriteBadRequest(w, "Invalid cluster ID")
		return
	}

	var cluster db.Cluster
	result := db.DB.Preload("Nodes").Preload("Events").First(&cluster, id)
	if result.Error != nil {
		WriteNotFound(w, "Cluster not found")
		return
	}

	WriteSuccess(w, cluster)
}

// CreateCluster creates a new cluster
func (h *ClusterHandler) CreateCluster(w http.ResponseWriter, r *http.Request) {
	var req CreateClusterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	if req.Name == "" {
		WriteBadRequest(w, "Cluster name is required")
		return
	}
	if len(req.ControlPlanes) == 0 {
		WriteBadRequest(w, "At least one control plane is required")
		return
	}

	// Create cluster record
	cluster := db.Cluster{
		Name:             req.Name,
		K8sVersion:       req.K8sVersion,
		PodNetworkCIDR:   req.PodNetworkCIDR,
		ServiceCIDR:      req.ServiceCIDR,
		CNI:              req.CNI,
		ContainerRuntime: req.ContainerRuntime,
		APIServerEndpoint: req.APIServerEndpoint,
		Provider:         "kubeadm",
		Status:           "pending",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// Set defaults
	if cluster.K8sVersion == "" {
		cluster.K8sVersion = "1.28.0"
	}
	if cluster.PodNetworkCIDR == "" {
		cluster.PodNetworkCIDR = "10.244.0.0/16"
	}
	if cluster.ServiceCIDR == "" {
		cluster.ServiceCIDR = "10.96.0.0/12"
	}
	if cluster.CNI == "" {
		cluster.CNI = "calico"
	}
	if cluster.ContainerRuntime == "" {
		cluster.ContainerRuntime = "containerd"
	}

	// Save to database
	if err := db.DB.Create(&cluster).Error; err != nil {
		WriteInternalError(w, "Failed to create cluster")
		return
	}

	// Create node records
	for _, cp := range req.ControlPlanes {
		node := db.Node{
			ClusterID: cluster.ID,
			Hostname:  cp.Hostname,
			Address:   cp.Address,
			User:      cp.User,
			SSHKeyPath: cp.SSHKeyPath,
			Port:      cp.Port,
			Role:      "control-plane",
			Status:    "provisioning",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if node.Port == 0 {
			node.Port = 22
		}
		db.DB.Create(&node)
	}

	for _, worker := range req.Workers {
		node := db.Node{
			ClusterID: cluster.ID,
			Hostname:  worker.Hostname,
			Address:   worker.Address,
			User:      worker.User,
			SSHKeyPath: worker.SSHKeyPath,
			Port:      worker.Port,
			Role:      "worker",
			Status:    "provisioning",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		if node.Port == 0 {
			node.Port = 22
		}
		db.DB.Create(&node)
	}

	// Create a job for async provisioning
	job := db.Job{
		ClusterID: cluster.ID,
		Type:      "provision",
		Status:    "pending",
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	db.DB.Create(&job)

	// Start provisioning in background (async)
	go h.provisionCluster(cluster.ID, req)

	// Return created cluster
	db.DB.Preload("Nodes").First(&cluster, cluster.ID)
	WriteCreated(w, cluster)
}

// provisionCluster provisions the cluster asynchronously
func (h *ClusterHandler) provisionCluster(clusterID uint, req CreateClusterRequest) {
	ctx := context.Background()

	// Update cluster status
	db.DB.Model(&db.Cluster{}).Where("id = ?", clusterID).Update("status", "provisioning")

	// Get provisioner
	provisioner, err := provision.GetProvisioner("kubeadm", nil)
	if err != nil {
		h.logError(clusterID, "Failed to get provisioner", err)
		return
	}

	// Build ClusterSpec
	spec := provision.ClusterSpec{
		Name:             req.Name,
		ControlPlanes:    req.ControlPlanes,
		Workers:          req.Workers,
		K8sVersion:       req.K8sVersion,
		PodNetworkCIDR:   req.PodNetworkCIDR,
		ServiceCIDR:      req.ServiceCIDR,
		CNI:              req.CNI,
		ContainerRuntime: req.ContainerRuntime,
		APIServerEndpoint: req.APIServerEndpoint,
	}

	// Validate spec
	if err := provisioner.ValidateSpec(&spec); err != nil {
		h.logError(clusterID, "Invalid cluster spec", err)
		return
	}

	// Prepare all hosts
	allHosts := append(spec.ControlPlanes, spec.Workers...)
	h.logEvent(clusterID, "info", "localhost", "prepare", "Preparing hosts")

	if err := provisioner.PrepareHosts(ctx, allHosts, spec.ContainerRuntime, spec.K8sVersion); err != nil {
		h.logError(clusterID, "Failed to prepare hosts", err)
		return
	}

	// Bootstrap first control plane
	h.logEvent(clusterID, "info", spec.ControlPlanes[0].Address, "bootstrap", "Bootstrapping control plane")

	result, err := provisioner.BootstrapControlPlane(ctx, spec.ControlPlanes[0], spec)
	if err != nil {
		h.logError(clusterID, "Failed to bootstrap control plane", err)
		return
	}

	// Save kubeconfig and join command
	db.DB.Model(&db.Cluster{}).Where("id = ?", clusterID).Updates(map[string]interface{}{
		"kubeconfig":      result.Kubeconfig,
		"join_command":    result.JoinCommand,
		"certificate_key": result.CertificateKey,
	})

	// Install CNI
	h.logEvent(clusterID, "info", spec.ControlPlanes[0].Address, "cni", "Installing CNI")
	if err := provisioner.InstallCNI(ctx, result.Kubeconfig, spec.CNI, spec.ControlPlanes[0]); err != nil {
		h.logError(clusterID, "Failed to install CNI", err)
		// Continue anyway, CNI can be installed manually
	}

	// Join additional control planes
	for i := 1; i < len(spec.ControlPlanes); i++ {
		cp := spec.ControlPlanes[i]
		h.logEvent(clusterID, "info", cp.Address, "join", "Joining control plane")

		if err := provisioner.JoinControlPlane(ctx, cp, result.JoinCommand, result.CertificateKey); err != nil {
			h.logError(clusterID, "Failed to join control plane", err)
			// Continue with other nodes
		}
	}

	// Join workers
	for _, worker := range spec.Workers {
		h.logEvent(clusterID, "info", worker.Address, "join", "Joining worker")

		if err := provisioner.JoinWorker(ctx, worker, result.JoinCommand); err != nil {
			h.logError(clusterID, "Failed to join worker", err)
			// Continue with other nodes
		}
	}

	// Update cluster status
	db.DB.Model(&db.Cluster{}).Where("id = ?", clusterID).Update("status", "ready")
	h.logEvent(clusterID, "info", "localhost", "complete", "Cluster provisioned successfully")
}

// DeleteCluster deletes a cluster
func (h *ClusterHandler) DeleteCluster(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		WriteBadRequest(w, "Invalid cluster ID")
		return
	}

	// TODO: Run kubeadm reset on all nodes before deleting

	if err := db.DB.Delete(&db.Cluster{}, id).Error; err != nil {
		WriteInternalError(w, "Failed to delete cluster")
		return
	}

	WriteSuccess(w, map[string]string{"message": "Cluster deleted"})
}

// AddNode adds a node to an existing cluster
func (h *ClusterHandler) AddNode(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	WriteError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not yet implemented")
}

// RemoveNode removes a node from a cluster
func (h *ClusterHandler) RemoveNode(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement
	WriteError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Not yet implemented")
}

// GetKubeconfig returns the kubeconfig for a cluster
func (h *ClusterHandler) GetKubeconfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		WriteBadRequest(w, "Invalid cluster ID")
		return
	}

	var cluster db.Cluster
	if err := db.DB.First(&cluster, id).Error; err != nil {
		WriteNotFound(w, "Cluster not found")
		return
	}

	if cluster.Kubeconfig == nil {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "Kubeconfig not available")
		return
	}

	w.Header().Set("Content-Type", "application/x-yaml")
	w.Header().Set("Content-Disposition", "attachment; filename=kubeconfig.yaml")
	w.Write(cluster.Kubeconfig)
}

// GetEvents returns events for a cluster
func (h *ClusterHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.ParseUint(vars["id"], 10, 32)
	if err != nil {
		WriteBadRequest(w, "Invalid cluster ID")
		return
	}

	var events []db.Event
	if err := db.DB.Where("cluster_id = ?", id).Order("timestamp desc").Limit(100).Find(&events).Error; err != nil {
		WriteInternalError(w, "Failed to retrieve events")
		return
	}

	WriteSuccess(w, events)
}

// Helper methods

func (h *ClusterHandler) logEvent(clusterID uint, level, host, step, message string) {
	event := db.Event{
		ClusterID: clusterID,
		Timestamp: time.Now(),
		Level:     level,
		Host:      host,
		Step:      step,
		Message:   message,
		CreatedAt: time.Now(),
	}
	db.DB.Create(&event)

	// Broadcast event to WebSocket clients
	Hub.BroadcastEvent(clusterID, event)
}

func (h *ClusterHandler) logError(clusterID uint, message string, err error) {
	h.logEvent(clusterID, "error", "localhost", "error", message+": "+err.Error())
	db.DB.Model(&db.Cluster{}).Where("id = ?", clusterID).Update("status", "failed")
}
