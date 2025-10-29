package provision

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHClient wraps an SSH connection to a remote host
type SSHClient struct {
	client *ssh.Client
	host   HostSpec
}

// NewSSHClient creates a new SSH client connection
func NewSSHClient(host HostSpec) (*SSHClient, error) {
	// Read SSH key
	var key []byte
	var err error

	if host.SSHKey != "" {
		key = []byte(host.SSHKey)
	} else if host.SSHKeyPath != "" {
		key, err = os.ReadFile(host.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key from %s: %w", host.SSHKeyPath, err)
		}
	} else {
		return nil, fmt.Errorf("no SSH key provided for host %s", host.Address)
	}

	// Parse SSH private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH key: %w", err)
	}

	// Configure SSH client
	config := &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Add proper host key verification
		Timeout:         30 * time.Second,
	}

	// Connect to the remote host
	addr := fmt.Sprintf("%s:%d", host.Address, host.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	return &SSHClient{
		client: client,
		host:   host,
	}, nil
}

// Close closes the SSH connection
func (c *SSHClient) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// RunCommand executes a command on the remote host and returns stdout, stderr, and error
func (c *SSHClient) RunCommand(ctx context.Context, command string) (stdout, stderr string, err error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdoutBuf, stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf

	// Run command with context
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return stdoutBuf.String(), stderrBuf.String(), ctx.Err()
	case err := <-done:
		return stdoutBuf.String(), stderrBuf.String(), err
	}
}

// RunCommandWithCallback executes a command and streams output via callback
func (c *SSHClient) RunCommandWithCallback(ctx context.Context, command string, callback func(line string)) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// Start command
	if err := session.Start(command); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Stream output
	done := make(chan error, 1)
	go func() {
		multiReader := io.MultiReader(stdout, stderr)
		buf := make([]byte, 1024)
		for {
			n, err := multiReader.Read(buf)
			if n > 0 && callback != nil {
				callback(string(buf[:n]))
			}
			if err != nil {
				break
			}
		}
		done <- session.Wait()
	}()

	// Wait for completion or cancellation
	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// UploadFile uploads a file to the remote host using SCP-like logic
func (c *SSHClient) UploadFile(ctx context.Context, localPath, remotePath string) error {
	// Read local file
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Create remote file using a simple approach (write via echo or heredoc)
	// For production, consider using proper SCP or SFTP
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdin = bytes.NewReader(content)
	command := fmt.Sprintf("cat > %s", remotePath)

	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// DownloadFile downloads a file from the remote host
func (c *SSHClient) DownloadFile(ctx context.Context, remotePath, localPath string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf

	done := make(chan error, 1)
	go func() {
		done <- session.Run(fmt.Sprintf("cat %s", remotePath))
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return err
		}
		return os.WriteFile(localPath, stdoutBuf.Bytes(), 0644)
	}
}

// TestConnection tests if the SSH connection is working
func (c *SSHClient) TestConnection(ctx context.Context) error {
	_, _, err := c.RunCommand(ctx, "echo 'test'")
	return err
}

// GetHostInfo retrieves basic host information
func (c *SSHClient) GetHostInfo(ctx context.Context) (map[string]string, error) {
	info := make(map[string]string)

	// Get hostname
	stdout, _, err := c.RunCommand(ctx, "hostname")
	if err == nil {
		info["hostname"] = stdout
	}

	// Get OS info
	stdout, _, err = c.RunCommand(ctx, "cat /etc/os-release | grep PRETTY_NAME | cut -d'=' -f2 | tr -d '\"'")
	if err == nil {
		info["os"] = stdout
	}

	// Get kernel version
	stdout, _, err = c.RunCommand(ctx, "uname -r")
	if err == nil {
		info["kernel"] = stdout
	}

	// Check if swap is enabled
	stdout, _, err = c.RunCommand(ctx, "swapon --show")
	if err == nil && stdout != "" {
		info["swap_enabled"] = "true"
	} else {
		info["swap_enabled"] = "false"
	}

	return info, nil
}
