package provision

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// KubeadmProvisioner implements IProvisioner for kubeadm-based clusters
type KubeadmProvisioner struct {
	eventCallback EventCallback
}

// NewKubeadmProvisioner creates a new kubeadm provisioner
func NewKubeadmProvisioner(config map[string]interface{}) (IProvisioner, error) {
	return &KubeadmProvisioner{}, nil
}

func init() {
	RegisterProvisioner("kubeadm", NewKubeadmProvisioner)
}

// Name returns the provisioner name
func (p *KubeadmProvisioner) Name() string {
	return "kubeadm"
}

// ValidateSpec validates the cluster specification
func (p *KubeadmProvisioner) ValidateSpec(spec *ClusterSpec) error {
	return spec.Validate()
}

// PrepareHosts prepares all hosts for Kubernetes installation
func (p *KubeadmProvisioner) PrepareHosts(ctx context.Context, hosts []HostSpec, runtime string, k8sVersion string) error {
	for _, host := range hosts {
		if err := p.prepareHost(ctx, host, runtime, k8sVersion); err != nil {
			return fmt.Errorf("failed to prepare host %s: %w", host.Address, err)
		}
	}
	return nil
}

// prepareHost prepares a single host
func (p *KubeadmProvisioner) prepareHost(ctx context.Context, host HostSpec, runtime string, k8sVersion string) error {
	client, err := NewSSHClient(host)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	p.emitEvent("info", host.Address, "prepare", "Connected to host")

	// Test connection
	if err := client.TestConnection(ctx); err != nil {
		return fmt.Errorf("connection test failed: %w", err)
	}

	// Get host info
	info, _ := client.GetHostInfo(ctx)
	if info["swap_enabled"] == "true" {
		p.emitEvent("info", host.Address, "prepare", "Disabling swap")
		if _, _, err := client.RunCommand(ctx, "swapoff -a && sed -i '/ swap / s/^/#/' /etc/fstab"); err != nil {
			return fmt.Errorf("failed to disable swap: %w", err)
		}
	}

	// Load kernel modules
	p.emitEvent("info", host.Address, "prepare", "Loading kernel modules")
	loadModules := `
cat <<EOF | tee /etc/modules-load.d/k8s.conf
overlay
br_netfilter
EOF
modprobe overlay
modprobe br_netfilter
`
	if _, _, err := client.RunCommand(ctx, loadModules); err != nil {
		return fmt.Errorf("failed to load kernel modules: %w", err)
	}

	// Configure sysctl
	p.emitEvent("info", host.Address, "prepare", "Configuring sysctl parameters")
	sysctl := `
cat <<EOF | tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-iptables  = 1
net.bridge.bridge-nf-call-ip6tables = 1
net.ipv4.ip_forward                 = 1
EOF
sysctl --system
`
	if _, _, err := client.RunCommand(ctx, sysctl); err != nil {
		return fmt.Errorf("failed to configure sysctl: %w", err)
	}

	// Install container runtime
	if err := p.installContainerRuntime(ctx, client, host, runtime); err != nil {
		return fmt.Errorf("failed to install container runtime: %w", err)
	}

	// Install kubeadm, kubelet, kubectl
	if err := p.installKubernetesTools(ctx, client, host, k8sVersion); err != nil {
		return fmt.Errorf("failed to install kubernetes tools: %w", err)
	}

	p.emitEvent("info", host.Address, "prepare", "Host prepared successfully")
	return nil
}

// installContainerRuntime installs the specified container runtime
func (p *KubeadmProvisioner) installContainerRuntime(ctx context.Context, client *SSHClient, host HostSpec, runtime string) error {
	p.emitEvent("info", host.Address, "install-runtime", fmt.Sprintf("Installing %s", runtime))

	switch runtime {
	case "containerd":
		return p.installContainerd(ctx, client, host)
	case "cri-o":
		return p.installCRIO(ctx, client, host)
	default:
		return fmt.Errorf("unsupported runtime: %s", runtime)
	}
}

