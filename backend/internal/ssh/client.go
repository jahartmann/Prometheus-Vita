package ssh

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ShellQuote wraps a string in single quotes for safe use in POSIX shell commands.
// Single quotes prevent ALL shell interpretation including $(), ``, etc.
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

const defaultTimeout = 60 * time.Second

// Client wraps an SSH connection and provides methods for remote command execution.
type Client struct {
	client  *ssh.Client
	done    chan struct{}
	hostKey string // server host key in authorized_keys format, captured during handshake
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

	// Build host key verification callback (TOFU pattern).
	var capturedHostKey string
	hostKeyCallback, err := buildHostKeyCallback(cfg.Host, cfg.HostKey, &capturedHostKey)
	if err != nil {
		return nil, fmt.Errorf("build host key callback: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
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

	c := &Client{
		client:  client,
		done:    make(chan struct{}),
		hostKey: capturedHostKey,
	}

	// Start SSH-level keepalive (sends request every 30s)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-c.done:
				return
			case <-ticker.C:
				_, _, err := client.SendRequest("keepalive@prometheus", true, nil)
				if err != nil {
					return
				}
			}
		}
	}()

	return c, nil
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
	result, err := c.RunCommand(ctx, "cat "+ShellQuote(remotePath))
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
		tmpPath := remotePath + ".prometheus-tmp"
		done <- session.Run("cat > " + ShellQuote(tmpPath) + " && mv " + ShellQuote(tmpPath) + " " + ShellQuote(remotePath))
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

// GetHostKey returns the remote server's SSH host key in authorized_keys format
// (e.g. "ssh-ed25519 AAAA..."). Useful for persisting the key after a TOFU connection.
func (c *Client) GetHostKey() string {
	return c.hostKey
}

// buildHostKeyCallback creates an ssh.HostKeyCallback implementing Trust-On-First-Use (TOFU).
// If expectedKey is non-empty, it parses the key and verifies the server presents it exactly.
// If expectedKey is empty, any key is accepted and a warning is logged. In both cases, the
// server's key (in authorized_keys format) is written to capturedKey for later persistence.
func buildHostKeyCallback(host, expectedKey string, capturedKey *string) (ssh.HostKeyCallback, error) {
	if expectedKey != "" {
		// Parse the trusted host key and verify against it.
		trustedKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(expectedKey))
		if err != nil {
			return nil, fmt.Errorf("parse expected host key: %w", err)
		}

		return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			*capturedKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
			if key.Type() != trustedKey.Type() || !bytes.Equal(key.Marshal(), trustedKey.Marshal()) {
				return fmt.Errorf(
					"ssh host key mismatch for %s: expected %s, got %s",
					host,
					ssh.FingerprintSHA256(trustedKey),
					ssh.FingerprintSHA256(key),
				)
			}
			return nil
		}, nil
	}

	// TOFU: no known key — accept and log a warning.
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		*capturedKey = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(key)))
		slog.Warn("accepting unverified SSH host key (TOFU)",
			slog.String("host", host),
			slog.String("fingerprint", ssh.FingerprintSHA256(key)),
		)
		return nil
	}, nil
}

// Close closes the underlying SSH connection and stops the keepalive goroutine.
func (c *Client) Close() error {
	select {
	case <-c.done:
		// Already closed
	default:
		close(c.done)
	}
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
