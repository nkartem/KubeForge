package db

import (
	"time"

	"gorm.io/gorm"
)

// Cluster represents a Kubernetes cluster
type Cluster struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	Name              string    `gorm:"uniqueIndex;not null" json:"name"`
	K8sVersion        string    `json:"k8s_version"`
	PodNetworkCIDR    string    `json:"pod_network_cidr"`
	ServiceCIDR       string    `json:"service_cidr"`
	CNI               string    `json:"cni"`
	ContainerRuntime  string    `json:"container_runtime"`
	APIServerEndpoint string    `json:"api_server_endpoint"`
	LoadBalancerIP    string    `json:"load_balancer_ip,omitempty"`
	Provider          string    `json:"provider"` // kubeadm, k3s, kind
	Status            string    `json:"status"`   // pending, provisioning, ready, failed, destroying
	Kubeconfig        []byte    `json:"-"`        // encrypted, not exposed in JSON
	JoinCommand       string    `json:"-"`        // not exposed in JSON
	CertificateKey    string    `json:"-"`        // not exposed in JSON
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Nodes  []Node  `gorm:"foreignKey:ClusterID" json:"nodes,omitempty"`
	Events []Event `gorm:"foreignKey:ClusterID" json:"events,omitempty"`
}

// Node represents a node in a cluster
type Node struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ClusterID        uint      `gorm:"index;not null" json:"cluster_id"`
	Hostname         string    `json:"hostname"`
	Address          string    `json:"address"`
	User             string    `json:"user"`
	SSHKeyPath       string    `json:"ssh_key_path,omitempty"`
	Port             int       `json:"port"`
	Role             string    `json:"role"` // control-plane, worker
	Status           string    `json:"status"` // ready, notready, unknown, provisioning
	K8sVersion       string    `json:"k8s_version"`
	ContainerRuntime string    `json:"container_runtime"`
	Labels           string    `json:"labels,omitempty"` // JSON encoded map
	Taints           string    `json:"taints,omitempty"` // JSON encoded array
	JoinedAt         *time.Time `json:"joined_at,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// Event represents a provisioning or cluster event
type Event struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ClusterID uint      `gorm:"index;not null" json:"cluster_id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"` // info, warn, error
	Host      string    `json:"host"`
	Step      string    `json:"step"`
	Message   string    `json:"message"`
	Output    string    `json:"output,omitempty" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
}

// SSHKey represents an SSH key for authentication
type SSHKey struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	PublicKey   string    `gorm:"type:text" json:"public_key"`
	PrivateKey  []byte    `json:"-"` // encrypted, not exposed
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

// User represents a user of the system (for future auth)
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"uniqueIndex" json:"email"`
	PasswordHash string    `json:"-"` // bcrypt hash
	Role         string    `json:"role"` // admin, user
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

// Job represents an async provisioning job
type Job struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	ClusterID  uint      `gorm:"index" json:"cluster_id,omitempty"`
	Type       string    `json:"type"` // provision, destroy, add-node, remove-node
	Status     string    `json:"status"` // pending, running, completed, failed, cancelled
	Progress   int       `json:"progress"` // 0-100
	Error      string    `json:"error,omitempty" gorm:"type:text"`
	Metadata   string    `json:"metadata,omitempty" gorm:"type:text"` // JSON encoded metadata
	StartedAt  *time.Time `json:"started_at,omitempty"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// TableName overrides (optional, GORM will pluralize by default)
func (Cluster) TableName() string {
	return "clusters"
}

func (Node) TableName() string {
	return "nodes"
}

func (Event) TableName() string {
	return "events"
}

func (SSHKey) TableName() string {
	return "ssh_keys"
}

func (User) TableName() string {
	return "users"
}

func (Job) TableName() string {
	return "jobs"
}