// installContainerd installs containerd runtime
func (p *KubeadmProvisioner) installContainerd(ctx context.Context, client *SSHClient, host HostSpec) error {
	script := `
# Install dependencies
apt-get update
apt-get install -y apt-transport-https ca-certificates curl gnupg lsb-release

# Add Docker's official GPG key
mkdir -p /etc/apt/keyrings
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | gpg --dearmor -o /etc/apt/keyrings/docker.gpg

# Set up the repository
echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null

# Install containerd
apt-get update
apt-get install -y containerd.io

# Configure containerd
mkdir -p /etc/containerd
containerd config default | tee /etc/containerd/config.toml
sed -i 's/SystemdCgroup = false/SystemdCgroup = true/g' /etc/containerd/config.toml

# Restart containerd
systemctl restart containerd
systemctl enable containerd
`
	_, stderr, err := client.RunCommand(ctx, script)
	if err != nil {
		return fmt.Errorf("containerd installation failed: %s: %w", stderr, err)
	}

	p.emitEvent("info", host.Address, "install-runtime", "Containerd installed successfully")
	return nil
}

// installCRIO installs CRI-O runtime
func (p *KubeadmProvisioner) installCRIO(ctx context.Context, client *SSHClient, host HostSpec) error {
	// TODO: Implement CRI-O installation
	return fmt.Errorf("CRI-O installation not yet implemented")
}

// installKubernetesTools installs kubeadm, kubelet, and kubectl
func (p *KubeadmProvisioner) installKubernetesTools(ctx context.Context, client *SSHClient, host HostSpec, k8sVersion string) error {
	p.emitEvent("info", host.Address, "install-k8s", fmt.Sprintf("Installing Kubernetes %s tools", k8sVersion))

	// Determine version major.minor (e.g., 1.28)
	versionParts := strings.Split(k8sVersion, ".")
	if len(versionParts) < 2 {
		return fmt.Errorf("invalid k8s version format: %s", k8sVersion)
	}
	majorMinor := fmt.Sprintf("%s.%s", versionParts[0], versionParts[1])

	script := fmt.Sprintf(`
# Add Kubernetes apt repository
apt-get update
apt-get install -y apt-transport-https ca-certificates curl gpg

mkdir -p /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v%s/deb/Release.key | gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg

echo "deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v%s/deb/ /" | tee /etc/apt/sources.list.d/kubernetes.list

# Install kubelet, kubeadm, kubectl
apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl

# Enable kubelet
systemctl enable kubelet
`, majorMinor, majorMinor)

	_, stderr, err := client.RunCommand(ctx, script)
	if err != nil {
		return fmt.Errorf("kubernetes tools installation failed: %s: %w", stderr, err)
	}

	p.emitEvent("info", host.Address, "install-k8s", "Kubernetes tools installed successfully")
	return nil
}

// BootstrapControlPlane initializes the first control plane node
func (p *KubeadmProvisioner) BootstrapControlPlane(ctx context.Context, host HostSpec, spec ClusterSpec) (*ProvisionResult, error) {
	client, err := NewSSHClient(host)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	p.emitEvent("info", host.Address, "bootstrap", "Initializing control plane")

	result := &ProvisionResult{
		Nodes:    []NodeInfo{},
		Events:   []ProvisionEvent{},
		Metadata: make(map[string]string),
	}

	// Build kubeadm init command
	initCmd := fmt.Sprintf("kubeadm init --pod-network-cidr=%s --kubernetes-version=%s",
		spec.PodNetworkCIDR, spec.K8sVersion)

	if spec.APIServerEndpoint != "" {
		initCmd += fmt.Sprintf(" --control-plane-endpoint=%s", spec.APIServerEndpoint)
	}

	initCmd += " --upload-certs" // For HA setup

	p.emitEvent("info", host.Address, "bootstrap", "Running kubeadm init (this may take a few minutes)")

	// Run kubeadm init
	stdout, stderr, err := client.RunCommand(ctx, initCmd)
	if err != nil {
		result.AddEvent("error", host.Address, "bootstrap", fmt.Sprintf("kubeadm init failed: %s", stderr))
		return result, fmt.Errorf("kubeadm init failed: %w", err)
	}

	result.AddEvent("info", host.Address, "bootstrap", "kubeadm init completed")

	// Extract join commands and certificate key from output
	result.JoinCommand = p.extractJoinCommand(stdout)
	result.CertificateKey = p.extractCertificateKey(stdout)

	// Copy kubeconfig
	p.emitEvent("info", host.Address, "bootstrap", "Retrieving kubeconfig")
	_, _, err = client.RunCommand(ctx, "mkdir -p $HOME/.kube && cp -i /etc/kubernetes/admin.conf $HOME/.kube/config && chown $(id -u):$(id -g) $HOME/.kube/config")
	if err != nil {
		return result, fmt.Errorf("failed to setup kubeconfig: %w", err)
	}

	// Download kubeconfig
	kubeconfigContent, _, err := client.RunCommand(ctx, "cat /etc/kubernetes/admin.conf")
	if err != nil {
		return result, fmt.Errorf("failed to retrieve kubeconfig: %w", err)
	}
	result.Kubeconfig = []byte(kubeconfigContent)

	p.emitEvent("info", host.Address, "bootstrap", "Control plane bootstrapped successfully")

	// Add node info
	result.Nodes = append(result.Nodes, NodeInfo{
		Hostname:   host.Hostname,
		Address:    host.Address,
		Role:       "control-plane",
		Status:     "ready",
		K8sVersion: spec.K8sVersion,
		JoinedAt:   time.Now(),
	})

	return result, nil
}

