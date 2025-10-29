package provision

import "time"

// ClusterSpec defines the desired state of a Kubernetes cluster using kubeadm
type ClusterSpec struct {
	Name          string     `json:"name"`
	ControlPlanes []HostSpec `json:"control_planes"`
	Workers       []HostSpec `json:"workers"`
	K8sVersion    string     `json:"k8s_version"` // e.g., "1.28.0"
	PodNetworkCIDR string    `json:"pod_network_cidr"` // default: "10.244.0.0/16"
	ServiceCIDR    string    `json:"service_cidr"` // default: "10.96.0.0/12"
	CNI           string     `json:"cni"` // calico, flannel, weave, cilium
	ContainerRuntime string `json:"container_runtime"` // containerd, cri-o, docker
	APIServerEndpoint string `json:"api_server_endpoint,omitempty"` // for HA setup
	LoadBalancerIP   string `json:"load_balancer_ip,omitempty"` // for HA control plane
	CertificateKey   string `json:"certificate_key,omitempty"` // for joining additional control planes
}

// HostSpec defines a single host/node in the cluster
type HostSpec struct {
	Hostname   string            `json:"hostname"`
	Address    string            `json:"address"` // IP or DNS
	User       string            `json:"user"` // SSH user
	SSHKey     string            `json:"ssh_key,omitempty"` // SSH private key content
	SSHKeyPath string            `json:"ssh_key_path,omitempty"` // or path to key file
	Port       int               `json:"port"` // SSH port, default 22
	Role       string            `json:"role"` // control-plane, worker
	Labels     map[string]string `json:"labels,omitempty"`
	Taints     []string          `json:"taints,omitempty"`
}

// ProvisionResult contains the result of a provision operation
type ProvisionResult struct {
	Kubeconfig     []byte            `json:"kubeconfig,omitempty"`
	JoinCommand    string            `json:"join_command,omitempty"` // kubeadm join command
	JoinToken      string            `json:"join_token,omitempty"`
	CertificateKey string            `json:"certificate_key,omitempty"` // for control plane join
	Nodes          []NodeInfo        `json:"nodes,omitempty"`
	Events         []ProvisionEvent  `json:"events,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Error          error             `json:"error,omitempty"`
}

// NodeInfo contains information about a provisioned node
type NodeInfo struct {
	Hostname       string    `json:"hostname"`
	Address        string    `json:"address"`
	Role           string    `json:"role"` // control-plane, worker
	Status         string    `json:"status"` // ready, notready, unknown
	K8sVersion     string    `json:"k8s_version"`
	ContainerRuntime string  `json:"container_runtime"`
	JoinedAt       time.Time `json:"joined_at"`
}

// ProvisionEvent represents a step in the provisioning process
type ProvisionEvent struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, warn, error
	Host      string    `json:"host"`
	Step      string    `json:"step"`
	Message   string    `json:"message"`
	Output    string    `json:"output,omitempty"` // command output
}

// ProvisionStatus represents the current state of a provision operation
type ProvisionStatus string

const (
	StatusPending    ProvisionStatus = "pending"
	StatusPreparing  ProvisionStatus = "preparing" // installing dependencies
	StatusBootstrapping ProvisionStatus = "bootstrapping" // kubeadm init
	StatusJoining    ProvisionStatus = "joining" // joining nodes
	StatusCompleted  ProvisionStatus = "completed"
	StatusFailed     ProvisionStatus = "failed"
	StatusCancelling ProvisionStatus = "cancelling"
	StatusCancelled  ProvisionStatus = "cancelled"
)

// Validate checks if the ClusterSpec is valid
func (cs *ClusterSpec) Validate() error {
	if cs.Name == "" {
		return ErrInvalidSpec("cluster name is required")
	}
	if len(cs.ControlPlanes) == 0 {
		return ErrInvalidSpec("at least one control plane is required")
	}

	// Set defaults
	if cs.K8sVersion == "" {
		cs.K8sVersion = "1.28.0" // default version
	}
	if cs.PodNetworkCIDR == "" {
		cs.PodNetworkCIDR = "10.244.0.0/16"
	}
	if cs.ServiceCIDR == "" {
		cs.ServiceCIDR = "10.96.0.0/12"
	}
	if cs.CNI == "" {
		cs.CNI = "calico" // default CNI
	}
	if cs.ContainerRuntime == "" {
		cs.ContainerRuntime = "containerd"
	}

	// Validate all hosts
	for _, host := range append(cs.ControlPlanes, cs.Workers...) {
		if err := host.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Validate checks if the HostSpec is valid
func (hs *HostSpec) Validate() error {
	if hs.Address == "" {
		return ErrInvalidSpec("host address is required")
	}
	if hs.User == "" {
		hs.User = "root" // default user
	}
	if hs.Port == 0 {
		hs.Port = 22 // default SSH port
	}
	if hs.SSHKey == "" && hs.SSHKeyPath == "" {
		return ErrInvalidSpec("SSH key or key path is required for host " + hs.Address)
	}
	if hs.Hostname == "" {
		hs.Hostname = hs.Address // use address as hostname if not specified
	}
	return nil
}

// NewProvisionEvent creates a new provision event
func NewProvisionEvent(level, host, step, message string) ProvisionEvent {
	return ProvisionEvent{
		Timestamp: time.Now(),
		Level:     level,
		Host:      host,
		Step:      step,
		Message:   message,
	}
}

// AddEvent adds an event to the provision result
func (pr *ProvisionResult) AddEvent(level, host, step, message string) {
	pr.Events = append(pr.Events, NewProvisionEvent(level, host, step, message))
}
