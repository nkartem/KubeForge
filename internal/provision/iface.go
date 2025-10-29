package provision

import (
	"context"
	"errors"
	"fmt"
)

// IProvisioner defines the interface for Kubernetes cluster provisioning
type IProvisioner interface {
	// Name returns the provisioner name (kubeadm, k3s, kind, etc.)
	Name() string

	// ValidateSpec validates the cluster specification for this provisioner
	ValidateSpec(spec *ClusterSpec) error

	// PrepareHosts prepares hosts for cluster installation
	// - Disables swap
	// - Installs container runtime (containerd, cri-o)
	// - Installs kubeadm, kubelet, kubectl
	// - Configures kernel modules and sysctl
	PrepareHosts(ctx context.Context, hosts []HostSpec, runtime string, k8sVersion string) error

	// BootstrapControlPlane initializes the first control plane node
	// - Runs kubeadm init
	// - Returns kubeconfig and join tokens
	BootstrapControlPlane(ctx context.Context, host HostSpec, spec ClusterSpec) (*ProvisionResult, error)

	// InstallCNI installs the CNI plugin (Calico, Flannel, Weave, Cilium)
	InstallCNI(ctx context.Context, kubeconfig []byte, cni string, controlPlane HostSpec) error

	// JoinControlPlane joins additional control plane nodes to the cluster
	// - Requires certificate key from bootstrap
	JoinControlPlane(ctx context.Context, host HostSpec, joinCommand string, certificateKey string) error

	// JoinWorker joins a worker node to the cluster
	JoinWorker(ctx context.Context, host HostSpec, joinCommand string) error

	// GetClusterInfo retrieves current cluster information using kubectl
	GetClusterInfo(ctx context.Context, kubeconfig []byte) (*ClusterInfo, error)

	// DestroyCluster removes the cluster from all hosts
	// - Runs kubeadm reset on all nodes
	// - Removes packages and configs
	DestroyCluster(ctx context.Context, spec ClusterSpec) error

	// RemoveNode removes a single node from the cluster
	// - Drains the node
	// - Runs kubeadm reset
	RemoveNode(ctx context.Context, host HostSpec, kubeconfig []byte) error

	// GenerateJoinToken generates a new join token for adding nodes
	GenerateJoinToken(ctx context.Context, kubeconfig []byte, controlPlane bool) (string, error)
}

// ClusterInfo contains runtime information about a cluster
type ClusterInfo struct {
	Version      string     `json:"version"`
	Nodes        []NodeInfo `json:"nodes"`
	APIServer    string     `json:"api_server"`
	Ready        bool       `json:"ready"`
	CNIInstalled bool       `json:"cni_installed"`
	NodeCount    int        `json:"node_count"`
}

// EventCallback is called for each provisioning event (for real-time streaming to UI)
type EventCallback func(event ProvisionEvent)

// Common errors
var (
	ErrNotImplemented      = errors.New("not implemented")
	ErrConnectionFailed    = errors.New("connection failed")
	ErrCommandFailed       = errors.New("command execution failed")
	ErrInvalidKubeconfig   = errors.New("invalid kubeconfig")
	ErrClusterNotReady     = errors.New("cluster is not ready")
	ErrNodeAlreadyExists   = errors.New("node already exists in cluster")
	ErrProvisionerNotFound = errors.New("provisioner not found")
	ErrSwapEnabled         = errors.New("swap is enabled on host")
	ErrKubeadmNotInstalled = errors.New("kubeadm is not installed")
)

// ErrInvalidSpec creates a new invalid spec error
func ErrInvalidSpec(msg string) error {
	return fmt.Errorf("invalid spec: %s", msg)
}

// ProvisionerFactory creates a provisioner by name
type ProvisionerFactory func(config map[string]interface{}) (IProvisioner, error)

var provisionerRegistry = make(map[string]ProvisionerFactory)

// RegisterProvisioner registers a new provisioner factory
func RegisterProvisioner(name string, factory ProvisionerFactory) {
	provisionerRegistry[name] = factory
}

// GetProvisioner returns a provisioner by name
func GetProvisioner(name string, config map[string]interface{}) (IProvisioner, error) {
	factory, ok := provisionerRegistry[name]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProvisionerNotFound, name)
	}
	return factory(config)
}

// ListProvisioners returns all registered provisioner names
func ListProvisioners() []string {
	names := make([]string, 0, len(provisionerRegistry))
	for name := range provisionerRegistry {
		names = append(names, name)
	}
	return names
}