// InstallCNI installs the CNI plugin on the control plane
func (p *KubeadmProvisioner) InstallCNI(ctx context.Context, kubeconfig []byte, cni string, controlPlane HostSpec) error {
	p.emitEvent("info", controlPlane.Address, "install-cni", fmt.Sprintf("Installing %s CNI", cni))

	var cniManifest string
	switch cni {
	case "calico":
		cniManifest = "https://raw.githubusercontent.com/projectcalico/calico/v3.26.1/manifests/calico.yaml"
	case "flannel":
		cniManifest = "https://github.com/flannel-io/flannel/releases/latest/download/kube-flannel.yml"
	case "weave":
		cniManifest = "https://github.com/weaveworks/weave/releases/download/v2.8.1/weave-daemonset-k8s.yaml"
	case "cilium":
		// Cilium requires Helm or cilium CLI
		return fmt.Errorf("cilium installation requires Helm or CLI, not yet implemented")
	default:
		return fmt.Errorf("unsupported CNI: %s", cni)
	}

	// Connect to control plane to apply CNI
	client, err := NewSSHClient(controlPlane)
	if err != nil {
		return fmt.Errorf("failed to connect to control plane: %w", err)
	}
	defer client.Close()

	// Apply CNI manifest using kubectl on control plane
	applyCmd := fmt.Sprintf("kubectl apply -f %s", cniManifest)
	stdout, stderr, err := client.RunCommand(ctx, applyCmd)
	if err != nil {
		p.emitEvent("error", controlPlane.Address, "install-cni", fmt.Sprintf("Failed to apply CNI: %s", stderr))
		return fmt.Errorf("failed to apply CNI manifest: %s: %w", stderr, err)
	}

	p.emitEvent("info", controlPlane.Address, "install-cni", fmt.Sprintf("CNI applied successfully: %s", stdout))

	// Wait for CNI pods to be ready (optional but recommended)
	waitCmd := "kubectl wait --for=condition=Ready pods --all -n kube-system --timeout=300s"
	_, _, err = client.RunCommand(ctx, waitCmd)
	if err != nil {
		p.emitEvent("warn", controlPlane.Address, "install-cni", "CNI pods may not be fully ready yet")
	} else {
		p.emitEvent("info", controlPlane.Address, "install-cni", "CNI pods are ready")
	}

	return nil
}

// JoinControlPlane joins an additional control plane node
func (p *KubeadmProvisioner) JoinControlPlane(ctx context.Context, host HostSpec, joinCommand string, certificateKey string) error {
	client, err := NewSSHClient(host)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	p.emitEvent("info", host.Address, "join-cp", "Joining control plane")

	// Add --control-plane and --certificate-key flags
	fullJoinCmd := fmt.Sprintf("%s --control-plane --certificate-key %s", joinCommand, certificateKey)

	_, stderr, err := client.RunCommand(ctx, fullJoinCmd)
	if err != nil {
		return fmt.Errorf("failed to join control plane: %s: %w", stderr, err)
	}

	p.emitEvent("info", host.Address, "join-cp", "Control plane joined successfully")
	return nil
}

