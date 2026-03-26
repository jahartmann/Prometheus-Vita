package ssh

import "time"

// CommandResult holds the output of an SSH command execution.
type CommandResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// SSHConfig holds the configuration for an SSH connection.
type SSHConfig struct {
	Host       string
	Port       int
	User       string
	PrivateKey string
	Password   string
	Timeout    time.Duration
	// HostKey is the expected SSH host key in authorized_keys format (base64-encoded).
	// If set, the connection verifies the server's key matches exactly.
	// If empty, any key is accepted (TOFU — caller should persist the key afterward).
	HostKey string
}

// PoolConfig holds the configuration for the SSH connection pool.
type PoolConfig struct {
	MaxConnections int
	IdleTimeout    time.Duration
}
