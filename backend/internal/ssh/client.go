package ssh

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

const defaultTimeout = 30 * time.Second

// Client wraps an SSH connection and provides methods for remote command execution.
type Client struct {
	client *ssh.Client
}

// NewClient establishes an SSH connection using the provided configuration.
// PrivateKey authentication takes priority over Password authentication.
func NewClient(cfg SSHConfig) (*Client, error) {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	port := cfg.Port
	if port == 0 {
		port = 22
	}

	user := cfg.User
	if user == "" {
		user = "root"
	}

	var authMethods []ssh.AuthMethod
	if cfg.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(cfg.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	} else {
		return nil, fmt.Errorf("no authentication method provided")
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         timeout,
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial tcp %s: %w", addr, err)
	}

	// Enable TCP keepalive to prevent connection drops during long transfers
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetKeepAlive(true)
		_ = tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("ssh handshake %s: %w", addr, err)
	}

	client := ssh.NewClient(sshConn, chans, reqs)

	// Start SSH-level keepalive (sends request every 30s)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_, _, err := client.SendRequest("keepalive@prometheus", true, nil)
			if err != nil {
				return
			}
		}
	}()

	return &Client{client: client}, nil
}

// RunCommand executes a command on the remote host and returns the result.
func (c *Client) RunCommand(ctx context.Context, cmd string) (*CommandResult, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return nil, ctx.Err()
	case err := <-done:
		result := &CommandResult{
			Stdout:   stdout.String(),
			Stderr:   stderr.String(),
			ExitCode: 0,
		}
		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				return nil, fmt.Errorf("run command: %w", err)
			}
		}
		return result, nil
	}
}

// CopyFrom reads the contents of a remote file.
func (c *Client) CopyFrom(ctx context.Context, remotePath string) ([]byte, error) {
	result, err := c.RunCommand(ctx, fmt.Sprintf("cat %q", remotePath))
	if err != nil {
		return nil, fmt.Errorf("copy from %s: %w", remotePath, err)
	}
	if result.ExitCode != 0 {
		return nil, fmt.Errorf("copy from %s: %s", remotePath, result.Stderr)
	}
	return []byte(result.Stdout), nil
}

// CopyTo writes data to a remote file via stdin pipe.
func (c *Client) CopyTo(ctx context.Context, data []byte, remotePath string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer session.Close()

	session.Stdin = bytes.NewReader(data)

	var stderr bytes.Buffer
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() {
		done <- session.Run(fmt.Sprintf("cat > %q", remotePath))
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return ctx.Err()
	case err := <-done:
		if err != nil {
			return fmt.Errorf("copy to %s: %s", remotePath, stderr.String())
		}
		return nil
	}
}

// Close closes the underlying SSH connection.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