// JoinWorker joins a worker node to the cluster
func (p *KubeadmProvisioner) JoinWorker(ctx context.Context, host HostSpec, joinCommand string) error {
	client, err := NewSSHClient(host)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	p.emitEvent("info", host.Address, "join-worker", "Joining worker node")

	_, stderr, err := client.RunCommand(ctx, joinCommand)
	if err != nil {
		return fmt.Errorf("failed to join worker: %s: %w", stderr, err)
	}

	p.emitEvent("info", host.Address, "join-worker", "Worker node joined successfully")
	return nil
}

// GetClusterInfo retrieves cluster information
func (p *KubeadmProvisioner) GetClusterInfo(ctx context.Context, kubeconfig []byte) (*ClusterInfo, error) {
	// TODO: Use client-go to query cluster
	return &ClusterInfo{
		Ready: true,
	}, nil
}

// DestroyCluster removes the cluster from all hosts
func (p *KubeadmProvisioner) DestroyCluster(ctx context.Context, spec ClusterSpec) error {
	allHosts := append(spec.ControlPlanes, spec.Workers...)

	for _, host := range allHosts {
		if err := p.resetNode(ctx, host); err != nil {
			p.emitEvent("warn", host.Address, "destroy", fmt.Sprintf("Failed to reset node: %v", err))
		}
	}

	return nil
}

// RemoveNode removes a node from the cluster
func (p *KubeadmProvisioner) RemoveNode(ctx context.Context, host HostSpec, kubeconfig []byte) error {
	// TODO: Drain node using client-go
	// kubectl drain <node> --ignore-daemonsets --delete-emptydir-data

	return p.resetNode(ctx, host)
}

// resetNode runs kubeadm reset on a node
func (p *KubeadmProvisioner) resetNode(ctx context.Context, host HostSpec) error {
	client, err := NewSSHClient(host)
	if err != nil {
		return err
	}
	defer client.Close()

	p.emitEvent("info", host.Address, "reset", "Running kubeadm reset")

	_, _, err = client.RunCommand(ctx, "kubeadm reset -f")
	if err != nil {
		return err
	}

	// Clean up
	_, _, _ = client.RunCommand(ctx, "rm -rf /etc/cni/net.d && rm -rf $HOME/.kube/config")

	return nil
}

// GenerateJoinToken generates a new join token
func (p *KubeadmProvisioner) GenerateJoinToken(ctx context.Context, kubeconfig []byte, controlPlane bool) (string, error) {
	// TODO: Use client-go or execute kubeadm token create
	return "", ErrNotImplemented
}

// Helper methods

func (p *KubeadmProvisioner) emitEvent(level, host, step, message string) {
	if p.eventCallback != nil {
		p.eventCallback(NewProvisionEvent(level, host, step, message))
	}
}

func (p *KubeadmProvisioner) extractJoinCommand(output string) string {
	// Extract "kubeadm join ..." from output
	lines := strings.Split(output, "\n")
	for i, line := range lines {
		if strings.Contains(line, "kubeadm join") {
			// Combine this line and possibly the next few lines
			joinCmd := strings.TrimSpace(line)
			for j := i + 1; j < len(lines) && j < i+5; j++ {
				nextLine := strings.TrimSpace(lines[j])
				if nextLine != "" && (strings.HasPrefix(nextLine, "--") || strings.HasPrefix(nextLine, "\\")) {
					joinCmd += " " + strings.TrimPrefix(nextLine, "\\")
				} else {
					break
				}
			}
			return strings.TrimSpace(joinCmd)
		}
	}
	return ""
}

func (p *KubeadmProvisioner) extractCertificateKey(output string) string {
	// Extract certificate key from output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "--certificate-key") {
			parts := strings.Split(line, "--certificate-key")
			if len(parts) > 1 {
				key := strings.TrimSpace(parts[1])
				return strings.Fields(key)[0]
			}
		}
	}
	return ""
}
